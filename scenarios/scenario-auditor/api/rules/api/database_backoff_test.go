//go:build ruletests
// +build ruletests

package api

import "testing"

func TestDatabaseBackoffDocCases(t *testing.T) {
	runDocTestsViolations(t, "database_backoff.go", "api/main.go", CheckDatabaseBackoff)
}
