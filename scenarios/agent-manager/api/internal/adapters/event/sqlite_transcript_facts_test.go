package event_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// The facts behind the transcript step-back triggers come from narrow reads:
// turns from turn-ID message events (closed by their terminal record, the
// newest open turn running to the latest event), compactions with their goal
// retention, and spawn_agent calls split by fork_turns.
func TestSQLiteStoreTranscriptFactsReadTurnsCompactionsAndSpawns(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID, quiet := uuid.New(), uuid.New()
	base := time.Date(2026, 11, 6, 20, 3, 48, 310_000_000, time.UTC)
	first, second := "01a1b275-d1d6-7960-9492-31d87173da33", "01a1b30a-3c72-7d60-89b0-edd5142c4e61"
	stamp := func(e *domain.RunEvent, minutes int) *domain.RunEvent {
		e.Timestamp = base.Add(time.Duration(minutes) * time.Minute)
		return e
	}
	message := func(turn string, terminal bool, minutes int) *domain.RunEvent {
		return stamp(domain.NewProviderMessageEvent(runID, "assistant", "working", domain.MessageEventData{TurnID: turn, Terminal: terminal}), minutes)
	}
	compaction := domain.NewCompactionEvent(runID, "", "auto", "", 0, 0, 0, "")
	compaction.Data.(*domain.CompactionEventData).GoalObjective = domain.CompactionGoalOmitted
	noStance := 0
	compaction.Data.(*domain.CompactionEventData).OperatorMessagesKept = &noStance
	events := []*domain.RunEvent{
		message(first, false, 1),
		stamp(domain.NewToolCallEvent(runID, "spawn_agent", "c1", map[string]interface{}{"fork_turns": "none"}), 2),
		stamp(domain.NewToolCallEvent(runID, "spawn_agent", "c2", map[string]interface{}{"message": "inherits by default"}), 3),
		stamp(domain.NewToolCallEvent(runID, "exec", "c3", map[string]interface{}{"input": "spawn_agent"}), 4),
		stamp(compaction, 60),
		message(first, true, 155),
		message(second, false, 162),
		stamp(domain.NewToolCallEvent(runID, "exec", "c4", nil), 200),
	}
	if err := store.Append(ctx, runID, events...); err != nil {
		t.Fatal(err)
	}
	facts, err := store.TranscriptFacts(ctx, []uuid.UUID{runID, quiet})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := facts[quiet]; ok {
		t.Fatal("a run without stored events must be absent, not empty")
	}
	got := facts[runID]
	if len(got.Turns) != 2 {
		t.Fatalf("turns = %+v, want 2", got.Turns)
	}
	closed, open := got.Turns[0], got.Turns[1]
	if closed.ID != first || !closed.Start.Equal(base) || !closed.Ended || !closed.End.Equal(base.Add(155*time.Minute)) {
		t.Fatalf("closed turn = %+v, want UUIDv7 start %v and terminal end", closed, base)
	}
	if open.ID != second || open.Ended || !open.End.Equal(base.Add(200*time.Minute)) {
		t.Fatalf("open turn = %+v, want it to run to the latest event", open)
	}
	if len(got.Compactions) != 1 || got.Compactions[0].GoalObjective != domain.CompactionGoalOmitted || !got.Compactions[0].At.Equal(base.Add(time.Hour)) ||
		got.Compactions[0].OperatorMessagesKept == nil || *got.Compactions[0].OperatorMessagesKept != 0 {
		t.Fatalf("compactions = %+v", got.Compactions)
	}
	if got.Spawns != 2 || got.SpawnsWithHistory != 1 {
		t.Fatalf("spawns = %d (inherited %d), want 2 (1)", got.Spawns, got.SpawnsWithHistory)
	}
}
