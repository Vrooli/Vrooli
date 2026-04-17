package services

import (
	"agent-inbox/domain"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// processStatusResult updates the operation based on status tool response.
// Returns (isTerminal, status) to avoid reading from op after lock release.
//
// The function extracts values from the result using dot-notation paths configured
// in the tool's AsyncBehavior.CompletionConditions:
//   - StatusField: Required. Path to the status string (e.g., "data.run.status")
//   - ErrorField: Optional. Path to error message if status indicates failure
//   - ResultField: Optional. Path to final result data on success
//
// Terminal status is determined by matching the extracted status against
// SuccessValues or FailureValues from the completion conditions.
func (s *AsyncTrackerService) processStatusResult(op *AsyncOperation, result interface{}, conditions *toolspb.CompletionConditions) (bool, string) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		log.Printf("[WARN] processStatusResult: result is not a map for operation %s: %T", op.ToolCallID, result)
		return false, ""
	}

	// Extract status using the configured field path
	status := ExtractStringField(resultMap, conditions.StatusField)
	if status == "" {
		log.Printf("[WARN] processStatusResult: failed to extract status using path %q for operation %s, result: %+v",
			conditions.StatusField, op.ToolCallID, resultMap)
		return false, ""
	}

	log.Printf("[DEBUG] processStatusResult: operation %s status updated to %q", op.ToolCallID, status)

	// Update operation and build update struct inside lock to avoid race
	s.mu.Lock()
	op.Status = status
	op.UpdatedAt = time.Now()

	// Extract progress if configured
	if op.AsyncBehavior.ProgressTracking != nil {
		if progress := ExtractIntField(resultMap, op.AsyncBehavior.ProgressTracking.ProgressField); progress != nil {
			op.Progress = progress
		}
		if message := ExtractStringField(resultMap, op.AsyncBehavior.ProgressTracking.MessageField); message != "" {
			op.Message = message
		}
		if phase := ExtractStringField(resultMap, op.AsyncBehavior.ProgressTracking.PhaseField); phase != "" {
			op.Phase = phase
		}
	}

	// Check for error
	if conditions.ErrorField != "" {
		if errMsg := ExtractStringField(resultMap, conditions.ErrorField); errMsg != "" {
			op.Error = errMsg
		}
	}

	// Check for result
	if conditions.ResultField != "" {
		if resultVal := ExtractField(resultMap, conditions.ResultField); resultVal != nil {
			op.Result = resultVal
		}
	}

	// Check terminal conditions - compare status against configured success/failure values
	isSuccess := ContainsString(conditions.SuccessValues, status)
	isFailure := ContainsString(conditions.FailureValues, status)
	isTerminal := isSuccess || isFailure

	if isTerminal {
		now := time.Now()
		op.CompletedAt = &now
	}

	// Build update struct while holding the lock to avoid race condition
	update := buildUpdateFromOp(op, isTerminal)
	repo := s.repo // Capture repo reference while holding lock
	toolCallID := op.ToolCallID
	chatID := op.ChatID
	s.mu.Unlock()

	// Persist update to database
	if repo != nil {
		var resultJSON json.RawMessage
		if op.Result != nil {
			if data, err := json.Marshal(op.Result); err == nil {
				resultJSON = data
			}
		}
		record := &AsyncOperationRecord{
			ToolCallID:  toolCallID,
			Status:      op.Status,
			Progress:    op.Progress,
			Message:     op.Message,
			Phase:       op.Phase,
			Result:      resultJSON,
			Error:       op.Error,
			UpdatedAt:   op.UpdatedAt,
			CompletedAt: op.CompletedAt,
		}
		if err := repo.UpdateAsyncOperation(context.Background(), record); err != nil {
			log.Printf("[WARN] Failed to persist operation update for %s: %v", toolCallID, err)
		}
	}

	// Push update to subscribers (using pre-built update)
	s.pushUpdateData(chatID, update)

	// If terminal, trigger completion callback for AI conversation resumption
	if isTerminal {
		s.triggerCompletionCallback(op, status)
	}

	return isTerminal, status
}

// handleTimeout marks an operation as timed out.
// Called when polling exceeds the configured MaxPollDurationSeconds.
func (s *AsyncTrackerService) handleTimeout(op *AsyncOperation) {
	s.mu.Lock()
	op.Status = AsyncStatusTimeout
	op.Error = "Operation timed out"
	now := time.Now()
	op.CompletedAt = &now
	op.UpdatedAt = now
	// Build update while holding lock to avoid race
	update := buildUpdateFromOp(op, true)
	chatID := op.ChatID
	toolCallID := op.ToolCallID
	repo := s.repo
	s.mu.Unlock()

	// Persist timeout to database
	if repo != nil {
		record := &AsyncOperationRecord{
			ToolCallID:  toolCallID,
			Status:      op.Status,
			Error:       op.Error,
			UpdatedAt:   op.UpdatedAt,
			CompletedAt: op.CompletedAt,
		}
		if err := repo.UpdateAsyncOperation(context.Background(), record); err != nil {
			log.Printf("[WARN] Failed to persist timeout for %s: %v", toolCallID, err)
		}
	}

	s.pushUpdateData(chatID, update)

	// Trigger completion callback for AI conversation resumption
	s.triggerCompletionCallback(op, AsyncStatusTimeout)

	log.Printf("Operation %s timed out", toolCallID)
}

// callStatusToolWithSnapshot invokes the status tool using immutable snapshot data.
// This avoids race conditions by not reading from the mutable AsyncOperation.
func (s *AsyncTrackerService) callStatusToolWithSnapshot(ctx context.Context, snap *OperationSnapshot) (interface{}, error) {
	polling := snap.AsyncBehavior.StatusPolling

	// Build arguments for status tool
	args := map[string]interface{}{
		polling.StatusToolIdParam: snap.ExternalRunID,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status tool args: %w", err)
	}

	// Execute the status tool
	record, err := s.toolExecutor.ExecuteTool(ctx, snap.ChatID, "", polling.StatusTool, string(argsJSON))
	if err != nil {
		return nil, err
	}
	if record.Status == domain.StatusFailed {
		return nil, fmt.Errorf("status tool failed: %s", record.ErrorMessage)
	}

	// Parse the result
	var result interface{}
	if err := json.Unmarshal([]byte(record.Result), &result); err != nil {
		return nil, fmt.Errorf("failed to parse status result: %w", err)
	}

	return result, nil
}
