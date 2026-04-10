// Package handlers provides HTTP handlers for the Agent Inbox API.
//
// This file provides tool approval and manual execution endpoints.
package handlers

import (
	"encoding/json"
	"net/http"

	"agent-inbox/domain"

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

// ExecuteToolManually executes a tool directly without going through AI.
// POST /api/v1/tools/execute
//
//	Body: {
//	  "scenario": "agent-manager",
//	  "tool_name": "spawn_coding_agent",
//	  "arguments": { ... tool parameters ... },
//	  "chat_id": "optional - if provided, adds result to chat history"
//	}
func (h *Handlers) ExecuteToolManually(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scenario  string                 `json:"scenario"`
		ToolName  string                 `json:"tool_name"`
		Arguments map[string]interface{} `json:"arguments"`
		ChatID    string                 `json:"chat_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Scenario == "" || req.ToolName == "" {
		h.JSONError(w, "scenario and tool_name are required", http.StatusBadRequest)
		return
	}

	// Validate the tool exists
	tool, err := h.ToolRegistry.GetTool(r.Context(), req.Scenario, req.ToolName)
	if err != nil || tool == nil {
		h.JSONError(w, "Tool not found", http.StatusNotFound)
		return
	}

	// Execute the tool via completion service
	svc := h.NewCompletionService()
	result, err := svc.ExecuteToolManually(r.Context(), req.ChatID, req.Scenario, req.ToolName, req.Arguments)
	if err != nil {
		h.JSONResponse(w, map[string]interface{}{
			"success":           false,
			"status":            "failed",
			"error":             err.Error(),
			"execution_time_ms": 0,
		}, http.StatusOK)
		return
	}

	response := map[string]interface{}{
		"success":           true,
		"result":            result.Result,
		"status":            result.Status,
		"execution_time_ms": result.ExecutionTimeMs,
	}

	if result.ToolCallRecord != nil {
		response["tool_call_record"] = map[string]interface{}{
			"id":         result.ToolCallRecord.ID,
			"message_id": result.ToolCallRecord.MessageID,
		}
	}

	h.JSONResponse(w, response, http.StatusOK)
}

// SetToolApproval updates the approval override for a tool.
// POST /api/v1/tools/config/approval
// Body: { "chat_id": "optional", "scenario": "agent-manager", "tool_name": "spawn_coding_agent", "approval_override": "require"|"skip"|"" }
func (h *Handlers) SetToolApproval(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChatID           string `json:"chat_id"`
		Scenario         string `json:"scenario"`
		ToolName         string `json:"tool_name"`
		ApprovalOverride string `json:"approval_override"` // "", "require", "skip"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Scenario == "" || req.ToolName == "" {
		h.JSONError(w, "scenario and tool_name are required", http.StatusBadRequest)
		return
	}

	// Validate approval_override value
	override := domain.ApprovalOverride(req.ApprovalOverride)
	if override != "" && override != domain.ApprovalRequire && override != domain.ApprovalSkip {
		h.JSONError(w, "approval_override must be '', 'require', or 'skip'", http.StatusBadRequest)
		return
	}

	if err := h.ToolRegistry.SetToolApprovalOverride(r.Context(), req.ChatID, req.Scenario, req.ToolName, override); err != nil {
		h.JSONError(w, "Failed to update tool approval configuration", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"success":           true,
		"chat_id":           req.ChatID,
		"scenario":          req.Scenario,
		"tool_name":         req.ToolName,
		"approval_override": req.ApprovalOverride,
	}, http.StatusOK)
}
