package main

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

// Test-only helpers moved from production files by the private-function debt lens.
// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Redirect log output to discard during tests (unless verbose)
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(ioutil.Discard)
		return func() {
			log.SetOutput(os.Stdout)
		}
	}
	return func() {}
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "metareasoning-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}
