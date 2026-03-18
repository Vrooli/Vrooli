// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains SSE event writers for async operations and tool calls.
package handlers

import "agent-inbox/domain"

// WriteToolCallStart sends a tool call start event.
func (sw *StreamWriter) WriteToolCallStart(tc domain.ToolCall) {
	sw.WriteEvent(map[string]interface{}{
		"type":          "tool_call_start",
		"tool_name":     tc.Function.Name,
		"tool_id":       tc.ID,
		"arguments":     tc.Function.Arguments,
		"completion_id": sw.completionID,
	})
}

// WriteToolCallResult sends a tool call result event.
func (sw *StreamWriter) WriteToolCallResult(result domain.ToolExecutionResult) {
	event := map[string]interface{}{
		"type":          "tool_call_result",
		"tool_name":     result.ToolName,
		"tool_id":       result.ToolCallID,
		"status":        result.Status,
		"completion_id": sw.completionID,
	}
	if result.Error != "" {
		event["error"] = result.Error
	} else {
		event["result"] = result.Result
	}
	if result.DeactivateTemplate {
		event["deactivate_template"] = true
	}
	sw.WriteEvent(event)
}

// WriteToolCallsComplete signals that all tool calls finished.
func (sw *StreamWriter) WriteToolCallsComplete() {
	sw.WriteEvent(map[string]interface{}{
		"type":          "tool_calls_complete",
		"continuing":    true,
		"completion_id": sw.completionID,
	})
}

// WriteToolCallPendingApproval sends an event indicating a tool requires approval.
func (sw *StreamWriter) WriteToolCallPendingApproval(record *domain.ToolCallRecord) {
	sw.WriteEvent(map[string]interface{}{
		"type":          "tool_pending_approval",
		"tool_call_id":  record.ID,
		"tool_name":     record.ToolName,
		"arguments":     record.Arguments,
		"completion_id": sw.completionID,
	})
}

// WriteAwaitingApprovals signals that tool calls are waiting for user approval.
func (sw *StreamWriter) WriteAwaitingApprovals() {
	sw.WriteEvent(map[string]interface{}{
		"type":          "awaiting_approvals",
		"continuing":    false,
		"completion_id": sw.completionID,
	})
}

// WriteAsyncWaiting signals that async tool operations are running.
// The AI conversation is paused while waiting for these operations to complete.
// Operations will be automatically injected when they finish.
func (sw *StreamWriter) WriteAsyncWaiting(operations []domain.AsyncOperationInfo) {
	ops := make([]map[string]interface{}, len(operations))
	for i, op := range operations {
		ops[i] = map[string]interface{}{
			"tool_call_id": op.ToolCallID,
			"tool_name":    op.ToolName,
			"run_id":       op.RunID,
			"scenario":     op.Scenario,
		}
	}
	sw.WriteEvent(map[string]interface{}{
		"type":          "async_waiting",
		"operations":    ops,
		"message":       "Tools are running asynchronously. Results will be injected automatically when complete.",
		"completion_id": sw.completionID,
	})
}

// WriteAsyncProgress sends a progress update for an async operation.
func (sw *StreamWriter) WriteAsyncProgress(toolCallID string, status string, progress *int, message string) {
	event := map[string]interface{}{
		"type":          "async_progress",
		"tool_call_id":  toolCallID,
		"status":        status,
		"completion_id": sw.completionID,
	}
	if progress != nil {
		event["progress"] = *progress
	}
	if message != "" {
		event["message"] = message
	}
	sw.WriteEvent(event)
}

// WriteAsyncCompleted signals that an async operation has completed.
// The result is automatically injected into the conversation.
func (sw *StreamWriter) WriteAsyncCompleted(toolCallID, toolName, status string, result interface{}, errMsg string) {
	event := map[string]interface{}{
		"type":          "async_completed",
		"tool_call_id":  toolCallID,
		"tool_name":     toolName,
		"status":        status,
		"completion_id": sw.completionID,
	}
	if result != nil {
		event["result"] = result
	}
	if errMsg != "" {
		event["error"] = errMsg
	}
	sw.WriteEvent(event)
}

// WriteSystemMessage sends a system message to be displayed to the user.
// Used for informational messages like async waiting notifications.
func (sw *StreamWriter) WriteSystemMessage(message string) {
	sw.WriteEvent(map[string]interface{}{
		"type":          "system_message",
		"message":       message,
		"completion_id": sw.completionID,
	})
}
