package event_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// [REQ:REQ-P0-008] Tests for event streaming storage implementation

func TestMemoryStore_Append(t *testing.T) {
	store := event.NewMemoryStore()
	ctx := context.Background()
	runID := uuid.New()

	// Create test event using helper function
	evt := domain.NewLogEvent(runID, "info", "test message")

	// Append event
	if err := store.Append(ctx, runID, evt); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify event was stored (AfterSequence: -1 to include sequence 0)
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

func TestMemoryStore_Append_MultipleEvents(t *testing.T) {
	store := event.NewMemoryStore()
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

func TestMemoryStore_Get_WithFilters(t *testing.T) {
	store := event.NewMemoryStore()
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

	// Filter by type (AfterSequence: -1 to include sequence 0)
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

func TestMemoryStore_Get_AfterSequence(t *testing.T) {
	store := event.NewMemoryStore()
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

func TestMemoryStore_Get_WithLimit(t *testing.T) {
	store := event.NewMemoryStore()
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

func TestMemoryStore_Count(t *testing.T) {
	store := event.NewMemoryStore()
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

func TestMemoryStore_Delete(t *testing.T) {
	store := event.NewMemoryStore()
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

func TestMemoryStore_Stream(t *testing.T) {
	store := event.NewMemoryStore()
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

func TestMemoryStore_Stream_ExistingEvents(t *testing.T) {
	store := event.NewMemoryStore()
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

func TestMemoryStore_IsolatesByRunID(t *testing.T) {
	store := event.NewMemoryStore()
	ctx := context.Background()
	runID1 := uuid.New()
	runID2 := uuid.New()

	// Add events to different runs
	evt1 := domain.NewLogEvent(runID1, "info", "run1")
	evt2 := domain.NewLogEvent(runID2, "info", "run2")

	store.Append(ctx, runID1, evt1)
	store.Append(ctx, runID2, evt2)

	// Verify isolation (AfterSequence: -1 to include sequence 0)
	events1, _ := store.Get(ctx, runID1, event.GetOptions{AfterSequence: -1})
	events2, _ := store.Get(ctx, runID2, event.GetOptions{AfterSequence: -1})

	if len(events1) != 1 || len(events2) != 1 {
		t.Errorf("expected 1 event per run, got %d and %d", len(events1), len(events2))
	}
}
