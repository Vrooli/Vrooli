//go:build ruletests
// +build ruletests

package config

import (
	"strings"
	"scenario-auditor/rules/testkit"
	"testing"
)

func TestMakefileStructureDebug(t *testing.T) {
	cases := testkit.LoadDocCases(t, "makefile_structure.go")
	
	if len(cases) == 0 {
		t.Fatal("No test cases found")
	}
	
	firstCase := cases[0]
	t.Logf("First test case ID: %s", firstCase.ID)
	t.Logf("Input length: %d characters", len(firstCase.Input))
	t.Logf("Input line count: %d lines", len(strings.Split(firstCase.Input, "\n")))
	
	lines := strings.Split(firstCase.Input, "\n")
	if len(lines) > 0 {
		t.Logf("Last 3 lines:")
		start := len(lines) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(lines); i++ {
			t.Logf("  Line %d: %q (len=%d)", i+1, lines[i], len(lines[i]))
		}
	}
	
	violations, err := CheckMakefileStructure(firstCase.Input, "test.mk")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	
	t.Logf("Violations found: %d", len(violations))
	for i, v := range violations {
		t.Logf("  %d. Line %d: %s", i+1, v.Line, v.Message)
	}
}
