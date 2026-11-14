package focus

import "testing"

func TestRankPutsIntegrityFirstAndStatesWhy(t *testing.T) {
	got := Rank([]Finding{{ID: "condition", Source: "condition", Stage: StageAvailability, Severity: 9}, {ID: "trust", Source: "condition", Stage: StageIntegrity, Severity: 1}})
	if len(got) != 2 || got[0].ID != "trust" {
		t.Fatalf("rank = %+v", got)
	}
	if got[0].RankExplanation == "" {
		t.Fatal("missing ranking rationale")
	}
}

func TestEfficacyUnmeasurableIsNotPass(t *testing.T) {
	if got := EvaluateEfficacy("IN_BAND", "IN_BAND", false, true); got != EfficacyUnmeasurable {
		t.Fatalf("got %q", got)
	}
	if got := EvaluateEfficacy("IN_BAND", "OUT_OF_BAND", true, true); got != EfficacyDidNotMove {
		t.Fatalf("got %q", got)
	}
}
