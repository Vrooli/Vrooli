//go:build ruletests
// +build ruletests

package ui

import "testing"

func TestSecureTunnelDocCases(t *testing.T) {
	runDocTestsViolations(t, "secure_tunnel.go", "ui/server.js", CheckSecureTunnelSetup)
}
