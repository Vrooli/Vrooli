package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scenario-completeness-scoring/pkg/config"
	"scenario-completeness-scoring/pkg/scoring"
	"scenario-completeness-scoring/pkg/validators"

	"github.com/gorilla/mux"
)

// HandleListScores returns a list of all scenarios with their scores
// [REQ:SCS-CORE-002] Score retrieval API endpoint
func (ctx *Context) HandleListScores(w http.ResponseWriter, r *http.Request) {
	scenariosDir := filepath.Join(ctx.VrooliRoot, "scenarios")

	// Load config for scoring options
	cfg, _ := ctx.ConfigLoader.LoadGlobal()
	scoringOpts := configToScoringOptions(cfg)

	// Discover scenarios by listing directories
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		http.Error(w, "Failed to read scenarios directory", http.StatusInternalServerError)
		return
	}

	var scenarios []map[string]interface{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		scenarioName := entry.Name()

		// Try to collect metrics (gracefully handle failures)
		metrics, err := ctx.Collector.Collect(scenarioName)
		if err != nil {
			// Log but continue with other scenarios
			log.Printf("Warning: failed to collect metrics for %s: %v", scenarioName, err)
			continue
		}

		thresholds := scoring.GetThresholds(metrics.Category)
		breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, 0, scoringOpts)

		scenarios = append(scenarios, map[string]interface{}{
			"scenario":       scenarioName,
			"category":       metrics.Category,
			"score":          breakdown.Score,
			"classification": breakdown.Classification,
		})
	}

	response := map[string]interface{}{
		"scenarios":     scenarios,
		"total":         len(scenarios),
		"calculated_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetScore returns the detailed score for a specific scenario
// [REQ:SCS-CORE-002] Score retrieval API endpoint
func (ctx *Context) HandleGetScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Collect real metrics from the scenario
	metrics, err := ctx.Collector.Collect(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect metrics for scenario %s: %v", scenarioName, err), http.StatusNotFound)
		return
	}

	// Load config for scoring options
	cfg, _ := ctx.ConfigLoader.LoadGlobal()
	scoringOpts := configToScoringOptions(cfg)

	// Perform validation quality analysis
	scenarioRoot := ctx.Collector.GetScenarioRoot(scenarioName)
	requirements := ctx.Collector.LoadRequirements(scenarioName)

	validationAnalysis := validators.AnalyzeValidationQuality(
		validators.MetricCounts{
			RequirementsTotal: metrics.Requirements.Total,
			TestsTotal:        metrics.Tests.Total,
		},
		requirements, // No conversion needed - both use domain.Requirement
		nil,          // No operational targets yet
		scenarioRoot,
	)

	// Pass penalty directly - the full analysis is included in response for details
	thresholds := scoring.GetThresholds(metrics.Category)
	breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, validationAnalysis.TotalPenalty, scoringOpts)
	recommendations := scoring.GenerateRecommendations(breakdown, thresholds)

	response := map[string]interface{}{
		"scenario":            scenarioName,
		"category":            metrics.Category,
		"score":               breakdown.Score,
		"base_score":          breakdown.BaseScore,
		"validation_penalty":  breakdown.ValidationPenalty,
		"classification":      breakdown.Classification,
		"breakdown":           breakdown,
		"metrics":             metrics,
		"validation_analysis": validationAnalysis,
		"recommendations":     recommendations,
		"calculated_at":       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleCalculateScore forces recalculation of score for a scenario
// [REQ:SCS-CORE-002] Score retrieval API endpoint
// [REQ:SCS-HIST-001] Save score snapshot to history
func (ctx *Context) HandleCalculateScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Force recalculation by collecting fresh metrics
	metrics, err := ctx.Collector.Collect(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to calculate score for scenario %s: %v", scenarioName, err), http.StatusNotFound)
		return
	}

	// Load config for scoring options
	cfg, _ := ctx.ConfigLoader.LoadGlobal()
	scoringOpts := configToScoringOptions(cfg)

	thresholds := scoring.GetThresholds(metrics.Category)
	breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, 0, scoringOpts)
	recommendations := scoring.GenerateRecommendations(breakdown, thresholds)

	// Save snapshot to history [REQ:SCS-HIST-001]
	var snapshotID int64
	if ctx.HistoryRepo != nil {
		snapshot, err := ctx.HistoryRepo.Save(scenarioName, &breakdown, nil)
		if err != nil {
			log.Printf("Warning: failed to save history snapshot: %v", err)
		} else if snapshot != nil {
			snapshotID = snapshot.ID
		}
	}

	response := map[string]interface{}{
		"scenario":        scenarioName,
		"category":        metrics.Category,
		"score":           breakdown.Score,
		"classification":  breakdown.Classification,
		"breakdown":       breakdown,
		"metrics":         metrics,
		"recommendations": recommendations,
		"calculated_at":   time.Now().UTC().Format(time.RFC3339),
		"recalculated":    true,
		"snapshot_id":     snapshotID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleValidationAnalysis returns validation quality analysis for a scenario
// This detects anti-patterns and gaming behaviors in test validation
func (ctx *Context) HandleValidationAnalysis(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Collect metrics to get counts
	metrics, err := ctx.Collector.Collect(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect metrics for scenario %s: %v", scenarioName, err), http.StatusNotFound)
		return
	}

	// Perform validation quality analysis
	scenarioRoot := ctx.Collector.GetScenarioRoot(scenarioName)
	requirements := ctx.Collector.LoadRequirements(scenarioName)

	analysis := validators.AnalyzeValidationQuality(
		validators.MetricCounts{
			RequirementsTotal: metrics.Requirements.Total,
			TestsTotal:        metrics.Tests.Total,
		},
		requirements, // No conversion needed - both use domain.Requirement
		nil,          // No operational targets yet
		scenarioRoot,
	)

	response := map[string]interface{}{
		"scenario":    scenarioName,
		"analysis":    analysis,
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// configToScoringOptions converts config.ScoringConfig to scoring.ScoringOptions
// Normalizes weights to sum to 100 when some dimensions are disabled
func configToScoringOptions(cfg *config.ScoringConfig) *scoring.ScoringOptions {
	if cfg == nil {
		return scoring.DefaultScoringOptions()
	}

	// Normalize weights to redistribute when dimensions are disabled
	// This ensures the total always sums to 100
	normalizedWeights := cfg.Weights.Normalize(cfg.Components)

	return &scoring.ScoringOptions{
		QualityEnabled:  cfg.Components.Quality.Enabled,
		CoverageEnabled: cfg.Components.Coverage.Enabled,
		QuantityEnabled: cfg.Components.Quantity.Enabled,
		UIEnabled:       cfg.Components.UI.Enabled,
		QualityWeight:   normalizedWeights.Quality,
		CoverageWeight:  normalizedWeights.Coverage,
		QuantityWeight:  normalizedWeights.Quantity,
		UIWeight:        normalizedWeights.UI,
	}
}

// HandleGetRecommendations returns recommendations for a specific scenario
// [REQ:SCS-ANALYSIS-002] Prioritized recommendations API endpoint
func (ctx *Context) HandleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Collect metrics and calculate score
	metrics, err := ctx.Collector.Collect(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect metrics for scenario %s: %v", scenarioName, err), http.StatusNotFound)
		return
	}

	// Load config for scoring options
	cfg, _ := ctx.ConfigLoader.LoadGlobal()
	scoringOpts := configToScoringOptions(cfg)

	thresholds := scoring.GetThresholds(metrics.Category)
	breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, 0, scoringOpts)
	recommendations := scoring.GenerateRecommendations(breakdown, thresholds)

	// Calculate total potential impact
	totalImpact := 0
	for _, rec := range recommendations {
		totalImpact += rec.Impact
	}

	response := map[string]interface{}{
		"scenario":        scenarioName,
		"current_score":   breakdown.Score,
		"recommendations": recommendations,
		"total_impact":    totalImpact,
		"potential_score": breakdown.Score + totalImpact,
		"generated_at":    time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetThresholds returns all category thresholds
func (ctx *Context) HandleGetThresholds(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"categories":       scoring.CategoryThresholds,
		"default_category": "utility",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetCategoryThresholds returns thresholds for a specific category
func (ctx *Context) HandleGetCategoryThresholds(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	category := vars["category"]

	thresholds := scoring.GetThresholds(category)
	response := map[string]interface{}{
		"category":   category,
		"thresholds": thresholds,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to round properly
func round(x float64) int {
	return int(math.Round(x))
}
