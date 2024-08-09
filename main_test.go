package main

import (
	"fmt"
	"log"
	"testing"
)

func TestExampleFailing(t *testing.T) {
	fmt.Println("This is a standard fmt.Println output.")

	log.Println("This is a log output using log.Println.")

	if 1 != 2 {
		t.Errorf("Failed: 1 should be equal to 1")
	}
}

func TestExample(t *testing.T) {
	fmt.Println("This is a standard fmt.Println output.")

	log.Println("This is a log output using log.Println.")

	if 1 != 1 {
		t.Errorf("Failed: 1 should be equal to 1")
	}
}
