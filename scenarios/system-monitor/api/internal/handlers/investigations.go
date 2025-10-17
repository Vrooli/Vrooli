package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

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

// ListInvestigations handles GET /api/investigations
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

// GetLatestInvestigation handles GET /api/investigations/latest
func (h *InvestigationHandler) GetLatestInvestigation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	investigation, err := h.investigationSvc.GetLatestInvestigation(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, investigation)
}

// TriggerInvestigation handles POST /api/investigations/trigger
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
		"api_base_url":     "http://localhost:8080",
		"message":          "Investigation queued for processing",
		"auto_fix":         req.AutoFix,
		"note":             req.Note,
	}

	respondWithJSON(w, http.StatusAccepted, response)
}

// GetInvestigation handles GET /api/investigations/{id}
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

// UpdateInvestigationStatus handles PUT /api/investigations/{id}/status
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

// UpdateInvestigationFindings handles PUT /api/investigations/{id}/findings
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

// UpdateInvestigationProgress handles PUT /api/investigations/{id}/progress
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

// AddInvestigationStep handles POST /api/investigations/{id}/step
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

// GetCooldownStatus handles GET /api/investigations/cooldown
func (h *InvestigationHandler) GetCooldownStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.investigationSvc.GetCooldownStatus(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, status)
}

// ResetCooldown handles POST /api/investigations/cooldown/reset
func (h *InvestigationHandler) ResetCooldown(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.investigationSvc.ResetCooldown(ctx); err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "cooldown_reset"})
}

// UpdateCooldownPeriod handles PUT /api/investigations/cooldown/period
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

// GetTriggers handles GET /api/investigations/triggers
func (h *InvestigationHandler) GetTriggers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	triggers, err := h.investigationSvc.GetTriggers(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, triggers)
}

// UpdateTrigger handles PUT /api/investigations/triggers/{id}
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

// UpdateTriggerThreshold handles PUT /api/investigations/triggers/{id}/threshold
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
