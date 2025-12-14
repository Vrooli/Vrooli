package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// LightScanRequest defines the request body for light scanning
type LightScanRequest struct {
	ScenarioPath string `json:"scenario_path"`
	TimeoutSec   int    `json:"timeout_sec,omitempty"`
	Timeout      string `json:"timeout,omitempty"`     // backwards compatibility with tests sending string
	Incremental  bool   `json:"incremental,omitempty"` // Only scan modified files
}

// ParseRequest defines request for parsing lint/type output
type ParseRequest struct {
	Scenario string `json:"scenario"`
	Tool     string `json:"tool"`
	Output   string `json:"output"`
}

// handleLightScan performs a complete light scan on a scenario
func (s *Server) handleLightScan(w http.ResponseWriter, r *http.Request) {
	var req LightScanRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.ScenarioPath) == "" {
		respondError(w, http.StatusBadRequest, "scenario_path is required")
		return
	}

	// Support legacy "timeout" field encoded as string
	if req.TimeoutSec == 0 && strings.TrimSpace(req.Timeout) != "" {
		if parsed, err := strconv.Atoi(req.Timeout); err == nil && parsed > 0 {
			req.TimeoutSec = parsed
		} else {
			respondError(w, http.StatusBadRequest, "timeout must be a positive integer")
			return
		}
	}

	if err := s.ensureScanCoordinator(); err != nil {
		respondError(w, http.StatusInternalServerError, "scan coordinator not initialized")
		return
	}

	result, scanErr := s.scanCoordinator.LightScan(r.Context(), req)
	if scanErr != nil {
		respondError(w, scanErr.Status, scanErr.Message)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// handleParseLint parses lint output into structured issues
func (s *Server) handleParseLint(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" || req.Tool == "" {
		respondError(w, http.StatusBadRequest, "scenario and tool are required")
		return
	}

	issues := ParseLintOutput(req.Scenario, req.Tool, req.Output)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

// handleParseType parses type checker output into structured issues
func (s *Server) handleParseType(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" || req.Tool == "" {
		respondError(w, http.StatusBadRequest, "scenario and tool are required")
		return
	}

	issues := ParseTypeOutput(req.Scenario, req.Tool, req.Output)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

// handleRefactorRecommendations returns prioritized refactor candidates, optionally
// triggering a light scan to seed metrics when none exist.
func (s *Server) handleRefactorRecommendations(w http.ResponseWriter, r *http.Request) {
	scenario := r.URL.Query().Get("scenario")
	if scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario parameter is required")
		return
	}

	if err := s.ensureScanCoordinator(); err != nil {
		respondError(w, http.StatusInternalServerError, "scan coordinator not initialized")
		return
	}

	scenarioName, err := s.scanCoordinator.NormalizeScenarioName(scenario)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	limit := parseIntParam(r, "limit", 10)
	sortBy := parseStringParam(r, "sort_by", "priority")
	minLines := parseIntParam(r, "min_lines", 0)
	maxVisits := parseIntParam(r, "max_visits", 0)
	autoScan := parseBoolParam(r, "auto_scan")

	if autoScan {
		if err := s.scanCoordinator.EnsureFileMetrics(r.Context(), scenarioName); err != nil {
			s.log("auto-scan failed", map[string]interface{}{
				"error":    err.Error(),
				"scenario": scenarioName,
			})
		}
	}

	recommender := NewRefactorRecommender(s.db, s.campaignMgr)
	recommendations, err := recommender.GetRecommendations(
		r.Context(),
		scenarioName,
		limit,
		sortBy,
		minLines,
		maxVisits,
	)
	if err != nil {
		s.log("failed to get refactor recommendations", map[string]interface{}{
			"error":    err.Error(),
			"scenario": scenarioName,
		})
		respondError(w, http.StatusInternalServerError, "failed to get recommendations")
		return
	}

	response := map[string]interface{}{
		"scenario":        scenarioName,
		"recommendations": recommendations,
		"count":           len(recommendations),
	}

	// Add warning if no data exists for scenario
	if len(recommendations) == 0 {
		hasMetrics, _ := s.scanCoordinator.HasMetricsForScenario(r.Context(), scenarioName)
		if !hasMetrics {
			response["warning"] = "no file metrics found for scenario - run 'tidiness-manager scan <scenario-path>' first"
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}

// decodeAndValidateJSON decodes JSON request body and validates required fields
// Security: Limits request body size to prevent DoS attacks
func decodeAndValidateJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	// Limit request body to 10MB to prevent memory exhaustion
	const maxBodySize = 10 * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	decoder := json.NewDecoder(r.Body)
	// Security: DisallowUnknownFields prevents injection of unexpected fields
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		// Don't leak parsing details to client
		respondError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// parseIntParam extracts and parses an integer query parameter with a default value
func parseIntParam(r *http.Request, key string, defaultValue int) int {
	if val := r.URL.Query().Get(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}

// parseStringParam extracts a string query parameter with a default value
func parseStringParam(r *http.Request, key, defaultValue string) string {
	if val := r.URL.Query().Get(key); val != "" {
		return val
	}
	return defaultValue
}

// parseBoolParam extracts and parses a boolean query parameter
func parseBoolParam(r *http.Request, key string) bool {
	return r.URL.Query().Get(key) == "true"
}

// handleSmartScan performs AI-powered smart scanning (TM-SS-001, TM-SS-002)
func (s *Server) handleSmartScan(w http.ResponseWriter, r *http.Request) {
	var req SmartScanRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	if len(req.Files) == 0 {
		respondError(w, http.StatusBadRequest, "files list cannot be empty")
		return
	}

	if err := s.ensureScanCoordinator(); err != nil {
		respondError(w, http.StatusInternalServerError, "scan coordinator not initialized")
		return
	}

	result, scanErr := s.scanCoordinator.SmartScan(r.Context(), req)
	if scanErr != nil {
		respondError(w, scanErr.Status, scanErr.Message)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// storeAIIssue stores an AI-discovered issue in the database (TM-SS-002, TM-API-006)
func (s *Server) storeAIIssue(ctx context.Context, scenario string, issue AIIssue, sessionID string, campaignID *int) error {
	if s.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return s.store.StoreAIIssue(ctx, scenario, issue, sessionID, campaignID)
}

// recordScanHistory records a scan in the audit trail
func (s *Server) recordScanHistory(ctx context.Context, scenario, scanType string, result *SmartScanResult, campaignID *int) error {
	if s.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return s.store.RecordScanHistory(ctx, scenario, scanType, result, campaignID)
}

// persistFileMetrics stores file metrics from a light scan in the database (TM-FM-001, TM-FM-002)
func (s *Server) persistFileMetrics(ctx context.Context, scenario string, metrics []FileMetric) error {
	if len(metrics) == 0 {
		return nil
	}
	if s.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return s.store.PersistFileMetrics(ctx, scenario, metrics)
}

// GenerateIssuesFromMetricsRequest defines request for generating issues from stored metrics
type GenerateIssuesFromMetricsRequest struct {
	Scenario string `json:"scenario"`
}

// handleGenerateIssuesFromMetrics generates issues from existing file metrics in the database
// This is useful when metrics exist but issues weren't generated (e.g., after incremental scans)
func (s *Server) handleGenerateIssuesFromMetrics(w http.ResponseWriter, r *http.Request) {
	var req GenerateIssuesFromMetricsRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	if s.store == nil {
		respondError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	// Get existing file metrics from database
	metrics, err := s.store.GetDetailedFileMetrics(r.Context(), req.Scenario)
	if err != nil {
		s.log("failed to get file metrics", map[string]interface{}{
			"error":    err.Error(),
			"scenario": req.Scenario,
		})
		respondError(w, http.StatusInternalServerError, "failed to get file metrics")
		return
	}

	if len(metrics) == 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"scenario":  req.Scenario,
			"generated": 0,
			"inserted":  0,
			"message":   "no file metrics found for scenario - run a scan first",
		})
		return
	}

	// Generate issues from metrics
	config := DefaultIssueGeneratorConfig()
	issues := GenerateIssuesFromMetrics(req.Scenario, metrics, config)

	if len(issues) == 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"scenario":      req.Scenario,
			"metrics_count": len(metrics),
			"generated":     0,
			"inserted":      0,
			"message":       "no issues exceeded thresholds",
		})
		return
	}

	// Persist the generated issues
	inserted, persistErr := s.store.StoreLintTypeIssues(r.Context(), req.Scenario, issues)
	if persistErr != nil {
		s.log("failed to persist metric-based issues", map[string]interface{}{
			"error":    persistErr.Error(),
			"scenario": req.Scenario,
			"count":    len(issues),
		})
		respondError(w, http.StatusInternalServerError, "failed to persist issues")
		return
	}

	s.log("generated issues from metrics", map[string]interface{}{
		"scenario":      req.Scenario,
		"metrics_count": len(metrics),
		"generated":     len(issues),
		"inserted":      inserted,
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"scenario":      req.Scenario,
		"metrics_count": len(metrics),
		"generated":     len(issues),
		"inserted":      inserted,
	})
}
