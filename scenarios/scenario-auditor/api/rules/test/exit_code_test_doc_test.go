//go:build ruletests
// +build ruletests

package testrules

import (
	"testing"

	"scenario-auditor/rules/testkit"
)

func TestExitCodeDocCases(t *testing.T) {
	testkit.RunDocViolationCases(t, "exit_code_test.go", ".vrooli/service.json", CheckExitCode)
}
