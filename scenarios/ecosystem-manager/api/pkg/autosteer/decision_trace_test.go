package autosteer

import (
	"testing"
	"time"
)

func TestTraceStore_PersistsCreditSplit(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	if pg == nil {
		return // Docker unavailable; SetupTestDatabase skipped.
	}
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
}
