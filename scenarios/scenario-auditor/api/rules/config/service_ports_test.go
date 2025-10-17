//go:build ruletests
// +build ruletests

package config

import "testing"

func TestServicePortDocCases(t *testing.T) {
	runDocTestsViolations(t, "service_ports.go", ".vrooli/service.json", CheckServicePortConfiguration)
}
