package handlers

// DOC: docs/reference/api-endpoints.md#metrics

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/convert"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
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

// GetMetricsTimeline handles GET /api/v1/metrics/timeline
func (h *MetricsHandler) GetMetricsTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	windowSeconds := 120
	if ws := r.URL.Query().Get("window"); ws != "" {
		if parsed, err := strconv.Atoi(ws); err == nil && parsed > 0 {
			windowSeconds = parsed
		}
	}

	sampleInterval := 5
	if si := r.URL.Query().Get("interval"); si != "" {
		if parsed, err := strconv.Atoi(si); err == nil && parsed > 0 {
			sampleInterval = parsed
		}
	}

	timeline, err := h.monitorSvc.GetMetricsTimeline(ctx, windowSeconds, sampleInterval)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.MetricsTimelineResponseToProto(timeline))
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

// processTimelineEntryJSON is the wire shape for one ranked consumer. Plain
// JSON (not proto) following the forensics/logs precedent in this scenario:
// the attribution timeline has no cross-scenario clients, so adding ~2 proto
// messages + a convert layer would be disproportionate. See forensics.go.
type processTimelineEntryJSON struct {
	Owner       string  `json:"owner"`
	Comm        string  `json:"comm"`
	PID         int     `json:"pid,omitempty"`
	Aggregated  bool    `json:"aggregated"`
	CPUPct      float64 `json:"cpu_pct"`
	RSSKB       int64   `json:"rss_kb"`
	SampleCount int64   `json:"sample_count"`
	FirstSeen   string  `json:"first_seen,omitempty"`
	LastSeen    string  `json:"last_seen,omitempty"`
}

type processTimelineResponseJSON struct {
	WindowSeconds int                        `json:"window_seconds"`
	Owner         string                     `json:"owner,omitempty"`
	Top           int                        `json:"top"`
	Count         int                        `json:"count"`
	Entries       []processTimelineEntryJSON `json:"entries"`
}

// GetProcessTimeline handles GET /api/v1/metrics/processes/timeline. Query
// params: window (duration, default 5m), owner (scenario filter), top (int).
// It returns ranked consumers over the window, grouped by owner/scenario —
// the standing replacement for the manual `ps`/`top` forensic.
func (h *MetricsHandler) GetProcessTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	window := 5 * time.Minute
	if ws := r.URL.Query().Get("window"); ws != "" {
		if parsed, err := time.ParseDuration(ws); err == nil && parsed > 0 {
			window = parsed
		} else if secs, err := strconv.Atoi(ws); err == nil && secs > 0 {
			// Accept a bare integer as seconds for parity with /metrics/timeline.
			window = time.Duration(secs) * time.Second
		}
	}

	owner := r.URL.Query().Get("owner")

	top := 20
	if t := r.URL.Query().Get("top"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil && parsed > 0 {
			top = parsed
		}
	}

	entries, err := h.monitorSvc.GetProcessTimeline(ctx, window, owner, top)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	out := make([]processTimelineEntryJSON, 0, len(entries))
	for _, e := range entries {
		row := processTimelineEntryJSON{
			Owner:       e.Owner,
			Comm:        e.Comm,
			PID:         e.PID,
			Aggregated:  e.Aggregated,
			CPUPct:      e.CPUPct,
			RSSKB:       e.RSSKB,
			SampleCount: e.SampleCount,
		}
		if !e.FirstSeen.IsZero() {
			row.FirstSeen = e.FirstSeen.UTC().Format(time.RFC3339)
		}
		if !e.LastSeen.IsZero() {
			row.LastSeen = e.LastSeen.UTC().Format(time.RFC3339)
		}
		out = append(out, row)
	}

	httputil.JSON(w, processTimelineResponseJSON{ //nolint:errcheck
		WindowSeconds: int(window.Seconds()),
		Owner:         owner,
		Top:           top,
		Count:         len(out),
		Entries:       out,
	})
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
