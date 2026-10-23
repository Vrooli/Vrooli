package event_test

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
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
	"agent-manager/internal/protoconv"
	"google.golang.org/protobuf/encoding/protojson"

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

// =============================================================================
// SQLiteStore Tests
// =============================================================================

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

func TestSQLiteStore_QuotaFactsSurviveDurableReadAndPublicConversion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	runID := uuid.New()
	reset := time.Date(2026, 10, 14, 3, 29, 20, 7, time.UTC)
	zero := 0.0
	quota := domain.NewRateLimitEvent(runID, "weekly", "native quota", &reset, 0)
	q := quota.Data.(*domain.RateLimitEventData)
	q.Provider, q.Pool, q.UsedPercent, q.WindowMinutes, q.Provenance = "openai", "primary", &zero, 10080, "native"
	store := event.NewSQLiteStore(db, newTestLogger())
	if err := store.Append(ctx, runID, quota); err != nil {
		t.Fatal(err)
	}
	generic := domain.NewErrorEvent(runID, "RATE_LIMIT", "ordinary error", true)
	if err := store.Append(ctx, runID, generic); err != nil {
		t.Fatal(err)
	}

	// A fresh store reads owner-persisted JSON, rather than the appended pointer.
	store = event.NewSQLiteStore(db, newTestLogger())
	got, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(got) != 2 {
		t.Fatalf("durable events=%+v err=%v", got, err)
	}
	data, ok := got[0].Data.(*domain.RateLimitEventData)
	if !ok || data.ResetTime == nil || !data.ResetTime.Equal(reset) || data.UsedPercent == nil || *data.UsedPercent != 0 || data.Provider != "openai" || data.Pool != "primary" || data.WindowMinutes != 10080 || data.Provenance != "native" || data.Limit != 0 || data.CurrentUsed != 0 {
		t.Fatalf("durable quota=%+v (%T)", got[0].Data, got[0].Data)
	}
	public := protoconv.RunEventToProto(got[0])
	wire, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(wire, &fields); err != nil {
		t.Fatal(err)
	}
	var rate map[string]json.RawMessage
	if err := json.Unmarshal(fields["rate_limit"], &rate); err != nil {
		t.Fatalf("public event=%s: %v", wire, err)
	}
	if string(rate["used_percent"]) != "0" || string(rate["provider"]) != `"openai"` || string(rate["pool"]) != `"primary"` || string(rate["window_minutes"]) != `"10080"` || string(rate["provenance"]) != `"native"` || public.GetRateLimit().GetResetTime() == nil || !public.GetRateLimit().GetResetTime().AsTime().Equal(reset) {
		t.Fatalf("public quota facts=%s", wire)
	}
	if e := protoconv.RunEventToProto(got[1]).GetError(); e == nil || e.Code != "RATE_LIMIT" || e.Message != "ordinary error" || !e.Retryable {
		t.Fatalf("generic error=%+v", e)
	}

	// Reader-stage proof uses literal tagged JSON independently of the writer constructor.
	tagged := `{"payloadKind":"quota","limitType":"weekly","message":"native quota","resetTime":"2026-10-14T03:29:20.000000007Z","provider":"openai","pool":"primary","usedPercent":0,"windowMinutes":10080,"provenance":"native"}`
	taggedID := uuid.New().String()
	if _, err := db.Exec(`INSERT INTO run_events (id,run_id,sequence,event_type,timestamp,data) VALUES (?,?,?,?,?,?)`, taggedID, runID.String(), 3, string(domain.EventTypeError), reset, tagged); err != nil {
		t.Fatal(err)
	}
	got, err = store.Get(ctx, runID, event.GetOptions{AfterSequence: 2, Limit: 1})
	if err != nil || len(got) != 1 {
		t.Fatalf("tagged read=%+v err=%v", got, err)
	}
	data, ok = got[0].Data.(*domain.RateLimitEventData)
	if !ok || data.ResetTime == nil || !data.ResetTime.Equal(reset) || data.UsedPercent == nil || *data.UsedPercent != 0 || data.Provider != "openai" || data.Pool != "primary" || data.WindowMinutes != 10080 || data.Provenance != "native" || data.Limit != 0 || data.CurrentUsed != 0 {
		t.Fatalf("tagged durable quota=%+v (%T)", got[0].Data, got[0].Data)
	}
	public = protoconv.RunEventToProto(got[0])
	wire, err = (protojson.MarshalOptions{UseProtoNames: true}).Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	fields = make(map[string]json.RawMessage)
	if err := json.Unmarshal(wire, &fields); err != nil {
		t.Fatal(err)
	}
	rate = make(map[string]json.RawMessage)
	if err := json.Unmarshal(fields["rate_limit"], &rate); err != nil {
		t.Fatalf("tagged public event=%s: %v", wire, err)
	}
	if string(rate["used_percent"]) != "0" || string(rate["provider"]) != `"openai"` || string(rate["pool"]) != `"primary"` || string(rate["window_minutes"]) != `"10080"` || string(rate["provenance"]) != `"native"` || public.GetRateLimit().GetResetTime() == nil || !public.GetRateLimit().GetResetTime().AsTime().Equal(reset) {
		t.Fatalf("tagged public quota facts=%s", wire)
	}
	var taggedPersisted string
	if err := db.Get(&taggedPersisted, `SELECT data FROM run_events WHERE id=?`, taggedID); err != nil || taggedPersisted != tagged {
		t.Fatalf("tagged bytes=%q err=%v", taggedPersisted, err)
	}

	// Historical limit-only JSON stays byte-identical and has no invented native usage.
	legacy := `{"limitType":"token","message":"limit","currentUsed":25,"limit":100}`
	legacyID := uuid.New().String()
	if _, err := db.Exec(`INSERT INTO run_events (id,run_id,sequence,event_type,timestamp,data) VALUES (?,?,?,?,?,?)`, legacyID, runID.String(), 4, string(domain.EventTypeError), reset, legacy); err != nil {
		t.Fatal(err)
	}
	got, err = store.Get(ctx, runID, event.GetOptions{AfterSequence: 3, Limit: 1})
	if err != nil || len(got) != 1 {
		t.Fatalf("legacy read=%+v err=%v", got, err)
	}
	data, ok = got[0].Data.(*domain.RateLimitEventData)
	if !ok || data.Limit != 100 || data.CurrentUsed != 25 || data.UsedPercent != nil || data.Provider != "" || data.Pool != "" || data.WindowMinutes != 0 || data.ResetTime != nil || data.Provenance != "" {
		t.Fatalf("legacy facts=%+v (%T)", got[0].Data, got[0].Data)
	}
	public = protoconv.RunEventToProto(got[0])
	if public.GetRateLimit() == nil || public.GetRateLimit().UsedPercent != nil || public.GetRateLimit().GetLimit() != 100 {
		t.Fatalf("legacy public quota=%+v", public)
	}
	var persisted string
	if err := db.Get(&persisted, `SELECT data FROM run_events WHERE id=?`, legacyID); err != nil || persisted != legacy {
		t.Fatalf("historical bytes=%q err=%v", persisted, err)
	}
}

func TestSQLiteStoreMalformedKnownPayloadIsDatabaseError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, logrus.New())
	runID, eventID := uuid.New(), uuid.New()
	raw := `{"payloadKind":"quota","limitType":"weekly","usedPercent":"bad"}`
	_, err := db.Exec(`INSERT INTO run_events(id,run_id,sequence,event_type,timestamp,data) VALUES(?,?,?,?,?,?)`, eventID.String(), runID.String(), 0, "error", time.Now(), raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), runID, event.GetOptions{AfterSequence: -1})
	var databaseError *domain.DatabaseError
	if !errors.As(err, &databaseError) || databaseError.Operation != "decode_event" || databaseError.EntityID != eventID.String() || got != nil {
		t.Fatalf("events=%+v error=%v", got, err)
	}
	var persisted string
	if err := db.Get(&persisted, `SELECT data FROM run_events WHERE id=?`, eventID.String()); err != nil || persisted != raw {
		t.Fatalf("persisted=%q error=%v", persisted, err)
	}
}

func TestSQLiteStoreToolTruthSurvivesPublicConversion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, logrus.New())
	runID := uuid.New()
	for index, exit := range []*int{nil, eventExitCode(0), eventExitCode(8)} {
		known := exit != nil
		var duration *int64
		if index > 0 {
			value := int64(0)
			if index == 2 {
				value = 125
			}
			duration = &value
		}
		evt := domain.NewToolResultEvent(runID, "exec", "call", "", nil)
		data := evt.Data.(*domain.ToolResultEventData)
		data.ExitCode = exit
		data.DurationMS = duration
		data.OutcomeKnown = &known
		data.Success = known && *exit == 0
		data.OutputBlocks = []domain.ToolOutputBlock{{Type: "input_text", Text: "one", RawJSON: `{"type":"input_text","text":"one"}`}, {Type: "image", RawJSON: `{"type":"image","future":true}`}}
		data.RawOutputJSON = `[ {"type":"input_text", "text":"one"}, {"type":"image", "future":true} ]`
		if err := store.Append(context.Background(), runID, evt); err != nil {
			t.Fatal(err)
		}
		rows, err := store.Get(context.Background(), runID, event.GetOptions{AfterSequence: int64(index) - 1, Limit: 1})
		if err != nil || len(rows) != 1 {
			t.Fatalf("read=%+v error=%v", rows, err)
		}
		public := protoconv.RunEventToProto(rows[0]).GetToolResult()
		if (public.ExitCode == nil) != (exit == nil) || exit != nil && int(public.GetExitCode()) != *exit || public.OutcomeKnown == nil || public.GetOutcomeKnown() != known || (public.DurationMs == nil) != (duration == nil) || duration != nil && public.GetDurationMs() != *duration || len(public.GetOutputBlocks()) != 2 || public.GetOutputBlocks()[1].GetRawJson() != `{"type":"image","future":true}` || public.GetRawOutputJson() != string(data.RawOutputJSON) {
			t.Fatalf("public=%+v", public)
		}
	}
}
func eventExitCode(n int) *int { return &n }

func TestSQLiteStoreImmutableReplaySuppressesNotificationsAndConflicts(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	run := uuid.New()
	stream, err := store.Stream(ctx, run, event.StreamOptions{BufferSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	original := domain.NewLogEvent(run, "warning", "same evidence")
	if err := store.Append(ctx, run, original); err != nil {
		t.Fatal(err)
	}
	select {
	case <-stream:
	case <-time.After(time.Second):
		t.Fatal("initial notification missing")
	}
	replay := domain.NewLogEvent(run, "warning", "same evidence")
	replay.ID = original.ID
	if err := store.Append(ctx, run, replay); err != nil || !replay.PersistenceReplay || replay.Sequence != original.Sequence {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	select {
	case evt := <-stream:
		t.Fatalf("duplicate notification=%+v", evt)
	default:
	}
	conflict := domain.NewLogEvent(run, "warning", "changed evidence")
	conflict.ID = original.ID
	err = store.Append(ctx, run, conflict)
	var dbErr *domain.DatabaseError
	if !errors.As(err, &dbErr) || dbErr.Operation != "replay_event_conflict" || dbErr.EntityID != original.ID.String() {
		t.Fatalf("conflict=%v", err)
	}
	another := domain.NewLogEvent(run, "warning", "same evidence")
	if err := store.Append(ctx, run, another); err != nil || another.Sequence != 1 {
		t.Fatalf("distinct evidence=%+v err=%v", another, err)
	}
}
