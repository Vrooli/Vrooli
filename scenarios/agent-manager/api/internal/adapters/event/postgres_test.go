package event_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // SQLite driver for tests
)

// setupTestDB creates a fresh SQLite database for testing the PostgresStore.
// SQLite is used for fast, isolated tests but the PostgresStore works with both.
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

	// Create the run_events table (simplified schema for SQLite)
	schema := `
		CREATE TABLE IF NOT EXISTS run_events (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			data TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_run_events_run_id ON run_events(run_id);
		CREATE INDEX IF NOT EXISTS idx_run_events_sequence ON run_events(run_id, sequence);
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
// PostgresStore Tests
// =============================================================================

func TestPostgresStore_Append(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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

func TestPostgresStore_Append_MultipleEvents(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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

func TestPostgresStore_Get_WithFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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

func TestPostgresStore_Get_AfterSequence(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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

func TestPostgresStore_Get_WithLimit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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

func TestPostgresStore_Get_WithOffset(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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

func TestPostgresStore_Count(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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

func TestPostgresStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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

func TestPostgresStore_Stream(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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
		store.Append(context.Background(), runID, evt)
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

func TestPostgresStore_Stream_ExistingEvents(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runID := uuid.New()

	// Add events before streaming
	for i := 0; i < 3; i++ {
		evt := domain.NewLogEvent(runID, "info", "test")
		store.Append(context.Background(), runID, evt)
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

func TestPostgresStore_IsolatesByRunID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
	ctx := context.Background()
	runID1 := uuid.New()
	runID2 := uuid.New()

	// Add events to different runs
	evt1 := domain.NewLogEvent(runID1, "info", "run1")
	evt2 := domain.NewLogEvent(runID2, "info", "run2")

	store.Append(ctx, runID1, evt1)
	store.Append(ctx, runID2, evt2)

	// Verify isolation
	events1, _ := store.Get(ctx, runID1, event.GetOptions{AfterSequence: -1})
	events2, _ := store.Get(ctx, runID2, event.GetOptions{AfterSequence: -1})

	if len(events1) != 1 || len(events2) != 1 {
		t.Errorf("expected 1 event per run, got %d and %d", len(events1), len(events2))
	}
}

func TestPostgresStore_EventDataRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
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
			event: domain.NewToolCallEvent(runID, "write_file", map[string]interface{}{"path": "/test.txt", "content": "hello"}),
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

func TestPostgresStore_SequentialAppend(t *testing.T) {
	// Note: SQLite doesn't handle concurrent writes well, so this test
	// uses sequential appends to verify sequence uniqueness.
	// For PostgreSQL, concurrent appends are fully supported.
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := event.NewPostgresStore(db, newTestLogger())
	ctx := context.Background()
	runID := uuid.New()

	// Append events sequentially from multiple "sources"
	const numSources = 5
	const eventsPerSource = 10

	for i := 0; i < numSources; i++ {
		for j := 0; j < eventsPerSource; j++ {
			evt := domain.NewLogEvent(runID, "info", fmt.Sprintf("source %d event %d", i, j))
			if err := store.Append(ctx, runID, evt); err != nil {
				t.Fatalf("append failed: %v", err)
			}
		}
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
