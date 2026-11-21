package event_test

import (
	"context"
	"database/sql/driver"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"modernc.org/sqlite"
)

// setupTestDB creates a fresh SQLite database for testing the SQLiteStore.
func setupTestDB(t *testing.T) (*sqlx.DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "event-store-test.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)",
		dbPath,
	)

	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}

	// Create the run_events table
	schema := `
		CREATE TABLE IF NOT EXISTS run_events (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			schema_version INTEGER NOT NULL DEFAULT 1,
			data TEXT,
			UNIQUE(run_id, sequence)
		);
		CREATE INDEX IF NOT EXISTS idx_run_events_run_id ON run_events(run_id);
		CREATE INDEX IF NOT EXISTS idx_run_events_sequence ON run_events(run_id, sequence);
		CREATE TABLE IF NOT EXISTS event_retention_state (
			singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
			generation INTEGER NOT NULL,
			floor_rowid INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		);
		INSERT INTO event_retention_state(singleton,generation,floor_rowid,updated_at) VALUES (1,1,0,datetime('now'));
		CREATE TABLE IF NOT EXISTS runs (
			id TEXT PRIMARY KEY,
			execution_mode TEXT NOT NULL DEFAULT 'codec_pipe'
		);
		CREATE TABLE run_event_highwater (run_id TEXT PRIMARY KEY, high_sequence INTEGER NOT NULL);
		CREATE TRIGGER run_event_highwater_insert AFTER INSERT ON run_events BEGIN
			INSERT INTO run_event_highwater VALUES (NEW.run_id, NEW.sequence) ON CONFLICT(run_id) DO UPDATE SET high_sequence = MAX(high_sequence, excluded.high_sequence);
		END;
		CREATE TRIGGER run_event_highwater_delete BEFORE DELETE ON run_events WHEN EXISTS (SELECT 1 FROM runs WHERE id = OLD.run_id) BEGIN
			INSERT INTO run_event_highwater VALUES (OLD.run_id, OLD.sequence) ON CONFLICT(run_id) DO UPDATE SET high_sequence = MAX(high_sequence, excluded.high_sequence);
		END;
		CREATE TRIGGER run_event_highwater_run_delete AFTER DELETE ON runs BEGIN DELETE FROM run_event_highwater WHERE run_id = OLD.id; END;
		CREATE TRIGGER run_event_highwater_orphan_delete AFTER DELETE ON run_events WHEN NOT EXISTS (SELECT 1 FROM runs WHERE id = OLD.run_id) AND NOT EXISTS (SELECT 1 FROM run_events WHERE run_id = OLD.run_id) BEGIN DELETE FROM run_event_highwater WHERE run_id = OLD.run_id; END;
	`
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		t.Fatalf("create schema: %v", err)
	}

	return db, func() {
		_ = db.Close()
	}
}

func newTestLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.PanicLevel) // Suppress logs during tests
	return log
}

func TestSQLiteStoreGetManyKeepsCompleteFilteredStreamsGroupedByRun(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runA, runB, unrelated := uuid.New(), uuid.New(), uuid.New()
	for _, item := range []struct {
		run uuid.UUID
		evt *domain.RunEvent
	}{
		{runA, domain.NewMetricEvent(runA, "tokens", 11, "tokens")},
		{runA, domain.NewLogEvent(runA, "info", "ignored")},
		{runB, domain.NewMetricEvent(runB, "tokens", 29, "tokens")},
		{unrelated, domain.NewMetricEvent(unrelated, "tokens", 99, "tokens")},
	} {
		if err := store.Append(ctx, item.run, item.evt); err != nil {
			t.Fatal(err)
		}
	}
	got, decodeErrs, err := store.GetMany(ctx, []uuid.UUID{runA, runB}, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMetric}})
	if err != nil {
		t.Fatal(err)
	}
	if len(decodeErrs) != 0 {
		t.Fatalf("valid event streams had decode errors: %v", decodeErrs)
	}
	if len(got) != 2 || len(got[runA]) != 1 || len(got[runB]) != 1 || got[runA][0].RunID != runA || got[runB][0].RunID != runB {
		t.Fatalf("batch event grouping/filter changed: %#v", got)
	}
	if _, ok := got[unrelated]; ok {
		t.Fatal("batch read included an unrequested run")
	}
}

func TestSQLiteStoreGetManyScopesCorruptPayloadFailureToItsRun(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	badRun, goodRun := uuid.New(), uuid.New()
	if err := store.Append(ctx, badRun, domain.NewMetricEvent(badRun, "tokens", 11, "tokens")); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, goodRun, domain.NewMetricEvent(goodRun, "tokens", 29, "tokens")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE run_events SET data='not-json' WHERE run_id=?`, badRun); err != nil {
		t.Fatal(err)
	}
	events, decodeErrs, err := store.GetMany(ctx, []uuid.UUID{badRun, goodRun}, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMetric}})
	if err != nil {
		t.Fatal(err)
	}
	if decodeErrs[badRun] == nil || len(events[badRun]) != 0 || decodeErrs[goodRun] != nil || len(events[goodRun]) != 1 {
		t.Fatalf("corrupt run poisoned or leaked into sibling accounting: events=%v errors=%v", events, decodeErrs)
	}
}

// =============================================================================
// SQLiteStore Tests
// =============================================================================

func TestSQLiteStoreAppendIfEmptyPreservesConcurrentEvidence(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID, otherID := uuid.New(), uuid.New()
	old := domain.NewLogEvent(runID, "info", "old")
	other := domain.NewLogEvent(otherID, "info", "other")
	if err := store.Append(ctx, runID, old); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, otherID, other); err != nil {
		t.Fatal(err)
	}
	conflict := domain.NewLogEvent(runID, "info", "different")
	conflict.ID = other.ID
	if err := store.AppendIfEmpty(ctx, runID, conflict); err == nil {
		t.Fatal("expected nonempty stream rejection")
	}
	got, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(got) != 1 || got[0].ID != old.ID {
		t.Fatalf("old stream lost: %+v %v", got, err)
	}
	emptyID := uuid.New()
	replacement := domain.NewLogEvent(emptyID, "info", "new")
	if err := store.AppendIfEmpty(ctx, emptyID, replacement); err != nil {
		t.Fatal(err)
	}
	got, err = store.Get(ctx, emptyID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(got) != 1 || got[0].ID != replacement.ID {
		t.Fatalf("replacement not committed: %+v %v", got, err)
	}
}

func TestSQLiteStoreAppendIfEmptyRacesAppend(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	id := uuid.New()
	regular := domain.NewLogEvent(id, "info", "concurrent")
	rebuilt := domain.NewLogEvent(id, "info", "rebuilt")
	start := make(chan struct{})
	appendResult, rebuildResult := make(chan error, 1), make(chan error, 1)
	go func() { <-start; appendResult <- store.Append(ctx, id, regular) }()
	go func() { <-start; rebuildResult <- store.AppendIfEmpty(ctx, id, rebuilt) }()
	close(start)
	if err := <-appendResult; err != nil {
		t.Fatal(err)
	}
	rebuildErr := <-rebuildResult
	got, err := store.Get(ctx, id, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range got {
		if item.ID == regular.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("concurrent append was lost: %+v (rebuild err %v)", got, rebuildErr)
	}
}

func TestSQLiteStreamReplacementFencesConcurrentAppend(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	id := uuid.New()
	old := domain.NewLogEvent(id, "info", "old")
	if err := store.Append(ctx, id, old); err != nil {
		t.Fatal(err)
	}
	token, err := store.StreamToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	concurrent := domain.NewLogEvent(id, "info", "concurrent")
	if err := store.Append(ctx, id, concurrent); err != nil {
		t.Fatal(err)
	}
	guard := store.LockReplacement()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	err = guard.ReplaceInTx(ctx, tx, id, token, []*domain.RunEvent{domain.NewLogEvent(id, "info", "replacement")})
	_ = tx.Rollback()
	guard.Finish(id, false)
	if err == nil {
		t.Fatal("stale replacement token accepted")
	}
	got, err := store.Get(ctx, id, event.GetOptions{AfterSequence: -1})
	if err != nil || len(got) != 2 || got[0].ID != old.ID || got[1].ID != concurrent.ID {
		t.Fatalf("concurrent evidence lost: %+v %v", got, err)
	}
}

func TestSQLiteRecallTokenDetectsAppendThenDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	id := uuid.New()
	if _, err := db.Exec(`INSERT INTO runs(id) VALUES (?)`, id); err != nil {
		t.Fatal(err)
	}
	token, err := store.StreamToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, id, domain.NewLogEvent(id, "info", "intervening")); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	guard := store.LockReplacement()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	err = guard.AppendEmptyInTx(ctx, tx, id, token, []*domain.RunEvent{domain.NewLogEvent(id, "info", "stale")})
	_ = tx.Rollback()
	guard.Finish(id, false)
	if err == nil {
		t.Fatal("append/delete generation change was not detected")
	}
	count, err := store.Count(ctx, id)
	if err != nil || count != 0 {
		t.Fatalf("stale events published: %d %v", count, err)
	}
}

func TestSQLiteStreamReplacementRollsBackBeforeCommit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	id := uuid.New()
	old := domain.NewLogEvent(id, "info", "old")
	if err := store.Append(ctx, id, old); err != nil {
		t.Fatal(err)
	}
	token, err := store.StreamToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	guard := store.LockReplacement()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	if err := guard.ReplaceInTx(ctx, tx, id, token, []*domain.RunEvent{domain.NewLogEvent(id, "info", "replacement")}); err != nil {
		_ = tx.Rollback()
		guard.Finish(id, false)
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	guard.Finish(id, false)
	got, err := store.Get(ctx, id, event.GetOptions{AfterSequence: -1})
	if err != nil || len(got) != 1 || got[0].ID != old.ID {
		t.Fatalf("rollback lost old event: %+v %v", got, err)
	}
}

func TestSQLiteEmptyReplacementAndEvictionPreserveSequenceWatermark(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	id := uuid.New()
	if _, err := db.Exec(`INSERT INTO runs(id) VALUES (?)`, id); err != nil {
		t.Fatal(err)
	}
	old := domain.NewLogEvent(id, "info", "old")
	if err := store.Append(ctx, id, old); err != nil {
		t.Fatal(err)
	}
	token, err := store.StreamToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	guard := store.LockReplacement()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	if err := guard.ReplaceInTx(ctx, tx, id, token, nil); err != nil {
		_ = tx.Rollback()
		guard.Finish(id, false)
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	guard.Finish(id, true)
	var generation int
	if err := db.Get(&generation, `SELECT generation FROM event_retention_state WHERE singleton=1`); err != nil || generation != 2 {
		t.Fatalf("replacement did not advance retention generation: %d %v", generation, err)
	}
	next := domain.NewLogEvent(id, "info", "next")
	if err := store.Append(ctx, id, next); err != nil {
		t.Fatal(err)
	}
	if next.Sequence <= old.Sequence {
		t.Fatalf("empty replacement reused sequence: old=%d next=%d", old.Sequence, next.Sequence)
	}
	if err := store.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&generation, `SELECT generation FROM event_retention_state WHERE singleton=1`); err != nil || generation != 3 {
		t.Fatalf("recall deletion did not advance retention generation: %d %v", generation, err)
	}
	if err := store.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&generation, `SELECT generation FROM event_retention_state WHERE singleton=1`); err != nil || generation != 3 {
		t.Fatalf("empty deletion changed retention generation: %d %v", generation, err)
	}
	afterEviction := domain.NewLogEvent(id, "info", "after eviction")
	if err := store.AppendIfEmpty(ctx, id, afterEviction); err != nil {
		t.Fatal(err)
	}
	if afterEviction.Sequence <= next.Sequence {
		t.Fatalf("eviction reused sequence: before=%d after=%d", next.Sequence, afterEviction.Sequence)
	}
	if _, err := db.Exec(`DELETE FROM runs WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := db.Get(&remaining, `SELECT COUNT(*) FROM run_event_highwater WHERE run_id = ?`, id); err != nil || remaining != 0 {
		t.Fatalf("deleted run retained watermark: %d %v", remaining, err)
	}
}

func TestSQLiteStreamReplacementDoesNotCloseNewSubscriber(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	id := uuid.New()
	old := domain.NewLogEvent(id, "info", "old")
	if err := store.Append(ctx, id, old); err != nil {
		t.Fatal(err)
	}
	token, err := store.StreamToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	oldStream, err := store.Stream(ctx, id, event.StreamOptions{FromSequence: 0})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-oldStream:
	case <-time.After(time.Second):
		t.Fatal("old stream did not deliver history")
	}
	guard := store.LockReplacement()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	newEvent := domain.NewLogEvent(id, "info", "new")
	if err := guard.ReplaceInTx(ctx, tx, id, token, []*domain.RunEvent{newEvent}); err != nil {
		_ = tx.Rollback()
		guard.Finish(id, false)
		t.Fatal(err)
	}
	newStream := make(chan (<-chan *domain.RunEvent), 1)
	go func() { stream, _ := store.Stream(ctx, id, event.StreamOptions{FromSequence: 0}); newStream <- stream }()
	if err := tx.Commit(); err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	guard.Finish(id, true)
	resumed, err := store.Get(ctx, id, event.GetOptions{AfterSequence: old.Sequence})
	if err != nil || len(resumed) != 1 || resumed[0].ID != newEvent.ID || resumed[0].Sequence <= old.Sequence {
		t.Fatalf("old cursor missed replacement: %+v %v", resumed, err)
	}
	select {
	case _, open := <-oldStream:
		if open {
			t.Fatal("old subscription was not closed")
		}
	case <-time.After(time.Second):
		t.Fatal("old subscription remained open")
	}
	select {
	case stream := <-newStream:
		select {
		case got, open := <-stream:
			if !open || got.ID != newEvent.ID {
				t.Fatalf("new subscriber did not receive replacement: %+v open=%v", got, open)
			}
		case <-time.After(time.Second):
			t.Fatal("new subscriber received no history")
		}
	case <-time.After(time.Second):
		t.Fatal("new subscriber remained blocked")
	}
}

func TestSQLiteReplacementStopsBlockedHistoricalStream(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	id := uuid.New()
	var originals []*domain.RunEvent
	for i := 0; i < 50; i++ {
		originals = append(originals, domain.NewLogEvent(id, "info", fmt.Sprint(i)))
	}
	if err := store.Append(ctx, id, originals...); err != nil {
		t.Fatal(err)
	}
	stream, err := store.Stream(ctx, id, event.StreamOptions{FromSequence: 0, BufferSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	token, err := store.StreamToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	guard := store.LockReplacement()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	if err := guard.ReplaceInTx(ctx, tx, id, token, []*domain.RunEvent{domain.NewLogEvent(id, "info", "replacement")}); err != nil {
		_ = tx.Rollback()
		guard.Finish(id, false)
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		guard.Finish(id, false)
		t.Fatal(err)
	}
	guard.Finish(id, true)
	deadline := time.After(time.Second)
	for {
		select {
		case _, open := <-stream:
			if !open {
				return
			}
		case <-deadline:
			t.Fatal("blocked historical stream did not close")
		}
	}
}

func TestSQLiteStoreAppendIfEmptyKeepsSubscribers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	id := uuid.New()
	ch, err := store.Stream(ctx, id, event.StreamOptions{FromSequence: 0})
	if err != nil {
		t.Fatal(err)
	}
	item := domain.NewLogEvent(id, "info", "rebuilt")
	if err := store.AppendIfEmpty(ctx, id, item); err != nil {
		t.Fatal(err)
	}
	select {
	case received, open := <-ch:
		if !open || received.ID != item.ID {
			t.Fatalf("subscriber missed rebuilt event: %+v open=%v", received, open)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber received no rebuilt event")
	}
}

func TestSQLiteSubscriberOverflowClosesForReplay(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(fmt.Sprintf("staged_%t", staged), func(t *testing.T) {
			db, cleanup := setupTestDB(t)
			defer cleanup()
			store := event.NewSQLiteStore(db, newTestLogger())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			id := uuid.New()
			stream, err := store.Stream(ctx, id, event.StreamOptions{FromSequence: 0, BufferSize: 1})
			if err != nil {
				t.Fatal(err)
			}
			events := []*domain.RunEvent{domain.NewLogEvent(id, "info", "one"), domain.NewLogEvent(id, "info", "two"), domain.NewLogEvent(id, "info", "three")}
			if staged {
				token, err := store.StreamToken(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				guard := store.LockReplacement()
				tx, err := db.BeginTx(ctx, nil)
				if err != nil {
					guard.Finish(id, false)
					t.Fatal(err)
				}
				if err := guard.AppendEmptyInTx(ctx, tx, id, token, events); err != nil {
					_ = tx.Rollback()
					guard.Finish(id, false)
					t.Fatal(err)
				}
				if err := tx.Commit(); err != nil {
					guard.Finish(id, false)
					t.Fatal(err)
				}
				guard.FinishAppend(id, true, events)
			} else if err := store.Append(ctx, id, events...); err != nil {
				t.Fatal(err)
			}
			deadline := time.After(time.Second)
			for {
				select {
				case _, open := <-stream:
					if !open {
						got, err := store.Get(ctx, id, event.GetOptions{AfterSequence: -1})
						if err != nil || len(got) != 3 {
							t.Fatalf("replay evidence missing: %+v %v", got, err)
						}
						return
					}
				case <-deadline:
					t.Fatal("overflowed stream stayed open")
				}
			}
		})
	}
}

func TestSQLiteStore_Append(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Create test event
	evt := domain.NewLogEvent(runID, "info", "test message")

	// Append event
	if err := store.Append(ctx, runID, evt); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify event was stored
	events, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	// Verify sequence was assigned
	if events[0].Sequence != 0 {
		t.Errorf("expected sequence 0, got %d", events[0].Sequence)
	}

	// Verify runID was set
	if events[0].RunID != runID {
		t.Errorf("expected runID %s, got %s", runID, events[0].RunID)
	}

	// Verify timestamp was set
	if events[0].Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}

	// Verify ID was generated
	if events[0].ID == uuid.Nil {
		t.Error("expected ID to be generated")
	}
}

func TestSQLiteStore_Append_MultipleEvents(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Append multiple events
	events := []*domain.RunEvent{
		domain.NewLogEvent(runID, "info", "first"),
		domain.NewLogEvent(runID, "info", "second"),
		domain.NewLogEvent(runID, "info", "third"),
	}

	if err := store.Append(ctx, runID, events...); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify all events were stored with sequential sequence numbers
	result, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 events, got %d", len(result))
	}

	for i, evt := range result {
		if evt.Sequence != int64(i) {
			t.Errorf("event %d: expected sequence %d, got %d", i, i, evt.Sequence)
		}
	}
}

func TestSQLiteStore_SummarizeUsesRetainedMetadataAndPageSelection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()
	corruptLargePayload := []byte(strings.Repeat("x", 2*1024*1024))
	rows := []struct {
		sequence int64
		typeName string
		data     []byte
	}{
		{0, string(domain.EventTypeLog), []byte{0xff, 0x00}},
		{1, "future.kind", corruptLargePayload},
		{2, string(domain.EventTypeMessage), []byte("not-json")},
		{3, "future.kind", []byte("also-not-json")},
	}
	for _, row := range rows {
		_, err := db.ExecContext(ctx, `INSERT INTO run_events (id,run_id,sequence,event_type,timestamp,schema_version,data) VALUES (?,?,?,?,?,?,?)`, uuid.New(), runID, row.sequence, row.typeName, time.Now().UTC().Format(time.RFC3339Nano), 1, row.data)
		if err != nil {
			t.Fatal(err)
		}
	}

	summary, err := store.Summarize(ctx, runID, event.GetOptions{AfterSequence: -1, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(summary.Counts, map[string]int64{string(domain.EventTypeLog): 1, "future.kind": 1}) {
		t.Fatalf("limited counts=%v", summary.Counts)
	}
	if summary.LatestSequence == nil || *summary.LatestSequence != 1 || !summary.HasMore {
		t.Fatalf("limited summary=%+v; want latest=1 has_more=true", summary)
	}
	zeroSequence, err := store.Summarize(ctx, runID, event.GetOptions{AfterSequence: -1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if zeroSequence.LatestSequence == nil || *zeroSequence.LatestSequence != 0 || !zeroSequence.HasMore {
		t.Fatalf("sequence-zero summary=%+v; zero must be present and distinguishable from empty", zeroSequence)
	}

	filtered, err := store.Summarize(ctx, runID, event.GetOptions{
		AfterSequence: 0, EventTypes: []domain.RunEventType{"future.kind"}, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(filtered.Counts, map[string]int64{"future.kind": 1}) || filtered.LatestSequence == nil || *filtered.LatestSequence != 1 || !filtered.HasMore {
		t.Fatalf("filtered summary=%+v; want first matching row and lookahead", filtered)
	}

	unlimited, err := store.Summarize(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(unlimited.Counts, map[string]int64{
		string(domain.EventTypeLog): 1, "future.kind": 2, string(domain.EventTypeMessage): 1,
	}) || unlimited.LatestSequence == nil || *unlimited.LatestSequence != 3 || unlimited.HasMore {
		t.Fatalf("unlimited summary=%+v", unlimited)
	}

	empty, err := store.Summarize(ctx, runID, event.GetOptions{AfterSequence: 9})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Counts) != 0 || empty.LatestSequence != nil || empty.HasMore {
		t.Fatalf("empty summary=%+v", empty)
	}

	if _, err := db.Exec(`CREATE TABLE invocation_read_model_watermarks (run_id TEXT PRIMARY KEY, projection_complete INTEGER NOT NULL DEFAULT 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteBefore(ctx, time.Now().Add(time.Hour), 10); err != nil {
		t.Fatalf("sweep retained rows: %v", err)
	}
	swept, err := store.Summarize(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(swept.Counts) != 0 || swept.LatestSequence != nil {
		t.Fatalf("post-sweep summary=%+v; want empty retained-row facts", swept)
	}
}

func TestSQLiteStore_SummarizeReturnsQueryFailure(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	if _, err := db.Exec(`DROP TABLE run_events`); err != nil {
		t.Fatal(err)
	}
	if summary, err := store.Summarize(context.Background(), uuid.New(), event.GetOptions{AfterSequence: -1}); err == nil || summary != nil {
		t.Fatalf("summary=%+v error=%v; query failure must not become measured zero", summary, err)
	}
}

func TestSQLiteStore_Get_WithFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Append events of different types
	events := []*domain.RunEvent{
		domain.NewLogEvent(runID, "info", "log1"),
		domain.NewMessageEvent(runID, "user", "hello"),
		domain.NewLogEvent(runID, "info", "log2"),
	}

	if err := store.Append(ctx, runID, events...); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Filter by type
	result, err := store.Get(ctx, runID, event.GetOptions{
		EventTypes:    []domain.RunEventType{domain.EventTypeLog},
		AfterSequence: -1,
	})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 log events, got %d", len(result))
	}
}

func TestSQLiteStore_Get_AfterSequence(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Append events
	for i := 0; i < 5; i++ {
		evt := &domain.RunEvent{EventType: domain.EventTypeLog}
		if err := store.Append(ctx, runID, evt); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	// Get events after sequence 2
	result, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: 2})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 events after sequence 2, got %d", len(result))
	}

	// Verify first returned event has sequence 3
	if len(result) > 0 && result[0].Sequence != 3 {
		t.Errorf("expected first event sequence 3, got %d", result[0].Sequence)
	}
}

func TestSQLiteStore_Get_WithLimit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Append events
	for i := 0; i < 10; i++ {
		evt := &domain.RunEvent{EventType: domain.EventTypeLog}
		if err := store.Append(ctx, runID, evt); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	// Get with limit
	result, err := store.Get(ctx, runID, event.GetOptions{Limit: 3})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 events with limit, got %d", len(result))
	}
}

func TestSQLiteStore_Get_WithOffset(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Append events (sequences 0-9)
	for i := 0; i < 10; i++ {
		evt := &domain.RunEvent{EventType: domain.EventTypeLog}
		if err := store.Append(ctx, runID, evt); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	// Get with offset and limit, starting from sequence -1 (include all)
	// AfterSequence: -1 means include sequence 0 onwards
	// Offset: 5 means skip first 5 results
	// So we get events 5-9 = 5 events
	result, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: -1, Offset: 5, Limit: 100})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(result) != 5 {
		t.Errorf("expected 5 events after offset, got %d", len(result))
	}

	// First event should have sequence 5
	if len(result) > 0 && result[0].Sequence != 5 {
		t.Errorf("expected first event sequence 5, got %d", result[0].Sequence)
	}
}

func TestSQLiteStore_Count(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Initially empty
	count, err := store.Count(ctx, runID)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	// Add events
	for i := 0; i < 5; i++ {
		evt := &domain.RunEvent{EventType: domain.EventTypeLog}
		if err := store.Append(ctx, runID, evt); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	// Count again
	count, err = store.Count(ctx, runID)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Add events
	evt := &domain.RunEvent{EventType: domain.EventTypeLog}
	if err := store.Append(ctx, runID, evt); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify event exists
	count, _ := store.Count(ctx, runID)
	if count != 1 {
		t.Fatalf("expected 1 event before delete, got %d", count)
	}

	// Delete
	if err := store.Delete(ctx, runID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	count, _ = store.Count(ctx, runID)
	if count != 0 {
		t.Errorf("expected 0 events after delete, got %d", count)
	}
}

func TestSQLiteStore_Stream(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runID := uuid.New()

	// Start streaming
	ch, err := store.Stream(ctx, runID, event.StreamOptions{BufferSize: 10})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Append event after stream started
	go func() {
		time.Sleep(50 * time.Millisecond)
		evt := domain.NewLogEvent(runID, "info", "streamed")
		if err := store.Append(context.Background(), runID, evt); err != nil {
			t.Errorf("Append failed: %v", err)
		}
	}()

	// Wait for event
	select {
	case evt := <-ch:
		if evt == nil {
			t.Error("received nil event")
		}
	case <-ctx.Done():
		t.Error("timeout waiting for streamed event")
	}
}

func TestSQLiteStore_Stream_ExistingEvents(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runID := uuid.New()

	// Add events before streaming
	for i := 0; i < 3; i++ {
		evt := domain.NewLogEvent(runID, "info", "test")
		if err := store.Append(context.Background(), runID, evt); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	// Start streaming from sequence 0
	ch, err := store.Stream(ctx, runID, event.StreamOptions{FromSequence: 0, BufferSize: 10})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Should receive existing events
	received := 0
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case <-ch:
			received++
			if received == 3 {
				return // Success
			}
		case <-timeout:
			t.Errorf("timeout: received %d of 3 expected events", received)
			return
		}
	}
}

func TestSQLiteStore_IsolatesByRunID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID1 := uuid.New()
	runID2 := uuid.New()

	// Add events to different runs
	evt1 := domain.NewLogEvent(runID1, "info", "run1")
	evt2 := domain.NewLogEvent(runID2, "info", "run2")

	if err := store.Append(ctx, runID1, evt1); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if err := store.Append(ctx, runID2, evt2); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify isolation
	events1, _ := store.Get(ctx, runID1, event.GetOptions{AfterSequence: -1})
	events2, _ := store.Get(ctx, runID2, event.GetOptions{AfterSequence: -1})

	if len(events1) != 1 || len(events2) != 1 {
		t.Errorf("expected 1 event per run, got %d and %d", len(events1), len(events2))
	}
}

func TestSQLiteStore_EventDataRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Test different event types with data
	testCases := []struct {
		name  string
		event *domain.RunEvent
	}{
		{
			name:  "log event",
			event: domain.NewLogEvent(runID, "info", "test log message"),
		},
		{
			name:  "message event",
			event: domain.NewMessageEvent(runID, "assistant", "Hello, I can help with that."),
		},
		{
			name:  "tool call event",
			event: domain.NewToolCallEvent(runID, "write_file", "", map[string]interface{}{"path": "/test.txt", "content": "hello"}),
		},
		{
			name:  "status event",
			event: domain.NewStatusEvent(runID, "pending", "running", "test transition"),
		},
		{
			name:  "error event",
			event: domain.NewErrorEvent(runID, "test_error", "something went wrong", true),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Append
			if err := store.Append(ctx, runID, tc.event); err != nil {
				t.Fatalf("Append failed: %v", err)
			}

			// Get
			events, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: tc.event.Sequence - 1, Limit: 1})
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}

			// Verify event type matches
			if events[0].EventType != tc.event.EventType {
				t.Errorf("event type mismatch: got %s, want %s", events[0].EventType, tc.event.EventType)
			}

			// Verify data is not nil
			if events[0].Data == nil {
				t.Error("expected data to be preserved, got nil")
			}
		})
	}
}

func TestSQLiteStore_ConcurrentAppendSequences(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	const numSources = 5
	const eventsPerSource = 10

	var wg sync.WaitGroup
	errs := make(chan error, numSources*eventsPerSource)
	for i := 0; i < numSources; i++ {
		source := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerSource; j++ {
				evt := domain.NewLogEvent(runID, "info", fmt.Sprintf("source %d event %d", source, j))
				if err := store.Append(ctx, runID, evt); err != nil {
					errs <- err
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("append failed: %v", err)
	}

	// Verify all events were stored
	count, err := store.Count(ctx, runID)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	expected := int64(numSources * eventsPerSource)
	if count != expected {
		t.Errorf("expected %d events, got %d", expected, count)
	}

	// Verify sequences are unique and sequential
	events, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	for i, evt := range events {
		if evt.Sequence != int64(i) {
			t.Errorf("expected sequence %d, got %d", i, evt.Sequence)
		}
	}
}

func TestSQLiteStore_DeleteBeforeIsBounded(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ensureProjectionWatermarksTable(t, db)
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()
	old := time.Now().Add(-48 * time.Hour)
	for range 3 {
		evt := domain.NewLogEvent(runID, "info", "expired")
		evt.Timestamp = old
		if err := store.Append(ctx, runID, evt); err != nil {
			t.Fatalf("append expired: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,projection_complete,projected_at) VALUES (?, 'event-3', ?, 'test', 1, ?)`, runID.String(), old, old); err != nil {
		t.Fatalf("mark projection complete: %v", err)
	}
	fresh := domain.NewLogEvent(runID, "info", "fresh")
	fresh.Timestamp = time.Now()
	if err := store.Append(ctx, runID, fresh); err != nil {
		t.Fatalf("append fresh: %v", err)
	}

	deleted, err := store.DeleteBefore(ctx, time.Now().Add(-24*time.Hour), 2)
	if err != nil || deleted != 2 {
		t.Fatalf("first bounded delete = %d, %v", deleted, err)
	}
	deleted, err = store.DeleteBefore(ctx, time.Now().Add(-24*time.Hour), 2)
	if err != nil || deleted != 1 {
		t.Fatalf("second bounded delete = %d, %v", deleted, err)
	}
	count, err := store.Count(ctx, runID)
	if err != nil || count != 1 {
		t.Fatalf("retained events = %d, %v", count, err)
	}
	var generation int64
	if err := db.Get(&generation, `SELECT generation FROM event_retention_state WHERE singleton=1`); err != nil || generation != 3 {
		t.Fatalf("retention generation = %d, %v", generation, err)
	}
}

func TestSQLiteStore_RetentionSelectionDoesNotBlockWrites(t *testing.T) {
	selecting, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	functionName := "retention_gate_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := sqlite.RegisterScalarFunction(functionName, 1, func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		once.Do(func() { close(selecting); <-release })
		return args[0], nil
	}); err != nil {
		t.Fatal(err)
	}
	db, cleanup := setupTestDB(t)
	defer cleanup()
	db.SetMaxOpenConns(2)
	ensureProjectionWatermarksTable(t, db)
	store := event.NewSQLiteStore(db, newTestLogger())
	runID := uuid.New()
	// A live run row routes selection through the projection gate below; an
	// orphan would be eligible without reading it.
	if _, err := db.Exec(`INSERT INTO runs(id) VALUES (?)`, runID.String()); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	evt := domain.NewLogEvent(runID, "info", "expired")
	evt.Timestamp = old
	if err := store.Append(t.Context(), runID, evt); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,projection_complete,projected_at) VALUES (?, 'event', ?, 'test', 1, ?)`, runID.String(), old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE invocation_read_model_watermarks RENAME TO test_watermarks;
CREATE VIEW invocation_read_model_watermarks AS SELECT run_id, ` + functionName + `(projection_complete) AS projection_complete FROM test_watermarks`); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { _, err := store.DeleteBefore(t.Context(), time.Now().Add(-24*time.Hour), 1); finished <- err }()
	select {
	case <-selecting:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("retention did not begin selecting candidates")
	}
	writerCtx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	_, writeErr := db.ExecContext(writerCtx, `INSERT INTO runs(id) VALUES (?)`, uuid.NewString())
	cancel()
	close(release)
	retentionErr := <-finished
	if writeErr != nil {
		t.Fatalf("ordinary writes blocked by retention candidate selection: %v", writeErr)
	}
	if retentionErr != nil {
		t.Fatalf("retention: %v", retentionErr)
	}
}

type retentionSelectionDB struct {
	*sqlx.DB
	afterSelection func()
}

func (db *retentionSelectionDB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	if err := db.DB.SelectContext(ctx, dest, query, args...); err != nil {
		return err
	}
	db.afterSelection()
	return nil
}

func TestSQLiteStore_RetentionRechecksCandidateEligibility(t *testing.T) {
	for _, change := range []string{
		`UPDATE test_watermarks SET projection_complete=0`,
		`UPDATE runs SET execution_mode='imported'`,
		`UPDATE run_events SET timestamp='2099-01-01T00:00:00Z'`,
	} {
		t.Run(change, func(t *testing.T) {
			db, cleanup := setupTestDB(t)
			defer cleanup()
			ensureProjectionWatermarksTable(t, db)
			store := event.NewSQLiteStore(db, newTestLogger())
			runID := uuid.New()
			old := time.Now().Add(-48 * time.Hour)
			evt := domain.NewLogEvent(runID, "info", "expired")
			evt.Timestamp = old
			if err := store.Append(t.Context(), runID, evt); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO runs(id) VALUES (?)`, runID.String()); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,projection_complete,projected_at) VALUES (?, 'event', ?, 'test', 1, ?)`, runID.String(), old, old); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`ALTER TABLE invocation_read_model_watermarks RENAME TO test_watermarks;
CREATE VIEW invocation_read_model_watermarks AS SELECT * FROM test_watermarks`); err != nil {
				t.Fatal(err)
			}
			observed := event.NewSQLiteStore(&retentionSelectionDB{DB: db, afterSelection: func() {
				if _, err := db.Exec(change); err != nil {
					t.Fatal(err)
				}
			}}, newTestLogger())
			deleted, err := observed.DeleteBefore(t.Context(), time.Now().Add(-24*time.Hour), 1)
			if err != nil || deleted != 0 {
				t.Fatalf("newly ineligible event deleted: %d, %v", deleted, err)
			}
			if count, err := store.Count(t.Context(), runID); err != nil || count != 1 {
				t.Fatalf("retained count: %d, %v", count, err)
			}
			var generation int
			if err := db.Get(&generation, `SELECT generation FROM event_retention_state WHERE singleton=1`); err != nil || generation != 1 {
				t.Fatalf("empty retention advanced generation: %d, %v", generation, err)
			}
		})
	}
}

func TestSQLiteStore_DeleteBeforePreservesEventsWithoutCompletedProjection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ensureProjectionWatermarksTable(t, db)
	if _, err := db.Exec(`CREATE TABLE invocation_read_model_facts (run_id TEXT NOT NULL, call_event_id TEXT NOT NULL, PRIMARY KEY (run_id, call_event_id))`); err != nil {
		t.Fatal(err)
	}
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	old := time.Now().Add(-48 * time.Hour)
	completeRun, incompleteRun := uuid.New(), uuid.New()
	for _, runID := range []uuid.UUID{completeRun, incompleteRun} {
		if _, err := db.Exec(`INSERT INTO runs (id) VALUES (?)`, runID.String()); err != nil {
			t.Fatal(err)
		}
		evt := domain.NewLogEvent(runID, "info", "expired")
		evt.Timestamp = old
		if err := store.Append(ctx, runID, evt); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,projection_complete,projected_at) VALUES (?, 'event', ?, 'test', 1, ?)`, completeRun.String(), old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO invocation_read_model_facts (run_id, call_event_id) VALUES (?, 'event')`, completeRun.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,projection_complete,projected_at) VALUES (?, 'event', ?, 'test', 0, ?)`, incompleteRun.String(), old, old); err != nil {
		t.Fatal(err)
	}
	deleted, err := store.DeleteBefore(ctx, time.Now().Add(-24*time.Hour), 10)
	if err != nil || deleted != 1 {
		t.Fatalf("deleted = %d, %v", deleted, err)
	}
	if count, _ := store.Count(ctx, completeRun); count != 0 {
		t.Fatalf("completed run events = %d, want 0", count)
	}
	if count, _ := store.Count(ctx, incompleteRun); count != 1 {
		t.Fatalf("incomplete run events = %d, want 1", count)
	}
	var derivedFacts int
	if err := db.Get(&derivedFacts, `SELECT COUNT(*) FROM invocation_read_model_facts WHERE run_id = ?`, completeRun.String()); err != nil || derivedFacts != 1 {
		t.Fatalf("derived facts after completed-run sweep = %d, %v; want the projection to remain queryable", derivedFacts, err)
	}
}

func TestSQLiteStore_DeleteBeforePreservesImportedEvents(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ensureProjectionWatermarksTable(t, db)
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	old := time.Now().Add(-48 * time.Hour)
	importedRun, nativeRun := uuid.New(), uuid.New()

	for _, runID := range []uuid.UUID{importedRun, nativeRun} {
		if _, err := db.Exec(`INSERT INTO runs (id) VALUES (?)`, runID.String()); err != nil {
			t.Fatal(err)
		}
		evt := domain.NewLogEvent(runID, "info", "expired")
		evt.Timestamp = old
		if err := store.Append(ctx, runID, evt); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,projection_complete,projected_at) VALUES (?, 'event', ?, 'test', 1, ?)`, runID.String(), old, old); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`UPDATE runs SET execution_mode = 'imported' WHERE id = ?`, importedRun.String()); err != nil {
		t.Fatal(err)
	}

	deleted, err := store.DeleteBefore(ctx, time.Now().Add(-24*time.Hour), 10)
	if err != nil || deleted != 1 {
		t.Fatalf("deleted = %d, %v; want only native event deleted", deleted, err)
	}
	if count, _ := store.Count(ctx, importedRun); count != 1 {
		t.Fatalf("imported run events = %d, want 1", count)
	}
	if count, _ := store.Count(ctx, nativeRun); count != 0 {
		t.Fatalf("native run events = %d, want 0", count)
	}
}

func ensureProjectionWatermarksTable(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS invocation_read_model_watermarks (run_id TEXT PRIMARY KEY, last_event_id TEXT NOT NULL, last_event_at TEXT NOT NULL, classifier_version TEXT NOT NULL, projection_complete INTEGER NOT NULL DEFAULT 0, projected_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create projection watermark fixture: %v", err)
	}
}
