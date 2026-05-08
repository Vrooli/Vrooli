// Tests for the stats engine.
//
// Coverage focuses on the four weakness fixes the design promises:
//
//   1. Dispatch table coverage — every (event_type, schema_version)
//      registered in eventlog has a stats processor.
//      → TestAllEmittedEventsAreProcessed
//   2. Typed Category enum at the HTTP edge.
//      → TestOperationalHandler_BadCategory400
//   3. Resumable replay (rebuild from saved checkpoint, not from zero).
//      → TestRebuildResumesFromCheckpoint
//   4. Per-event correctness — at least one fixture-backed end-to-end
//      assertion that fold-by-(event,version) lands in the right metric.
//      → TestEngine_AggregatesFallbackInsights, TestEngine_HistoryWindow

package stats

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func newTestEngine(t *testing.T) (*Engine, *eventlog.SQLiteRepository, *sqlx.DB) {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(dir, "stats.db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
	CREATE TABLE IF NOT EXISTS run_events (
		id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		sequence INTEGER NOT NULL,
		event_type TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		schema_version INTEGER NOT NULL DEFAULT 1,
		data TEXT NOT NULL,
		UNIQUE(run_id, sequence)
	);
	CREATE TABLE IF NOT EXISTS stats_checkpoint (
		name TEXT PRIMARY KEY,
		last_rowid INTEGER NOT NULL,
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}

	repo := eventlog.NewSQLiteRepository(db)
	checkpoint := NewSQLiteCheckpointStore(db)
	engine := NewEngine(repo, checkpoint, "test")
	return engine, repo, db
}

func insertEvent(t *testing.T, db *sqlx.DB, runID uuid.UUID, seq int64, evt *domain.RunEvent) {
	t.Helper()
	body := evt.Data.(*domain.TypedEventData).Body
	_, err := db.Exec(
		`INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, schema_version, data)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		evt.ID, runID, seq, string(evt.EventType),
		evt.Timestamp.UTC().Format(time.RFC3339Nano),
		evt.SchemaVersion, []byte(body),
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
}

// TestAllEmittedEventsAreProcessed pins the contract that every
// (event_type, schema_version) registered in eventlog has a stats
// processor. If a new typed event is added without wiring a processor,
// this test fails — preventing the silent zero-counter bug that
// motivated the fix.
func TestAllEmittedEventsAreProcessed(t *testing.T) {
	registered := eventlog.RegisteredKeys()
	processors := RegisteredProcessorKeys()

	have := make(map[eventlog.RegisteredKey]struct{}, len(processors))
	for _, k := range processors {
		have[k] = struct{}{}
	}

	for _, k := range registered {
		if _, ok := have[k]; !ok {
			t.Errorf("eventlog registers %s v%d but stats has no processor for it",
				k.EventType, k.SchemaVersion)
		}
	}
}

func TestEngine_AggregatesFallbackInsights(t *testing.T) {
	engine, _, db := newTestEngine(t)
	runID := uuid.New()

	events := []eventlog.Payload{
		eventlog.ModelFallbackAttemptedPayload{
			From:          "claude-3.5-sonnet",
			To:            "claude-3.5-haiku",
			Reason:        "rate_limit",
			AttemptNo:     1,
			ChainPosition: 1,
			ChainLength:   3,
		},
		eventlog.ModelFallbackAttemptedPayload{
			From:          "claude-3.5-haiku",
			To:            "claude-3-opus",
			Reason:        "model_unknown",
			AttemptNo:     2,
			ChainPosition: 2,
			ChainLength:   3,
		},
		eventlog.RunnerFallbackAttemptedPayload{
			From:      "codex",
			To:        "claude-code",
			Reason:    "binary_missing",
			AttemptNo: 1,
		},
	}
	for i, p := range events {
		evt, err := eventlog.BuildEvent(runID, p)
		if err != nil {
			t.Fatalf("BuildEvent: %v", err)
		}
		evt.Timestamp = time.Now().UTC().Add(-time.Hour)
		insertEvent(t, db, runID, int64(i+1), evt)
	}

	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	fb := engine.GetFallback()
	if fb.ModelAttempts != 2 {
		t.Errorf("ModelAttempts = %d, want 2", fb.ModelAttempts)
	}
	if fb.RunnerAttempts != 1 {
		t.Errorf("RunnerAttempts = %d, want 1", fb.RunnerAttempts)
	}
	if fb.ModelByReason["rate_limit"] != 1 {
		t.Errorf("ModelByReason[rate_limit] = %d, want 1", fb.ModelByReason["rate_limit"])
	}
	if fb.ModelByReason["model_unknown"] != 1 {
		t.Errorf("ModelByReason[model_unknown] = %d, want 1", fb.ModelByReason["model_unknown"])
	}
	if fb.RunnerByReason["binary_missing"] != 1 {
		t.Errorf("RunnerByReason[binary_missing] = %d, want 1", fb.RunnerByReason["binary_missing"])
	}
	if got := fb.History.MinSampleMeaningful; got != MinSampleMeaningful {
		t.Errorf("MinSampleMeaningful = %d, want %d", got, MinSampleMeaningful)
	}
	if !fb.History.HasHistory {
		t.Error("HasHistory should be true after processing events")
	}
}

func TestRebuildResumesFromCheckpoint(t *testing.T) {
	engine, _, db := newTestEngine(t)
	runID := uuid.New()

	// Insert two events.
	for i, reason := range []string{"rate_limit", "auth_failure"} {
		evt, _ := eventlog.BuildEvent(runID, eventlog.ModelFallbackAttemptedPayload{
			From: "a", To: "b", Reason: reason, AttemptNo: i + 1,
		})
		evt.Timestamp = time.Now().UTC()
		insertEvent(t, db, runID, int64(i+1), evt)
	}
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("first rebuild: %v", err)
	}
	wm := engine.Watermark()
	if wm == 0 {
		t.Fatal("watermark should be advanced after first rebuild")
	}

	// Add a third event; rebuild should pick up the saved checkpoint and
	// process only the new event.
	evt, _ := eventlog.BuildEvent(runID, eventlog.ModelFallbackAttemptedPayload{
		From: "b", To: "c", Reason: "quota_exhausted", AttemptNo: 3,
	})
	evt.Timestamp = time.Now().UTC()
	insertEvent(t, db, runID, 3, evt)

	// Construct a fresh engine sharing the checkpoint store; this
	// simulates a process restart.
	fresh := NewEngine(eventlog.NewSQLiteRepository(db), NewSQLiteCheckpointStore(db), "test")
	if err := fresh.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild after restart: %v", err)
	}
	if fresh.Watermark() <= wm {
		t.Errorf("watermark did not advance: before=%d after=%d", wm, fresh.Watermark())
	}
	if fresh.EventCount() != 1 {
		// Only the newly-appended event should have been processed —
		// the checkpoint stopped us replaying the first two.
		t.Errorf("EventCount = %d, want 1 (only the new event after checkpoint)", fresh.EventCount())
	}
}

func TestEngine_HistoryWindow_EmptyState(t *testing.T) {
	engine, _, _ := newTestEngine(t)
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	s := engine.GetSummary()
	if s.History.HasHistory {
		t.Error("HasHistory should be false on an empty engine")
	}
	if s.History.MinSampleMeaningful != MinSampleMeaningful {
		t.Errorf("MinSampleMeaningful = %d, want %d", s.History.MinSampleMeaningful, MinSampleMeaningful)
	}
}

func TestEngine_HealthSummary_TracksLatestStatus(t *testing.T) {
	engine, _, db := newTestEngine(t)
	runID := uuid.New()

	// Two transitions for the same model: ok→failed→ok. The latest
	// status (ok) should be the one in the snapshot.
	transitions := []eventlog.ModelHealthTransitionPayload{
		{Runner: "claude-code", Model: "sonnet", FromStatus: "ok", ToStatus: "failed", Reason: "rate_limit"},
		{Runner: "claude-code", Model: "sonnet", FromStatus: "failed", ToStatus: "ok"},
	}
	for i, p := range transitions {
		evt, _ := eventlog.BuildEvent(runID, p)
		evt.Timestamp = time.Now().UTC().Add(time.Duration(i) * time.Minute)
		insertEvent(t, db, runID, int64(i+1), evt)
	}
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	h := engine.GetHealth()
	if len(h.Models) != 1 {
		t.Fatalf("Models = %d, want 1", len(h.Models))
	}
	got := h.Models[0]
	if got.Status != "ok" {
		t.Errorf("Status = %s, want ok", got.Status)
	}
	if got.TransitionsObserved != 2 {
		t.Errorf("TransitionsObserved = %d, want 2", got.TransitionsObserved)
	}
}

