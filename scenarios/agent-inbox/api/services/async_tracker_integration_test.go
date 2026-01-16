package services

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// -----------------------------------------------------------------------------
// Integration Tests
// -----------------------------------------------------------------------------

// TestAsyncFlow_OperationLifecycle verifies the full lifecycle of an async operation
// from creation through completion including subscriber notifications.
func TestAsyncFlow_OperationLifecycle(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	// Subscribe to updates
	sub := svc.SubscribeWithID("chat-1")
	defer svc.UnsubscribeByID(sub)

	// Register completion callback
	completionCh := svc.RegisterCompletionCallback("chat-1")
	defer svc.UnregisterCompletionCallback("chat-1")

	// Add a test operation
	now := time.Now()
	svc.AddTestOperation(&AsyncOperation{
		ToolCallID:    "tc-lifecycle",
		ChatID:        "chat-1",
		ToolName:      "test-tool",
		Scenario:      "test-scenario",
		ExternalRunID: "run-123",
		Status:        "running",
		StartedAt:     now,
		UpdatedAt:     now,
	})

	// Verify operation is active
	active := svc.GetActiveOperations("chat-1")
	if len(active) != 1 {
		t.Fatalf("expected 1 active operation, got %d", len(active))
	}

	// Push an update
	update := AsyncStatusUpdate{
		ToolCallID: "tc-lifecycle",
		ChatID:     "chat-1",
		ToolName:   "test-tool",
		Status:     "running",
		Message:    "Processing...",
		UpdatedAt:  time.Now(),
	}
	svc.pushUpdateData("chat-1", update)

	// Verify subscriber received update
	select {
	case received := <-sub.Channel:
		if received.Message != "Processing..." {
			t.Errorf("expected message 'Processing...', got '%s'", received.Message)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for subscriber update")
	}

	// Stop tracking (simulates completion)
	svc.StopTracking("tc-lifecycle")

	// Verify completion callback was triggered
	select {
	case event := <-completionCh:
		if event.Status != "cancelled" {
			t.Errorf("expected status 'cancelled', got '%s'", event.Status)
		}
		if event.ToolCallID != "tc-lifecycle" {
			t.Errorf("expected tool_call_id 'tc-lifecycle', got '%s'", event.ToolCallID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for completion callback")
	}

	// Verify operation is still in map but marked as completed
	op := svc.GetOperation("tc-lifecycle")
	if op == nil {
		t.Fatal("expected operation to still exist")
	}
	if op.Status != "cancelled" {
		t.Errorf("expected status 'cancelled', got '%s'", op.Status)
	}
	if op.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

// TestAsyncFlow_MultipleOperationsParallel verifies handling of multiple concurrent operations.
func TestAsyncFlow_MultipleOperationsParallel(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	const numOps = 10
	var wg sync.WaitGroup

	// Subscribe to updates
	sub := svc.SubscribeWithID("chat-parallel")
	defer svc.UnsubscribeByID(sub)

	// Start multiple operations concurrently
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc.AddTestOperation(&AsyncOperation{
				ToolCallID:    idString("tc-parallel", id),
				ChatID:        "chat-parallel",
				ToolName:      "test-tool",
				Status:        "running",
				StartedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			})
		}(i)
	}
	wg.Wait()

	// Verify all operations are tracked
	active := svc.GetActiveOperations("chat-parallel")
	if len(active) != numOps {
		t.Errorf("expected %d active operations, got %d", numOps, len(active))
	}

	// Send updates for all operations
	for i := 0; i < numOps; i++ {
		svc.pushUpdateData("chat-parallel", AsyncStatusUpdate{
			ToolCallID: idString("tc-parallel", i),
			ChatID:     "chat-parallel",
			Status:     "running",
			UpdatedAt:  time.Now(),
		})
	}

	// Drain and count updates
	updateCount := 0
	timeout := time.After(500 * time.Millisecond)
drainLoop:
	for {
		select {
		case <-sub.Channel:
			updateCount++
			if updateCount == numOps {
				break drainLoop
			}
		case <-timeout:
			break drainLoop
		}
	}

	if updateCount != numOps {
		t.Errorf("expected %d updates, got %d", numOps, updateCount)
	}

	// Stop all operations concurrently
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc.StopTracking(idString("tc-parallel", id))
		}(i)
	}
	wg.Wait()

	// Verify all operations are marked as cancelled
	for i := 0; i < numOps; i++ {
		op := svc.GetOperation(idString("tc-parallel", i))
		if op == nil {
			t.Errorf("operation %d not found", i)
			continue
		}
		if op.Status != "cancelled" {
			t.Errorf("operation %d: expected status 'cancelled', got '%s'", i, op.Status)
		}
	}
}

// TestAsyncFlow_CleanupRemovesStaleOperations verifies cleanup of old operations.
func TestAsyncFlow_CleanupRemovesStaleOperations(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	now := time.Now()
	old := now.Add(-2 * time.Hour)
	recent := now.Add(-5 * time.Minute)

	// Add a mix of old and recent operations
	svc.AddTestOperation(&AsyncOperation{
		ToolCallID:  "tc-old-1",
		ChatID:      "chat-cleanup",
		Status:      "completed",
		CompletedAt: &old,
		UpdatedAt:   old,
	})
	svc.AddTestOperation(&AsyncOperation{
		ToolCallID:  "tc-old-2",
		ChatID:      "chat-cleanup",
		Status:      "failed",
		CompletedAt: &old,
		UpdatedAt:   old,
	})
	svc.AddTestOperation(&AsyncOperation{
		ToolCallID:  "tc-recent",
		ChatID:      "chat-cleanup",
		Status:      "completed",
		CompletedAt: &recent,
		UpdatedAt:   recent,
	})
	svc.AddTestOperation(&AsyncOperation{
		ToolCallID: "tc-running",
		ChatID:     "chat-cleanup",
		Status:     "running",
		UpdatedAt:  now,
	})

	// Run cleanup with 1 hour retention
	removed := svc.CleanupStaleOperations(time.Hour)

	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	// Verify correct operations remain
	if svc.GetOperation("tc-old-1") != nil {
		t.Error("tc-old-1 should have been removed")
	}
	if svc.GetOperation("tc-old-2") != nil {
		t.Error("tc-old-2 should have been removed")
	}
	if svc.GetOperation("tc-recent") == nil {
		t.Error("tc-recent should remain")
	}
	if svc.GetOperation("tc-running") == nil {
		t.Error("tc-running should remain")
	}

	// Total count should be 2
	if svc.GetOperationCount() != 2 {
		t.Errorf("expected 2 operations, got %d", svc.GetOperationCount())
	}
}

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

// TestAsyncTracker_NoGoroutineLeakOnStop verifies goroutines are cleaned up.
func TestAsyncTracker_NoGoroutineLeakOnStop(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	svc := NewAsyncTrackerService(nil, nil)

	// Create operations with cancel funcs
	const ops = 10
	for i := 0; i < ops; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		svc.mu.Lock()
		svc.operations[idString("tc-leak", i)] = &AsyncOperation{
			ToolCallID: idString("tc-leak", i),
			ChatID:     "chat-leak",
			Status:     "running",
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					PollIntervalSeconds: 1,
				},
			},
		}
		svc.cancelFuncs[idString("tc-leak", i)] = cancel
		svc.mu.Unlock()

		// Simulate a goroutine that would be spawned
		go func(ctx context.Context) {
			<-ctx.Done()
		}(ctx)
	}

	// Stop all operations
	for i := 0; i < ops; i++ {
		svc.StopTracking(idString("tc-leak", i))
	}

	// Allow goroutines to clean up
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	// Check goroutine count
	final := runtime.NumGoroutine()
	leaked := final - baseline

	// Allow for some variance due to runtime goroutines
	if leaked > 2 {
		t.Errorf("potential goroutine leak: started with %d, ended with %d (delta: %d)",
			baseline, final, leaked)
	}
}

// TestAsyncTracker_ConcurrentOperationAccess verifies concurrent read/write of operations.
func TestAsyncTracker_ConcurrentOperationAccess(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	// Pre-populate some operations
	for i := 0; i < 10; i++ {
		svc.AddTestOperation(&AsyncOperation{
			ToolCallID: idString("tc-access", i),
			ChatID:     "chat-access",
			Status:     "running",
			UpdatedAt:  time.Now(),
		})
	}

	var wg sync.WaitGroup
	const readers = 20
	const writers = 5

	// Start concurrent readers
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = svc.GetActiveOperations("chat-access")
				_ = svc.GetOperation(idString("tc-access", j%10))
				_ = svc.GetOperationCount()
			}
		}()
	}

	// Start concurrent writers
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				svc.AddTestOperation(&AsyncOperation{
					ToolCallID: idString("tc-access-new", id*20+j),
					ChatID:     "chat-access",
					Status:     "running",
					UpdatedAt:  time.Now(),
				})
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	count := svc.GetOperationCount()
	if count != 10+(writers*20) {
		t.Errorf("expected %d operations, got %d", 10+(writers*20), count)
	}
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
