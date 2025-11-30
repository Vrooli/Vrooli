package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"scenario-completeness-scoring/pkg/analysis"
	apierrors "scenario-completeness-scoring/pkg/errors"
	"scenario-completeness-scoring/pkg/history"

	"github.com/gorilla/mux"
)

// maxLimitDefault is the default maximum for limit parameters
// ASSUMPTION: Users may request very large limits that could cause memory issues
// HARDENED: Cap at a reasonable maximum to prevent excessive memory usage
const maxLimitDefault = 1000

// HandleGetHistory returns score history for a scenario
// [REQ:SCS-HIST-001] Score history storage
// [REQ:SCS-HIST-004] History API endpoint
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
//
// Query parameters:
//   - limit: Max number of snapshots (default 30, max 1000)
//   - source: Filter by source system (e.g., "ecosystem-manager")
//   - tag: Filter by tag (can be repeated for AND logic, e.g., ?tag=task:abc&tag=iteration:5)
func (ctx *Context) HandleGetHistory(w http.ResponseWriter, r *http.Request) {
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

	// Parse limit from query params with bounds checking
	// ASSUMPTION: Users may request very large limits
	// HARDENED: Cap at maxLimitDefault to prevent excessive memory usage
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = min(parsed, maxLimitDefault)
		}
	}

	// Parse source and tags for filtering
	filter := history.HistoryFilter{
		Limit:  limit,
		Source: r.URL.Query().Get("source"),
		Tags:   parseTags(r),
	}

	historyData, err := ctx.HistoryRepo.GetHistoryWithFilter(scenarioName, filter)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeDatabaseError,
			fmt.Sprintf("Failed to retrieve history for scenario '%s'", scenarioName),
			apierrors.CategoryDatabase,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"The scenario may not have any history yet",
			"Calculate a score first to create a history entry",
			"Check database connectivity",
		), http.StatusInternalServerError)
		return
	}

	// Get counts (filtered vs total)
	filteredCount, _ := ctx.HistoryRepo.CountWithFilter(scenarioName, filter)
	totalCount, _ := ctx.HistoryRepo.Count(scenarioName)

	response := map[string]interface{}{
		"scenario":   scenarioName,
		"snapshots":  historyData,
		"count":      len(historyData),
		"filtered":   filteredCount,
		"total":      totalCount,
		"limit":      limit,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	}

	// Include filter info in response if filters were applied
	if filter.Source != "" || len(filter.Tags) > 0 {
		filterInfo := map[string]interface{}{}
		if filter.Source != "" {
			filterInfo["source"] = filter.Source
		}
		if len(filter.Tags) > 0 {
			filterInfo["tags"] = filter.Tags
		}
		response["filter"] = filterInfo
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// parseTags extracts tag values from query string (supports multiple ?tag=X&tag=Y)
func parseTags(r *http.Request) []string {
	// First try the standard multi-value approach
	tags := r.URL.Query()["tag"]

	// Also support comma-separated tags in a single parameter
	if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	return tags
}

// HandleGetTrends returns trend analysis for a scenario
// [REQ:SCS-HIST-003] Trend detection
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
//
// Query parameters:
//   - limit: Max number of snapshots to analyze (default 30, max 1000)
//   - source: Filter by source system (e.g., "ecosystem-manager")
//   - tag: Filter by tag (can be repeated for AND logic)
func (ctx *Context) HandleGetTrends(w http.ResponseWriter, r *http.Request) {
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

	// Parse limit from query params with bounds checking
	// ASSUMPTION: Users may request very large limits
	// HARDENED: Cap at maxLimitDefault to prevent excessive memory usage
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = min(parsed, maxLimitDefault)
		}
	}

	// Parse source and tags for filtering
	filter := history.HistoryFilter{
		Limit:  limit,
		Source: r.URL.Query().Get("source"),
		Tags:   parseTags(r),
	}

	trendAnalysis, err := ctx.TrendAnalyzer.AnalyzeWithFilter(scenarioName, filter)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeDatabaseError,
			fmt.Sprintf("Failed to analyze trends for scenario '%s'", scenarioName),
			apierrors.CategoryDatabase,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Trend analysis requires at least 2 history snapshots",
			"Calculate scores multiple times to generate trend data",
			"Check database connectivity",
		), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"scenario":    scenarioName,
		"analysis":    trendAnalysis,
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	}

	// Include filter info in response if filters were applied
	if filter.Source != "" || len(filter.Tags) > 0 {
		filterInfo := map[string]interface{}{}
		if filter.Source != "" {
			filterInfo["source"] = filter.Source
		}
		if len(filter.Tags) > 0 {
			filterInfo["tags"] = filter.Tags
		}
		response["filter"] = filterInfo
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetAllTrends returns trend summary across all scenarios
// [REQ:SCS-HIST-003] Trend detection
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func (ctx *Context) HandleGetAllTrends(w http.ResponseWriter, r *http.Request) {
	// Parse limit from query params with bounds checking
	// ASSUMPTION: Users may request very large limits
	// HARDENED: Cap at maxLimitDefault to prevent excessive memory usage
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = min(parsed, maxLimitDefault)
		}
	}

	summary, err := ctx.TrendAnalyzer.GetTrendSummary(limit)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeDatabaseError,
			"Failed to retrieve trend summary",
			apierrors.CategoryDatabase,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Check database connectivity",
			"Ensure history snapshots exist",
		), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"summary":     summary,
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleWhatIf performs what-if analysis for a scenario
// [REQ:SCS-ANALYSIS-001] What-if analysis API endpoint
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func (ctx *Context) HandleWhatIf(w http.ResponseWriter, r *http.Request) {
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Failed to read request body",
			apierrors.CategoryValidation,
		).WithDetails(err.Error()), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req analysis.WhatIfRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid JSON in request body",
			apierrors.CategoryValidation,
		).WithDetails(err.Error()).WithNextSteps(
			"Verify the JSON syntax is correct",
			"Example: {\"changes\": [{\"component\": \"quality.test_pass_rate\", \"new_value\": 0.95}]}",
		), http.StatusBadRequest)
		return
	}

	if len(req.Changes) == 0 {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"No changes specified for what-if analysis",
			apierrors.CategoryValidation,
		).WithNextSteps(
			"Provide at least one change in the 'changes' array",
			"Use GET /api/v1/analysis/components to see available components",
			"Example: {\"changes\": [{\"component\": \"quality.test_pass_rate\", \"new_value\": 1.0}]}",
		), http.StatusBadRequest)
		return
	}

	result, err := ctx.WhatIfAnalyzer.Analyze(scenarioName, req.Changes)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			fmt.Sprintf("What-if analysis failed for scenario '%s'", scenarioName),
			apierrors.CategoryInternal,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Verify the scenario exists",
			"Check that component paths are valid",
			"Try with a simpler change first",
		), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"scenario":    scenarioName,
		"result":      result,
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleBulkRefresh recalculates scores for all scenarios
// [REQ:SCS-ANALYSIS-003] Bulk score refresh API endpoint
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func (ctx *Context) HandleBulkRefresh(w http.ResponseWriter, r *http.Request) {
	result, err := ctx.BulkRefresher.RefreshAll()
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			"Bulk refresh failed",
			apierrors.CategoryInternal,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Some scenarios may have still refreshed successfully",
			"Check individual scenario scores",
			"Try refreshing scenarios individually",
		), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"result": result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleCompare compares multiple scenarios
// [REQ:SCS-ANALYSIS-004] Cross-scenario comparison API endpoint
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func (ctx *Context) HandleCompare(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Failed to read request body",
			apierrors.CategoryValidation,
		).WithDetails(err.Error()), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Scenarios []string `json:"scenarios"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid JSON in request body",
			apierrors.CategoryValidation,
		).WithDetails(err.Error()).WithNextSteps(
			"Verify the JSON syntax is correct",
			"Example: {\"scenarios\": [\"scenario-a\", \"scenario-b\"]}",
		), http.StatusBadRequest)
		return
	}

	if len(req.Scenarios) < 2 {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"At least 2 scenarios required for comparison",
			apierrors.CategoryValidation,
		).WithNextSteps(
			"Provide at least 2 scenario names in the 'scenarios' array",
			"Example: {\"scenarios\": [\"scenario-a\", \"scenario-b\", \"scenario-c\"]}",
		), http.StatusBadRequest)
		return
	}

	// Validate all scenario names before comparison
	// ASSUMPTION: Scenario names in arrays are also user-controlled
	// HARDENED: Validate each name to prevent path traversal via array elements
	for _, name := range req.Scenarios {
		if errMsg := ValidateScenarioName(name); errMsg != "" {
			writeAPIError(w, apierrors.NewAPIError(
				apierrors.ErrCodeValidationFailed,
				fmt.Sprintf("Invalid scenario name in comparison list: '%s'", name),
				apierrors.CategoryValidation,
			).WithDetails(errMsg).WithNextSteps(
				"Scenario names must start with a letter or number",
				"Use only letters, numbers, hyphens, and underscores",
				"Maximum length is 64 characters",
			), http.StatusBadRequest)
			return
		}
	}

	result, err := ctx.BulkRefresher.Compare(req.Scenarios)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			"Scenario comparison failed",
			apierrors.CategoryInternal,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Verify all scenario names exist",
			"Check that scenarios have been scored at least once",
			"Try comparing fewer scenarios",
		), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"comparison": result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleListAnalysisComponents returns the list of components available for what-if analysis
func (ctx *Context) HandleListAnalysisComponents(w http.ResponseWriter, r *http.Request) {
	components := analysis.AvailableComponents()

	response := map[string]interface{}{
		"components": components,
		"total":      len(components),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
