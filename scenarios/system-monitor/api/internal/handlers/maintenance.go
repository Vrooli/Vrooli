package handlers

// DOC: docs/reference/api-endpoints.md#maintenance

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"connectrpc.com/connect"
	maintenancepb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/maintenance"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/convert"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

const defaultRetentionDays = 30

// MaintenanceHandler exposes metrics-lifecycle maintenance endpoints.
type MaintenanceHandler struct {
	log   *slog.Logger
	maint MaintenanceProvider
}

// NewMaintenanceHandler creates a maintenance handler.
func NewMaintenanceHandler(maint MaintenanceProvider, log *slog.Logger) *MaintenanceHandler {
	return &MaintenanceHandler{log: log, maint: maint}
}

// MetricsRetentionPreview handles the typed Connect-RPC retention preview contract.
func (h *MaintenanceHandler) MetricsRetentionPreview(ctx context.Context, req *connect.Request[maintenancepb.MetricsRetentionPreviewRequest]) (*connect.Response[maintenancepb.MetricsRetentionPreviewResponse], error) {
	days := int(req.Msg.GetRetentionDays())
	if days <= 0 {
		days = defaultRetentionDays
	}

	estimate, stats, err := h.maint.RetentionPreview(ctx, days)
	if err != nil {
		return nil, connectError(mapMaintenanceError(err))
	}

	return connect.NewResponse(&maintenancepb.MetricsRetentionPreviewResponse{
		Success:       true,
		Estimate:      convert.RetentionEstimateToProto(estimate),
		DatabaseStats: convert.DatabaseStatsToProto(stats),
	}), nil
}

// MetricsRetentionApply handles the typed Connect-RPC retention apply contract.
func (h *MaintenanceHandler) MetricsRetentionApply(ctx context.Context, req *connect.Request[maintenancepb.MetricsRetentionApplyRequest]) (*connect.Response[maintenancepb.MetricsRetentionApplyResponse], error) {
	days := int(req.Msg.GetRetentionDays())
	if days <= 0 {
		days = defaultRetentionDays
	}

	result, before, after, err := h.maint.RetentionApply(ctx, days, req.Msg.GetConfirm())
	if err != nil {
		return nil, connectError(mapMaintenanceError(err))
	}

	return connect.NewResponse(&maintenancepb.MetricsRetentionApplyResponse{
		Success:             true,
		Result:              convert.RetentionResultToProto(result),
		DatabaseStatsBefore: convert.DatabaseStatsToProto(before),
		DatabaseStatsAfter:  convert.DatabaseStatsToProto(after),
	}), nil
}

// MetricsCompactionPreview handles the typed Connect-RPC compaction preview contract.
func (h *MaintenanceHandler) MetricsCompactionPreview(ctx context.Context, _ *connect.Request[maintenancepb.MetricsCompactionPreviewRequest]) (*connect.Response[maintenancepb.MetricsCompactionPreviewResponse], error) {
	stats, reclaimable, err := h.maint.CompactionPreview(ctx)
	if err != nil {
		return nil, connectError(mapMaintenanceError(err))
	}

	return connect.NewResponse(&maintenancepb.MetricsCompactionPreviewResponse{
		Success:                   true,
		DatabaseStats:             convert.DatabaseStatsToProto(stats),
		EstimatedReclaimableBytes: reclaimable,
	}), nil
}

// MetricsCompactionApply handles the typed Connect-RPC compaction apply contract.
func (h *MaintenanceHandler) MetricsCompactionApply(ctx context.Context, req *connect.Request[maintenancepb.MetricsCompactionApplyRequest]) (*connect.Response[maintenancepb.MetricsCompactionApplyResponse], error) {
	result, err := h.maint.CompactionApply(ctx, req.Msg.GetConfirm())
	if err != nil {
		return nil, connectError(mapMaintenanceError(err))
	}

	return connect.NewResponse(&maintenancepb.MetricsCompactionApplyResponse{
		Success:             true,
		DatabaseStatsBefore: convert.DatabaseStatsToProto(result.StatsBefore),
		DatabaseStatsAfter:  convert.DatabaseStatsToProto(result.StatsAfter),
		ReclaimedBytes:      result.ReclaimedBytes,
	}), nil
}

// mapMaintenanceError translates service errors to API errors.
func mapMaintenanceError(err error) error {
	switch {
	case errors.Is(err, services.ErrConfirmationRequired):
		return apierrors.Validation("confirm", "must be true to perform this destructive operation")
	case errors.Is(err, services.ErrInvalidRetentionDays):
		return apierrors.Validation("retention_days", "must be greater than 0")
	case errors.Is(err, repository.ErrNotSupported):
		return apierrors.Unavailable("compaction (storage backend does not support it)")
	default:
		return apierrors.Internal("maintenance operation failed", err)
	}
}

// RetentionPreview handles GET /api/v1/maintenance/metrics/retention/preview
func (h *MaintenanceHandler) RetentionPreview(w http.ResponseWriter, r *http.Request) {
	days := defaultRetentionDays
	if v := r.URL.Query().Get("days"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			httputil.HandleError(w, h.log, r, apierrors.Validation("days", "must be a positive integer"))
			return
		}
		days = parsed
	}

	estimate, stats, err := h.maint.RetentionPreview(r.Context(), days)
	if err != nil {
		httputil.HandleError(w, h.log, r, mapMaintenanceError(err))
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, &maintenancepb.MetricsRetentionPreviewResponse{
		Success:       true,
		Estimate:      convert.RetentionEstimateToProto(estimate),
		DatabaseStats: convert.DatabaseStatsToProto(stats),
	})
}

// RetentionApply handles POST /api/v1/maintenance/metrics/retention/apply
func (h *MaintenanceHandler) RetentionApply(w http.ResponseWriter, r *http.Request) {
	var req maintenancepb.MetricsRetentionApplyRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", "Invalid JSON payload"))
		return
	}

	days := int(req.GetRetentionDays())
	if days <= 0 {
		days = defaultRetentionDays
	}

	result, before, after, err := h.maint.RetentionApply(r.Context(), days, req.GetConfirm())
	if err != nil {
		httputil.HandleError(w, h.log, r, mapMaintenanceError(err))
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, &maintenancepb.MetricsRetentionApplyResponse{
		Success:             true,
		Result:              convert.RetentionResultToProto(result),
		DatabaseStatsBefore: convert.DatabaseStatsToProto(before),
		DatabaseStatsAfter:  convert.DatabaseStatsToProto(after),
	})
}

// CompactionPreview handles GET /api/v1/maintenance/metrics/compaction/preview
func (h *MaintenanceHandler) CompactionPreview(w http.ResponseWriter, r *http.Request) {
	stats, reclaimable, err := h.maint.CompactionPreview(r.Context())
	if err != nil {
		httputil.HandleError(w, h.log, r, mapMaintenanceError(err))
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, &maintenancepb.MetricsCompactionPreviewResponse{
		Success:                   true,
		DatabaseStats:             convert.DatabaseStatsToProto(stats),
		EstimatedReclaimableBytes: reclaimable,
	})
}

// CompactionApply handles POST /api/v1/maintenance/metrics/compaction/apply
func (h *MaintenanceHandler) CompactionApply(w http.ResponseWriter, r *http.Request) {
	var req maintenancepb.MetricsCompactionApplyRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", "Invalid JSON payload"))
		return
	}

	result, err := h.maint.CompactionApply(r.Context(), req.GetConfirm())
	if err != nil {
		httputil.HandleError(w, h.log, r, mapMaintenanceError(err))
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, &maintenancepb.MetricsCompactionApplyResponse{
		Success:             true,
		DatabaseStatsBefore: convert.DatabaseStatsToProto(result.StatsBefore),
		DatabaseStatsAfter:  convert.DatabaseStatsToProto(result.StatsAfter),
		ReclaimedBytes:      result.ReclaimedBytes,
	})
}
