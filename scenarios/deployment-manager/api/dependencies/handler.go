package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"deployment-manager/fitness"
	"deployment-manager/shared"

	"github.com/gorilla/mux"
)

// Handler handles dependency analysis requests.
type Handler struct {
	log func(string, map[string]interface{})
}

// NewHandler creates a new dependencies handler.
func NewHandler(log func(string, map[string]interface{})) *Handler {
	return &Handler{log: log}
}

// AnalyzeDependencies analyzes dependencies for a scenario.
// [REQ:DM-P0-001,DM-P0-002,DM-P0-003,DM-P0-006]
func (h *Handler) AnalyzeDependencies(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	if scenarioName == "" {
		http.Error(w, `{"error":"scenario parameter required"}`, http.StatusBadRequest)
		return
	}

	// Resolve analyzer URL
	analyzerBaseURL, err := shared.GetConfigResolver().ResolveAnalyzerURL()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusServiceUnavailable)
		return
	}

	analyzerURL := fmt.Sprintf("%s/api/v1/analyze/%s", analyzerBaseURL, scenarioName)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", analyzerURL, nil)
	if err != nil {
		http.Error(w, `{"error":"failed to create request"}`, http.StatusInternalServerError)
		return
	}

	resp, err := shared.GetHTTPClient(ctx).Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to call dependency analyzer: %s"}`, err.Error()), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Pass through error status codes from dependency analyzer
	if resp.StatusCode == http.StatusNotFound {
		http.Error(w, fmt.Sprintf(`{"error":"scenario '%s' not found"}`, scenarioName), http.StatusNotFound)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf(`{"error":"dependency analyzer returned status %d"}`, resp.StatusCode), resp.StatusCode)
		return
	}

	var analysisData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&analysisData); err != nil {
		http.Error(w, `{"error":"failed to decode analyzer response"}`, http.StatusInternalServerError)
		return
	}

	// Add circular dependency detection
	circularDeps := DetectCircularDependencies(analysisData)

	// [REQ:DM-P0-002] Return error if circular dependencies detected
	if len(circularDeps) > 0 {
		response := map[string]interface{}{
			"error":                 "Circular dependencies detected",
			"circular_dependencies": circularDeps,
			"remediation":           "Review and break circular dependency chain by restructuring scenario dependencies",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Calculate aggregate resources
	aggregateReqs := CalculateAggregateRequirements(analysisData)

	// Calculate fitness scores for all tiers
	tiers := fitness.AllTiers()
	fitnessScores := make(map[string]interface{})
	for _, tier := range tiers {
		score := fitness.CalculateScore(scenarioName, tier)
		tierName := fitness.GetTierDisplayName(tier)
		fitnessScores[tierName] = map[string]interface{}{
			"overall":          score.Overall,
			"portability":      score.Portability,
			"resources":        score.Resources,
			"licensing":        score.Licensing,
			"platform_support": score.PlatformSupport,
		}
	}

	response := map[string]interface{}{
		"scenario":               scenarioName,
		"dependencies":           analysisData,
		"circular_dependencies":  circularDeps,
		"aggregate_requirements": aggregateReqs,
		"tiers":                  fitnessScores,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
