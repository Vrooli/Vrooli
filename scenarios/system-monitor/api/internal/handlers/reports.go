package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/services"
)

// ReportHandler handles report-related HTTP requests
type ReportHandler struct {
	config *config.Config
	reportService *services.ReportService
}

// NewReportHandler creates a new report handler
func NewReportHandler(cfg *config.Config, reportService *services.ReportService) *ReportHandler {
	return &ReportHandler{
		config: cfg,
		reportService: reportService,
	}
}

// ReportRequest represents the request structure for report generation
type ReportRequest struct {
	Type string `json:"type"`
}

// GenerateReport handles POST /api/reports/generate
func (h *ReportHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate report type
	if req.Type != "daily" && req.Type != "weekly" {
		http.Error(w, "Invalid report type. Must be 'daily' or 'weekly'", http.StatusBadRequest)
		return
	}

	// Generate the report using real historical data
	report, err := h.reportService.GenerateReport(r.Context(), req.Type)
	if err != nil {
		http.Error(w, "Failed to generate report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the complete report
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// ListReports handles GET /api/reports
func (h *ReportHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.reportService.ListReports(r.Context())
	if err != nil {
		http.Error(w, "Failed to list reports: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure reports is not nil
	if reports == nil {
		reports = []*models.EnhancedSystemReport{}
	}

	response := map[string]interface{}{
		"reports": reports,
		"count":   len(reports),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log error but response is already partially written
		return
	}
}

// GetReport handles GET /api/reports/{id}
func (h *ReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	// Extract report ID from URL parameters
	vars := mux.Vars(r)
	id := vars["id"]
	
	if id == "" {
		http.Error(w, "Report ID is required", http.StatusBadRequest)
		return
	}

	report, err := h.reportService.GetReport(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to get report: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}