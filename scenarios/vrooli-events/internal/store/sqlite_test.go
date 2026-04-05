package store

import (
	"context"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLiteStore(context.Background(), SQLiteConfig{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func makeEvent(id, eventType, source string) Event {
	return Event{
		EventID:        id,
		SourceScenario: source,
		TargetScenario: "target-1",
		EventType:      eventType,
		CorrelationID:  "corr-1",
		Payload:        []byte(`{"test": true}`),
		Metadata:       map[string]string{"key": "value"},
	}
}

func TestInsertAndQuery(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.Insert(ctx, makeEvent("evt-1", "test.domain.action.v1", "source-1"))
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if id != 1 {
		t.Fatalf("expected id=1, got %d", id)
	}

	events, err := s.Query(ctx, QueryFilters{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventID != "evt-1" {
		t.Fatalf("expected event_id=evt-1, got %s", events[0].EventID)
	}
	if events[0].Metadata["key"] != "value" {
		t.Fatalf("expected metadata key=value, got %v", events[0].Metadata)
	}
}

func TestQueryFilters(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, _ = s.Insert(ctx, makeEvent("evt-1", "app.domain.created.v1", "source-a"))
	_, _ = s.Insert(ctx, makeEvent("evt-2", "app.domain.deleted.v1", "source-a"))
	_, _ = s.Insert(ctx, makeEvent("evt-3", "other.domain.created.v1", "source-b"))

	// Filter by source
	events, _ := s.Query(ctx, QueryFilters{Source: "source-a"})
	if len(events) != 2 {
		t.Fatalf("source filter: expected 2, got %d", len(events))
	}

	// Filter by correlation_id
	events, _ = s.Query(ctx, QueryFilters{CorrelationID: "corr-1"})
	if len(events) != 3 {
		t.Fatalf("correlation filter: expected 3, got %d", len(events))
	}

	// Filter by since
	events, _ = s.Query(ctx, QueryFilters{Since: 1})
	if len(events) != 2 {
		t.Fatalf("since filter: expected 2, got %d", len(events))
	}

	// Filter by limit
	events, _ = s.Query(ctx, QueryFilters{Limit: 1})
	if len(events) != 1 {
		t.Fatalf("limit filter: expected 1, got %d", len(events))
	}
}

func TestQueryEventTypeGlob(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, _ = s.Insert(ctx, makeEvent("evt-1", "app.domain.created.v1", "src"))
	_, _ = s.Insert(ctx, makeEvent("evt-2", "app.domain.deleted.v1", "src"))
	_, _ = s.Insert(ctx, makeEvent("evt-3", "other.domain.created.v1", "src"))

	// Glob: app.domain.*.v1 should match first two
	events, _ := s.Query(ctx, QueryFilters{EventType: "app.domain.*.v1"})
	if len(events) != 2 {
		t.Fatalf("glob *.v1: expected 2, got %d", len(events))
	}

	// Glob: **.created.v1 should match evt-1 and evt-3
	events, _ = s.Query(ctx, QueryFilters{EventType: "**.created.v1"})
	if len(events) != 2 {
		t.Fatalf("glob **.created: expected 2, got %d", len(events))
	}
}

func TestGetSince(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, _ = s.Insert(ctx, makeEvent("evt-1", "test.action.v1", "src"))
	_, _ = s.Insert(ctx, makeEvent("evt-2", "test.action.v1", "src"))
	_, _ = s.Insert(ctx, makeEvent("evt-3", "test.action.v1", "src"))

	events, err := s.GetSince(ctx, 1, 10)
	if err != nil {
		t.Fatalf("getSince: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2, got %d", len(events))
	}
	if events[0].EventID != "evt-2" {
		t.Fatalf("expected evt-2, got %s", events[0].EventID)
	}
}

func TestDuplicateEventID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Insert(ctx, makeEvent("evt-1", "test.v1", "src"))
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	_, err = s.Insert(ctx, makeEvent("evt-1", "test.v1", "src"))
	if err == nil {
		t.Fatal("expected duplicate error, got nil")
	}
}

func TestStoreMetaTracking(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	payload := []byte("hello world")
	evt := Event{
		EventID:        "evt-1",
		SourceScenario: "src",
		EventType:      "test.v1",
		Payload:        payload,
	}
	_, _ = s.Insert(ctx, evt)

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalPayloadBytes != int64(len(payload)) {
		t.Fatalf("expected %d bytes, got %d", len(payload), stats.TotalPayloadBytes)
	}
}

func TestStats(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	stats, _ := s.Stats(ctx)
	if stats.TotalEvents != 0 {
		t.Fatalf("expected 0 events, got %d", stats.TotalEvents)
	}

	_, _ = s.Insert(ctx, makeEvent("evt-1", "test.v1", "src"))
	stats, _ = s.Stats(ctx)
	if stats.TotalEvents != 1 {
		t.Fatalf("expected 1 event, got %d", stats.TotalEvents)
	}
	if stats.OldestEvent == nil || stats.NewestEvent == nil {
		t.Fatal("expected non-nil oldest/newest")
	}
}

func TestPruneByTime(t *testing.T) {
	s, err := NewSQLiteStore(context.Background(), SQLiteConfig{MaxAge: 1 * time.Second})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	_, _ = s.Insert(ctx, makeEvent("evt-1", "test.v1", "src"))
	time.Sleep(2 * time.Second)

	result, err := s.Prune(ctx)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if result.TimeDeletedCount != 1 {
		t.Fatalf("expected 1 time-deleted, got %d", result.TimeDeletedCount)
	}

	events, _ := s.Query(ctx, QueryFilters{})
	if len(events) != 0 {
		t.Fatalf("expected 0 events after prune, got %d", len(events))
	}
}

func TestPruneBySize(t *testing.T) {
	// Max 100 bytes of payload
	s, err := NewSQLiteStore(context.Background(), SQLiteConfig{MaxSizeBytes: 100})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	bigPayload := make([]byte, 60)
	for i := range bigPayload {
		bigPayload[i] = 'A'
	}

	_, _ = s.Insert(ctx, Event{EventID: "evt-1", SourceScenario: "src", EventType: "test.v1", Payload: bigPayload})
	_, _ = s.Insert(ctx, Event{EventID: "evt-2", SourceScenario: "src", EventType: "test.v1", Payload: bigPayload})

	result, err := s.Prune(ctx)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if result.SizeDeletedCount < 1 {
		t.Fatalf("expected at least 1 size-deleted, got %d", result.SizeDeletedCount)
	}

	stats, _ := s.Stats(ctx)
	if stats.TotalPayloadBytes > 100 {
		t.Fatalf("expected payload <= 100 bytes after prune, got %d", stats.TotalPayloadBytes)
	}
}

func TestReconcileMeta(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, _ = s.Insert(ctx, Event{EventID: "evt-1", SourceScenario: "src", EventType: "test.v1", Payload: []byte("data")})

	// Corrupt the meta value
	_, _ = s.db.ExecContext(ctx, `UPDATE store_meta SET value = 999 WHERE key = 'total_payload_bytes'`)

	// Reconcile should fix it
	if err := s.reconcileMeta(ctx); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	stats, _ := s.Stats(ctx)
	if stats.TotalPayloadBytes != 4 { // len("data")
		t.Fatalf("expected 4 bytes after reconcile, got %d", stats.TotalPayloadBytes)
	}
}
