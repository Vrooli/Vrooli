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
	apierrors "scenario-completeness-scoring/pkg/errors"
	"scenario-completeness-scoring/pkg/history"
	"scenario-completeness-scoring/pkg/scoring"
	"scenario-completeness-scoring/pkg/validators"

	"github.com/gorilla/mux"
)

// CalculateRequest represents the optional request body for score calculation
// Allows callers to specify source and tags for history correlation
type CalculateRequest struct {
	Source string   `json:"source,omitempty"` // Source system (e.g., "ecosystem-manager", "cli")
	Tags   []string `json:"tags,omitempty"`   // Tags for filtering (e.g., ["task:abc123", "iteration:5"])
}

// HandleListScores returns a list of all scenarios with their scores
// [REQ:SCS-CORE-002] Score retrieval API endpoint
// [REQ:SCS-CORE-003] Graceful degradation - continues on individual scenario failures
// [REQ:SCS-CORE-004] Reports partial results and skipped scenarios
func (ctx *Context) HandleListScores(w http.ResponseWriter, r *http.Request) {
	scenariosDir := filepath.Join(ctx.VrooliRoot, "scenarios")

	// Load config for scoring options, defaulting on error
	// ASSUMPTION: Config loading may fail (disk issues, corruption)
	// HARDENED: Fall back to defaults instead of nil
	cfg, err := ctx.ConfigLoader.LoadGlobal()
	if err != nil {
		log.Printf("config_load_warning | error=%v | using defaults", err)
	}
	scoringOpts := configToScoringOptions(cfg) // configToScoringOptions handles nil safely

	// Discover scenarios by listing directories
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			"Failed to read scenarios directory",
			apierrors.CategoryFileSystem,
		).WithDetails(err.Error()), http.StatusInternalServerError)
		return
	}

	var scenarios []map[string]interface{}
	var skippedScenarios []map[string]interface{} // Track failed scenarios for transparency
	var partialCount int                          // Count scenarios with partial data

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		scenarioName := entry.Name()

		// Try to collect metrics with partial result info
		result, err := ctx.Collector.CollectWithPartialResults(scenarioName)
		if err != nil {
			// Log and track skipped scenarios for transparency
			log.Printf("collection_failed | scenario=%s error=%v", scenarioName, err)
			skippedScenarios = append(skippedScenarios, map[string]interface{}{
				"scenario": scenarioName,
				"error":    err.Error(),
				"reason":   "collection_failed",
			})
			continue
		}

		metrics := result.Metrics
		thresholds := scoring.GetThresholds(metrics.Category)
		breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, 0, scoringOpts)

		scenarioData := map[string]interface{}{
			"scenario":       scenarioName,
			"category":       metrics.Category,
			"score":          breakdown.Score,
			"classification": breakdown.Classification,
		}

		// Include partial result info if not complete
		if !result.PartialResult.IsComplete {
			scenarioData["partial"] = true
			scenarioData["confidence"] = result.PartialResult.Confidence
			scenarioData["missing_collectors"] = result.PartialResult.Missing
			partialCount++
		}

		scenarios = append(scenarios, scenarioData)
	}

	response := map[string]interface{}{
		"scenarios":     scenarios,
		"total":         len(scenarios),
		"calculated_at": time.Now().UTC().Format(time.RFC3339),
	}

	// Include degradation info if there are issues
	if len(skippedScenarios) > 0 || partialCount > 0 {
		response["degradation"] = map[string]interface{}{
			"skipped":         skippedScenarios,
			"skipped_count":   len(skippedScenarios),
			"partial_count":   partialCount,
			"is_complete":     len(skippedScenarios) == 0 && partialCount == 0,
			"message":         getDegradationMessage(len(skippedScenarios), partialCount),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getDegradationMessage generates a user-friendly message about data completeness
func getDegradationMessage(skipped, partial int) string {
	if skipped == 0 && partial == 0 {
		return ""
	}
	parts := []string{}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d scenario(s) skipped due to errors", skipped))
	}
	if partial > 0 {
		parts = append(parts, fmt.Sprintf("%d scenario(s) have partial data", partial))
	}
	return strings.Join(parts, "; ")
}

// writeAPIError writes a structured API error response
// [REQ:SCS-CORE-003] User-friendly error responses with actionable guidance
func writeAPIError(w http.ResponseWriter, apiErr *apierrors.APIError, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": apiErr,
	})
}

// HandleGetScore returns the detailed score for a specific scenario
// [REQ:SCS-CORE-002] Score retrieval API endpoint
// [REQ:SCS-CORE-003] Graceful degradation - returns partial results when possible
// [REQ:SCS-CORE-004] Includes data completeness info
func (ctx *Context) HandleGetScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Validate scenario name
	// ASSUMPTION: Scenario names are user-controlled input
	// HARDENED: Explicit validation prevents path traversal and injection
	if errMsg := ValidateScenarioName(scenarioName); errMsg != "" {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid scenario name",
			apierrors.CategoryValidation,
		).WithDetails(errMsg).WithNextSteps(
			"Scenario names must start with a letter or number",
			"Use only letters, numbers, hyphens, and underscores",
			"Maximum length is 64 characters",
		), http.StatusBadRequest)
		return
	}

	// Collect real metrics from the scenario with partial result info
	result, err := ctx.Collector.CollectWithPartialResults(scenarioName)
	if err != nil {
		// Check for specific error types to provide better messages
		if scoringErr, ok := err.(*apierrors.ScoringError); ok {
			writeAPIError(w, apierrors.NewAPIError(
				apierrors.ErrCodeScenarioNotFound,
				fmt.Sprintf("Failed to collect metrics for scenario '%s'", scenarioName),
				scoringErr.Category,
			).WithDetails(scoringErr.Message).WithNextSteps(
				"Check that the scenario name is correct",
				"Verify the scenario directory exists",
				"Run 'vrooli scenario list' to see available scenarios",
			), http.StatusNotFound)
			return
		}
		writeAPIError(w, apierrors.ErrScenarioNotFound.WithDetails(err.Error()), http.StatusNotFound)
		return
	}

	metrics := result.Metrics

	// Load config for scoring options, defaulting on error
	// ASSUMPTION: Config loading may fail (disk issues, corruption)
	// HARDENED: Fall back to defaults instead of nil
	cfg, err := ctx.ConfigLoader.LoadGlobal()
	if err != nil {
		log.Printf("config_load_warning | scenario=%s error=%v | using defaults", scenarioName, err)
	}
	scoringOpts := configToScoringOptions(cfg) // configToScoringOptions handles nil safely

	// Perform validation quality analysis
	scenarioRoot := ctx.Collector.GetScenarioRoot(scenarioName)
	requirements := ctx.Collector.LoadRequirements(scenarioName)

	validationAnalysis := validators.AnalyzeValidationQuality(
		validators.ValidationInputCounts{
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

	// Include partial result info if data is incomplete
	// [REQ:SCS-CORE-004] Report partial results to UI
	if !result.PartialResult.IsComplete {
		response["partial_result"] = map[string]interface{}{
			"is_complete":       false,
			"confidence":        result.PartialResult.Confidence,
			"available":         result.PartialResult.Available,
			"missing":           result.PartialResult.Missing,
			"collector_errors":  result.PartialResult.CollectorErrors,
			"message":           result.PartialResult.GetMessage(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleCalculateScore forces recalculation of score for a scenario
// [REQ:SCS-CORE-002] Score retrieval API endpoint
// [REQ:SCS-HIST-001] Save score snapshot to history
// Accepts optional JSON body with source and tags for history correlation:
//
//	{
//	  "source": "ecosystem-manager",
//	  "tags": ["task:abc123", "iteration:5"]
//	}
func (ctx *Context) HandleCalculateScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Validate scenario name
	// ASSUMPTION: Scenario names are user-controlled input
	// HARDENED: Explicit validation prevents path traversal and injection
	if errMsg := ValidateScenarioName(scenarioName); errMsg != "" {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid scenario name",
			apierrors.CategoryValidation,
		).WithDetails(errMsg).WithNextSteps(
			"Scenario names must start with a letter or number",
			"Use only letters, numbers, hyphens, and underscores",
			"Maximum length is 64 characters",
		), http.StatusBadRequest)
		return
	}

	// Parse optional request body for source/tags
	// Empty body is valid - source and tags are optional
	var calcReq CalculateRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&calcReq); err != nil {
			writeAPIError(w, apierrors.NewAPIError(
				apierrors.ErrCodeValidationFailed,
				"Invalid request body",
				apierrors.CategoryValidation,
			).WithDetails(err.Error()).WithNextSteps(
				"Request body must be valid JSON",
				"Example: {\"source\": \"ecosystem-manager\", \"tags\": [\"task:abc123\"]}",
			), http.StatusBadRequest)
			return
		}
	}

	// Force recalculation by collecting fresh metrics
	metrics, err := ctx.Collector.Collect(scenarioName)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeScenarioNotFound,
			fmt.Sprintf("Failed to calculate score for scenario '%s'", scenarioName),
			apierrors.CategoryCollector,
		).WithDetails(err.Error()).WithNextSteps(
			"Verify the scenario exists",
			"Run 'vrooli scenario list' to see available scenarios",
		), http.StatusNotFound)
		return
	}

	// Load config for scoring options, defaulting on error
	// ASSUMPTION: Config loading may fail (disk issues, corruption)
	// HARDENED: Fall back to defaults instead of nil
	cfg, cfgErr := ctx.ConfigLoader.LoadGlobal()
	if cfgErr != nil {
		log.Printf("config_load_warning | scenario=%s error=%v | using defaults", scenarioName, cfgErr)
	}
	scoringOpts := configToScoringOptions(cfg) // configToScoringOptions handles nil safely

	thresholds := scoring.GetThresholds(metrics.Category)
	breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, 0, scoringOpts)
	recommendations := scoring.GenerateRecommendations(breakdown, thresholds)

	// Save snapshot to history with source/tags for correlation [REQ:SCS-HIST-001]
	var snapshotID int64
	var snapshotSource string
	var snapshotTags []string
	if ctx.HistoryRepo != nil {
		saveOpts := history.SaveOptions{
			Source: calcReq.Source,
			Tags:   calcReq.Tags,
		}
		snapshot, err := ctx.HistoryRepo.SaveWithOptions(scenarioName, &breakdown, saveOpts)
		if err != nil {
			log.Printf("Warning: failed to save history snapshot: %v", err)
		} else if snapshot != nil {
			snapshotID = snapshot.ID
			snapshotSource = snapshot.Source
			snapshotTags = snapshot.Tags
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

	// Include source/tags in response if provided
	if snapshotSource != "" {
		response["source"] = snapshotSource
	}
	if len(snapshotTags) > 0 {
		response["tags"] = snapshotTags
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleValidationAnalysis returns validation quality analysis for a scenario
// This detects anti-patterns and gaming behaviors in test validation
func (ctx *Context) HandleValidationAnalysis(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Validate scenario name
	// ASSUMPTION: Scenario names are user-controlled input
	// HARDENED: Explicit validation prevents path traversal and injection
	if errMsg := ValidateScenarioName(scenarioName); errMsg != "" {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid scenario name",
			apierrors.CategoryValidation,
		).WithDetails(errMsg).WithNextSteps(
			"Scenario names must start with a letter or number",
			"Use only letters, numbers, hyphens, and underscores",
			"Maximum length is 64 characters",
		), http.StatusBadRequest)
		return
	}

	// Collect metrics to get counts
	metrics, err := ctx.Collector.Collect(scenarioName)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeScenarioNotFound,
			fmt.Sprintf("Failed to collect metrics for scenario '%s'", scenarioName),
			apierrors.CategoryCollector,
		).WithDetails(err.Error()).WithNextSteps(
			"Verify the scenario exists",
			"Run 'vrooli scenario list' to see available scenarios",
		), http.StatusNotFound)
		return
	}

	// Perform validation quality analysis
	scenarioRoot := ctx.Collector.GetScenarioRoot(scenarioName)
	requirements := ctx.Collector.LoadRequirements(scenarioName)

	analysis := validators.AnalyzeValidationQuality(
		validators.ValidationInputCounts{
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

	// Validate scenario name
	// ASSUMPTION: Scenario names are user-controlled input
	// HARDENED: Explicit validation prevents path traversal and injection
	if errMsg := ValidateScenarioName(scenarioName); errMsg != "" {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid scenario name",
			apierrors.CategoryValidation,
		).WithDetails(errMsg).WithNextSteps(
			"Scenario names must start with a letter or number",
			"Use only letters, numbers, hyphens, and underscores",
			"Maximum length is 64 characters",
		), http.StatusBadRequest)
		return
	}

	// Collect metrics and calculate score
	metrics, err := ctx.Collector.Collect(scenarioName)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeScenarioNotFound,
			fmt.Sprintf("Failed to collect metrics for scenario '%s'", scenarioName),
			apierrors.CategoryCollector,
		).WithDetails(err.Error()).WithNextSteps(
			"Verify the scenario exists",
			"Run 'vrooli scenario list' to see available scenarios",
		), http.StatusNotFound)
		return
	}

	// Load config for scoring options, defaulting on error
	// ASSUMPTION: Config loading may fail (disk issues, corruption)
	// HARDENED: Fall back to defaults instead of nil
	cfg, cfgErr := ctx.ConfigLoader.LoadGlobal()
	if cfgErr != nil {
		log.Printf("config_load_warning | scenario=%s error=%v | using defaults", scenarioName, cfgErr)
	}
	scoringOpts := configToScoringOptions(cfg) // configToScoringOptions handles nil safely

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
