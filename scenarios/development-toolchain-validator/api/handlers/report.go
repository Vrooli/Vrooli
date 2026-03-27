package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"development-toolchain-validator/domain/report"
)

// ReportHandler handles HTTP requests for report generation.
type ReportHandler struct {
	service *report.Service
}

// NewReportHandler creates a new report handler.
func NewReportHandler(service *report.Service) *ReportHandler {
	return &ReportHandler{service: service}
}

// RegisterRoutes adds report routes to the router.
func (h *ReportHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/reports/conflicts", h.Conflicts).Methods("GET")
	r.HandleFunc("/api/v1/reports/drift", h.Drift).Methods("POST")
	r.HandleFunc("/api/v1/reports/maturity", h.Maturity).Methods("GET")
	r.HandleFunc("/api/v1/reports/tool-baselines", h.ToolBaselines).Methods("GET")
}

// Conflicts returns cross-skill contradictions.
func (h *ReportHandler) Conflicts(w http.ResponseWriter, r *http.Request) {
	opts := report.ListOptions{
		ReferenceID: r.URL.Query().Get("reference_id"),
		SkillID:     r.URL.Query().Get("skill_id"),
	}

	rpt, err := h.service.Conflicts(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate conflicts report")
		return
	}

	writeJSON(w, http.StatusOK, rpt)
}

// DriftRequest is the request body for the drift report endpoint.
type DriftRequest struct {
	// CurrentHashes maps skill_id to current content hash.
	// Optionally includes "skill_id_version" keys for version tracking.
	CurrentHashes map[string]string `json:"current_hashes"`
}

// Drift returns aggregated drift status across connections.
// POST because it requires current hashes in the request body.
func (h *ReportHandler) Drift(w http.ResponseWriter, r *http.Request) {
	var req DriftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.CurrentHashes == nil {
		writeError(w, http.StatusBadRequest, "current_hashes is required")
		return
	}

	opts := report.ListOptions{
		ReferenceID: r.URL.Query().Get("reference_id"),
		SkillID:     r.URL.Query().Get("skill_id"),
	}

	rpt, err := h.service.Drift(r.Context(), opts, req.CurrentHashes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate drift report")
		return
	}

	writeJSON(w, http.StatusOK, rpt)
}

// Maturity returns skill maturity scores.
func (h *ReportHandler) Maturity(w http.ResponseWriter, r *http.Request) {
	opts := report.ListOptions{
		ReferenceID: r.URL.Query().Get("reference_id"),
		SkillID:     r.URL.Query().Get("skill_id"),
	}

	rpt, err := h.service.Maturity(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate maturity report")
		return
	}

	writeJSON(w, http.StatusOK, rpt)
}

// ToolBaselines returns tool accuracy regression checks.
func (h *ReportHandler) ToolBaselines(w http.ResponseWriter, r *http.Request) {
	opts := report.ListOptions{
		ReferenceID: r.URL.Query().Get("reference_id"),
		SkillID:     r.URL.Query().Get("skill_id"),
	}

	rpt, err := h.service.ToolBaselines(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate tool baselines report")
		return
	}

	writeJSON(w, http.StatusOK, rpt)
}
