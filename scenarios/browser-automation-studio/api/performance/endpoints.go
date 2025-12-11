// Package performance provides debug endpoints for performance monitoring.
package performance

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
)

// Endpoints handles HTTP endpoints for debug performance data.
type Endpoints struct {
	registry *CollectorRegistry
	log      *logrus.Logger
}

// NewEndpoints creates a new Endpoints handler.
func NewEndpoints(registry *CollectorRegistry, log *logrus.Logger) *Endpoints {
	return &Endpoints{
		registry: registry,
		log:      log,
	}
}

// DebugPerformanceResponse is the response for GET /debug/performance/{sessionId}
type DebugPerformanceResponse struct {
	Enabled      bool                  `json:"enabled"`
	SessionID    string                `json:"session_id"`
	Stats        *FrameStatsAggregated `json:"stats,omitempty"`
	RecentFrames []FrameTimings        `json:"recent_frames,omitempty"`
	Message      string                `json:"message,omitempty"`
}

// GetPerformance handles GET /debug/performance/{sessionId}
// Returns aggregated stats and recent frame timings for a session.
func (e *Endpoints) GetPerformance(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	cfg := config.Load()

	// Check if endpoint is exposed
	if !cfg.Performance.ExposeEndpoint {
		http.Error(w, "Performance endpoint disabled", http.StatusForbidden)
		return
	}

	// Get collector for session
	collector := e.registry.Get(sessionID)
	if collector == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DebugPerformanceResponse{
			Enabled:   cfg.Performance.Enabled,
			SessionID: sessionID,
			Message:   "No performance data for session (session not found or no frames recorded)",
		})
		return
	}

	// Get aggregated stats
	stats := collector.GetAggregated()
	recentFrames := collector.GetRecentFrames(20)

	response := DebugPerformanceResponse{
		Enabled:      cfg.Performance.Enabled,
		SessionID:    sessionID,
		Stats:        &stats,
		RecentFrames: recentFrames,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetAllPerformance handles GET /debug/performance
// Returns a summary of all active session collectors.
func (e *Endpoints) GetAllPerformance(w http.ResponseWriter, r *http.Request) {
	cfg := config.Load()

	// Check if endpoint is exposed
	if !cfg.Performance.ExposeEndpoint {
		http.Error(w, "Performance endpoint disabled", http.StatusForbidden)
		return
	}

	response := map[string]interface{}{
		"enabled":         cfg.Performance.Enabled,
		"active_sessions": e.registry.Count(),
		"config": map[string]interface{}{
			"buffer_size":          cfg.Performance.BufferSize,
			"log_summary_interval": cfg.Performance.LogSummaryInterval,
			"stream_to_websocket":  cfg.Performance.StreamToWebSocket,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the debug performance routes on a chi router.
func (e *Endpoints) RegisterRoutes(r chi.Router) {
	r.Get("/debug/performance", e.GetAllPerformance)
	r.Get("/debug/performance/{sessionId}", e.GetPerformance)
}
