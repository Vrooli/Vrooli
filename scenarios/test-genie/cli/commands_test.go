package main

import "testing"

func TestNormalizePhaseSelection(t *testing.T) {
	t.Run("dedupes and normalizes", func(t *testing.T) {
		phases, err := normalizePhaseSelection([]string{"Unit", "unit", "integration"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(phases) != 2 || phases[0] != "unit" || phases[1] != "integration" {
			t.Fatalf("unexpected phases: %#v", phases)
		}
	})

	t.Run("rejects unknown", func(t *testing.T) {
		if _, err := normalizePhaseSelection([]string{"unknown"}); err == nil {
			t.Fatalf("expected error for unknown phase")
		}
	})
}

func TestParseExecuteArgsMergesPhases(t *testing.T) {
	args := []string{"demo", "--phases", "structure,unit", "business"}
	out, err := parseExecuteArgs(args)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got := out.Phases; len(got) != 3 || got[0] != "structure" || got[1] != "unit" || got[2] != "business" {
		t.Fatalf("unexpected phases: %#v", got)
	}
}
