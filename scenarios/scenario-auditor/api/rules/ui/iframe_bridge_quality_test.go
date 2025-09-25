//go:build ruletests
// +build ruletests

package ui

import "testing"

func TestIframeBridgeDocCases(t *testing.T) {
	runDocTestsViolations(t, "iframe_bridge_quality.go", "ui/main.tsx", func(input []byte, path string) []Violation {
		return CheckIframeBridgeQuality(input, path)
	})
}
