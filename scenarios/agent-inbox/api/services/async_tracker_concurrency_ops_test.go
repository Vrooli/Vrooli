package services

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

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
			AsyncBehavior: &AsyncBehavior{
				StatusPolling: &StatusPolling{
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
