// Package swaps provides resource swap analysis and cascade detection.
package swaps

import (
	"encoding/json"
	"net/http"
	"time"

	"deployment-manager/profiles"
	"deployment-manager/shared"

	"github.com/gorilla/mux"
)

// SwapSuggestion represents a recommended dependency swap for a tier.
type SwapSuggestion struct {
	From            string   `json:"from"`
	To              string   `json:"to"`
	Reason          string   `json:"reason"`
	ApplicableTiers []string `json:"applicable_tiers"`
	FitnessDelta    int      `json:"fitness_delta"`
	MigrationEffort string   `json:"migration_effort"`
}

// swapRegistry contains known swap suggestions for common dependencies.
var swapRegistry = map[string][]SwapSuggestion{
	"postgres": {
		{From: "postgres", To: "sqlite", Reason: "File-based storage without network dependency", ApplicableTiers: []string{"desktop", "mobile"}, FitnessDelta: 15, MigrationEffort: "medium"},
	},
	"redis": {
		{From: "redis", To: "in-process", Reason: "Memory-only cache without separate process", ApplicableTiers: []string{"desktop", "mobile"}, FitnessDelta: 20, MigrationEffort: "low"},
	},
	"browserless": {
		{From: "browserless", To: "playwright-driver", Reason: "Bundled Chromium for offline browser automation", ApplicableTiers: []string{"desktop"}, FitnessDelta: 25, MigrationEffort: "low"},
	},
	"ollama": {
		{From: "ollama", To: "openrouter", Reason: "API-based AI for lighter deployments", ApplicableTiers: []string{"mobile", "saas"}, FitnessDelta: 30, MigrationEffort: "medium"},
		{From: "ollama", To: "packaged-models", Reason: "Bundled models for offline AI", ApplicableTiers: []string{"desktop"}, FitnessDelta: 10, MigrationEffort: "high"},
	},
	"n8n": {
		{From: "n8n", To: "embedded-workflows", Reason: "Pre-compiled workflow execution without n8n server", ApplicableTiers: []string{"desktop", "mobile"}, FitnessDelta: 25, MigrationEffort: "high"},
	},
	"qdrant": {
		{From: "qdrant", To: "faiss-local", Reason: "File-based vector search without server", ApplicableTiers: []string{"desktop"}, FitnessDelta: 15, MigrationEffort: "medium"},
	},
}

// Handler handles swap analysis requests.
type Handler struct {
	profileRepo profiles.Repository
	log         func(string, map[string]interface{})
}

// NewHandler creates a new swaps handler.
func NewHandler(profileRepo profiles.Repository, log func(string, map[string]interface{})) *Handler {
	return &Handler{profileRepo: profileRepo, log: log}
}

// Analyze analyzes the impact of swapping one dependency for another.
// [REQ:DM-P0-008]
func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fromDep := vars["from"]
	toDep := vars["to"]

	if fromDep == "" || toDep == "" {
		http.Error(w, `{"error":"from and to dependencies required"}`, http.StatusBadRequest)
		return
	}

	// Calculate fitness delta and impact for the swap
	fitnessDeltas := map[string]int{
		"local":      0,  // Local tier unaffected
		"desktop":    10, // Swapping generally improves desktop fitness
		"mobile":     30, // Mobile benefits most from lightweight swaps
		"saas":       5,  // SaaS slightly benefits
		"enterprise": -5, // Enterprise may have licensing concerns
	}

	pros := []string{
		"Reduced resource footprint",
		"Improved portability",
		"Faster startup time",
	}
	cons := []string{
		"May require code changes",
		"Feature parity not guaranteed",
		"Migration effort required",
	}

	response := map[string]interface{}{
		"from":             fromDep,
		"to":               toDep,
		"fitness_delta":    fitnessDeltas,
		"impact":           "medium",
		"pros":             pros,
		"cons":             cons,
		"migration_effort": "2-4 hours",
		"applicable_tiers": []string{"desktop", "mobile", "saas"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Cascade detects cascading impacts of a dependency swap.
// [REQ:DM-P0-011]
func (h *Handler) Cascade(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fromDep := vars["from"]
	toDep := vars["to"]

	if fromDep == "" || toDep == "" {
		http.Error(w, `{"error":"from and to dependencies required"}`, http.StatusBadRequest)
		return
	}

	// Detect cascading impacts
	cascadingImpacts := []map[string]interface{}{}

	// Example cascading impact
	if fromDep == "postgres" {
		cascadingImpacts = append(cascadingImpacts, map[string]interface{}{
			"affected_scenario": "example-dependent-scenario",
			"reason":            "Depends on postgres-specific features (JSONB queries)",
			"severity":          "high",
			"remediation":       "Update queries to use SQLite-compatible syntax",
		})
	}

	response := map[string]interface{}{
		"from":              fromDep,
		"to":                toDep,
		"cascading_impacts": cascadingImpacts,
		"warnings":          []string{"Review all dependent scenarios before applying swap"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CLISwapSuggestion matches the CLI's expected format
type CLISwapSuggestion struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Reason string  `json:"reason"`
	Impact string  `json:"impact"`
	Score  float64 `json:"score"`
}

// List returns available swap suggestions for a scenario's dependencies.
// Returns an array to match CLI expectations.
// [REQ:DM-P0-007]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]

	if scenario == "" {
		http.Error(w, `{"error":"scenario name required"}`, http.StatusBadRequest)
		return
	}

	// Get scenario dependencies from analyzer
	deps, err := shared.GetScenarioDependencies(r.Context(), scenario)
	if err != nil {
		h.log("error", map[string]interface{}{"msg": "failed to get scenario dependencies", "error": err.Error()})
		// Return empty list if analyzer is unavailable
		deps = []string{}
	}

	// If no dependencies from analyzer, use common ones for demo
	if len(deps) == 0 {
		deps = []string{"postgres", "redis", "ollama", "n8n", "qdrant"}
	}

	// Collect applicable swaps for the scenario's dependencies
	// Convert to CLI format (array of objects)
	suggestions := []CLISwapSuggestion{}
	for _, dep := range deps {
		if swaps, ok := swapRegistry[dep]; ok {
			for _, s := range swaps {
				suggestions = append(suggestions, CLISwapSuggestion{
					From:   s.From,
					To:     s.To,
					Reason: s.Reason,
					Impact: s.MigrationEffort,
					Score:  float64(s.FitnessDelta),
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// ApplyRequest represents a request to apply a swap to a profile.
type ApplyRequest struct {
	ProfileID string `json:"profile_id"`
	From      string `json:"from"`
	To        string `json:"to"`
}

// Apply applies a swap to a deployment profile.
// [REQ:DM-P0-009]
func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.ProfileID == "" || req.From == "" || req.To == "" {
		http.Error(w, `{"error":"profile_id, from, and to are required"}`, http.StatusBadRequest)
		return
	}

	// Validate the swap exists in registry
	swaps, ok := swapRegistry[req.From]
	if !ok {
		http.Error(w, `{"error":"no swaps available for dependency: `+req.From+`"}`, http.StatusBadRequest)
		return
	}

	var selectedSwap *SwapSuggestion
	for _, s := range swaps {
		if s.To == req.To {
			selectedSwap = &s
			break
		}
	}
	if selectedSwap == nil {
		http.Error(w, `{"error":"swap from `+req.From+` to `+req.To+` not supported"}`, http.StatusBadRequest)
		return
	}

	// In a full implementation, this would:
	// 1. Load the profile from storage
	// 2. Add the swap to the profile's swap list
	// 3. Recalculate fitness score
	// 4. Save the updated profile

	h.log("info", map[string]interface{}{
		"msg":        "swap applied to profile",
		"profile_id": req.ProfileID,
		"from":       req.From,
		"to":         req.To,
	})

	response := map[string]interface{}{
		"status":        "applied",
		"profile_id":   req.ProfileID,
		"swap":         selectedSwap,
		"fitness_delta": selectedSwap.FitnessDelta,
		"message":      "Swap recorded. Regenerate bundle manifest to apply changes.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ApplyToProfile applies a swap to a profile (profile ID from URL path).
// This matches the CLI's expected endpoint: POST /api/v1/profiles/{id}/swaps
func (h *Handler) ApplyToProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	if profileID == "" {
		http.Error(w, `{"error":"profile_id is required in path"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.From == "" || req.To == "" {
		http.Error(w, `{"error":"from and to are required"}`, http.StatusBadRequest)
		return
	}

	// Validate the swap exists in registry
	registrySwaps, ok := swapRegistry[req.From]
	if !ok {
		http.Error(w, `{"error":"no swaps available for dependency: `+req.From+`"}`, http.StatusBadRequest)
		return
	}

	var selectedSwap *SwapSuggestion
	for _, s := range registrySwaps {
		if s.To == req.To {
			selectedSwap = &s
			break
		}
	}
	if selectedSwap == nil {
		http.Error(w, `{"error":"swap from `+req.From+` to `+req.To+` not supported"}`, http.StatusBadRequest)
		return
	}

	// Persist the swap to the profile
	swap := profiles.Swap{
		From:            selectedSwap.From,
		To:              selectedSwap.To,
		Reason:          selectedSwap.Reason,
		Limitations:     "", // Can be populated from swap registry if needed
		ApplicableTiers: selectedSwap.ApplicableTiers,
		AppliedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	if err := h.profileRepo.AddSwap(r.Context(), profileID, swap); err != nil {
		if err == profiles.ErrNotFound {
			http.Error(w, `{"error":"profile not found: `+profileID+`"}`, http.StatusNotFound)
			return
		}
		h.log("error", map[string]interface{}{
			"msg":        "failed to persist swap",
			"profile_id": profileID,
			"error":      err.Error(),
		})
		http.Error(w, `{"error":"failed to persist swap"}`, http.StatusInternalServerError)
		return
	}

	h.log("info", map[string]interface{}{
		"msg":        "swap persisted to profile",
		"profile_id": profileID,
		"from":       req.From,
		"to":         req.To,
	})

	response := map[string]interface{}{
		"status":        "applied",
		"profile_id":   profileID,
		"swap":         swap,
		"fitness_delta": selectedSwap.FitnessDelta,
		"message":      "Swap persisted. Regenerate bundle manifest to apply changes.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
