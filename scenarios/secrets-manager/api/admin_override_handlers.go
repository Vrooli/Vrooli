// Package main provides admin HTTP handlers for scenario override management.
package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// AdminOverrideHandlers handles admin operations for scenario overrides.
type AdminOverrideHandlers struct {
	store  *ScenarioOverrideStore
	logger *Logger
}

// NewAdminOverrideHandlers creates handlers for admin override operations.
func NewAdminOverrideHandlers(db *sql.DB, logger *Logger) *AdminOverrideHandlers {
	return &AdminOverrideHandlers{
		store:  NewScenarioOverrideStore(db, logger),
		logger: logger,
	}
}

// RegisterRoutes mounts admin override endpoints under the provided router.
// Routes:
//
//	GET  /overrides/orphans  - List overrides for non-existent scenarios or removed dependencies
//	POST /overrides/cleanup  - Remove orphan overrides (with dry-run option)
func (h *AdminOverrideHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/overrides/orphans", h.ListOrphans).Methods("GET")
	router.HandleFunc("/overrides/cleanup", h.CleanupOrphans).Methods("POST")
}

// ListOrphans returns all overrides that reference non-existent scenarios or dependencies.
func (h *AdminOverrideHandlers) ListOrphans(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	orphans, err := h.findOrphanOverrides(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"orphans": orphans,
		"count":   len(orphans),
	})
}

// CleanupOrphans removes orphan overrides from the database.
func (h *AdminOverrideHandlers) CleanupOrphans(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	var req CleanupOrphansRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to dry run if no body
		req.DryRun = true
	}

	orphans, err := h.findOrphanOverrides(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(orphans) == 0 {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"dry_run":     req.DryRun,
			"deleted":     0,
			"orphan_ids":  []string{},
			"message":     "no orphan overrides found",
		})
		return
	}

	// Extract orphan IDs
	orphanIDs := make([]string, len(orphans))
	for i, orphan := range orphans {
		orphanIDs[i] = orphan.Override.ID
	}

	var deleted int
	if !req.DryRun {
		deleted, err = h.store.DeleteOverridesByID(r.Context(), orphanIDs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"dry_run":     req.DryRun,
		"deleted":     deleted,
		"would_delete": len(orphans),
		"orphan_ids":  orphanIDs,
		"orphans":     orphans,
	})
}

// findOrphanOverrides identifies overrides that no longer have valid scenarios or dependencies.
func (h *AdminOverrideHandlers) findOrphanOverrides(r *http.Request) ([]OrphanOverride, error) {
	// Get all overrides
	allOverrides, err := h.store.FetchAllOverrides(r.Context())
	if err != nil {
		return nil, err
	}

	if len(allOverrides) == 0 {
		return []OrphanOverride{}, nil
	}

	// Get list of valid scenarios
	scenarios, err := fetchScenarioList(r.Context())
	if err != nil {
		// Can't fetch scenarios - assume all are valid to avoid false positives
		if h.logger != nil {
			h.logger.Info("could not fetch scenarios for orphan detection: %v", err)
		}
		scenarios = nil
	}

	// Build set of valid scenario names
	validScenarios := make(map[string]bool)
	for _, sc := range scenarios {
		validScenarios[sc.Name] = true
	}

	// Cache for scenario resource dependencies
	scenarioResources := make(map[string][]string)

	var orphans []OrphanOverride
	for _, override := range allOverrides {
		// Check if scenario exists
		if scenarios != nil && !validScenarios[override.ScenarioName] {
			orphans = append(orphans, OrphanOverride{
				Override: override,
				Reason:   "scenario not found: " + override.ScenarioName,
			})
			continue
		}

		// Check if scenario depends on the resource
		resources, cached := scenarioResources[override.ScenarioName]
		if !cached {
			resolver := NewResourceResolver(nil, h.logger)
			resources = resolver.resolveScenarioResources(override.ScenarioName)
			scenarioResources[override.ScenarioName] = resources
		}

		// If no resources defined, can't validate - skip
		if len(resources) == 0 {
			continue
		}

		// Check if resource is in dependencies
		found := false
		for _, res := range resources {
			if res == override.ResourceName {
				found = true
				break
			}
		}

		if !found {
			orphans = append(orphans, OrphanOverride{
				Override: override,
				Reason:   "scenario '" + override.ScenarioName + "' no longer depends on resource '" + override.ResourceName + "'",
			})
		}
	}

	return orphans, nil
}
