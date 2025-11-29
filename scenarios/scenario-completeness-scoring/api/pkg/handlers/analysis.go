package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"scenario-completeness-scoring/pkg/analysis"

	"github.com/gorilla/mux"
)

// HandleGetHistory returns score history for a scenario
// [REQ:SCS-HIST-001] Score history storage
// [REQ:SCS-HIST-004] History API endpoint
func (ctx *Context) HandleGetHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Parse limit from query params
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := ctx.HistoryRepo.GetHistory(scenarioName, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get history: %v", err), http.StatusInternalServerError)
		return
	}

	count, _ := ctx.HistoryRepo.Count(scenarioName)

	response := map[string]interface{}{
		"scenario":   scenarioName,
		"snapshots":  history,
		"count":      len(history),
		"total":      count,
		"limit":      limit,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetTrends returns trend analysis for a scenario
// [REQ:SCS-HIST-003] Trend detection
func (ctx *Context) HandleGetTrends(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Parse limit from query params
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	analysis, err := ctx.TrendAnalyzer.Analyze(scenarioName, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze trends: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"scenario":    scenarioName,
		"analysis":    analysis,
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetAllTrends returns trend summary across all scenarios
// [REQ:SCS-HIST-003] Trend detection
func (ctx *Context) HandleGetAllTrends(w http.ResponseWriter, r *http.Request) {
	// Parse limit from query params
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	summary, err := ctx.TrendAnalyzer.GetTrendSummary(limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get trend summary: %v", err), http.StatusInternalServerError)
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
func (ctx *Context) HandleWhatIf(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req analysis.WhatIfRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Changes) == 0 {
		http.Error(w, "No changes specified", http.StatusBadRequest)
		return
	}

	result, err := ctx.WhatIfAnalyzer.Analyze(scenarioName, req.Changes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
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
func (ctx *Context) HandleBulkRefresh(w http.ResponseWriter, r *http.Request) {
	result, err := ctx.BulkRefresher.RefreshAll()
	if err != nil {
		http.Error(w, fmt.Sprintf("Bulk refresh failed: %v", err), http.StatusInternalServerError)
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
func (ctx *Context) HandleCompare(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Scenarios []string `json:"scenarios"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Scenarios) < 2 {
		http.Error(w, "At least 2 scenarios required for comparison", http.StatusBadRequest)
		return
	}

	result, err := ctx.BulkRefresher.Compare(req.Scenarios)
	if err != nil {
		http.Error(w, fmt.Sprintf("Comparison failed: %v", err), http.StatusInternalServerError)
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
