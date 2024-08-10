package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

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
	cmd := exec.Command("go", "test", "./...", "-json")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stdout pipe: %v\n", err)
		os.Exit(1)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting command: %v\n", err)
		os.Exit(1)
	}

	packageText, _ := pterm.DefaultArea.Start()
	testText, _ := pterm.DefaultArea.Start()

	// package + test name to result map
	packages := make(map[string]*PackageResult)
	results := make(map[string]*TestResult)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var event TestEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
			continue
		}

		// Package information
		if event.Test == "" {
			switch event.Action {
			case "start":
				packageText.Update(pterm.Sprintf("%s", event.Package))
				packages[event.Package] = &PackageResult{}
			case "pass", "fail":
				packageText.Clear()
				pkg := packages[event.Package]
				pkg.Elapsed = event.Elapsed

				failedTests := []*TestResult{}
				passedCount := 0
				for _, testResult := range pkg.tests {
					if testResult.State == "fail" {
						failedTests = append(failedTests, testResult)
					} else {
						passedCount++
					}
				}
				pterm.DefaultBasicText.Println(pterm.Sprintf("%s: Passed: %d, Failed: %d", event.Package, passedCount, len(failedTests)))
			}
			continue
		}

		testKey := event.Package + "." + event.Test
		switch event.Action {
		case "run":
			testText.Clear()
			testText.Update(pterm.Sprintf("%s\n%s\n", event.Package, event.Test))
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
		case "skip", "pause", "cont":
			continue
		default:
			panic("Unknown action: " + event.Action)
		}
	}

	if err := cmd.Wait(); err != nil {
		exitErr := err.(*exec.ExitError)
		if exitErr.ExitCode() == 1 {
			fmt.Fprintln(os.Stderr, "Some tests failed")
		} else {
			fmt.Fprintf(os.Stderr, "Command finished with error: %v\n", err)
		}
	}
	packageText.Stop()

	PrintFailedPackageTests(packages)
}

func PrintFailedPackageTests(packages map[string]*PackageResult) {
	fmt.Println()
	failedTests := []*TestResult{}
	for _, pkgResult := range packages {
		successCount := 0
		for _, pkgTest := range pkgResult.tests {
			if pkgTest.State == "pass" {
				successCount++
			} else {
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
