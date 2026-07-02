package services

import (
	"context"
	"sync"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// Concurrency Tests (run with -race)
// -----------------------------------------------------------------------------

// TestAsyncTracker_ConcurrentStartTracking verifies thread safety of StartTracking.
func TestAsyncTracker_ConcurrentStartTracking(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	const goroutines = 20
	var wg sync.WaitGroup

	// Try to start many trackings concurrently
	// Note: These will fail because they don't have valid async behavior,
	// but we're testing that concurrent access doesn't cause races
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			// This will fail validation but shouldn't race
			_ = svc.StartTracking(
				ctx,
				idString("tc-race", id),
				"chat-race",
				"tool",
				"scenario",
				nil,
				nil, // nil async behavior will cause validation error
			)
		}(i)
	}

	wg.Wait()
}

// TestAsyncTracker_ConcurrentSubscribeUnsubscribe verifies thread safety of subscriptions.
func TestAsyncTracker_ConcurrentSubscribeUnsubscribe(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	const goroutines = 50
	var wg sync.WaitGroup

	// Concurrent subscribe/unsubscribe operations
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Subscribe
			sub := svc.SubscribeWithID("chat-race-sub")

			// Simulate some work
			time.Sleep(time.Millisecond)

			// Unsubscribe
			svc.UnsubscribeByID(sub)
		}(i)
	}

	wg.Wait()

	// Verify clean state
	svc.mu.RLock()
	subCount := len(svc.subscriptions)
	chatSubCount := len(svc.chatSubs["chat-race-sub"])
	svc.mu.RUnlock()

	if subCount != 0 {
		t.Errorf("expected 0 subscriptions, got %d", subCount)
	}
	if chatSubCount != 0 {
		t.Errorf("expected 0 chat subscriptions, got %d", chatSubCount)
	}
}

// TestAsyncTracker_ConcurrentPushUpdate verifies thread safety of update pushing.
func TestAsyncTracker_ConcurrentPushUpdate(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	// Create multiple subscribers
	const subscribers = 5
	subs := make([]*Subscription, subscribers)
	for i := 0; i < subscribers; i++ {
		subs[i] = svc.SubscribeWithID("chat-push-race")
	}
	defer func() {
		for _, sub := range subs {
			svc.UnsubscribeByID(sub)
		}
	}()

	// Push many updates concurrently
	const updates = 100
	var wg sync.WaitGroup

	for i := 0; i < updates; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc.pushUpdateData("chat-push-race", AsyncStatusUpdate{
				ToolCallID: "tc-push",
				ChatID:     "chat-push-race",
				Status:     "running",
				UpdatedAt:  time.Now(),
			})
		}(i)
	}

	wg.Wait()

	// Drain all channels
	for _, sub := range subs {
		drainCount := 0
	drainLoop:
		for {
			select {
			case <-sub.Channel:
				drainCount++
			default:
				break drainLoop
			}
		}
		// Each subscriber should have received some updates
		// (may not be all due to non-blocking sends)
		if drainCount == 0 {
			t.Error("subscriber received no updates")
		}
	}
}

// TestAsyncTracker_ConcurrentCompletionCallbacks verifies callback thread safety.
func TestAsyncTracker_ConcurrentCompletionCallbacks(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	const goroutines = 20
	var wg sync.WaitGroup

	// Register and unregister completion callbacks concurrently
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			chatID := idString("chat-callback", id%5)

			// Register
			ch := svc.RegisterCompletionCallback(chatID)
			if ch == nil {
				return
			}

			// Simulate some work
			time.Sleep(time.Millisecond)

			// Unregister
			svc.UnregisterCompletionCallback(chatID)
		}(i)
	}

	wg.Wait()
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// idString creates a unique ID string for tests.
func idString(prefix string, id int) string {
	return prefix + "-" + itoa(id)
}

// itoa converts int to string (simple implementation).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
