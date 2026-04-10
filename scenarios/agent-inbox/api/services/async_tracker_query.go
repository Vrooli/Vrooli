package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CancelOperation cancels a running async operation using the configured cancel tool.
func (s *AsyncTrackerService) CancelOperation(ctx context.Context, toolCallID string) error {
	s.mu.RLock()
	op := s.operations[toolCallID]
	s.mu.RUnlock()

	if op == nil {
		return fmt.Errorf("operation not found: %s", toolCallID)
	}

	// Check if cancellation is configured
	if op.AsyncBehavior == nil || op.AsyncBehavior.Cancellation == nil {
		// No cancel tool configured - just stop tracking
		s.StopTracking(toolCallID)
		return nil
	}

	cancel := op.AsyncBehavior.Cancellation

	// Build arguments for cancel tool
	args := map[string]interface{}{
		cancel.CancelToolIdParam: op.ExternalRunID,
	}
	argsJSON, _ := json.Marshal(args)

	// Execute the cancel tool
	_, err := s.toolExecutor.ExecuteTool(ctx, op.ChatID, "", cancel.CancelTool, string(argsJSON))
	if err != nil {
		return fmt.Errorf("failed to cancel operation: %w", err)
	}

	// Stop tracking
	s.StopTracking(toolCallID)

	return nil
}

// ForceRefresh performs an immediate status poll for an operation, bypassing the normal interval.
// This is useful for manual refresh requests from the UI.
// Returns the updated AsyncStatusUpdate and any error encountered.
func (s *AsyncTrackerService) ForceRefresh(ctx context.Context, toolCallID string) (*AsyncStatusUpdate, error) {
	// Get the operation
	s.mu.RLock()
	op := s.operations[toolCallID]
	s.mu.RUnlock()

	if op == nil {
		return nil, fmt.Errorf("operation not found: %s", toolCallID)
	}

	// Check if operation is already terminal
	if op.CompletedAt != nil {
		// Return current status without polling
		s.mu.RLock()
		update := buildUpdateFromOp(op, true)
		s.mu.RUnlock()
		return &update, nil
	}

	// Get snapshot for polling
	snap, ok := s.snapshotOperation(toolCallID)
	if !ok {
		return nil, fmt.Errorf("failed to snapshot operation: %s", toolCallID)
	}

	// Verify we have the necessary configuration
	if snap.AsyncBehavior == nil || snap.AsyncBehavior.StatusPolling == nil {
		return nil, fmt.Errorf("operation %s does not have status polling configured", toolCallID)
	}

	// Call the status tool immediately
	statusResult, err := s.callStatusToolWithSnapshot(ctx, snap)
	if err != nil {
		// Still return current status with the error
		s.mu.RLock()
		update := buildUpdateFromOp(op, false)
		update.Error = fmt.Sprintf("Refresh failed: %v", err)
		s.mu.RUnlock()
		return &update, nil // Return update with error, not an error return
	}

	// Process the result (this updates the operation and pushes to subscribers)
	conditions := snap.AsyncBehavior.CompletionConditions
	isTerminal, _ := s.processStatusResult(op, statusResult, conditions)

	// Return the updated status
	s.mu.RLock()
	update := buildUpdateFromOp(op, isTerminal)
	s.mu.RUnlock()

	return &update, nil
}

// extractOperationID extracts the operation ID from the tool result.
//
// The fieldPath uses dot notation to navigate nested structures.
// For example, if the tool returns {"data": {"run": {"id": "abc123"}}},
// the fieldPath "data.run.id" would extract "abc123".
func (s *AsyncTrackerService) extractOperationID(result interface{}, fieldPath string) (string, error) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("result is not a map")
	}

	value := ExtractStringField(resultMap, fieldPath)
	if value == "" {
		return "", fmt.Errorf("field %s not found or empty", fieldPath)
	}

	return value, nil
}

// AddTestOperation adds an operation directly to the tracker for testing.
// This bypasses the normal StartTracking flow which requires external dependencies.
func (s *AsyncTrackerService) AddTestOperation(op *AsyncOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.operations[op.ToolCallID] = op
}

// GetOperationCount returns the number of tracked operations (for monitoring/testing).
func (s *AsyncTrackerService) GetOperationCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.operations)
}

// GetOperation returns an operation by ID.
func (s *AsyncTrackerService) GetOperation(toolCallID string) *AsyncOperation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.operations[toolCallID]
}

// GetActiveOperations returns all active operations for a chat.
func (s *AsyncTrackerService) GetActiveOperations(chatID string) []*AsyncOperation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*AsyncOperation
	for _, op := range s.operations {
		if op.ChatID == chatID && op.CompletedAt == nil {
			result = append(result, op)
		}
	}
	return result
}

// RemoveOperation removes an operation from tracking.
// This should only be called after the operation has completed and results have been processed.
func (s *AsyncTrackerService) RemoveOperation(toolCallID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.operations, toolCallID)
	delete(s.cancelFuncs, toolCallID)
}

// CleanupStaleOperations removes completed operations older than the retention duration.
// Returns the number of operations removed.
func (s *AsyncTrackerService) CleanupStaleOperations(retention time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retention)
	var removed int

	for id, op := range s.operations {
		// Only remove completed operations that are older than the retention period
		if op.CompletedAt != nil && op.CompletedAt.Before(cutoff) {
			delete(s.operations, id)
			delete(s.cancelFuncs, id)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("[INFO] Cleaned up %d stale async operations", removed)
	}
	return removed
}

// StartCleanupRoutine starts a background routine that periodically cleans up stale operations.
// Call this once during service initialization.
func (s *AsyncTrackerService) StartCleanupRoutine(ctx context.Context, interval, retention time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("[INFO] Async tracker cleanup routine stopped")
				return
			case <-ticker.C:
				s.CleanupStaleOperations(retention)
			}
		}
	}()
	log.Printf("[INFO] Started async tracker cleanup routine (interval=%v, retention=%v)", interval, retention)
}
