package autosteer

import (
	"testing"
	"time"
)

func TestTraceStore_PersistsCreditSplit(t *testing.T) { // [REQ:EM-P1-006]
	pg, cleanup := SetupTestDatabase(t)
	if pg == nil {
		return // Docker unavailable; SetupTestDatabase skipped.
	}
	defer cleanup()

	store := NewTraceStore(pg.db)
	taskID := "trace-task"

	entry := DecisionTraceEntry{
		Iteration:          1,
		Timestamp:          time.Now(),
		DimensionScores:    map[string]float64{"standards": 8},
		HeaviestDimension:  "standards",
		ChosenSkill:        "refactor",
		Rationale:          "heaviest open cluster",
		Fingerprint:        "fp-1",
		ScoreBefore:        8,
		DTVVerdict:         "yellow",
		DTVPrior:           0.42,
		DTVExcluded:        map[string]string{"lint-fix": "dtv:red"},
		DTVGateOverride:    true,
		DTVDegraded:        false,
		GateDegradedCause:  GateCauseAllRed,
		PredictedReduction: 3.5,
	}
	if err := store.Append(taskID, "demo", "demo", entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Fill the realized outcome including the per-dimension findings flow.
	entry.ScoreAfter = 4
	entry.RealizedDelta = 4
	entry.TokensUsed = 2500
	entry.ClosedByDimension = map[string]int{"standards": 1}
	entry.IntroducedByDimension = map[string]int{"tests": 1}
	if err := store.SetRealized(taskID, entry); err != nil {
		t.Fatalf("SetRealized: %v", err)
	}

	got, err := store.GetTrace(taskID)
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 trace row, got %d", len(got))
	}
	e := got[0]
	if e.TokensUsed != 2500 {
		t.Fatalf("tokens_used not persisted: %d", e.TokensUsed)
	}
	if e.ClosedByDimension["standards"] != 1 {
		t.Fatalf("closed_by_dimension not persisted: %+v", e.ClosedByDimension)
	}
	if e.IntroducedByDimension["tests"] != 1 {
		t.Fatalf("introduced_by_dimension not persisted: %+v", e.IntroducedByDimension)
	}
	if e.RealizedDelta != 4 || e.ScoreAfter != 4 {
		t.Fatalf("realized fields not persisted: %+v", e)
	}
	// DTV provenance + P2/P4 columns survive the durable round-trip.
	if e.DTVVerdict != "yellow" || e.DTVPrior != 0.42 {
		t.Fatalf("DTV verdict/prior not persisted: %q %v", e.DTVVerdict, e.DTVPrior)
	}
	if e.DTVExcluded["lint-fix"] != "dtv:red" {
		t.Fatalf("dtv_excluded not persisted: %+v", e.DTVExcluded)
	}
	if !e.DTVGateOverride || e.GateDegradedCause != GateCauseAllRed {
		t.Fatalf("degraded-gate fields not persisted: override=%v cause=%q", e.DTVGateOverride, e.GateDegradedCause)
	}
	if e.PredictedReduction != 3.5 {
		t.Fatalf("predicted_reduction not persisted: %v", e.PredictedReduction)
	}
}

// TestSkillPick_PredictedReduction pins the EM-P4 forward estimate math.
func TestSkillPick_PredictedReduction(t *testing.T) {
	pick := skillPick{
		method: "effectiveness",
		candidates: []candidate{
			{id: "a", efficacy: 2.0, avgTokens: 1500},
			{id: "b", efficacy: 5.0, avgTokens: 0}, // cold tokens ⇒ default estimate
		},
	}
	// a: 2.0 · 1500/1000 · weight 2.0 = 6.0
	if got := pick.predictedReduction("a", 2.0); got != 6.0 {
		t.Fatalf("predicted(a) = %v, want 6.0", got)
	}
	// b: cold tokens fall back to defaultEstimatedRunTokens; weight 0 ⇒ 1.0
	want := 5.0 * (float64(defaultEstimatedRunTokens) / 1000.0) * 1.0
	if got := pick.predictedReduction("b", 0); got != want {
		t.Fatalf("predicted(b) = %v, want %v", got, want)
	}
	// Greedy / unknown candidate ⇒ no estimate.
	if got := (skillPick{method: "greedy"}).predictedReduction("a", 1.0); got != 0 {
		t.Fatalf("greedy predicted = %v, want 0", got)
	}
}
