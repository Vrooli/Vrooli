package main

import (
	"testing"

	"test-genie/cli/execute"
	"test-genie/cli/internal/phases"
)

func TestNormalizePhaseSelection(t *testing.T) {
	t.Run("dedupes and normalizes", func(t *testing.T) {
		result, err := phases.NormalizeSelection([]string{"Unit", "unit", "integration"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 || result[0] != "unit" || result[1] != "integration" {
			t.Fatalf("unexpected phases: %#v", result)
		}
	})

	t.Run("accepts standards phase", func(t *testing.T) {
		result, err := phases.NormalizeSelection([]string{"standards"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "standards" {
			t.Fatalf("unexpected phases: %#v", result)
		}
	})

	t.Run("rejects unknown", func(t *testing.T) {
		if _, err := phases.NormalizeSelection([]string{"unknown"}); err == nil {
			t.Fatalf("expected error for unknown phase")
		}
	})
}

func TestParseExecuteArgsMergesPhases(t *testing.T) {
	args := []string{"demo", "--phases", "structure,unit", "business"}
	out, err := execute.ParseArgs(args)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got := out.Phases; len(got) != 3 || got[0] != "structure" || got[1] != "unit" || got[2] != "business" {
		t.Fatalf("unexpected phases: %#v", got)
	}
}
