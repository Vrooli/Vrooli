//go:build ruletests
// +build ruletests

package testrules

import (
	"testing"

	"scenario-auditor/internal/testutil"
)

func TestExitCodeDocCases(t *testing.T) {
	testutil.RunDocViolationCases(t, "exit_code_test.go", ".vrooli/service.json", CheckExitCode)
}
