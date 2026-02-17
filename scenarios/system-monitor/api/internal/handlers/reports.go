package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/convert"
	"system-monitor-api/internal/httputil"
	"system-monitor-api/internal/services"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/api"
)

// ReportHandler handles report-related HTTP requests
type ReportHandler struct {
	config        *config.Config
	reportService *services.ReportService
}

// NewReportHandler creates a new report handler
func NewReportHandler(cfg *config.Config, reportService *services.ReportService) *ReportHandler {
	return &ReportHandler{
		config:        cfg,
		reportService: reportService,
	}
}

// GenerateReport handles POST /api/reports/generate
func (h *ReportHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var pbReq apipb.GenerateReportRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.BadRequest(w, "GenerateReport", "Invalid request body")
		return
	}

	// Validate report type
	if pbReq.Type != "daily" && pbReq.Type != "weekly" {
		httputil.BadRequest(w, "GenerateReport", "Invalid report type. Must be 'daily' or 'weekly'")
		return
	}

	// Generate the report using real historical data
	report, err := h.reportService.GenerateReport(r.Context(), pbReq.Type)
	if err != nil {
		httputil.InternalError(w, "GenerateReport", "Failed to generate report: "+err.Error())
		return
	}

	httputil.ProtoJSON(w, convert.EnhancedSystemReportToProto(report)) //nolint:errcheck
}

// ListReports handles GET /api/reports
func (h *ReportHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.reportService.ListReports(r.Context())
	if err != nil {
		httputil.InternalError(w, "ListReports", "Failed to list reports: "+err.Error())
		return
	}

	resp := &apipb.ListReportsResponse{
		Reports: convert.EnhancedSystemReportsToProto(reports),
		Count:   int32(len(reports)),
	}
	httputil.ProtoJSON(w, resp) //nolint:errcheck
}

// GetReport handles GET /api/reports/{id}
func (h *ReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	// Extract report ID from URL parameters
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		httputil.BadRequest(w, "GetReport", "Report ID is required")
		return
	}

	report, err := h.reportService.GetReport(r.Context(), id)
	if err != nil {
		httputil.NotFound(w, "GetReport", "Failed to get report: "+err.Error())
		return
	}

	httputil.ProtoJSON(w, convert.EnhancedSystemReportToProto(report)) //nolint:errcheck
}
