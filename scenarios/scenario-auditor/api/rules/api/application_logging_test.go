//go:build ruletests
// +build ruletests

package api

import "testing"

func TestAPIApplicationLoggingDocCases(t *testing.T) {
	runDocTestsViolations(t, "application_logging.go", "api/handlers.go", CheckAPIApplicationLogging)
}
