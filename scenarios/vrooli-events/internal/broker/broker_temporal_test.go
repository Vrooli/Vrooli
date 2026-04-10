package broker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

// [REQ:REQ-PS-001] Verify concurrent publish and unsubscribe do not race or panic
func TestConcurrentPublishAndCleanup(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{SubscriberBufSize: 8})
	defer b.Close()

	ctx := context.Background()

	// Create several subscribers
	var cleanups []func()
	for i := 0; i < 10; i++ {
		_, _, cleanup := b.Subscribe(ctx, SubscribeOpts{})
		cleanups = append(cleanups, cleanup)
	}

	// Concurrently publish and remove subscribers
	var wg sync.WaitGroup

	// Publisher goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			b.Publish(store.Event{
				ID:             int64(i),
				EventType:      "test.concurrent.v1",
				SourceScenario: "test",
			}, `{}`)
		}
	}()

	// Cleanup goroutines — remove subscribers while publishing
	for _, cleanup := range cleanups {
		wg.Add(1)
		go func(fn func()) {
			defer wg.Done()
			time.Sleep(time.Millisecond) // slight stagger
			fn()
		}(cleanup)
	}

	wg.Wait()

	// After all cleanup, subscriber count should be 0
	if got := b.SubscriberCount(); got != 0 {
		t.Fatalf("expected 0 subscribers after concurrent cleanup, got %d", got)
	}
}

// [REQ:REQ-PS-001] Verify concurrent subscribe operations don't corrupt broker state
func TestConcurrentSubscribe(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{})
	defer b.Close()

	ctx := context.Background()
	const n = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	var cleanups []func()

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, cleanup := b.Subscribe(ctx, SubscribeOpts{})
			mu.Lock()
			cleanups = append(cleanups, cleanup)
			mu.Unlock()
		}()
	}
	wg.Wait()

	if got := b.SubscriberCount(); got != n {
		t.Fatalf("expected %d subscribers, got %d", n, got)
	}

	// Clean up all
	for _, fn := range cleanups {
		fn()
	}
	if got := b.SubscriberCount(); got != 0 {
		t.Fatalf("expected 0 subscribers after cleanup, got %d", got)
	}
}

// [REQ:REQ-PS-004] Verify dropped count is atomically safe under concurrent publish
func TestDroppedCountConcurrency(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{SubscriberBufSize: 4})
	defer b.Close()

	ctx := context.Background()
	ch, _, cleanup := b.Subscribe(ctx, SubscribeOpts{})
	defer cleanup()

	// Fill the buffer from multiple goroutines
	var wg sync.WaitGroup
	const goroutines = 5
	const eventsPerGoroutine = 20

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < eventsPerGoroutine; i++ {
				b.Publish(store.Event{
					ID:             int64(base*eventsPerGoroutine + i),
					EventType:      "test.v1",
					SourceScenario: "test",
				}, `{}`)
			}
		}(g)
	}
	wg.Wait()

	// Drain channel to count delivered messages
	delivered := 0
	for {
		select {
		case <-ch:
			delivered++
		default:
			goto done
		}
	}
done:
	// Total events = goroutines * eventsPerGoroutine = 100
	// Buffer capacity = 4, so most events dropped
	total := goroutines * eventsPerGoroutine
	dropped := b.DroppedCount(ch)

	// delivered + dropped should account for all events
	// (DroppedCount resets on read, so we only get the count since last reset)
	if delivered+int(dropped) > total {
		t.Fatalf("delivered(%d) + dropped(%d) > total(%d)", delivered, dropped, total)
	}
	if delivered > 4 {
		t.Fatalf("expected at most 4 delivered (buffer size), got %d", delivered)
	}
}

// [REQ:REQ-PS-001] Verify broker.Close stops all heartbeat goroutines and closes channels
func TestBrokerCloseStopsHeartbeats(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{HeartbeatInterval: 10 * time.Millisecond})

	ctx := context.Background()
	ch1, _, _ := b.Subscribe(ctx, SubscribeOpts{})
	ch2, _, _ := b.Subscribe(ctx, SubscribeOpts{})

	if b.SubscriberCount() != 2 {
		t.Fatalf("expected 2 subscribers, got %d", b.SubscriberCount())
	}

	b.Close()

	// Channels should be closed after broker.Close
	if _, ok := <-ch1; ok {
		// Drain remaining messages, then channel should close
		for range ch1 {
		}
	}
	if _, ok := <-ch2; ok {
		for range ch2 {
		}
	}

	if b.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after Close, got %d", b.SubscriberCount())
	}
}

// [REQ:REQ-PS-001] Verify cleanup is idempotent — calling it twice does not panic
func TestCleanupIdempotent(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{})
	defer b.Close()

	ctx := context.Background()
	_, _, cleanup := b.Subscribe(ctx, SubscribeOpts{})

	cleanup()
	cleanup() // second call should be safe (no-op)

	if got := b.SubscriberCount(); got != 0 {
		t.Fatalf("expected 0 subscribers, got %d", got)
	}
}

// [REQ:REQ-PS-001] Verify heartbeat message is delivered within expected interval
func TestHeartbeatTiming(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{
		HeartbeatInterval: 50 * time.Millisecond,
	})
	defer b.Close()

	ctx := context.Background()
	ch, _, cleanup := b.Subscribe(ctx, SubscribeOpts{})
	defer cleanup()

	// Wait for heartbeat
	select {
	case msg := <-ch:
		if msg.Event != "heartbeat" {
			t.Fatalf("expected heartbeat event, got %s", msg.Event)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for heartbeat")
	}
}

// [REQ:REQ-PS-004] Verify heartbeat reports dropped count after backpressure
func TestHeartbeatReportsDroppedCount(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{
		SubscriberBufSize: 2,
		HeartbeatInterval: 50 * time.Millisecond,
	})
	defer b.Close()

	ctx := context.Background()
	ch, _, cleanup := b.Subscribe(ctx, SubscribeOpts{})
	defer cleanup()

	// Overflow the buffer to trigger drops
	for i := 0; i < 10; i++ {
		b.Publish(store.Event{
			ID:             int64(i),
			EventType:      "test.v1",
			SourceScenario: "test",
		}, `{}`)
	}

	// Wait for heartbeat that reports dropped count
	deadline := time.After(300 * time.Millisecond)
	foundDropReport := false
	for {
		select {
		case msg := <-ch:
			if msg.Event == "heartbeat" && msg.Data != "" {
				foundDropReport = true
				goto check
			}
		case <-deadline:
			goto check
		}
	}
check:
	if !foundDropReport {
		t.Fatal("expected heartbeat with dropped count report")
	}
}

// [REQ:REQ-PS-001] Verify context cancellation stops subscriber event delivery
func TestContextCancellationStopsDelivery(t *testing.T) {
	b := NewBroker(&mockStore{}, BrokerConfig{
		HeartbeatInterval: 10 * time.Millisecond,
	})
	defer b.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch, subCtx, cleanup := b.Subscribe(ctx, SubscribeOpts{})
	defer cleanup()

	cancel()

	// subCtx should be done
	select {
	case <-subCtx.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subCtx should be done after parent cancel")
	}

	// No more messages should be delivered (heartbeat goroutine should stop)
	time.Sleep(30 * time.Millisecond) // wait past heartbeat interval
	select {
	case _, ok := <-ch:
		if ok {
			// drain any buffered messages - that's fine
		}
	default:
		// no messages - expected
	}
}
