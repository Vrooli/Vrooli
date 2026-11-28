package main

import "os"

// setupTestLogger initializes a test logger with cleanup
func setupTestLogger() func() {
	// Disable logging during tests to reduce noise
	return func() {}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
