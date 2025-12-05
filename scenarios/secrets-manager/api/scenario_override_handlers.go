// Package main provides HTTP handlers for scenario secret strategy overrides.
package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// ScenarioOverrideHandlers handles HTTP requests for scenario overrides.
type ScenarioOverrideHandlers struct {
	store  *ScenarioOverrideStore
	logger *Logger
}

// NewScenarioOverrideHandlers creates handlers for scenario override operations.
func NewScenarioOverrideHandlers(db *sql.DB, logger *Logger) *ScenarioOverrideHandlers {
	return &ScenarioOverrideHandlers{
		store:  NewScenarioOverrideStore(db, logger),
		logger: logger,
	}
}

// RegisterRoutes mounts scenario override endpoints under the provided router.
// Routes:
//
//	GET    /{scenario}/overrides                            - List ALL overrides (all tiers)
//	GET    /{scenario}/overrides/{tier}                     - List overrides for scenario+tier
//	GET    /{scenario}/overrides/{tier}/{resource}/{secret} - Get override for specific secret
//	POST   /{scenario}/overrides/{tier}/{resource}/{secret} - Create/update override
//	DELETE /{scenario}/overrides/{tier}/{resource}/{secret} - Remove override (revert to default)
//	GET    /{scenario}/effective/{tier}                     - Get effective strategies for deployment
//	POST   /{scenario}/overrides/copy-from-tier             - Copy overrides between tiers
//	POST   /{scenario}/overrides/copy-from-scenario         - Copy overrides from another scenario
func (h *ScenarioOverrideHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/{scenario}/overrides", h.ListAllOverrides).Methods("GET")
	router.HandleFunc("/{scenario}/overrides/{tier}", h.ListOverrides).Methods("GET")
	router.HandleFunc("/{scenario}/overrides/{tier}/{resource}/{secret}", h.GetOverride).Methods("GET")
	router.HandleFunc("/{scenario}/overrides/{tier}/{resource}/{secret}", h.SetOverride).Methods("POST")
	router.HandleFunc("/{scenario}/overrides/{tier}/{resource}/{secret}", h.DeleteOverride).Methods("DELETE")
	router.HandleFunc("/{scenario}/effective/{tier}", h.GetEffectiveStrategies).Methods("GET")
	router.HandleFunc("/{scenario}/overrides/copy-from-tier", h.CopyFromTier).Methods("POST")
	router.HandleFunc("/{scenario}/overrides/copy-from-scenario", h.CopyFromScenario).Methods("POST")
}

// ListAllOverrides returns all overrides for a scenario across all tiers.
func (h *ScenarioOverrideHandlers) ListAllOverrides(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	scenario := vars["scenario"]

	overrides, err := h.store.FetchOverridesByScenario(r.Context(), scenario)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"scenario":  scenario,
		"overrides": overrides,
		"count":     len(overrides),
	})
}

// ListOverrides returns all overrides for a scenario and tier.
func (h *ScenarioOverrideHandlers) ListOverrides(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	scenario := vars["scenario"]
	tier := vars["tier"]

	overrides, err := h.store.FetchOverridesByScenarioTier(r.Context(), scenario, tier)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"scenario":  scenario,
		"tier":      tier,
		"overrides": overrides,
		"count":     len(overrides),
	})
}

// GetOverride returns a specific override.
func (h *ScenarioOverrideHandlers) GetOverride(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	scenario := vars["scenario"]
	tier := vars["tier"]
	resource := vars["resource"]
	secret := vars["secret"]

	override, err := h.store.FetchOverride(r.Context(), scenario, tier, resource, secret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if override == nil {
		http.Error(w, "override not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(override)
}

// SetOverride creates or updates a scenario override.
func (h *ScenarioOverrideHandlers) SetOverride(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	scenario := vars["scenario"]
	tier := vars["tier"]
	resource := vars["resource"]
	secret := vars["secret"]

	// Parse request body
	var req SetOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate handling_strategy if provided
	if req.HandlingStrategy != nil {
		validStrategies := map[string]bool{
			"strip": true, "generate": true, "prompt": true, "delegate": true,
		}
		if !validStrategies[*req.HandlingStrategy] {
			http.Error(w, "invalid handling_strategy: must be strip, generate, prompt, or delegate", http.StatusBadRequest)
			return
		}
	}

	// Validate scenario exists and depends on resource
	if validationErr := h.store.ValidateScenarioDependency(r.Context(), scenario, resource); validationErr != "" {
		http.Error(w, validationErr, http.StatusBadRequest)
		return
	}

	// Create/update the override
	override, err := h.store.UpsertOverride(r.Context(), scenario, tier, resource, secret, req)
	if err != nil {
		if err.Error() == "secret "+resource+"/"+secret+" not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(override)
}

// DeleteOverride removes a scenario override.
func (h *ScenarioOverrideHandlers) DeleteOverride(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	scenario := vars["scenario"]
	tier := vars["tier"]
	resource := vars["resource"]
	secret := vars["secret"]

	if err := h.store.DeleteOverride(r.Context(), scenario, tier, resource, secret); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "override deleted",
	})
}

// GetEffectiveStrategies returns the merged strategies for a scenario and tier.
func (h *ScenarioOverrideHandlers) GetEffectiveStrategies(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	scenario := vars["scenario"]
	tier := vars["tier"]

	// Optional resource filter from query param
	var resources []string
	if resourceParam := r.URL.Query().Get("resources"); resourceParam != "" {
		resources = splitAndTrim(resourceParam, ",")
	}

	strategies, err := h.store.FetchEffectiveStrategies(r.Context(), scenario, tier, resources)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"scenario":   scenario,
		"tier":       tier,
		"strategies": strategies,
		"count":      len(strategies),
	})
}

// CopyFromTier copies overrides from one tier to another within the same scenario.
func (h *ScenarioOverrideHandlers) CopyFromTier(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	scenario := vars["scenario"]

	var req CopyFromTierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.SourceTier == "" || req.TargetTier == "" {
		http.Error(w, "source_tier and target_tier are required", http.StatusBadRequest)
		return
	}

	if req.SourceTier == req.TargetTier {
		http.Error(w, "source_tier and target_tier must be different", http.StatusBadRequest)
		return
	}

	count, err := h.store.CopyOverridesFromTier(r.Context(), scenario, req.SourceTier, req.TargetTier, req.Overwrite)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"copied":      count,
		"source_tier": req.SourceTier,
		"target_tier": req.TargetTier,
		"overwrite":   req.Overwrite,
	})
}

// CopyFromScenario copies overrides from another scenario.
func (h *ScenarioOverrideHandlers) CopyFromScenario(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		http.Error(w, "database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	scenario := vars["scenario"]

	var req CopyFromScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.SourceScenario == "" || req.Tier == "" {
		http.Error(w, "source_scenario and tier are required", http.StatusBadRequest)
		return
	}

	if req.SourceScenario == scenario {
		http.Error(w, "source_scenario and target scenario must be different", http.StatusBadRequest)
		return
	}

	count, err := h.store.CopyOverridesFromScenario(r.Context(), req.SourceScenario, scenario, req.Tier, req.Overwrite)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"copied":          count,
		"source_scenario": req.SourceScenario,
		"target_scenario": scenario,
		"tier":            req.Tier,
		"overwrite":       req.Overwrite,
	})
}

// splitAndTrim splits a string by separator and trims whitespace from each part.
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range split(s, sep) {
		part = trim(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
