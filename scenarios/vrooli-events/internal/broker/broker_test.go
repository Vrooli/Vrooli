package broker

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

// mockStore implements store.Store for broker tests with no-op behavior.
// Broker only needs a Store reference; it never calls Store methods directly
// during pub-sub operations, so a zero-value mock is sufficient here.
type mockStore struct{}

func (m *mockStore) Insert(_ context.Context, _ store.Event) (int64, error) { return 0, nil }
func (m *mockStore) Query(_ context.Context, _ store.QueryFilters) ([]store.Event, error) {
	return nil, nil
}

func (m *mockStore) GetSince(_ context.Context, _ int64, _ int) ([]store.Event, error) {
	return nil, nil
}

func (m *mockStore) Prune(_ context.Context) (store.PruneResult, error) {
	return store.PruneResult{}, nil
}
func (m *mockStore) Stats(_ context.Context) (store.Stats, error) { return store.Stats{}, nil }
func (m *mockStore) Close() error                                 { return nil }

// [REQ:REQ-PS-001] Verify subscriber receives published events via SSE channel
func TestSubscribeAndPublish(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{})
	defer b.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, _, cleanup := b.Subscribe(ctx, SubscribeOpts{EventTypePattern: "app.**"})
	defer cleanup()

	event := store.Event{
		ID:             1,
		EventType:      "app.domain.action.v1",
		SourceScenario: "test",
	}
	b.Publish(event, `{"test":true}`)

	select {
	case msg := <-ch:
		if msg.Event != "app.domain.action.v1" {
			t.Fatalf("expected event type app.domain.action.v1, got %s", msg.Event)
		}
		if msg.Data != `{"test":true}` {
			t.Fatalf("expected data, got %s", msg.Data)
		}
		if msg.ID != 1 {
			t.Fatalf("expected id 1, got %d", msg.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// [REQ:REQ-PS-002] Verify glob-pattern filtering on event type during publish
func TestSubscribeFiltering(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{})
	defer b.Close()

	ctx := context.Background()

	ch, _, cleanup := b.Subscribe(ctx, SubscribeOpts{EventTypePattern: "app.domain.*"})
	defer cleanup()

	// This should NOT match
	b.Publish(store.Event{ID: 1, EventType: "other.domain.action.v1"}, `{}`)
	// This should match
	b.Publish(store.Event{ID: 2, EventType: "app.domain.created"}, `{"matched":true}`)

	select {
	case msg := <-ch:
		if msg.ID != 2 {
			t.Fatalf("expected event 2, got %d", msg.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for filtered event")
	}
}

// [REQ:REQ-PS-004] Verify backpressure drops events when subscriber buffer is full
func TestBackpressureDrop(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{})
	defer b.Close()

	ctx := context.Background()
	ch, _, cleanup := b.Subscribe(ctx, SubscribeOpts{})
	defer cleanup()

	// Fill the channel beyond capacity
	for i := 0; i < 64 /* default subscriber buf size */ +10; i++ {
		b.Publish(store.Event{ID: int64(i), EventType: "test.v1"}, `{}`)
	}

	// Drain channel
	drained := 0
	for {
		select {
		case <-ch:
			drained++
		default:
			goto done
		}
	}
done:
	if drained != 64 /* default subscriber buf size */ {
		t.Fatalf("expected %d messages (buffer capacity), got %d", 64 /* default subscriber buf size */, drained)
	}
}

// [REQ:REQ-PS-001] Verify subscriber count tracking on subscribe and cleanup
func TestSubscriberCount(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{})
	defer b.Close()

	ctx := context.Background()

	if b.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers, got %d", b.SubscriberCount())
	}

	_, _, cleanup1 := b.Subscribe(ctx, SubscribeOpts{})
	_, _, cleanup2 := b.Subscribe(ctx, SubscribeOpts{})

	if b.SubscriberCount() != 2 {
		t.Fatalf("expected 2 subscribers, got %d", b.SubscriberCount())
	}

	cleanup1()
	if b.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after cleanup, got %d", b.SubscriberCount())
	}

	cleanup2()
	if b.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after all cleanup, got %d", b.SubscriberCount())
	}
}

// [REQ:REQ-PS-002] Verify glob-pattern filtering on source_scenario during publish
func TestSourceAndTargetFiltering(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{})
	defer b.Close()

	ctx := context.Background()
	ch, _, cleanup := b.Subscribe(ctx, SubscribeOpts{
		SourceScenarioPattern: "agent-manager",
	})
	defer cleanup()

	// Should not match
	b.Publish(store.Event{ID: 1, EventType: "test.v1", SourceScenario: "other"}, `{}`)
	// Should match
	b.Publish(store.Event{ID: 2, EventType: "test.v1", SourceScenario: "agent-manager"}, `{"ok":true}`)

	select {
	case msg := <-ch:
		if msg.ID != 2 {
			t.Fatalf("expected event 2, got %d", msg.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
