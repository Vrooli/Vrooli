package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/vrooli/system-monitor/internal/config"
	"github.com/vrooli/system-monitor/internal/services"
)

// MetricsHandler handles metrics-related requests
type MetricsHandler struct {
	config     *config.Config
	monitorSvc *services.MonitorService
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(cfg *config.Config, monitorSvc *services.MonitorService) *MetricsHandler {
	return &MetricsHandler{
		config:     cfg,
		monitorSvc: monitorSvc,
	}
}

// GetCurrentMetrics handles GET /api/metrics/current
func (h *MetricsHandler) GetCurrentMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	metrics, err := h.monitorSvc.GetCurrentMetrics(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}
	
	respondWithJSON(w, http.StatusOK, metrics)
}

// GetDetailedMetrics handles GET /api/metrics/detailed
func (h *MetricsHandler) GetDetailedMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	metrics, err := h.monitorSvc.GetDetailedMetrics(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}
	
	respondWithJSON(w, http.StatusOK, metrics)
}

// GetProcessMonitor handles GET /api/metrics/processes
func (h *MetricsHandler) GetProcessMonitor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	data, err := h.monitorSvc.GetProcessMonitorData(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}
	
	respondWithJSON(w, http.StatusOK, data)
}

// GetInfrastructureMonitor handles GET /api/metrics/infrastructure
func (h *MetricsHandler) GetInfrastructureMonitor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	data, err := h.monitorSvc.GetInfrastructureMonitorData(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}
	
	respondWithJSON(w, http.StatusOK, data)
}