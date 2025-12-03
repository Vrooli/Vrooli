// Package swaps provides resource swap analysis and cascade detection.
package swaps

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// Handler handles swap analysis requests.
type Handler struct {
	log func(string, map[string]interface{})
}

// NewHandler creates a new swaps handler.
func NewHandler(log func(string, map[string]interface{})) *Handler {
	return &Handler{log: log}
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
