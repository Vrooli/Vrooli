// Package handlers provides HTTP handlers for the Agent Inbox scenario.
//
// This file implements the SSE endpoint for async tool status streaming.
// Clients connect to this endpoint to receive real-time updates about
// long-running tool operations.
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"agent-inbox/services"

	"github.com/gorilla/mux"
)

// StreamAsyncStatus handles GET /api/v1/chats/{id}/async-status
// This endpoint streams Server-Sent Events for async tool operations.
func (h *Handlers) StreamAsyncStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	if chatID == "" {
		http.Error(w, "chat ID is required", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to updates
	updates := h.AsyncTracker.Subscribe(chatID)
	defer h.AsyncTracker.Unsubscribe(chatID, updates)

	// Send initial state of active operations
	activeOps := h.AsyncTracker.GetActiveOperations(chatID)
	for _, op := range activeOps {
		data, err := json.Marshal(operationToUpdate(op))
		if err != nil {
			log.Printf("[WARN] Failed to marshal async operation status for %s: %v", op.ToolCallID, err)
			continue
		}
		fmt.Fprintf(w, "event: status\ndata: %s\n\n", data)
		flusher.Flush()
	}

	// Send connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"chat_id\":\"%s\"}\n\n", chatID)
	flusher.Flush()

	// Stream updates until client disconnects
	for {
		select {
		case <-r.Context().Done():
			// Client disconnected
			return
		case update, ok := <-updates:
			if !ok {
				// Channel closed
				return
			}

			data, err := json.Marshal(update)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "event: status\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// GetAsyncOperations handles GET /api/v1/chats/{id}/async-operations
// Returns the current state of all async operations for a chat.
func (h *Handlers) GetAsyncOperations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	if chatID == "" {
		http.Error(w, "chat ID is required", http.StatusBadRequest)
		return
	}

	operations := h.AsyncTracker.GetActiveOperations(chatID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"operations": operations,
		"count":      len(operations),
	})
}

// CancelAsyncOperation handles POST /api/v1/chats/{id}/async-operations/{toolCallId}/cancel
// Cancels a running async operation.
func (h *Handlers) CancelAsyncOperation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]
	toolCallID := vars["toolCallId"]

	if chatID == "" || toolCallID == "" {
		http.Error(w, "chat ID and tool call ID are required", http.StatusBadRequest)
		return
	}

	// Verify the operation belongs to this chat
	op := h.AsyncTracker.GetOperation(toolCallID)
	if op == nil {
		http.Error(w, "operation not found", http.StatusNotFound)
		return
	}
	if op.ChatID != chatID {
		http.Error(w, "operation does not belong to this chat", http.StatusForbidden)
		return
	}

	// Cancel the operation
	if err := h.AsyncTracker.CancelOperation(r.Context(), toolCallID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"tool_call_id": toolCallID,
		"message":      "Operation cancelled",
	})
}

// operationToUpdate converts an AsyncOperation to an AsyncStatusUpdate.
func operationToUpdate(op *services.AsyncOperation) services.AsyncStatusUpdate {
	return services.AsyncStatusUpdate{
		ToolCallID: op.ToolCallID,
		ChatID:     op.ChatID,
		ToolName:   op.ToolName,
		Status:     op.Status,
		Progress:   op.Progress,
		Message:    op.Message,
		Phase:      op.Phase,
		Result:     op.Result,
		Error:      op.Error,
		IsTerminal: op.CompletedAt != nil,
		UpdatedAt:  op.UpdatedAt,
	}
}
