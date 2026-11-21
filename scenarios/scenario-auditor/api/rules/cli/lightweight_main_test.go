//go:build ruletests
// +build ruletests

package cli

import (
	"testing"

	"scenario-auditor/rules/testkit"
)

func TestLightweightMainDocCases(t *testing.T) {
	testkit.RunDocViolationCases(t, "lightweight_main.go", "cli/main.go", CheckLightweightMain)
}
