package event_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// A parser resumed mid-file can derive a different payload for a line already
// stored. A transcript event keeps its first record; any other event is refused.
func TestReplayKeepsStoredTranscriptEvidence(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID, id := uuid.New(), uuid.NewSHA1(uuid.NameSpaceOID, []byte("line"))
	make := func(input int, keep bool) *domain.RunEvent {
		return &domain.RunEvent{ID: id, RunID: runID, EventType: domain.EventTypeMetric, Timestamp: time.Now().UTC(), KeepStoredOnReplay: keep, Data: &domain.UsageEventData{PayloadKind: domain.PayloadKindUsage, InputTokens: input}}
	}
	if err := store.Append(ctx, runID, make(100, true)); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, runID, make(0, false)); err == nil {
		t.Fatal("a changed payload without the transcript mark must be refused")
	}
	replay := make(0, true)
	if err := store.Append(ctx, runID, replay); err != nil || !replay.PersistenceReplay {
		t.Fatalf("transcript replay: %v %+v", err, replay)
	}
	events, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(events) != 1 {
		t.Fatalf("events: %d %v", len(events), err)
	}
	if usage, ok := events[0].Data.(*domain.UsageEventData); !ok || usage.InputTokens != 100 {
		t.Fatalf("stored evidence changed: %+v", events[0].Data)
	}
}
