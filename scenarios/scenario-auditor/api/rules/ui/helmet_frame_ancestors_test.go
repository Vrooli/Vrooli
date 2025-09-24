//go:build ruletests
// +build ruletests

package ui

import "testing"

func TestHelmetFrameAncestorsDocCases(t *testing.T) {
	runDocTestsViolations(t, "helmet_frame_ancestors.go", "ui/server.js", CheckHelmetFrameAncestors)
}
