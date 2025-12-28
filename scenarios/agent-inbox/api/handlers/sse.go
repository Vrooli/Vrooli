// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains Server-Sent Events (SSE) streaming utilities.
//
// SSE Error Design:
//   - Error events include machine-readable codes for automated handling
//   - Recovery hints guide clients on next actions
//   - Fatal vs recoverable errors are clearly distinguished
//   - Request IDs enable correlation with server logs
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"agent-inbox/domain"
)

// StreamWriter wraps http.ResponseWriter for SSE event emission.
// It provides typed methods for sending different event types to clients.
//
// TEMPORAL FLOW DESIGN:
// - completionID enables client-side correlation of events from the same completion
// - requestID enables log correlation for debugging
// - All events include these IDs for traceability across async operations
type StreamWriter struct {
	w            http.ResponseWriter
	flusher      http.Flusher
	requestID    string
	completionID string
}

// SetupSSEResponse configures the response for Server-Sent Events.
// Returns nil if streaming is not supported by the response writer.
//
// A unique completionID is generated for each SSE stream to enable
// client-side correlation of events from the same completion request.
func SetupSSEResponse(w http.ResponseWriter, r *http.Request) *StreamWriter {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}

	return &StreamWriter{
		w:            w,
		flusher:      flusher,
		requestID:    GetRequestID(r),
		completionID: generateCompletionID(),
	}
}

// generateCompletionID creates a unique ID for tracking a completion stream.
// Uses timestamp + random suffix for uniqueness without external dependencies.
func generateCompletionID() string {
	return fmt.Sprintf("cmp_%d", time.Now().UnixNano())
}

// StreamingEvent is the base structure for all streaming events.
// All events include a type field for client-side routing.
//
// TEMPORAL FLOW: CompletionID is included in all events to enable
// client-side guards against stale events from cancelled requests.
type StreamingEvent struct {
	// Type identifies the event kind for client-side handling.
	Type string `json:"type"`

	// CompletionID uniquely identifies this completion stream.
	// Clients should verify this matches their current request.
	CompletionID string `json:"completion_id,omitempty"`

	// RequestID enables log correlation (included in error events).
	RequestID string `json:"request_id,omitempty"`
}

// StreamingErrorEvent is a structured error event for SSE.
type StreamingErrorEvent struct {
	StreamingEvent

	// Code is a machine-readable error identifier.
	Code string `json:"code"`

	// Category groups the error type.
	Category string `json:"category"`

	// Message is a user-friendly error description.
	Message string `json:"message"`

	// Recovery suggests what action to take next.
	Recovery string `json:"recovery"`

	// Fatal indicates if the stream should be terminated.
	Fatal bool `json:"fatal"`

	// Details provides additional context when available.
	Details map[string]interface{} `json:"details,omitempty"`
}

// WriteEvent sends a JSON event to the stream.
func (sw *StreamWriter) WriteEvent(data interface{}) {
	eventData, _ := json.Marshal(data)
	fmt.Fprintf(sw.w, "data: %s\n\n", eventData)
	sw.flusher.Flush()
}

// WriteContentChunk sends a content chunk event.
func (sw *StreamWriter) WriteContentChunk(content string) {
	sw.WriteEvent(map[string]interface{}{
		"content":       content,
		"type":          "content",
		"completion_id": sw.completionID,
	})
}

// WriteImageGenerated sends an event when an AI-generated image is received.
func (sw *StreamWriter) WriteImageGenerated(imageURL string) {
	log.Printf("[DEBUG] WriteImageGenerated: sending image_generated event (url length: %d)", len(imageURL))
	sw.WriteEvent(map[string]interface{}{
		"type":          "image_generated",
		"image_url":     imageURL,
		"completion_id": sw.completionID,
	})
}

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

// WriteError sends a structured error event.
// For backwards compatibility, also includes the error string.
func (sw *StreamWriter) WriteError(err error) {
	// Check if it's an AppError for structured output
	if appErr, ok := err.(*domain.AppError); ok {
		sw.WriteAppError(appErr, false)
		return
	}

	// Wrap unknown errors as internal errors
	sw.WriteEvent(StreamingErrorEvent{
		StreamingEvent: StreamingEvent{
			Type:         "error",
			CompletionID: sw.completionID,
			RequestID:    sw.requestID,
		},
		Code:     string(domain.ErrCodeInternalError),
		Category: string(domain.CategoryInternal),
		Message:  err.Error(),
		Recovery: string(domain.ActionRetry),
		Fatal:    false,
	})
}

// WriteAppError sends a structured AppError event.
func (sw *StreamWriter) WriteAppError(appErr *domain.AppError, fatal bool) {
	sw.WriteEvent(StreamingErrorEvent{
		StreamingEvent: StreamingEvent{
			Type:         "error",
			CompletionID: sw.completionID,
			RequestID:    sw.requestID,
		},
		Code:     string(appErr.Code),
		Category: string(appErr.Category),
		Message:  appErr.Message,
		Recovery: string(appErr.Recovery),
		Fatal:    fatal,
		Details:  appErr.Details,
	})
}

// WriteFatalError sends a fatal error event that indicates stream termination.
func (sw *StreamWriter) WriteFatalError(err error) {
	if appErr, ok := err.(*domain.AppError); ok {
		sw.WriteAppError(appErr, true)
	} else {
		sw.WriteEvent(StreamingErrorEvent{
			StreamingEvent: StreamingEvent{
				Type:         "error",
				CompletionID: sw.completionID,
				RequestID:    sw.requestID,
			},
			Code:     string(domain.ErrCodeInternalError),
			Category: string(domain.CategoryInternal),
			Message:  err.Error(),
			Recovery: string(domain.ActionEscalate),
			Fatal:    true,
		})
	}
}

// WriteWarning sends a non-fatal warning event.
// Warnings indicate issues that don't stop the stream.
func (sw *StreamWriter) WriteWarning(code domain.ErrorCode, message string) {
	sw.WriteEvent(map[string]interface{}{
		"type":          "warning",
		"code":          string(code),
		"message":       message,
		"completion_id": sw.completionID,
		"request_id":    sw.requestID,
	})
}

// WriteProgress sends a progress event for long-running operations.
func (sw *StreamWriter) WriteProgress(phase string, message string) {
	sw.WriteEvent(map[string]interface{}{
		"type":          "progress",
		"phase":         phase,
		"message":       message,
		"completion_id": sw.completionID,
	})
}

// WriteDone sends the stream completion marker.
func (sw *StreamWriter) WriteDone() {
	sw.WriteEvent(map[string]interface{}{
		"done":          true,
		"completion_id": sw.completionID,
	})
}
