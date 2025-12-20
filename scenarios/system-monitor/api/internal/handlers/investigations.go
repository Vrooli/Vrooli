package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/services"
)

// InvestigationHandler handles investigation-related requests
type InvestigationHandler struct {
	config           *config.Config
	investigationSvc *services.InvestigationService
}

// NewInvestigationHandler creates a new investigation handler
func NewInvestigationHandler(cfg *config.Config, investigationSvc *services.InvestigationService) *InvestigationHandler {
	return &InvestigationHandler{
		config:           cfg,
		investigationSvc: investigationSvc,
	}
}

// ListInvestigations handles GET /api/v1/investigations
func (h *InvestigationHandler) ListInvestigations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := 20
	if queryLimit := r.URL.Query().Get("limit"); queryLimit != "" {
		if parsed, err := strconv.Atoi(queryLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	investigations, err := h.investigationSvc.ListInvestigations(ctx, limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, investigations)
}

// GetLatestInvestigation handles GET /api/v1/investigations/latest
func (h *InvestigationHandler) GetLatestInvestigation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	investigation, err := h.investigationSvc.GetLatestInvestigation(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, investigation)
}

// TriggerInvestigation handles POST /api/v1/investigations/trigger
func (h *InvestigationHandler) TriggerInvestigation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req struct {
		AutoFix bool   `json:"auto_fix"`
		Note    string `json:"note,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for backwards compatibility
		req.AutoFix = false
		req.Note = ""
	}

	investigation, err := h.investigationSvc.TriggerInvestigation(ctx, req.AutoFix, req.Note)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	// Return immediate response with API info
	response := map[string]interface{}{
		"status":           "queued",
		"investigation_id": investigation.ID,
		"api_base_url":     h.resolveAPIBaseURL(),
		"message":          "Investigation queued for processing",
		"auto_fix":         req.AutoFix,
		"note":             req.Note,
	}

	respondWithJSON(w, http.StatusAccepted, response)
}

// GetInvestigation handles GET /api/v1/investigations/{id}
func (h *InvestigationHandler) GetInvestigation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	investigation, err := h.investigationSvc.GetInvestigation(ctx, id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err)
		return
	}

	respondWithJSON(w, http.StatusOK, investigation)
}

// UpdateInvestigationStatus handles PUT /api/v1/investigations/{id}/status
func (h *InvestigationHandler) UpdateInvestigationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.investigationSvc.UpdateInvestigationStatus(ctx, id, req.Status); err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// UpdateInvestigationFindings handles PUT /api/v1/investigations/{id}/findings
func (h *InvestigationHandler) UpdateInvestigationFindings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var req struct {
		Findings string                 `json:"findings"`
		Details  map[string]interface{} `json:"details,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.investigationSvc.UpdateInvestigationFindings(ctx, id, req.Findings, req.Details); err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// UpdateInvestigationProgress handles PUT /api/v1/investigations/{id}/progress
func (h *InvestigationHandler) UpdateInvestigationProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var req struct {
		Progress int `json:"progress"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.investigationSvc.UpdateInvestigationProgress(ctx, id, req.Progress); err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// AddInvestigationStep handles POST /api/v1/investigations/{id}/step
func (h *InvestigationHandler) AddInvestigationStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var step models.InvestigationStep
	if err := json.NewDecoder(r.Body).Decode(&step); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.investigationSvc.AddInvestigationStep(ctx, id, step); err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "step_added"})
}

// GetCooldownStatus handles GET /api/v1/investigations/cooldown
func (h *InvestigationHandler) GetCooldownStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.investigationSvc.GetCooldownStatus(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, status)
}

// ResetCooldown handles POST /api/v1/investigations/cooldown/reset
func (h *InvestigationHandler) ResetCooldown(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.investigationSvc.ResetCooldown(ctx); err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "cooldown_reset"})
}

// UpdateCooldownPeriod handles PUT /api/v1/investigations/cooldown/period
func (h *InvestigationHandler) UpdateCooldownPeriod(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		CooldownPeriodSeconds int `json:"cooldown_period_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.investigationSvc.UpdateCooldownPeriod(ctx, req.CooldownPeriodSeconds); err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// GetTriggers handles GET /api/v1/investigations/triggers
func (h *InvestigationHandler) GetTriggers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	triggers, err := h.investigationSvc.GetTriggers(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, triggers)
}

// UpdateTrigger handles PUT /api/v1/investigations/triggers/{id}
func (h *InvestigationHandler) UpdateTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var req struct {
		Enabled   *bool    `json:"enabled,omitempty"`
		AutoFix   *bool    `json:"auto_fix,omitempty"`
		Threshold *float64 `json:"threshold,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.investigationSvc.UpdateTrigger(ctx, id, req.Enabled, req.AutoFix, req.Threshold); err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// UpdateTriggerThreshold handles PUT /api/v1/investigations/triggers/{id}/threshold
func (h *InvestigationHandler) UpdateTriggerThreshold(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var req struct {
		Threshold float64 `json:"threshold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.investigationSvc.UpdateTrigger(ctx, id, nil, nil, &req.Threshold); err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *InvestigationHandler) resolveAPIBaseURL() string {
	port := strings.TrimSpace(h.config.Server.APIPort)
	if port == "" {
		port = "8080"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// =============================================================================
// Agent Configuration Endpoints
// =============================================================================

// GetAgentConfig handles GET /api/agent/config
func (h *InvestigationHandler) GetAgentConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config, err := h.investigationSvc.GetAgentConfig(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, config)
}

// GetAvailableRunners handles GET /api/agent/runners
func (h *InvestigationHandler) GetAvailableRunners(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	runners, err := h.investigationSvc.GetAvailableRunners(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, runners)
}

// UpdateAgentConfig handles PUT /api/agent/config
func (h *InvestigationHandler) UpdateAgentConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		RunnerType       string   `json:"runner_type,omitempty"`
		Model            string   `json:"model,omitempty"`
		MaxTurns         int32    `json:"max_turns,omitempty"`
		TimeoutSeconds   int32    `json:"timeout_seconds,omitempty"`
		AllowedTools     []string `json:"allowed_tools,omitempty"`
		SkipPermissions  bool     `json:"skip_permissions,omitempty"`
		RequiresSandbox  bool     `json:"requires_sandbox,omitempty"`
		RequiresApproval bool     `json:"requires_approval,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	config, err := h.investigationSvc.UpdateAgentConfig(ctx, req.RunnerType, req.Model, req.MaxTurns, req.TimeoutSeconds, req.AllowedTools, req.SkipPermissions, req.RequiresSandbox, req.RequiresApproval)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, config)
}

// GetAgentStatus handles GET /api/agent/status
func (h *InvestigationHandler) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.investigationSvc.GetAgentStatus(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, status)
}

// =============================================================================
// Agent Current & Scripts Endpoints
// =============================================================================

// GetCurrentAgent handles GET /api/v1/investigations/agent/current
// Returns current running investigation agent status (if any)
func (h *InvestigationHandler) GetCurrentAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the latest in-progress investigation
	investigation, err := h.investigationSvc.GetLatestInvestigation(ctx)
	if err != nil {
		// Return empty if no investigations
		respondWithJSON(w, http.StatusOK, nil)
		return
	}

	// Only return if there's an active investigation
	if investigation != nil && investigation.Status == "in_progress" {
		respondWithJSON(w, http.StatusOK, investigation)
		return
	}

	// No active agent
	respondWithJSON(w, http.StatusOK, nil)
}

// ListScripts handles GET /api/v1/investigations/scripts
// Returns list of available investigation scripts
func (h *InvestigationHandler) ListScripts(w http.ResponseWriter, r *http.Request) {
	// Return empty scripts list for now (feature placeholder)
	response := map[string]interface{}{
		"scripts": []interface{}{},
	}
	respondWithJSON(w, http.StatusOK, response)
}

// GetScript handles GET /api/v1/investigations/scripts/{id}
// Returns a specific script's content
func (h *InvestigationHandler) GetScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Script feature not implemented yet
	respondWithError(w, http.StatusNotFound, fmt.Errorf("script not found: %s", id))
}

// ExecuteScript handles POST /api/v1/investigations/scripts/{id}/execute
// Executes a specific investigation script
func (h *InvestigationHandler) ExecuteScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Script feature not implemented yet
	respondWithError(w, http.StatusNotFound, fmt.Errorf("script not found: %s", id))
}
