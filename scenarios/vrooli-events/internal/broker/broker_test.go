package broker

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

// mockStore implements store.Store for broker tests (only GetSince needed).
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

func TestSubscribeAndPublish(t *testing.T) {
	b := NewBroker(&mockStore{})
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

func TestSubscribeFiltering(t *testing.T) {
	b := NewBroker(&mockStore{})
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

func TestBackpressureDrop(t *testing.T) {
	b := NewBroker(&mockStore{})
	defer b.Close()

	ctx := context.Background()
	ch, _, cleanup := b.Subscribe(ctx, SubscribeOpts{})
	defer cleanup()

	// Fill the channel beyond capacity
	for i := 0; i < subscriberBufSize+10; i++ {
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
	if drained != subscriberBufSize {
		t.Fatalf("expected %d messages (buffer capacity), got %d", subscriberBufSize, drained)
	}
}

func TestSubscriberCount(t *testing.T) {
	b := NewBroker(&mockStore{})
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

func TestSourceAndTargetFiltering(t *testing.T) {
	b := NewBroker(&mockStore{})
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
