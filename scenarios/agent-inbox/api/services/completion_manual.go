// Package services contains business logic orchestration.
// This file handles manual tool execution and async tool result saving.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"agent-inbox/domain"
)

// ManualExecutionResult contains the result of manually executing a tool.
type ManualExecutionResult struct {
	// Result is the tool's return value (JSON encoded).
	Result interface{} `json:"result"`
	// Status is "completed" or "failed".
	Status string `json:"status"`
	// Error contains the error message if failed.
	Error string `json:"error,omitempty"`
	// ExecutionTimeMs is how long execution took.
	ExecutionTimeMs int64 `json:"execution_time_ms"`
	// ToolCallRecord is set if chat_id was provided (tool call added to chat).
	ToolCallRecord *domain.ToolCallRecord `json:"tool_call_record,omitempty"`
}

// ExecuteToolManually executes a tool directly without going through the AI.
// If chatID is provided, the tool call and result are added to the chat history.
// If chatID is empty, the tool is executed standalone without persistence.
func (s *CompletionService) ExecuteToolManually(ctx context.Context, chatID, scenario, toolName string, arguments map[string]interface{}) (*ManualExecutionResult, error) {
	startTime := time.Now()

	argsJSON, err := json.Marshal(arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize arguments: %w", err)
	}

	toolCallID := fmt.Sprintf("manual_%s_%d", toolName, time.Now().UnixNano())

	// If chatID is provided, create a synthetic message for the tool call
	var messageID string
	if chatID != "" {
		messageID, err = s.createManualToolCallMessage(ctx, chatID, toolName, toolCallID, string(argsJSON))
		if err != nil {
			return nil, err
		}
	}

	// Execute the tool
	record, execErr := s.executor.ExecuteTool(ctx, chatID, toolCallID, toolName, string(argsJSON))
	executionTime := time.Since(startTime).Milliseconds()

	// Build result
	result := s.buildManualExecutionResult(ctx, chatID, toolName, toolCallID, record, execErr, executionTime)

	// If chatID was provided, save the record and response message
	if chatID != "" && messageID != "" {
		s.persistManualToolResult(ctx, chatID, messageID, toolCallID, record, result)
	}

	return result, nil
}

// createManualToolCallMessage creates a synthetic assistant message for manual tool invocation.
func (s *CompletionService) createManualToolCallMessage(ctx context.Context, chatID, toolName, toolCallID, argsJSON string) (string, error) {
	syntheticContent := fmt.Sprintf("[Manual tool execution: %s]", toolName)
	toolCall := domain.ToolCall{
		ID:   toolCallID,
		Type: "function",
		Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{
			Name:      toolName,
			Arguments: argsJSON,
		},
	}
	msg, err := s.repo.SaveAssistantMessageWithToolCalls(
		ctx, chatID, "", syntheticContent, []domain.ToolCall{toolCall},
		"", "manual_tool_call", 0, "",
	)
	if err != nil {
		return "", fmt.Errorf("failed to save tool call message: %w", err)
	}
	return msg.ID, nil
}

// buildManualExecutionResult constructs the result from a manual tool execution.
func (s *CompletionService) buildManualExecutionResult(
	ctx context.Context,
	chatID, toolName, toolCallID string,
	record *domain.ToolCallRecord,
	execErr error,
	executionTime int64,
) *ManualExecutionResult {
	result := &ManualExecutionResult{
		ExecutionTimeMs: executionTime,
	}

	if execErr != nil {
		result.Status = domain.StatusFailed
		result.Error = execErr.Error()
		result.Result = map[string]string{"error": execErr.Error()}
	} else {
		result.Status = domain.StatusCompleted
		var parsedResult interface{}
		if err := json.Unmarshal([]byte(record.Result), &parsedResult); err != nil {
			result.Result = record.Result
		} else {
			result.Result = parsedResult
		}

		// Start async tracking if applicable
		if chatID != "" && s.asyncTracker != nil {
			s.maybeStartAsyncTracking(ctx, chatID, toolCallID, toolName, record)
		}
	}

	return result
}

// persistManualToolResult saves the manual tool call record and response message.
func (s *CompletionService) persistManualToolResult(
	ctx context.Context,
	chatID, messageID, toolCallID string,
	record *domain.ToolCallRecord,
	result *ManualExecutionResult,
) {
	record.MessageID = messageID
	if saveErr := s.repo.SaveToolCallRecord(ctx, messageID, record); saveErr != nil {
		log.Printf("warning: failed to save manual tool call record: %v", saveErr)
	}

	toolMsg, toolMsgErr := s.repo.SaveToolResponseMessage(ctx, chatID, toolCallID, record.Result, messageID)
	if toolMsgErr != nil {
		log.Printf("warning: failed to save manual tool response message: %v", toolMsgErr)
	} else if toolMsg != nil {
		_ = s.repo.SetActiveLeaf(ctx, chatID, toolMsg.ID)
	}

	result.ToolCallRecord = record
}

// SaveToolResult saves a tool execution result as a tool response message.
// Used for async tool results that need to be injected into the conversation.
func (s *CompletionService) SaveToolResult(ctx context.Context, chatID string, result domain.ToolExecutionResult, parentMessageID string) error {
	resultJSON := formatToolResultJSON(result)

	toolMsg, err := s.repo.SaveToolResponseMessage(ctx, chatID, result.ToolCallID, resultJSON, parentMessageID)
	if err != nil {
		return fmt.Errorf("failed to save tool response message: %w", err)
	}

	if toolMsg != nil {
		if err := s.repo.SetActiveLeaf(ctx, chatID, toolMsg.ID); err != nil {
			log.Printf("[WARN] Failed to set active leaf for async tool result: %v", err)
		}
	}

	return nil
}

// formatToolResultJSON converts a ToolExecutionResult to a JSON string.
func formatToolResultJSON(result domain.ToolExecutionResult) string {
	if result.Error != "" {
		return fmt.Sprintf(`{"status": %q, "error": %q}`, result.Status, result.Error)
	}
	resultBytes, err := json.Marshal(result.Result)
	if err != nil {
		return fmt.Sprintf(`{"status": %q, "result": "marshal error: %s"}`, result.Status, err.Error())
	}
	return fmt.Sprintf(`{"status": %q, "result": %s}`, result.Status, string(resultBytes))
}
