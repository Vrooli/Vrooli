package store

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// [REQ:REQ-ES-002] Verify duplicate event_id is safely rejected under concurrent insertion
func TestConcurrentDuplicateInsert(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	evt := makeEvent("evt-concurrent-dup", "test.v1", "src")

	// Try inserting the same event from multiple goroutines
	const goroutines = 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	failures := 0

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Insert(ctx, evt)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failures++
			} else {
				successes++
			}
		}()
	}
	wg.Wait()

	// Exactly one insert should succeed; rest should fail with duplicate
	if successes != 1 {
		t.Fatalf("expected exactly 1 success, got %d successes and %d failures", successes, failures)
	}

	// Verify only one row exists
	events, _ := s.Query(ctx, QueryFilters{})
	if len(events) != 1 {
		t.Fatalf("expected 1 event in store, got %d", len(events))
	}
}

// [REQ:REQ-ES-002] Verify duplicate insert returns ErrDuplicateEvent sentinel
func TestDuplicateInsertReturnsSentinel(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	evt := makeEvent("evt-dup-sentinel", "test.v1", "src")
	_, err := s.Insert(ctx, evt)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = s.Insert(ctx, evt)
	if err == nil {
		t.Fatal("expected error on duplicate insert")
	}
	if !errors.Is(err, ErrDuplicateEvent) {
		t.Fatalf("expected ErrDuplicateEvent, got: %v", err)
	}
	if !strings.Contains(err.Error(), "evt-dup-sentinel") {
		t.Fatalf("expected event_id in error message, got: %v", err)
	}
}

// [REQ:REQ-ES-001] Verify concurrent inserts of unique events maintain store consistency
func TestConcurrentUniqueInserts(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	const n = 20
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			evt := Event{
				EventID:        eventID(idx),
				SourceScenario: "src",
				EventType:      "test.concurrent.v1",
				Payload:        []byte(`{"idx":` + intToStr(idx) + `}`),
			}
			_, err := s.Insert(ctx, evt)
			if err != nil {
				t.Errorf("insert %d: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	events, err := s.Query(ctx, QueryFilters{Limit: n + 10})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(events) != n {
		t.Fatalf("expected %d events, got %d", n, len(events))
	}

	// Verify meta tracking is consistent
	stats, _ := s.Stats(ctx)
	if stats.TotalEvents != int64(n) {
		t.Fatalf("expected %d total events, got %d", n, stats.TotalEvents)
	}
}

// [REQ:REQ-ES-003] Verify prune is idempotent — running twice yields same state
func TestPruneIdempotent(t *testing.T) {
	s, err := NewSQLiteStore(context.Background(), SQLiteConfig{MaxAge: 1 * time.Second})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	_, _ = s.Insert(ctx, makeEvent("evt-1", "test.v1", "src"))
	time.Sleep(2 * time.Second)

	// First prune should delete the event
	result1, err := s.Prune(ctx)
	if err != nil {
		t.Fatalf("first prune: %v", err)
	}
	if result1.TimeDeletedCount != 1 {
		t.Fatalf("expected 1 deleted on first prune, got %d", result1.TimeDeletedCount)
	}

	// Second prune should be a no-op
	result2, err := s.Prune(ctx)
	if err != nil {
		t.Fatalf("second prune: %v", err)
	}
	if result2.TimeDeletedCount != 0 || result2.SizeDeletedCount != 0 {
		t.Fatalf("expected 0 deleted on second prune, got time=%d size=%d",
			result2.TimeDeletedCount, result2.SizeDeletedCount)
	}

	// Store should be empty after both prunes
	stats, _ := s.Stats(ctx)
	if stats.TotalEvents != 0 {
		t.Fatalf("expected 0 events after idempotent prune, got %d", stats.TotalEvents)
	}
	if stats.TotalPayloadBytes != 0 {
		t.Fatalf("expected 0 bytes after idempotent prune, got %d", stats.TotalPayloadBytes)
	}
}

// [REQ:REQ-ES-001] Verify GetSince is stable/idempotent — same lastID returns same results
func TestGetSinceIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, _ = s.Insert(ctx, makeEvent("evt-1", "test.v1", "src"))
	_, _ = s.Insert(ctx, makeEvent("evt-2", "test.v1", "src"))
	_, _ = s.Insert(ctx, makeEvent("evt-3", "test.v1", "src"))

	// Call GetSince twice with the same lastID
	events1, _ := s.GetSince(ctx, 1, 10)
	events2, _ := s.GetSince(ctx, 1, 10)

	if len(events1) != len(events2) {
		t.Fatalf("GetSince not stable: got %d then %d", len(events1), len(events2))
	}
	for i := range events1 {
		if events1[i].EventID != events2[i].EventID {
			t.Fatalf("GetSince not stable at index %d: %s != %s",
				i, events1[i].EventID, events2[i].EventID)
		}
	}
}

// [REQ:REQ-ES-001] Verify Insert with context cancellation rolls back cleanly
func TestInsertContextCancellation(t *testing.T) {
	s := newTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := s.Insert(ctx, makeEvent("evt-cancelled", "test.v1", "src"))
	if err == nil {
		// If insert succeeded despite cancellation, that's also valid for SQLite
		// (the operation may complete before context check). Verify consistency.
		stats, _ := s.Stats(context.Background())
		if stats.TotalEvents != 1 {
			t.Fatal("inconsistent state after cancelled insert that succeeded")
		}
		return
	}

	// If insert failed, verify no partial state
	stats, _ := s.Stats(context.Background())
	if stats.TotalEvents != 0 {
		t.Fatalf("expected 0 events after cancelled insert, got %d", stats.TotalEvents)
	}
}

// [REQ:REQ-ES-003] Verify meta tracking stays accurate through insert + prune cycles
func TestMetaConsistencyThroughPruneCycle(t *testing.T) {
	s, err := NewSQLiteStore(context.Background(), SQLiteConfig{
		MaxAge:       1 * time.Second,
		MaxSizeBytes: 1 << 30, // 1GB - won't trigger size prune
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	// Insert events with known payload sizes
	payload10 := []byte("0123456789") // 10 bytes
	for i := 0; i < 5; i++ {
		_, _ = s.Insert(ctx, Event{
			EventID:        eventID(i),
			SourceScenario: "src",
			EventType:      "test.v1",
			Payload:        payload10,
		})
	}

	stats, _ := s.Stats(ctx)
	if stats.TotalPayloadBytes != 50 {
		t.Fatalf("expected 50 bytes, got %d", stats.TotalPayloadBytes)
	}

	// Wait for events to expire
	time.Sleep(2 * time.Second)

	// Prune expired events
	_, err = s.Prune(ctx)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}

	// Meta should track 0 bytes
	stats, _ = s.Stats(ctx)
	if stats.TotalPayloadBytes != 0 {
		t.Fatalf("expected 0 bytes after prune, got %d", stats.TotalPayloadBytes)
	}

	// Insert new events and verify meta is still accurate
	_, _ = s.Insert(ctx, Event{
		EventID:        "evt-post-prune",
		SourceScenario: "src",
		EventType:      "test.v1",
		Payload:        payload10,
	})

	stats, _ = s.Stats(ctx)
	if stats.TotalPayloadBytes != 10 {
		t.Fatalf("expected 10 bytes after post-prune insert, got %d", stats.TotalPayloadBytes)
	}
}

// [REQ:REQ-ES-002] Verify event_id uniqueness constraint error message is recognizable
func TestDuplicateEventIDErrorMessage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, _ = s.Insert(ctx, makeEvent("evt-dup-check", "test.v1", "src"))
	_, err := s.Insert(ctx, makeEvent("evt-dup-check", "test.v1", "src"))

	if err == nil {
		t.Fatal("expected error for duplicate event_id")
	}
	// Error should wrap ErrDuplicateEvent and include the event_id
	if !errors.Is(err, ErrDuplicateEvent) {
		t.Fatalf("expected ErrDuplicateEvent, got: %s", err)
	}
	if !strings.Contains(err.Error(), "evt-dup-check") {
		t.Fatalf("expected event_id in error, got: %s", err)
	}
}

// Helper: generate deterministic event IDs for concurrent tests
func eventID(idx int) string {
	return "evt-" + intToStr(idx)
}

func intToStr(n int) string {
	if n < 0 {
		return "-" + intToStr(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return intToStr(n/10) + string(rune('0'+n%10))
}
