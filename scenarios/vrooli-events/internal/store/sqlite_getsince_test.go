package store

import (
	"context"
	"testing"
)

// [REQ:REQ-ES-001] GetSince returns events after a given ID
func TestGetSince_ReturnsEventsAfterID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Insert 5 events
	for i := 1; i <= 5; i++ {
		_, err := s.Insert(ctx, makeEvent("gs-"+string(rune('0'+i)), "test.getsince.v1", "src"))
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	// Get events since ID 3
	events, err := s.GetSince(ctx, 3, 100)
	if err != nil {
		t.Fatalf("GetSince: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events after ID 3, got %d", len(events))
	}
	if events[0].ID != 4 {
		t.Fatalf("expected first event ID=4, got %d", events[0].ID)
	}
	if events[1].ID != 5 {
		t.Fatalf("expected second event ID=5, got %d", events[1].ID)
	}
}

// [REQ:REQ-ES-001] GetSince with limit caps results
func TestGetSince_RespectsLimit(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		_, err := s.Insert(ctx, makeEvent("gsl-"+string(rune('0'+i)), "test.getsince.v1", "src"))
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	events, err := s.GetSince(ctx, 0, 2)
	if err != nil {
		t.Fatalf("GetSince: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events with limit=2, got %d", len(events))
	}
}

// [REQ:REQ-ES-001] GetSince with zero/negative limit uses default (100)
func TestGetSince_DefaultLimit(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Insert(ctx, makeEvent("gsd-1", "test.getsince.v1", "src"))
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	events, err := s.GetSince(ctx, 0, 0)
	if err != nil {
		t.Fatalf("GetSince with limit=0: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event with default limit, got %d", len(events))
	}

	events, err = s.GetSince(ctx, 0, -5)
	if err != nil {
		t.Fatalf("GetSince with limit=-5: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event with negative limit, got %d", len(events))
	}
}

// [REQ:REQ-ES-001] GetSince with ID beyond max returns empty
func TestGetSince_BeyondMaxID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Insert(ctx, makeEvent("gsb-1", "test.getsince.v1", "src"))
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	events, err := s.GetSince(ctx, 999999, 100)
	if err != nil {
		t.Fatalf("GetSince beyond max: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events beyond max ID, got %d", len(events))
	}
}

// [REQ:REQ-ES-001] GetSince returns events in ascending order
func TestGetSince_AscendingOrder(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		_, err := s.Insert(ctx, makeEvent("gso-"+string(rune('0'+i)), "test.getsince.v1", "src"))
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	events, err := s.GetSince(ctx, 0, 100)
	if err != nil {
		t.Fatalf("GetSince: %v", err)
	}
	for i := 1; i < len(events); i++ {
		if events[i].ID <= events[i-1].ID {
			t.Fatalf("events not in ascending order: ID %d after ID %d", events[i].ID, events[i-1].ID)
		}
	}
}

// [REQ:REQ-ES-002] NewSQLiteStore with custom config applies settings
func TestNewSQLiteStore_CustomConfig(t *testing.T) {
	ctx := context.Background()
	s, err := NewSQLiteStore(ctx, SQLiteConfig{
		QueryLimit:    50,
		QueryLimitMax: 500,
	})
	if err != nil {
		t.Fatalf("new store with custom config: %v", err)
	}
	defer s.Close()

	// Verify the store is functional
	id, err := s.Insert(ctx, makeEvent("cfg-1", "test.config.v1", "src"))
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if id != 1 {
		t.Fatalf("expected id=1, got %d", id)
	}
}

// [REQ:REQ-ES-002] Stats returns correct byte count after inserts
func TestStats_ByteCountAccuracy(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	evt := makeEvent("stats-1", "test.stats.v1", "src")
	_, err := s.Insert(ctx, evt)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalEvents != 1 {
		t.Fatalf("expected 1 event, got %d", stats.TotalEvents)
	}
	if stats.TotalPayloadBytes != int64(len(evt.Payload)) {
		t.Fatalf("expected %d bytes, got %d", len(evt.Payload), stats.TotalPayloadBytes)
	}
}

// [REQ:REQ-ES-002] DB() returns the underlying sql.DB
func TestDB_ReturnsNonNil(t *testing.T) {
	s := newTestStore(t)
	db := s.DB()
	if db == nil {
		t.Fatal("expected non-nil *sql.DB from DB()")
	}
	// Verify it's usable
	if err := db.Ping(); err != nil {
		t.Fatalf("DB().Ping() failed: %v", err)
	}
}
