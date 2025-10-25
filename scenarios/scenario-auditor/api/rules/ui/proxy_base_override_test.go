//go:build ruletests
// +build ruletests

package ui

import "testing"

func TestProxyBasePreservationDocCases(t *testing.T) {
	runDocTestsViolations(t, "proxy_base_override.go", "ui/src/App.tsx", CheckProxyBasePreservation)
}
