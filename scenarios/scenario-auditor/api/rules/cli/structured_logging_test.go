//go:build ruletests
// +build ruletests

package cli

import "testing"

func TestStructuredLoggingDocCases(t *testing.T) {
	runDocTestsViolations(t, "structured_logging.go", "cli/main.go", CheckStructuredLogging)
}
