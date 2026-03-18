package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// StartTracking begins tracking an async tool operation.
// This should be called after executing a tool that has AsyncBehavior defined.
func (s *AsyncTrackerService) StartTracking(
	ctx context.Context,
	toolCallID string,
	chatID string,
	toolName string,
	scenario string,
	toolResult interface{},
	asyncBehavior *toolspb.AsyncBehavior,
) error {
	if asyncBehavior == nil || asyncBehavior.StatusPolling == nil {
		return fmt.Errorf("no async behavior configuration provided")
	}

	// Extract the operation ID from the tool result
	externalRunID, err := s.extractOperationID(toolResult, asyncBehavior.StatusPolling.OperationIdField)
	if err != nil {
		return fmt.Errorf("failed to extract operation ID: %w", err)
	}

	// Create the operation record
	op := &AsyncOperation{
		ToolCallID:    toolCallID,
		ChatID:        chatID,
		ToolName:      toolName,
		Scenario:      scenario,
		ExternalRunID: externalRunID,
		AsyncBehavior: asyncBehavior,
		Status:        AsyncStatusPending,
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Store the operation and build initial update while holding lock
	s.mu.Lock()
	s.operations[toolCallID] = op
	initialUpdate := buildUpdateFromOp(op, false)
	repo := s.repo // Capture repo reference while holding lock
	s.mu.Unlock()

	// Persist to database if repository is configured (graceful degradation if fails)
	if repo != nil {
		asyncBehaviorJSON, err := json.Marshal(asyncBehavior)
		if err != nil {
			log.Printf("[WARN] Failed to marshal async behavior for %s: %v", toolCallID, err)
		} else {
			record := &AsyncOperationRecord{
				ToolCallID:    toolCallID,
				ChatID:        chatID,
				ToolName:      toolName,
				ScenarioName:  scenario,
				OperationID:   externalRunID,
				Status:        AsyncStatusPending,
				AsyncBehavior: asyncBehaviorJSON,
				StartedAt:     op.StartedAt,
				UpdatedAt:     op.UpdatedAt,
			}
			if err := repo.CreateAsyncOperation(ctx, record); err != nil {
				log.Printf("[WARN] Failed to persist async operation %s: %v", toolCallID, err)
				// Continue with in-memory tracking
			}
		}
	}

	// Push initial update to subscribers so UI shows the operation immediately
	s.pushUpdateData(chatID, initialUpdate)

	// Create cancellable context for polling
	pollCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancelFuncs[toolCallID] = cancel
	s.mu.Unlock()

	// Start background polling
	go s.pollLoop(pollCtx, op)

	log.Printf("Started async tracking for %s/%s (toolCallID=%s, runID=%s)",
		scenario, toolName, toolCallID, externalRunID)

	return nil
}

// StopTracking cancels tracking for an operation and marks it as cancelled.
// The operation is kept in memory for a grace period to allow clients to query its status.
func (s *AsyncTrackerService) StopTracking(toolCallID string) {
	s.mu.Lock()
	// Cancel the polling goroutine
	if cancel, ok := s.cancelFuncs[toolCallID]; ok {
		cancel()
		delete(s.cancelFuncs, toolCallID)
	}

	// Mark the operation as cancelled if it exists and hasn't completed yet
	op := s.operations[toolCallID]
	if op != nil && op.CompletedAt == nil {
		op.Status = AsyncStatusCancelled
		op.Error = "Operation cancelled"
		now := time.Now()
		op.CompletedAt = &now
		op.UpdatedAt = now
	}
	repo := s.repo // Capture repo reference while holding lock
	s.mu.Unlock()

	// Persist the cancellation to database
	if repo != nil && op != nil {
		record := &AsyncOperationRecord{
			ToolCallID:  toolCallID,
			Status:      op.Status,
			Error:       op.Error,
			UpdatedAt:   op.UpdatedAt,
			CompletedAt: op.CompletedAt,
		}
		if err := repo.UpdateAsyncOperation(context.Background(), record); err != nil {
			log.Printf("[WARN] Failed to persist cancelled operation %s: %v", toolCallID, err)
		}
	}

	// Trigger completion callback outside of lock
	if op != nil {
		s.triggerCompletionCallback(op, AsyncStatusCancelled)
	}
}
