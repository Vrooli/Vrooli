// Package handlers provides HTTP request handlers for the autoheal API
// [REQ:CLI-TICK-001] [REQ:CLI-TICK-002] [REQ:CLI-STATUS-001] [REQ:CLI-STATUS-002]
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/persistence"
	"vrooli-autoheal/internal/platform"
)

// Handlers wraps the dependencies needed by HTTP handlers
type Handlers struct {
	registry *checks.Registry
	store    *persistence.Store
	platform *platform.Capabilities
}

// New creates a new Handlers instance
func New(registry *checks.Registry, store *persistence.Store, plat *platform.Capabilities) *Handlers {
	return &Handlers{
		registry: registry,
		store:    store,
		platform: plat,
	}
}

// Health returns basic service health for lifecycle checks
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := h.store.Ping(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Vrooli Autoheal API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Platform returns detected platform capabilities
func (h *Handlers) Platform(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.platform)
}

// Status returns the current health summary
func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	summary := h.registry.GetSummary()

	response := map[string]interface{}{
		"status":   summary.Status,
		"platform": h.platform,
		"summary": map[string]interface{}{
			"total":    summary.TotalCount,
			"ok":       summary.OkCount,
			"warning":  summary.WarnCount,
			"critical": summary.CritCount,
		},
		"checks":    summary.Checks,
		"timestamp": summary.Timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Tick runs a single health check cycle
func (h *Handlers) Tick(w http.ResponseWriter, r *http.Request) {
	// Parse force parameter
	forceAll := r.URL.Query().Get("force") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	results := h.registry.RunAll(ctx, forceAll)

	// Store results in database
	for _, result := range results {
		if err := h.store.SaveResult(ctx, result); err != nil {
			// Log but don't fail the request
			_ = err // TODO: Add proper logging
		}
	}

	// Get updated summary
	summary := h.registry.GetSummary()

	response := map[string]interface{}{
		"success": true,
		"status":  summary.Status,
		"summary": map[string]interface{}{
			"total":    summary.TotalCount,
			"ok":       summary.OkCount,
			"warning":  summary.WarnCount,
			"critical": summary.CritCount,
		},
		"results":   results,
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListChecks returns all registered checks
func (h *Handlers) ListChecks(w http.ResponseWriter, r *http.Request) {
	checks := h.registry.ListChecks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checks)
}

// CheckResult returns the result for a specific check
func (h *Handlers) CheckResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]

	result, exists := h.registry.GetResult(checkID)
	if !exists {
		http.Error(w, "Check not found or not yet run", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
