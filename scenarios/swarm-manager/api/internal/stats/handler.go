package stats

import (
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Handler provides the HTTP endpoint for stats.
type Handler struct {
	engine *Engine
}

// NewHandler creates a handler backed by the given stats engine.
func NewHandler(engine *Engine) *Handler {
	return &Handler{engine: engine}
}

// RegisterRoutes registers stats endpoints on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/stats", h.GetStats).Methods("GET")
}

// GetStats refreshes metrics from the event log and returns the computed stats.
// Optional query param: ?category=throughput,blocking (comma-separated) to filter.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	if err := h.engine.Refresh(r.Context()); err != nil {
		apierr.MapError(w, "[stats] refresh", apierr.Internal("failed to refresh stats"))
		return
	}

	resp := h.engine.GetStats()

	// Category filtering: if specified, zero out non-requested sections.
	if cats := r.URL.Query().Get("category"); cats != "" {
		requested := make(map[string]bool)
		for _, c := range strings.Split(cats, ",") {
			requested[strings.TrimSpace(c)] = true
		}
		if !requested["throughput"] {
			resp.Throughput = ThroughputStats{}
		}
		if !requested["timing"] {
			resp.Timing = TimingStats{}
		}
		if !requested["scope"] {
			resp.Scope = ScopeStats{}
		}
		if !requested["blocking"] {
			resp.Blocking = BlockingStats{}
		}
		if !requested["agent"] {
			resp.Agent = AgentStats{}
		}
		if !requested["dashboard"] {
			resp.Dashboard = DashboardStats{}
		}
		if !requested["session"] && !requested["sessions"] {
			resp.Session = SessionStats{}
		}
	}

	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[stats] encode", apierr.Internal("failed to encode response"))
	}
}
