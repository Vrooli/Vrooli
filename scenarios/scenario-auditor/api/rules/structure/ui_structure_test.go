//go:build ruletests
// +build ruletests

package structure

import (
	"testing"

	rules "scenario-auditor/rules"
)

func TestUIStructureDocCases(t *testing.T) {
	runDocTests(t, "ui_structure.go", "scenario", func(input string, path string, scenario string) ([]rules.Violation, error) {
		return CheckUIStructure([]byte(input), path, scenario)
	})
}
