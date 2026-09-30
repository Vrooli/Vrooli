// Package handlers provides HTTP handlers for the Agent Inbox API.
//
// This file provides runtime tool-call approval endpoints.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// GetPendingApprovals returns all pending tool call approvals for a chat.
// GET /api/v1/chats/{id}/pending-approvals
func (h *Handlers) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["id"]
	if chatID == "" {
		h.JSONError(w, "Chat ID is required", http.StatusBadRequest)
		return
	}

	svc := h.NewCompletionService()
	pending, err := svc.GetPendingApprovals(r.Context(), chatID)
	if err != nil {
		h.JSONError(w, "Failed to get pending approvals", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"pending_approvals": pending,
		"count":             len(pending),
	}, http.StatusOK)
}

// ApproveToolCall approves and executes a pending tool call.
// POST /api/v1/tool-calls/{id}/approve
// Query params:
//   - chat_id: required chat ID for validation
func (h *Handlers) ApproveToolCall(w http.ResponseWriter, r *http.Request) {
	toolCallID := mux.Vars(r)["id"]
	if toolCallID == "" {
		h.JSONError(w, "Tool call ID is required", http.StatusBadRequest)
		return
	}

	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		h.JSONError(w, "chat_id query param is required", http.StatusBadRequest)
		return
	}

	svc := h.NewCompletionService()
	result, err := svc.ApproveToolCall(r.Context(), chatID, toolCallID)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert pending approvals to response format
	var pendingList []map[string]interface{}
	for _, p := range result.PendingApprovals {
		pendingList = append(pendingList, map[string]interface{}{
			"id":         p.ID,
			"tool_name":  p.ToolName,
			"arguments":  p.Arguments,
			"status":     p.Status,
			"started_at": p.StartedAt,
		})
	}

	h.JSONResponse(w, map[string]interface{}{
		"success": true,
		"tool_result": map[string]interface{}{
			"id":        result.ToolResult.ID,
			"tool_name": result.ToolResult.ToolName,
			"status":    result.ToolResult.Status,
			"result":    result.ToolResult.Result,
		},
		"pending_approvals": pendingList,
		"auto_continued":    result.AutoContinued,
	}, http.StatusOK)
}

// RejectToolCall rejects a pending tool call.
// POST /api/v1/tool-calls/{id}/reject
// Query params:
//   - chat_id: required chat ID for validation
//
// Body: { "reason": "optional rejection reason" }
func (h *Handlers) RejectToolCall(w http.ResponseWriter, r *http.Request) {
	toolCallID := mux.Vars(r)["id"]
	if toolCallID == "" {
		h.JSONError(w, "Tool call ID is required", http.StatusBadRequest)
		return
	}

	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		h.JSONError(w, "chat_id query param is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req) // Ignore error - reason is optional

	svc := h.NewCompletionService()
	if err := svc.RejectToolCall(r.Context(), chatID, toolCallID, req.Reason); err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"success":      true,
		"tool_call_id": toolCallID,
		"rejected":     true,
		"reason":       req.Reason,
	}, http.StatusOK)
}
