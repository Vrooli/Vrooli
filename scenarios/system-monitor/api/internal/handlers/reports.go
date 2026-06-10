package handlers

// DOC: docs/reference/api-endpoints.md#reports

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/convert"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/api"
)

// ReportHandler handles report-related HTTP requests
type ReportHandler struct {
	log           *slog.Logger
	config        *config.Config
	reportService ReportGenerator
}

// NewReportHandler creates a new report handler
func NewReportHandler(cfg *config.Config, reportService ReportGenerator, log *slog.Logger) *ReportHandler {
	return &ReportHandler{
		log:           log,
		config:        cfg,
		reportService: reportService,
	}
}

// GenerateReport handles POST /api/reports/generate
func (h *ReportHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var pbReq apipb.GenerateReportRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", "Invalid request body"))
		return
	}

	// Validate report type
	if pbReq.Type != "daily" && pbReq.Type != "weekly" {
		httputil.HandleError(w, h.log, r, apierrors.Validation("type", "Must be 'daily' or 'weekly'"))
		return
	}

	// Generate the report using real historical data
	report, err := h.reportService.GenerateReport(r.Context(), pbReq.Type)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.EnhancedSystemReportToProto(report))
}

// ListReports handles GET /api/reports
func (h *ReportHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.reportService.ListReports(r.Context())
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.ListReportsResponse{
		Reports: convert.EnhancedSystemReportsToProto(reports),
		Count:   int32(len(reports)),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// GetReport handles GET /api/reports/{id}
func (h *ReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	// Extract report ID from URL parameters
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		httputil.HandleError(w, h.log, r, apierrors.Validation("id", "Report ID is required"))
		return
	}

	report, err := h.reportService.GetReport(r.Context(), id)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.EnhancedSystemReportToProto(report))
}
