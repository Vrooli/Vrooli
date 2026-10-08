// Package services contains business logic orchestration.
// This file handles tool call execution flows.
package services

import (
	"context"
	"fmt"
	"log"

	"agent-inbox/domain"
)

// ToolExecutionOutcome represents the result of attempting to execute tool calls.
// Some tools may execute immediately, others may require approval, and some may run asynchronously.
type ToolExecutionOutcome struct {
	// Results contains execution results for tools that ran immediately.
	Results []domain.ToolExecutionResult
	// PendingApprovals contains tool calls that require user approval.
	PendingApprovals []*domain.ToolCallRecord
	// HasPendingApprovals indicates if any tools are waiting for approval.
	HasPendingApprovals bool
	// AsyncOperations contains info about tools running asynchronously.
	AsyncOperations []domain.AsyncOperationInfo
	// HasAsyncOperations indicates if any tools are running in the background.
	HasAsyncOperations bool
}

// ExecuteToolCalls runs all tool calls from a completion result.
//
// For each tool call, it checks approval requirements, executes if allowed,
// saves records, and starts async tracking if applicable.
// Partial success is supported: individual tool errors are captured in each result.
// The returned error is non-nil if ANY tool call failed.
func (s *CompletionService) ExecuteToolCalls(ctx context.Context, chatID, messageID string, toolCalls []domain.ToolCall, parentMessageID string) (*ToolExecutionOutcome, error) {
	outcome := &ToolExecutionOutcome{
		Results:          make([]domain.ToolExecutionResult, 0, len(toolCalls)),
		PendingApprovals: make([]*domain.ToolCallRecord, 0),
	}
	var executionErrors []error
	var lastToolMsgID string

	for _, tc := range toolCalls {
		result, toolMsgID, err := s.executeSingleToolCall(ctx, chatID, messageID, tc, parentMessageID, outcome)
		if err != nil {
			executionErrors = append(executionErrors, err)
		}
		if toolMsgID != "" {
			lastToolMsgID = toolMsgID
		}
		outcome.Results = append(outcome.Results, result)
	}

	// Update active leaf to the last tool message
	if lastToolMsgID != "" {
		_ = s.repo.SetActiveLeaf(ctx, chatID, lastToolMsgID) // Ignore error: leaf update is best-effort
	}

	// Return aggregated error if any tool failed
	if len(executionErrors) > 0 {
		return outcome, fmt.Errorf("%d of %d tool calls failed: %v", len(executionErrors), len(toolCalls), executionErrors[0])
	}

	return outcome, nil
}

// executeSingleToolCall handles one runtime tool call: execution and persistence.
// Returns the execution result, the tool message ID (if saved), and any error.
func (s *CompletionService) executeSingleToolCall(
	ctx context.Context,
	chatID, messageID string,
	tc domain.ToolCall,
	parentMessageID string,
	outcome *ToolExecutionOutcome,
) (domain.ToolExecutionResult, string, error) {
	return s.executeAndPersistToolCall(ctx, chatID, messageID, tc, parentMessageID, outcome)
}

// handlePendingApproval creates a pending approval record instead of executing.
func (s *CompletionService) handlePendingApproval(
	ctx context.Context,
	chatID, messageID string,
	tc domain.ToolCall,
	outcome *ToolExecutionOutcome,
) domain.ToolExecutionResult {
	record := s.createPendingApprovalRecord(chatID, messageID, tc)
	if messageID != "" {
		if saveErr := s.repo.SaveToolCallRecord(ctx, messageID, record); saveErr != nil {
			log.Printf("[ERROR] Failed to save pending approval record for %s: %v", tc.Function.Name, saveErr)
		}
	}
	outcome.PendingApprovals = append(outcome.PendingApprovals, record)
	outcome.HasPendingApprovals = true

	return domain.ToolExecutionResult{
		ToolCallID: tc.ID,
		ToolName:   tc.Function.Name,
		Status:     domain.StatusPendingApproval,
	}
}

// executeAndPersistToolCall executes a tool and saves the result.
// Returns the execution result, the tool message ID, and any error.
func (s *CompletionService) executeAndPersistToolCall(
	ctx context.Context,
	chatID, messageID string,
	tc domain.ToolCall,
	parentMessageID string,
	outcome *ToolExecutionOutcome,
) (domain.ToolExecutionResult, string, error) {
	// Inject skills as context attachments into tool arguments
	enhancedArgs := s.injectSkillsIntoArgs(tc.Function.Name, tc.Function.Arguments)

	// Execute immediately
	record, err := s.executor.ExecuteTool(ctx, chatID, tc.ID, tc.Function.Name, enhancedArgs)
	var execErr error
	if err != nil {
		execErr = fmt.Errorf("tool %s failed: %w", tc.Function.Name, err)
	}

	// Check for async behavior and start tracking if applicable
	var asyncOpInfo *domain.AsyncOperationInfo
	if err == nil && s.asyncTracker != nil {
		asyncOpInfo = s.maybeStartAsyncTracking(ctx, chatID, tc.ID, tc.Function.Name, record)
		if asyncOpInfo != nil {
			outcome.AsyncOperations = append(outcome.AsyncOperations, *asyncOpInfo)
			outcome.HasAsyncOperations = true
		}
	}

	// Persist the tool call record and response message
	toolMsgID := s.persistToolResult(ctx, chatID, messageID, tc, record, parentMessageID)

	// Build result using centralized factory (with async info if applicable)
	execResult := NewToolExecutionResult(tc.ID, tc.Function.Name, record, err)
	if asyncOpInfo != nil {
		execResult.IsAsync = true
		execResult.AsyncRunID = asyncOpInfo.RunID
	}

	return execResult, toolMsgID, execErr
}

// persistToolResult saves the tool call record and response message.
// Returns the tool message ID if saved successfully.
func (s *CompletionService) persistToolResult(
	ctx context.Context,
	chatID, messageID string,
	tc domain.ToolCall,
	record *domain.ToolCallRecord,
	parentMessageID string,
) string {
	if messageID == "" {
		log.Printf("[WARN] No messageID for tool call %s, skipping record save", tc.Function.Name)
		return ""
	}

	if s.toolPersistence != nil {
		// Use atomic save operation (record + message in one transaction)
		toolMsg, saveErr := s.toolPersistence.SaveToolResultWithoutLeafUpdate(ctx, SaveToolResultParams{
			ChatID:          chatID,
			MessageID:       messageID,
			ToolCallID:      tc.ID,
			Record:          record,
			Result:          record.Result,
			ParentMessageID: parentMessageID,
		})
		if saveErr != nil {
			log.Printf("[ERROR] Failed to atomically save tool result for %s (tool_call_id=%s): %v",
				tc.Function.Name, tc.ID, saveErr)
			return ""
		}
		if toolMsg != nil {
			log.Printf("[DEBUG] Atomically saved tool result: id=%s, tool_call_id=%s, parent=%s",
				toolMsg.ID, tc.ID, parentMessageID)
			return toolMsg.ID
		}
		return ""
	}

	// Fallback to non-atomic saves (legacy behavior)
	if saveErr := s.repo.SaveToolCallRecord(ctx, messageID, record); saveErr != nil {
		log.Printf("[ERROR] Failed to save tool call record for %s: %v", tc.Function.Name, saveErr)
	}

	toolMsg, toolMsgErr := s.repo.SaveToolResponseMessage(ctx, chatID, tc.ID, record.Result, parentMessageID)
	if toolMsgErr != nil {
		log.Printf("[ERROR] Failed to save tool response message for %s (tool_call_id=%s): %v",
			tc.Function.Name, tc.ID, toolMsgErr)
		return ""
	}
	if toolMsg != nil {
		log.Printf("[DEBUG] Saved tool response message: id=%s, tool_call_id=%s, parent=%s",
			toolMsg.ID, tc.ID, parentMessageID)
		return toolMsg.ID
	}
	return ""
}
