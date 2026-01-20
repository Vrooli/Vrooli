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

	// Subscribe to updates using ID-based tracking (preferred over deprecated pointer comparison)
	sub := h.AsyncTracker.SubscribeWithID(chatID)
	defer h.AsyncTracker.UnsubscribeByID(sub)

	// Send initial state of active operations
	activeOps := h.AsyncTracker.GetActiveOperations(chatID)
	for _, op := range activeOps {
		data, err := json.Marshal(services.BuildUpdateFromOperation(op))
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
		case update, ok := <-sub.Channel:
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

// RefreshAsyncOperation handles POST /api/v1/chats/{id}/async-operations/{toolCallId}/refresh
// Performs an immediate status poll for an operation, bypassing the normal polling interval.
func (h *Handlers) RefreshAsyncOperation(w http.ResponseWriter, r *http.Request) {
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

	// Force refresh the operation
	update, err := h.AsyncTracker.ForceRefresh(r.Context(), toolCallID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(update)
}

// GetAsyncOperationHistory handles GET /api/v1/chats/{id}/async-operations/history
// Returns completed async operations for a chat with pagination.
func (h *Handlers) GetAsyncOperationHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	if chatID == "" {
		http.Error(w, "chat ID is required", http.StatusBadRequest)
		return
	}

	// Parse pagination params
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := parseIntParam(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get completed operations from database
	operations, total, err := h.Repo.GetCompletedAsyncOperationsByChatID(r.Context(), chatID, limit, offset)
	if err != nil {
		log.Printf("[ERROR] GetAsyncOperationHistory: failed to get history for chat %s: %v", chatID, err)
		http.Error(w, "failed to get operation history", http.StatusInternalServerError)
		return
	}

	// Convert to status updates
	var updates []services.AsyncStatusUpdate
	for _, op := range operations {
		updates = append(updates, services.AsyncStatusUpdate{
			ToolCallID: op.ToolCallID,
			ChatID:     op.ChatID,
			ToolName:   op.ToolName,
			Status:     op.Status,
			Progress:   op.Progress,
			Message:    op.Message,
			Phase:      op.Phase,
			Result:     unmarshalJSONOrNil(op.Result),
			Error:      op.Error,
			IsTerminal: true, // History only contains completed operations
			UpdatedAt:  op.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"operations": updates,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"has_more":   offset+len(operations) < total,
	})
}

// parseIntParam parses a string to int, returning error if invalid.
func parseIntParam(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// unmarshalJSONOrNil attempts to unmarshal JSON, returning nil on error.
func unmarshalJSONOrNil(data []byte) interface{} {
	if len(data) == 0 {
		return nil
	}
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

