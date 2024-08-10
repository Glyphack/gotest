package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	Output  []string
	Elapsed string
	State   string
}

type PackageResult struct {
	tests   []string
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

	// package + test name to result map
	results := make(map[string]*TestResult)
	packages := make(map[string]*PackageResult)

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
				packages[event.Package] = &PackageResult{}
			case "pass":
				packages[event.Package].Elapsed = event.Elapsed
			case "fail":
				packages[event.Package].Elapsed = event.Elapsed
			}
			continue
		}

		testKey := event.Package + "." + event.Test
		switch event.Action {
		case "run":
			results[testKey] = &TestResult{
				Name:    event.Test,
				Output:  []string{},
				Elapsed: "",
			}
			packages[event.Package].tests = append(packages[event.Package].tests, event.Test)
		case "output":
			results[testKey].Output = append(results[testKey].Output, event.Output)
		case "fail":
			results[testKey].State = "fail"
		case "pass":
			results[testKey].State = "pass"
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

	failedTests := []*TestResult{}
	for pkg, pkgResult := range packages {
		oldFailedTestCount := len(failedTests)
		fmt.Println("--------------------------")
		successCount := 0
		for pkgTest := range pkgResult.tests {
			testResult := results[pkg+"."+pkgResult.tests[pkgTest]]
			if testResult.State == "pass" {
				successCount++
			} else {
				failedTests = append(failedTests, testResult)
			}
		}
		packageFailedTests := len(failedTests) - oldFailedTestCount
		fmt.Printf("Package: %s, Passed tests: %d, Failed tests: %d\n", pkg, successCount, packageFailedTests)

		if len(failedTests) > 0 {
			fmt.Println("Failed:")
			for _, testResult := range failedTests {
				fmt.Printf("===========%s===========\n", testResult.Name)
				for _, output := range testResult.Output {
					fmt.Println(output)
				}
				fmt.Printf("=========================================\n")
			}
		}
	}
}
