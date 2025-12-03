package fitness

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler handles fitness scoring requests.
type Handler struct {
	log func(string, map[string]interface{})
}

// NewHandler creates a new fitness handler.
func NewHandler(log func(string, map[string]interface{})) *Handler {
	return &Handler{log: log}
}

// ScoreFitness calculates fitness scores for a scenario across tiers.
// [REQ:DM-P0-003,DM-P0-004,DM-P0-005,DM-P0-006]
func (h *Handler) ScoreFitness(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scenario string `json:"scenario"`
		Tiers    []int  `json:"tiers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Scenario == "" {
		http.Error(w, `{"error":"scenario field required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Tiers) == 0 {
		req.Tiers = AllTiers()
	}

	// Calculate fitness scores using hard-coded rules
	scores := make(map[int]interface{})
	blockers := []string{}
	warnings := []string{}

	for _, tier := range req.Tiers {
		fitnessScore := CalculateScore(req.Scenario, tier)

		scores[tier] = map[string]interface{}{
			"overall":          fitnessScore.Overall,
			"portability":      fitnessScore.Portability,
			"resources":        fitnessScore.Resources,
			"licensing":        fitnessScore.Licensing,
			"platform_support": fitnessScore.PlatformSupport,
		}

		if fitnessScore.Overall == 0 {
			blockers = append(blockers, fmt.Sprintf("Tier %d: %s", tier, fitnessScore.BlockerReason))
		} else if IsTierWarningLevel(fitnessScore.Overall) {
			warnings = append(warnings, fmt.Sprintf("Tier %d: Low fitness score (%d/100)", tier, fitnessScore.Overall))
		}
	}

	response := map[string]interface{}{
		"scenario": req.Scenario,
		"scores":   scores,
		"blockers": blockers,
		"warnings": warnings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
