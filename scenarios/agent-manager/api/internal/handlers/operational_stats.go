// HTTP handler for the operational-events stats engine.
//
// The handler enforces a typed Category enum at the edge: unknown
// categories return HTTP 400 with the list of known categories instead
// of silently returning empty stats. This is one of the four weakness
// fixes ported from swarm-manager (silent typo'd ?category=… on that
// scenario rendered a misleading zero).
//
// Refresh policy: each request triggers a Refresh before reading. The
// engine's Refresh is cheap when no new events are present (one SQL
// COUNT-shape query) and it ensures the response reflects the latest
// state without callers having to schedule background refreshes.

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"agent-manager/internal/stats"

	"github.com/gorilla/mux"
)

// OperationalCategory names a slice of the operational stats response.
// Closed enum so the HTTP layer can reject unknown values.
type OperationalCategory string

const (
	CategorySummary    OperationalCategory = "summary"
	CategoryFallback   OperationalCategory = "fallback"
	CategoryHealth     OperationalCategory = "health"
	CategorySandbox    OperationalCategory = "sandbox"
	CategoryHeartbeat  OperationalCategory = "heartbeat"
	CategoryCheckpoint OperationalCategory = "checkpoint"
	CategoryRetry      OperationalCategory = "retry"
)

// allCategories is the canonical ordered list returned in error
// responses so clients can correct their query.
var allCategories = []OperationalCategory{
	CategorySummary,
	CategoryFallback,
	CategoryHealth,
	CategorySandbox,
	CategoryHeartbeat,
	CategoryCheckpoint,
	CategoryRetry,
}

func parseCategory(raw string) (OperationalCategory, bool) {
	if raw == "" {
		return CategorySummary, true
	}
	for _, c := range allCategories {
		if string(c) == raw {
			return c, true
		}
	}
	return "", false
}

// OperationalStatsHandler exposes the new typed-event stats endpoints.
// It does NOT replace the existing StatsHandler (run-row-derived stats
// for runs/profiles/runners/etc.); the two coexist as separate views.
type OperationalStatsHandler struct {
	engine *stats.Engine
}

// NewOperationalStatsHandler wires a new handler.
func NewOperationalStatsHandler(engine *stats.Engine) *OperationalStatsHandler {
	return &OperationalStatsHandler{engine: engine}
}

// RegisterRoutes registers the operational-event stats routes.
func (h *OperationalStatsHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/stats/operational", h.GetOperational).Methods("GET")
	r.HandleFunc("/api/v1/stats/fallback", h.GetFallback).Methods("GET")
}

// GetOperational dispatches by ?category=... to one of the per-category
// builders. Unknown categories return 400.
func (h *OperationalStatsHandler) GetOperational(w http.ResponseWriter, r *http.Request) {
	cat, ok := parseCategory(r.URL.Query().Get("category"))
	if !ok {
		writeBadCategory(w, r.URL.Query().Get("category"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.engine.Refresh(ctx); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "stats refresh failed: "+err.Error())
		return
	}
	switch cat {
	case CategorySummary:
		writeJSON(w, http.StatusOK, h.engine.GetSummary())
	case CategoryFallback:
		writeJSON(w, http.StatusOK, h.engine.GetFallback())
	case CategoryHealth:
		writeJSON(w, http.StatusOK, h.engine.GetHealth())
	case CategorySandbox:
		writeJSON(w, http.StatusOK, h.engine.GetSandbox())
	case CategoryHeartbeat:
		writeJSON(w, http.StatusOK, h.engine.GetHeartbeat())
	case CategoryCheckpoint:
		writeJSON(w, http.StatusOK, h.engine.GetCheckpoint())
	case CategoryRetry:
		writeJSON(w, http.StatusOK, h.engine.GetRetry())
	default:
		writeJSONError(w, http.StatusInternalServerError, "unhandled category: "+string(cat))
	}
}

// GetFallback is the dedicated fallback-insights endpoint. Equivalent
// to /api/v1/stats/operational?category=fallback but stable enough to
// have its own URL.
func (h *OperationalStatsHandler) GetFallback(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.engine.Refresh(ctx); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "stats refresh failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.engine.GetFallback())
}

type categoryErrorResponse struct {
	Error           string   `json:"error"`
	KnownCategories []string `json:"known_categories"`
}

func writeBadCategory(w http.ResponseWriter, raw string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	known := make([]string, len(allCategories))
	for i, c := range allCategories {
		known[i] = string(c)
	}
	sort.Strings(known)
	_ = json.NewEncoder(w).Encode(categoryErrorResponse{
		Error:           fmt.Sprintf("unknown category %q", raw),
		KnownCategories: known,
	})
}
