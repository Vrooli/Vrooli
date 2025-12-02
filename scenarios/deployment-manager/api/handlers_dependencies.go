package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleAnalyzeDependencies analyzes dependencies for a scenario.
// [REQ:DM-P0-001,DM-P0-002,DM-P0-003,DM-P0-006]
func (s *Server) handleAnalyzeDependencies(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	if scenarioName == "" {
		http.Error(w, `{"error":"scenario parameter required"}`, http.StatusBadRequest)
		return
	}

	// Resolve analyzer port
	analyzerPort, err := resolveAnalyzerPort()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusServiceUnavailable)
		return
	}

	analyzerURL := fmt.Sprintf("http://localhost:%s/api/v1/analyze/%s", analyzerPort, scenarioName)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", analyzerURL, nil)
	if err != nil {
		http.Error(w, `{"error":"failed to create request"}`, http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultClient.Do(req)
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
	circularDeps := detectCircularDependencies(analysisData)

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
	aggregateReqs := calculateAggregateRequirements(analysisData)

	// Calculate fitness scores for all tiers
	tiers := []int{1, 2, 3, 4, 5}
	fitnessScores := make(map[string]interface{})
	for _, tier := range tiers {
		score := calculateFitnessScore(scenarioName, tier)
		tierName := getTierName(tier)
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

// handleScoreFitness calculates fitness scores for a scenario across tiers.
// [REQ:DM-P0-003,DM-P0-004,DM-P0-005,DM-P0-006]
func (s *Server) handleScoreFitness(w http.ResponseWriter, r *http.Request) {
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
		req.Tiers = []int{1, 2, 3, 4, 5} // all tiers by default
	}

	// Calculate fitness scores using hard-coded rules
	scores := make(map[int]interface{})
	blockers := []string{}
	warnings := []string{}

	for _, tier := range req.Tiers {
		fitnessScore := calculateFitnessScore(req.Scenario, tier)

		scores[tier] = map[string]interface{}{
			"overall":          fitnessScore.Overall,
			"portability":      fitnessScore.Portability,
			"resources":        fitnessScore.Resources,
			"licensing":        fitnessScore.Licensing,
			"platform_support": fitnessScore.PlatformSupport,
		}

		if fitnessScore.Overall == 0 {
			blockers = append(blockers, fmt.Sprintf("Tier %d: %s", tier, fitnessScore.BlockerReason))
		} else if fitnessScore.Overall < 50 {
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

// detectCircularDependencies finds circular references in the dependency graph.
func detectCircularDependencies(analysisData map[string]interface{}) []string {
	circular := []string{}

	// Extract dependencies map if it exists
	deps, ok := analysisData["dependencies"].(map[string]interface{})
	if !ok {
		return circular
	}

	// Check for circular reference patterns in the dependency structure
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(string, []string) bool
	dfs = func(node string, path []string) bool {
		if stack[node] {
			// Found a cycle - build the circular dependency path
			cycleStart := -1
			for i, p := range path {
				if p == node {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cyclePath := append(path[cycleStart:], node)
				circular = append(circular, strings.Join(cyclePath, " → "))
			}
			return true
		}

		if visited[node] {
			return false
		}

		visited[node] = true
		stack[node] = true
		path = append(path, node)

		// Check if this dependency has sub-dependencies
		if nodeDeps, ok := deps[node].(map[string]interface{}); ok {
			if subDeps, ok := nodeDeps["dependencies"].(map[string]interface{}); ok {
				for depName := range subDeps {
					if dfs(depName, path) {
						stack[node] = false
						return true
					}
				}
			}
		}

		stack[node] = false
		return false
	}

	// Start DFS from each top-level dependency
	for depName := range deps {
		if !visited[depName] {
			dfs(depName, []string{})
		}
	}

	return circular
}

// calculateAggregateRequirements computes total resource requirements.
func calculateAggregateRequirements(analysisData map[string]interface{}) map[string]interface{} {
	// Placeholder for aggregate requirement calculation
	return map[string]interface{}{
		"memory":  "512MB",
		"cpu":     "1 core",
		"gpu":     "none",
		"storage": "1GB",
		"network": "broadband",
	}
}
