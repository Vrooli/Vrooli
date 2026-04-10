// Package handlers contains HTTP request handlers for the agent-inbox API.
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// SSEWriter provides a structured way to write Server-Sent Events.
// It handles proper formatting and concurrent write safety.
//
// Usage:
//
//	writer := NewSSEWriter(w)
//	writer.SetHeaders()
//	writer.SendContent("Hello")
//	writer.SendToolCallStart("call_1", "tool-name", "{}")
//	writer.SendDone()
type SSEWriter struct {
	w  io.Writer
	mu sync.Mutex
}

// NewSSEWriter creates a new SSEWriter that writes to the given io.Writer.
func NewSSEWriter(w io.Writer) *SSEWriter {
	return &SSEWriter{w: w}
}

// SetHeaders configures the HTTP response headers for SSE streaming.
// This should be called before writing any events.
func (s *SSEWriter) SetHeaders() {
	if rw, ok := s.w.(http.ResponseWriter); ok {
		rw.Header().Set("Content-Type", "text/event-stream")
		rw.Header().Set("Cache-Control", "no-cache")
		rw.Header().Set("Connection", "keep-alive")
		rw.Header().Set("X-Accel-Buffering", "no")
	}
}

// writeEvent writes a raw SSE event. Thread-safe.
func (s *SSEWriter) writeEvent(data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := fmt.Fprintf(s.w, "data: %s\n\n", data)
	if flusher, ok := s.w.(http.Flusher); ok && err == nil {
		flusher.Flush()
	}
	return err
}

// writeJSON marshals the event to JSON and writes it. Thread-safe.
func (s *SSEWriter) writeJSON(event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return s.writeEvent(string(data))
}

// SendContent sends a content event with streaming text.
func (s *SSEWriter) SendContent(content string) error {
	return s.writeJSON(map[string]interface{}{
		"type":    "content",
		"content": content,
	})
}

// SendImageGenerated sends an image_generated event.
func (s *SSEWriter) SendImageGenerated(imageURL string) error {
	return s.writeJSON(map[string]interface{}{
		"type":      "image_generated",
		"image_url": imageURL,
	})
}

// SendToolCallStart sends a tool_call_start event.
func (s *SSEWriter) SendToolCallStart(toolID, toolName, arguments string) error {
	return s.writeJSON(map[string]interface{}{
		"type":      "tool_call_start",
		"tool_id":   toolID,
		"tool_name": toolName,
		"arguments": arguments,
	})
}

// SendToolCallResult sends a tool_call_result event.
func (s *SSEWriter) SendToolCallResult(toolID, status, result, errorMsg string) error {
	event := map[string]interface{}{
		"type":    "tool_call_result",
		"tool_id": toolID,
		"status":  status,
	}
	if result != "" {
		event["result"] = result
	}
	if errorMsg != "" {
		event["error"] = errorMsg
	}
	return s.writeJSON(event)
}

// SendToolCallsComplete sends a tool_calls_complete event.
func (s *SSEWriter) SendToolCallsComplete(continuing bool) error {
	return s.writeJSON(map[string]interface{}{
		"type":       "tool_calls_complete",
		"continuing": continuing,
	})
}

// SendToolPendingApproval sends a tool_pending_approval event.
func (s *SSEWriter) SendToolPendingApproval(toolCallID, toolName, arguments string) error {
	return s.writeJSON(map[string]interface{}{
		"type":         "tool_pending_approval",
		"tool_call_id": toolCallID,
		"tool_name":    toolName,
		"arguments":    arguments,
	})
}

// SendAwaitingApprovals sends an awaiting_approvals event.
func (s *SSEWriter) SendAwaitingApprovals() error {
	return s.writeJSON(map[string]interface{}{
		"type": "awaiting_approvals",
	})
}

// SendError sends an error event.
func (s *SSEWriter) SendError(errorMsg, code string) error {
	event := map[string]interface{}{
		"type":  "error",
		"error": errorMsg,
	}
	if code != "" {
		event["code"] = code
	}
	return s.writeJSON(event)
}

// SendWarning sends a warning event.
func (s *SSEWriter) SendWarning(message, code string) error {
	event := map[string]interface{}{
		"type":    "warning",
		"message": message,
	}
	if code != "" {
		event["code"] = code
	}
	return s.writeJSON(event)
}

// SendProgress sends a progress event.
func (s *SSEWriter) SendProgress(phase, message string) error {
	return s.writeJSON(map[string]interface{}{
		"type":    "progress",
		"phase":   phase,
		"message": message,
	})
}

// SendDone sends the [DONE] sentinel to signal end of stream.
func (s *SSEWriter) SendDone() error {
	return s.writeEvent("[DONE]")
}
