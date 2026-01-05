// Package handlers provides HTTP handlers for investigation endpoints.
package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// InvestigationHandler provides HTTP handlers for investigation endpoints.
type InvestigationHandler struct {
	svc orchestration.InvestigationService
}

// NewInvestigationHandler creates a new investigation handler.
func NewInvestigationHandler(svc orchestration.InvestigationService) *InvestigationHandler {
	return &InvestigationHandler{svc: svc}
}

// RegisterRoutes registers investigation API routes on the given router.
func (h *InvestigationHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/investigations", h.CreateInvestigation).Methods("POST")
	r.HandleFunc("/api/v1/investigations", h.ListInvestigations).Methods("GET")
	r.HandleFunc("/api/v1/investigations/active", h.GetActiveInvestigation).Methods("GET")
	r.HandleFunc("/api/v1/investigations/{id}", h.GetInvestigation).Methods("GET")
	r.HandleFunc("/api/v1/investigations/{id}", h.DeleteInvestigation).Methods("DELETE")
	r.HandleFunc("/api/v1/investigations/{id}/stop", h.StopInvestigation).Methods("POST")
	r.HandleFunc("/api/v1/investigations/{id}/apply-fixes", h.ApplyFixes).Methods("POST")
}

// =============================================================================
// Request/Response Types
// =============================================================================

// CreateInvestigationRequest is the HTTP request body for creating an investigation.
type CreateInvestigationRequest struct {
	RunIDs        []string               `json:"runIds"`
	AnalysisType  domain.AnalysisType    `json:"analysisType"`
	ReportSections domain.ReportSections `json:"reportSections"`
	CustomContext string                 `json:"customContext,omitempty"`
}

// ApplyFixesHTTPRequest is the HTTP request body for applying fixes.
type ApplyFixesHTTPRequest struct {
	RecommendationIDs []string `json:"recommendationIds"`
	Note              string   `json:"note,omitempty"`
}

// InvestigationResponse is the HTTP response for a single investigation.
type InvestigationResponse struct {
	Investigation *domain.Investigation `json:"investigation"`
}

// InvestigationsListResponse is the HTTP response for listing investigations.
type InvestigationsListResponse struct {
	Investigations []*domain.Investigation `json:"investigations"`
	Total          int                     `json:"total"`
}

// =============================================================================
// Handlers
// =============================================================================

// CreateInvestigation handles POST /api/v1/investigations
func (h *InvestigationHandler) CreateInvestigation(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req CreateInvestigationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	// Parse run IDs
	runIDs := make([]uuid.UUID, 0, len(req.RunIDs))
	for _, idStr := range req.RunIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeSimpleError(w, r, "runIds", "invalid UUID format: "+idStr)
			return
		}
		runIDs = append(runIDs, id)
	}

	// Create domain request
	domainReq := &domain.CreateInvestigationRequest{
		RunIDs:         runIDs,
		AnalysisType:   req.AnalysisType,
		ReportSections: req.ReportSections,
		CustomContext:  req.CustomContext,
	}

	investigation, err := h.svc.TriggerInvestigation(r.Context(), domainReq)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, InvestigationResponse{Investigation: investigation})
}

// GetInvestigation handles GET /api/v1/investigations/{id}
func (h *InvestigationHandler) GetInvestigation(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format")
		return
	}

	investigation, err := h.svc.GetInvestigation(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, InvestigationResponse{Investigation: investigation})
}

// ListInvestigations handles GET /api/v1/investigations
func (h *InvestigationHandler) ListInvestigations(w http.ResponseWriter, r *http.Request) {
	filter := repository.InvestigationListFilter{}

	// Parse status filter
	if statusStr := queryFirst(r, "status"); statusStr != "" {
		status := domain.InvestigationStatus(statusStr)
		if !status.IsValid() {
			writeSimpleError(w, r, "status", "invalid investigation status")
			return
		}
		filter.Status = &status
	}

	// Parse source_investigation_id filter
	if sourceIDStr := queryFirst(r, "source_investigation_id", "sourceInvestigationId"); sourceIDStr != "" {
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			writeSimpleError(w, r, "source_investigation_id", "invalid UUID format")
			return
		}
		filter.SourceInvestigationID = &sourceID
	}

	// Parse pagination
	if limit, ok := parseQueryInt(r, "limit"); ok {
		filter.Limit = limit
	}
	if offset, ok := parseQueryInt(r, "offset"); ok {
		filter.Offset = offset
	}

	investigations, err := h.svc.ListInvestigations(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, InvestigationsListResponse{
		Investigations: investigations,
		Total:          len(investigations),
	})
}

// GetActiveInvestigation handles GET /api/v1/investigations/active
func (h *InvestigationHandler) GetActiveInvestigation(w http.ResponseWriter, r *http.Request) {
	investigation, err := h.svc.GetActiveInvestigation(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	if investigation == nil {
		writeJSON(w, http.StatusOK, InvestigationResponse{Investigation: nil})
		return
	}

	writeJSON(w, http.StatusOK, InvestigationResponse{Investigation: investigation})
}

// StopInvestigation handles POST /api/v1/investigations/{id}/stop
func (h *InvestigationHandler) StopInvestigation(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format")
		return
	}

	if err := h.svc.StopInvestigation(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "cancelled",
	})
}

// DeleteInvestigation handles DELETE /api/v1/investigations/{id}
func (h *InvestigationHandler) DeleteInvestigation(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format")
		return
	}

	if err := h.svc.DeleteInvestigation(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// ApplyFixes handles POST /api/v1/investigations/{id}/apply-fixes
func (h *InvestigationHandler) ApplyFixes(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req ApplyFixesHTTPRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	domainReq := &domain.ApplyFixesRequest{
		RecommendationIDs: req.RecommendationIDs,
		Note:              req.Note,
	}

	investigation, err := h.svc.ApplyFixes(r.Context(), id, domainReq)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, InvestigationResponse{Investigation: investigation})
}
