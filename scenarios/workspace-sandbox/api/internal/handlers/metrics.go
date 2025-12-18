package handlers

import (
	"net/http"
)

// --- Metrics Endpoint [OT-P1-008] ---

// MetricsCollector is an interface for collecting and exporting metrics.
type MetricsCollector interface {
	ExportPrometheus() string
	Snapshot() map[string]interface{}
	SetActiveSandboxes(count int64)
	SetTotalDiskUsage(bytes int64)
	SetProcessCount(count int64)
}

// Metrics handles exporting Prometheus-format metrics.
// [OT-P1-008] Metrics/Observability
func (h *Handlers) Metrics(w http.ResponseWriter, r *http.Request, collector MetricsCollector) {
	if collector == nil {
		h.JSONError(w, "metrics not available", http.StatusServiceUnavailable)
		return
	}

	// Update gauges from current state before export
	if h.StatsGetter != nil {
		if stats, err := h.StatsGetter.GetStats(r.Context()); err == nil {
			collector.SetActiveSandboxes(stats.ActiveCount)
			collector.SetTotalDiskUsage(stats.TotalSizeBytes)
		}
	}
	if h.ProcessTracker != nil {
		stats := h.ProcessTracker.GetAllStats()
		collector.SetProcessCount(int64(stats.TotalRunning))
	}

	// Check if JSON format is requested
	if r.URL.Query().Get("format") == "json" {
		h.JSONSuccess(w, collector.Snapshot())
		return
	}

	// Default to Prometheus text format
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(collector.ExportPrometheus()))
}
