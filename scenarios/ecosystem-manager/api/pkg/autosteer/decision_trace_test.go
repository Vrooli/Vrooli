package autosteer

import (
	"testing"
	"time"
)

func TestTraceStore_PersistsSlimTrace(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	defer cleanup()

	store := NewTraceStore(pg.db)
	taskID := "trace-task"

	entry := DecisionTraceEntry{
		Iteration:         1,
		Timestamp:         time.Now(),
		DimensionScores:   map[string]float64{"standards": 8},
		HeaviestDimension: "standards",
		ChosenSkill:       "refactor",
		Rationale:         "heaviest open cluster",
		Fingerprint:       "fp-1",
		ScoreBefore:       8,
	}
	if err := store.Append(taskID, "demo", "demo", entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Fill the realized outcome, including the anti-gaming verdict.
	entry.ScoreAfter = 4
	entry.RealizedDelta = 4
	entry.GamingCause = "gamed:test-weakening"
	if err := store.SetRealized(taskID, entry); err != nil {
		t.Fatalf("SetRealized: %v", err)
	}
	if err := store.SetHalt(taskID, 1, StopObjectiveMet); err != nil {
		t.Fatalf("SetHalt: %v", err)
	}

	got, err := store.GetTrace(taskID)
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 trace row, got %d", len(got))
	}
	e := got[0]
	if e.ChosenSkill != "refactor" || e.HeaviestDimension != "standards" {
		t.Fatalf("identity columns not persisted: %+v", e)
	}
	if e.DimensionScores["standards"] != 8 {
		t.Fatalf("dimension_scores not persisted: %+v", e.DimensionScores)
	}
	if e.RealizedDelta != 4 || e.ScoreAfter != 4 {
		t.Fatalf("realized fields not persisted: %+v", e)
	}
	if e.GamingCause != "gamed:test-weakening" {
		t.Fatalf("gaming_cause not persisted: %q", e.GamingCause)
	}
	if e.HaltReason != StopObjectiveMet {
		t.Fatalf("halt_reason not persisted: %q", e.HaltReason)
	}
}
