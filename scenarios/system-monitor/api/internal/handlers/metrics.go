package handlers
// DOC: docs/reference/api-endpoints.md#metrics

import (
	"log/slog"
	"net/http"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/convert"
	"system-monitor-api/internal/httputil"
	"system-monitor-api/internal/models"
)

// MetricsHandler handles metrics-related requests
type MetricsHandler struct {
	log        *slog.Logger
	config     *config.Config
	monitorSvc MonitorQuerier
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(cfg *config.Config, monitorSvc MonitorQuerier, log *slog.Logger) *MetricsHandler {
	return &MetricsHandler{
		log:        log,
		config:     cfg,
		monitorSvc: monitorSvc,
	}
}

// GetCurrentMetrics handles GET /api/v1/metrics/current
func (h *MetricsHandler) GetCurrentMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fresh := r.URL.Query().Get("fresh")
	var (
		metrics *models.MetricsResponse
		err     error
	)
	if fresh == "1" || fresh == "true" {
		metrics, err = h.monitorSvc.GetCurrentMetricsFresh(ctx)
	} else {
		metrics, err = h.monitorSvc.GetCurrentMetrics(ctx)
	}
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.MetricsResponseToProto(metrics))
}

// GetDetailedMetrics handles GET /api/v1/metrics/detailed
func (h *MetricsHandler) GetDetailedMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metrics, err := h.monitorSvc.GetDetailedMetrics(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.DetailedMetricsToProto(metrics))
}

// GetProcessMonitor handles GET /api/v1/metrics/processes
func (h *MetricsHandler) GetProcessMonitor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := h.monitorSvc.GetProcessMonitorData(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.ProcessMonitorDataToProto(data))
}

// GetInfrastructureMonitor handles GET /api/v1/metrics/infrastructure
func (h *MetricsHandler) GetInfrastructureMonitor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := h.monitorSvc.GetInfrastructureMonitorData(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.InfrastructureMonitorDataToProto(data))
}
