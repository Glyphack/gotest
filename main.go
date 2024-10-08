package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pterm/pterm"
)

// TestEvent represents a test event in JSON output from 'go test -json'
type TestEvent struct {
	Time    string
	Action  string
	Package string
	Test    string
	Output  string
	Elapsed float64
}

// TestResult holds aggregated results for each test
type TestResult struct {
	Name    string
	Package string
	Output  []string
	Elapsed string
	State   string
}

type PackageResult struct {
	tests   []*TestResult
	Elapsed float64
}

func main() {
	var args []string

	if len(os.Args) > 1 {
		args = os.Args[1:]
	} else {
		args = []string{"./..."}
	}
	args = append([]string{"test", "-json"}, args...)

	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stdout pipe: %v\n", err)
		os.Exit(1)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting command: %v\n", err)
		os.Exit(1)
	}

	testResultArea, err := pterm.DefaultArea.Start()

	packages := make(map[string]*PackageResult)
	results := make(map[string]*TestResult)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var event TestEvent
		read := scanner.Bytes()
		if err := json.Unmarshal(read, &event); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
			continue
		}

		// Package information
		if event.Test == "" {
			switch event.Action {
			case "start":
				packages[event.Package] = &PackageResult{}
			case "pass", "fail":
				pkg := packages[event.Package]
				pkg.Elapsed = event.Elapsed

				failedCount := 0
				passedCount := 0
				skippedCount := 0
				for _, testResult := range pkg.tests {
					if testResult.State == "fail" {
						failedCount++
					} else if testResult.State == "skip" {
						skippedCount++
					} else {
						passedCount++
					}
				}
				if passedCount == 0 && failedCount == 0 {
					continue
				}
				text := pterm.Sprintf(
					"%s %s",
					event.Package,
					pterm.Green(pterm.Sprintf("Passed: %d", passedCount)),
				)

				if failedCount > 0 {
					text = pterm.Sprintf("%s %s",
						text,
						pterm.Red(pterm.Sprintf("Failed: %d", failedCount)),
					)
				}
				if skippedCount > 0 {
					text = pterm.Sprintf("%s %s",
						text,
						pterm.Yellow(pterm.Sprintf("Skipped: %d", skippedCount)),
					)
				}
				text = testResultArea.GetContent() + "\n" + text
				testResultArea.Clear()
				testResultArea.Update(text)
			case "output":
				if strings.Contains(event.Output, "build failed") {
					fmt.Println("Build failed for package " + event.Package)
					continue
				}
			}
			continue
		}

		testKey := event.Package + "." + event.Test
		switch event.Action {
		case "run":
			results[testKey] = &TestResult{
				Package: event.Package,
				Name:    event.Test,
				Output:  []string{},
				Elapsed: "",
			}
			packages[event.Package].tests = append(packages[event.Package].tests, results[testKey])
		case "output":
			results[testKey].Output = append(results[testKey].Output, event.Output)
		case "fail":
			results[testKey].State = "fail"
		case "pass":
			results[testKey].State = "pass"
		case "skip":
			results[testKey].State = "skip"
		case "pause", "cont":
			continue
		default:
			panic("Unknown action: " + event.Action)
		}
	}

	exitCode := 0
	if err := cmd.Wait(); err != nil {
		exitErr := err.(*exec.ExitError)
		exitCode = exitErr.ExitCode()
		if exitErr.ExitCode() != 1 {
			fmt.Fprintf(os.Stderr, "Command finished with error: %v\n", err)
		}
	}
	PrintFailedPackageTests(packages)

	os.Exit(exitCode)
}

func PrintFailedPackageTests(packages map[string]*PackageResult) {
	failedTests := []*TestResult{}
	for _, pkgResult := range packages {
		for _, pkgTest := range pkgResult.tests {
			if pkgTest.State == "fail" {
				failedTests = append(failedTests, pkgTest)
			}
		}

		if len(failedTests) > 0 {
			pterm.DefaultCenter.Printf(pterm.Bold.Sprintf(pterm.Red("Failures")))
			for _, testResult := range failedTests {
				pterm.DefaultSection.Println(testResult.Name)
				for _, output := range testResult.Output {
					fmt.Println(output)
				}
			}
		}
	}
}
