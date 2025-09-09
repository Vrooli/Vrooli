package handlers

import (
	"encoding/json"
	"net/http"

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
	
	investigation, err := h.investigationSvc.TriggerInvestigation(ctx)
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