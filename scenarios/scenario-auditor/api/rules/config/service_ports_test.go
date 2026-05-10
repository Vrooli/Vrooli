//go:build ruletests
// +build ruletests

package config

import "testing"

func TestServicePortDocCases(t *testing.T) {
	runDocTestsViolations(t, "service_ports.go", ".vrooli/service.json", func(content []byte, path string) []Violation {
		return CheckServicePortConfiguration(content, path)
	})
}
