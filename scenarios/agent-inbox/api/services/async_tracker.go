// Package services provides application services for the Agent Inbox scenario.
//
// This file implements the AsyncTrackerService for tracking long-running tool
// operations and providing status updates via Server-Sent Events (SSE).
//
// # ARCHITECTURE OVERVIEW
//
// The async tracking system enables tools to run asynchronously while keeping
// the AI conversation loop informed of progress and results. This is essential
// for long-running operations like code generation, browser automation, or
// data processing.
//
// ## Key Components
//
//   - AsyncTrackerService: Central coordinator managing operation lifecycle
//   - AsyncOperation: State container for a single tracked operation
//   - Subscription: SSE channel for real-time UI updates
//   - CompletionCallback: Channel for AI conversation loop notification
//
// ## Data Flow
//
//	┌─────────────────────────────────────────────────────────────────────┐
//	│                    AI Requests Tool                                  │
//	│                          │                                           │
//	│                          ▼                                           │
//	│            ┌─────────────────────────────┐                          │
//	│            │   Tool Execution Returns    │                          │
//	│            │   { run_id: "abc123" }      │                          │
//	│            └─────────────┬───────────────┘                          │
//	│                          │                                           │
//	│                          ▼                                           │
//	│            ┌─────────────────────────────┐                          │
//	│            │    StartTracking() called   │                          │
//	│            │    - Extracts run_id        │                          │
//	│            │    - Starts poll goroutine  │                          │
//	│            └─────────────┬───────────────┘                          │
//	│                          │                                           │
//	│           ┌──────────────┴──────────────┐                           │
//	│           ▼                              ▼                           │
//	│  ┌─────────────────┐          ┌─────────────────┐                   │
//	│  │ SSE Subscribers │          │  Poll Loop      │                   │
//	│  │ (UI updates)    │◀─────────│  (background)   │                   │
//	│  └─────────────────┘  updates └────────┬────────┘                   │
//	│                                        │                             │
//	│                                        ▼                             │
//	│                          ┌─────────────────────────┐                │
//	│                          │ Calls status tool       │                │
//	│                          │ repeatedly until done   │                │
//	│                          └─────────────┬───────────┘                │
//	│                                        │                             │
//	│                                        ▼                             │
//	│                          ┌─────────────────────────┐                │
//	│                          │ On terminal status:     │                │
//	│                          │ - Notify SSE subs       │                │
//	│                          │ - Trigger completion CB │                │
//	│                          │ - AI loop continues     │                │
//	│                          └─────────────────────────┘                │
//	└─────────────────────────────────────────────────────────────────────┘
//
// ## Subscription Systems
//
// There are two notification systems serving different consumers:
//
// 1. SSE Subscribers (SubscribeWithID/UnsubscribeByID):
//   - Used by UI clients for real-time progress display
//   - Buffered channels prevent blocking on slow consumers
//   - ID-based tracking for safe unsubscription
//
// 2. Completion Callbacks (RegisterCompletionCallback/UnregisterCompletionCallback):
//   - Used by AI conversation loop to wait for results
//   - One callback per chat (multiple operations fan into same channel)
//   - Enables auto-continuation after async tools complete
//
// Note: The deprecated Subscribe/Unsubscribe methods use pointer comparison
// which is fragile. Use SubscribeWithID/UnsubscribeByID for new code.
//
// ## Concurrency Model
//
// Operations are accessed from multiple goroutines:
//   - HTTP handlers (read operations, start tracking)
//   - Poll goroutines (update status, trigger callbacks)
//   - Cleanup routine (remove stale operations)
//
// Thread safety is ensured via:
//   - sync.RWMutex for operation map access
//   - OperationSnapshot for lock-free polling config access
//   - Non-blocking channel sends (with logging on full channels)
//
// ## Configuration
//
// Tunable parameters are defined in async_config.go:
//   - Poll intervals and timeouts
//   - Channel buffer sizes
//   - Cleanup intervals and retention
//
// ## File Organization
//
// The async tracker is split across several files:
//   - async_tracker.go: Package documentation and recovery/lifecycle methods
//   - async_tracker_types.go: Type definitions, constructor, and configuration
//   - async_tracker_operations.go: Operation lifecycle (start, stop, cancel, query)
//   - async_tracker_polling.go: Background polling loop and status processing
//   - async_tracker_subscriptions.go: SSE subscriptions and completion callbacks
//   - async_config.go: Configuration constants
//
// ## Testing
//
// The AsyncTrackerInterface in interfaces.go enables mocking for unit tests.
// Integration tests can use AddTestOperation() to inject test data.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// RecoverOperations loads active operations from the database and resumes polling.
// Called during service initialization for crash recovery.
// Uses fresh status check approach - queries current status before resuming polling.
func (s *AsyncTrackerService) RecoverOperations(ctx context.Context) error {
	s.mu.RLock()
	repo := s.repo
	s.mu.RUnlock()

	if repo == nil {
		return nil // No repository configured, nothing to recover
	}

	records, err := repo.GetAllActiveAsyncOperations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active operations for recovery: %w", err)
	}

	if len(records) == 0 {
		log.Printf("[INFO] No async operations to recover")
		return nil
	}

	log.Printf("[INFO] Recovering %d async operations", len(records))

	for _, record := range records {
		// Reconstruct the AsyncOperation from the record
		var asyncBehavior AsyncBehavior
		if err := json.Unmarshal(record.AsyncBehavior, &asyncBehavior); err != nil {
			log.Printf("[WARN] Failed to unmarshal async behavior for %s: %v", record.ToolCallID, err)
			continue
		}

		op := &AsyncOperation{
			ToolCallID:    record.ToolCallID,
			ChatID:        record.ChatID,
			ToolName:      record.ToolName,
			Scenario:      record.ScenarioName,
			ExternalRunID: record.OperationID,
			AsyncBehavior: &asyncBehavior,
			Status:        record.Status,
			Progress:      record.Progress,
			Message:       record.Message,
			Phase:         record.Phase,
			Error:         record.Error,
			StartedAt:     record.StartedAt,
			UpdatedAt:     record.UpdatedAt,
			CompletedAt:   record.CompletedAt,
		}

		// Parse result if present
		if len(record.Result) > 0 {
			var result interface{}
			if err := json.Unmarshal(record.Result, &result); err == nil {
				op.Result = result
			}
		}

		// Store in memory
		s.mu.Lock()
		s.operations[record.ToolCallID] = op
		s.mu.Unlock()

		// Start recovery goroutine with fresh status check
		go s.recoverOperation(ctx, op)
	}

	return nil
}

// recoverOperation performs a fresh status check and resumes polling if still active.
// This is safer than resuming mid-poll as it gets the current state from the external service.
func (s *AsyncTrackerService) recoverOperation(ctx context.Context, op *AsyncOperation) {
	log.Printf("[INFO] Recovering operation %s (status=%s)", op.ToolCallID, op.Status)

	// Create snapshot for the status check
	snap, ok := s.snapshotOperation(op.ToolCallID)
	if !ok {
		log.Printf("[WARN] Recovery: operation %s disappeared before recovery", op.ToolCallID)
		return
	}

	// Perform immediate fresh status check
	statusResult, err := s.callStatusToolWithSnapshot(ctx, snap)
	if err != nil {
		log.Printf("[WARN] Recovery status check failed for %s: %v", op.ToolCallID, err)
		// Continue to poll loop anyway - it will handle retries
	} else {
		// Process the result - this may mark as complete if operation finished while we were down
		conditions := snap.AsyncBehavior.CompletionConditions
		isTerminal, status := s.processStatusResult(op, statusResult, conditions)
		if isTerminal {
			log.Printf("[INFO] Recovery: operation %s already completed (status=%s)", op.ToolCallID, status)
			return
		}
	}

	// Create cancellable context for polling
	pollCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancelFuncs[op.ToolCallID] = cancel
	s.mu.Unlock()

	// Start polling loop
	s.pollLoop(pollCtx, op)
}

// Shutdown gracefully stops all active polling and persists state.
// Should be called before server shutdown.
func (s *AsyncTrackerService) Shutdown(ctx context.Context) error {
	log.Printf("[INFO] AsyncTrackerService shutting down...")

	s.mu.Lock()
	repo := s.repo
	toolCallIDs := make([]string, 0, len(s.cancelFuncs))

	// Cancel all polling goroutines
	for toolCallID, cancel := range s.cancelFuncs {
		toolCallIDs = append(toolCallIDs, toolCallID)
		cancel()
	}
	s.mu.Unlock()

	// Mark interrupted operations in database (outside lock to avoid blocking)
	if repo != nil {
		for _, toolCallID := range toolCallIDs {
			if err := repo.UpdateAsyncOperationStatus(ctx, toolCallID, "polling"); err != nil {
				log.Printf("[WARN] Failed to update status for %s during shutdown: %v", toolCallID, err)
			}
		}
	}

	log.Printf("[INFO] AsyncTrackerService shutdown complete (%d operations paused)", len(toolCallIDs))
	return nil
}
