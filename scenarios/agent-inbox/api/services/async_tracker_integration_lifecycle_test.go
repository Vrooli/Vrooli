package services

import (
	"sync"
	"testing"
	"time"
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
				ToolCallID: idString("tc-parallel", id),
				ChatID:     "chat-parallel",
				ToolName:   "test-tool",
				Status:     "running",
				StartedAt:  time.Now(),
				UpdatedAt:  time.Now(),
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
