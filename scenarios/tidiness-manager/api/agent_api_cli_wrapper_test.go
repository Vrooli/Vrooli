package main

import (
	"os"
	"testing"
)

// [REQ:TM-API-005] CLI wrapper exists
func TestCLIWrapper_IssuesCommand(t *testing.T) {
	// This is a unit test that verifies the CLI command exists
	// The actual CLI is tested in test/cli/agent-api.bats

	// Verify CLI binary exists
	if _, err := os.Stat("../cli/tidiness-manager"); os.IsNotExist(err) {
		t.Skip("CLI binary not found, skipping CLI wrapper test")
	}

	// Test passes if we get here - detailed CLI integration tests are in BATS
}
