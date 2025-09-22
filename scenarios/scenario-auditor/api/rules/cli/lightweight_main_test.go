//go:build ruletests
// +build ruletests

package cli

import "testing"

func TestLightweightMainDocCases(t *testing.T) {
	runDocTestsViolations(t, "lightweight_main.go", "cli/main.go", CheckLightweightMain)
}
