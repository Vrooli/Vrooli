//go:build ruletests
// +build ruletests

package config

import "testing"

func TestMakefileStructureDocCases(t *testing.T) {
	runDocTestsCustom(t, "makefile_structure.go", "Makefile", func(input string, path string) ([]MakefileStructureViolation, error) {
		return CheckMakefileStructure(input, path)
	}, func(v MakefileStructureViolation) string {
		return v.Message
	})
}
