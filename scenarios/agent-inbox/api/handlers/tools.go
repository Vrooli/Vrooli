// Package handlers provides HTTP handlers for the Agent Inbox API.
//
// This file provides tool configuration management endpoints.
// These enable users to configure which tools are enabled globally
// and per-chat.
package handlers

import (
	"encoding/json"
	"net/http"
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

	// If chat ID is provided, merge with chat-specific configurations
	if chatID != "" {
		tools, err := h.ToolRegistry.GetEffectiveTools(r.Context(), chatID)
		if err != nil {
			h.JSONError(w, "Failed to get effective tools", http.StatusInternalServerError)
			return
		}
		toolSet.Tools = tools
	}

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
