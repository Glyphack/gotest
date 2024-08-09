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
}

// TestResult holds aggregated results for each test
type TestResult struct {
	Passed []string
	Failed map[string]string
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

	results := make(map[string]*TestResult)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var event TestEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
			continue
		}

		if _, ok := results[event.Package]; !ok {
			results[event.Package] = &TestResult{
				Passed: make([]string, 0),
				Failed: make(map[string]string),
			}
		}

		switch event.Action {
		case "pass":
			results[event.Package].Passed = append(results[event.Package].Passed, event.Test)
		case "fail":
			results[event.Package].Failed[event.Test] += event.Output
		}
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Command finished with error: %v\n", err)
	}

	for pkg, result := range results {
		fmt.Printf("Package: %s, Passed tests: %d\n", pkg, len(result.Passed))
	}

	for pkg, result := range results {
		for test, output := range result.Failed {
			fmt.Printf("Failed in package %s, test %s: \nOutput: %s\n", pkg, test, output)
		}
	}
}
