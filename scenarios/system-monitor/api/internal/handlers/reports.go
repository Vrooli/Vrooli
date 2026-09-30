package handlers

// DOC: docs/reference/api-endpoints.md#reports

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	reportspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/reports"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/convert"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
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

// GenerateReport handles the typed Connect-RPC report generation contract.
func (h *ReportHandler) GenerateReport(ctx context.Context, req *connect.Request[reportspb.GenerateReportRequest]) (*connect.Response[reportspb.GenerateReportResponse], error) {
	reportType := req.Msg.GetType()
	if reportType != "daily" && reportType != "weekly" {
		return nil, connectError(apierrors.Validation("type", "Must be 'daily' or 'weekly'"))
	}

	report, err := h.reportService.GenerateReport(ctx, reportType)
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&reportspb.GenerateReportResponse{
		Report: convert.EnhancedSystemReportToProto(report),
	}), nil
}

// ListReports handles the typed Connect-RPC report listing contract.
func (h *ReportHandler) ListReports(ctx context.Context, _ *connect.Request[reportspb.ListReportsRequest]) (*connect.Response[reportspb.ListReportsResponse], error) {
	reports, err := h.reportService.ListReports(ctx)
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&reportspb.ListReportsResponse{
		Reports: convert.EnhancedSystemReportsToProto(reports),
		Count:   int32(len(reports)),
	}), nil
}

// GetReport handles the typed Connect-RPC report retrieval contract.
func (h *ReportHandler) GetReport(ctx context.Context, req *connect.Request[reportspb.GetReportRequest]) (*connect.Response[reportspb.GetReportResponse], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, connectError(apierrors.Validation("id", "Report ID is required"))
	}

	report, err := h.reportService.GetReport(ctx, id)
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&reportspb.GetReportResponse{
		Report: convert.EnhancedSystemReportToProto(report),
	}), nil
}

// HandleGenerateReport handles POST /api/reports/generate.
func (h *ReportHandler) HandleGenerateReport(w http.ResponseWriter, r *http.Request) {
	var pbReq reportspb.GenerateReportRequest
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

// HandleListReports handles GET /api/reports.
func (h *ReportHandler) HandleListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.reportService.ListReports(r.Context())
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &reportspb.ListReportsResponse{
		Reports: convert.EnhancedSystemReportsToProto(reports),
		Count:   int32(len(reports)),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleGetReport handles GET /api/reports/{id}.
func (h *ReportHandler) HandleGetReport(w http.ResponseWriter, r *http.Request) {
	// Extract report ID from URL parameters
	id := r.PathValue("id")

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
