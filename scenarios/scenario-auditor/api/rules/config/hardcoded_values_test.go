//go:build ruletests
// +build ruletests

package config

import "testing"

func TestHardcodedValuesDocCases(t *testing.T) {
	runDocTestsViolations(t, "hardcoded_values.go", "api/main.go", func(content []byte, path string) []Violation {
		return CheckHardcodedValues(content, path)
	})
}
