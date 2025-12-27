// Package handlers provides HTTP handlers for the Agent Inbox API.
//
// This file provides tool configuration management endpoints.
// These enable users to configure which tools are enabled globally
// and per-chat, including approval settings.
package handlers

import (
	"encoding/json"
	"net/http"

	"agent-inbox/domain"
	"agent-inbox/services"

	"github.com/gorilla/mux"
)

// GetToolSet returns all available tools with their effective enabled states.
// GET /api/v1/tools/set
// Query params:
//   - chat_id: optional chat ID for chat-specific configurations
func (h *Handlers) GetToolSet(w http.ResponseWriter, r *http.Request) {
	chatID := r.URL.Query().Get("chat_id")

	toolSet, err := h.ToolRegistry.GetToolSet(r.Context())
	if err != nil {
		h.JSONError(w, "Failed to get tool set", http.StatusInternalServerError)
		return
	}

	// Always apply configurations (global if no chatID, chat-specific if provided)
	tools, err := h.ToolRegistry.GetEffectiveTools(r.Context(), chatID)
	if err != nil {
		h.JSONError(w, "Failed to get effective tools", http.StatusInternalServerError)
		return
	}
	toolSet.Tools = tools

	h.JSONResponse(w, toolSet, http.StatusOK)
}

// GetScenarioStatuses returns the availability status of all configured scenarios.
// GET /api/v1/tools/scenarios
func (h *Handlers) GetScenarioStatuses(w http.ResponseWriter, r *http.Request) {
	statuses := h.ToolRegistry.GetScenarioStatuses(r.Context())
	h.JSONResponse(w, statuses, http.StatusOK)
}

// SetToolEnabled updates the enabled state for a tool.
// POST /api/v1/tools/config
// Body: { "chat_id": "optional", "scenario": "agent-manager", "tool_name": "spawn_coding_agent", "enabled": true }
func (h *Handlers) SetToolEnabled(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChatID   string `json:"chat_id"`
		Scenario string `json:"scenario"`
		ToolName string `json:"tool_name"`
		Enabled  bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Scenario == "" || req.ToolName == "" {
		h.JSONError(w, "scenario and tool_name are required", http.StatusBadRequest)
		return
	}

	if err := h.ToolRegistry.SetToolEnabled(r.Context(), req.ChatID, req.Scenario, req.ToolName, req.Enabled); err != nil {
		h.JSONError(w, "Failed to update tool configuration", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"success":   true,
		"chat_id":   req.ChatID,
		"scenario":  req.Scenario,
		"tool_name": req.ToolName,
		"enabled":   req.Enabled,
	}, http.StatusOK)
}

// ResetToolConfig removes a tool configuration, reverting to default.
// DELETE /api/v1/tools/config
// Query params:
//   - chat_id: optional chat ID (empty for global)
//   - scenario: required scenario name
//   - tool_name: required tool name
func (h *Handlers) ResetToolConfig(w http.ResponseWriter, r *http.Request) {
	chatID := r.URL.Query().Get("chat_id")
	scenario := r.URL.Query().Get("scenario")
	toolName := r.URL.Query().Get("tool_name")

	if scenario == "" || toolName == "" {
		h.JSONError(w, "scenario and tool_name query params are required", http.StatusBadRequest)
		return
	}

	if err := h.ToolRegistry.ResetToolConfiguration(r.Context(), chatID, scenario, toolName); err != nil {
		h.JSONError(w, "Failed to reset tool configuration", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"success":   true,
		"chat_id":   chatID,
		"scenario":  scenario,
		"tool_name": toolName,
		"reset":     true,
	}, http.StatusOK)
}

// RefreshTools triggers a refresh of the tool registry cache.
// POST /api/v1/tools/refresh
func (h *Handlers) RefreshTools(w http.ResponseWriter, r *http.Request) {
	if err := h.ToolRegistry.RefreshTools(r.Context()); err != nil {
		h.JSONError(w, "Failed to refresh tools", http.StatusInternalServerError)
		return
	}

	toolSet, err := h.ToolRegistry.GetToolSet(r.Context())
	if err != nil {
		h.JSONError(w, "Failed to get refreshed tool set", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"success":         true,
		"scenarios_count": len(toolSet.Scenarios),
		"tools_count":     len(toolSet.Tools),
	}, http.StatusOK)
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

// GetPendingApprovals returns all pending tool call approvals for a chat.
// GET /api/v1/chats/{id}/pending-approvals
func (h *Handlers) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["id"]
	if chatID == "" {
		h.JSONError(w, "Chat ID is required", http.StatusBadRequest)
		return
	}

	svc := services.NewCompletionService(h.Repo)
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

	svc := services.NewCompletionService(h.Repo)
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
	json.NewDecoder(r.Body).Decode(&req) // Ignore error - reason is optional

	svc := services.NewCompletionService(h.Repo)
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
