//go:build ruletests
// +build ruletests

package cli

import "testing"

func TestCLIDiagnosticLoggingDocCases(t *testing.T) {
	runDocTestsViolations(t, "diagnostic_logging.go", "cli/main.go", CheckCLIDiagnosticLogging)
}
