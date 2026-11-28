//go:build ruletests
// +build ruletests

package cli

import (
	"testing"

	"scenario-auditor/internal/testutil"
)

func TestLightweightMainDocCases(t *testing.T) {
	testutil.RunDocViolationCases(t, "lightweight_main.go", "cli/main.go", CheckLightweightMain)
}
