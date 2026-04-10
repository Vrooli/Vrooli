//go:build ruletests
// +build ruletests

package ui

import "testing"

func TestFocusVisibleStylesDocCases(t *testing.T) {
	runDocTestsViolations(t, "focus_visible_styles.go", "ui/src/components/button.tsx", CheckFocusVisibleStyles)
}
