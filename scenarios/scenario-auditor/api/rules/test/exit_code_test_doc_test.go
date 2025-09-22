//go:build ruletests
// +build ruletests

package testrules

import "testing"

func TestExitCodeDocCases(t *testing.T) {
	runDocTestsViolations(t, "exit_code_test.go", "test/run-tests.sh", CheckExitCode)
}
