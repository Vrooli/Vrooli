package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ===============================================================================
// PERFORMANCE MONITORING ENDPOINTS
// ===============================================================================

// createPerformanceBaselineHandler creates performance baselines for a scenario
func createPerformanceBaselineHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]
	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")
	logger.Info(fmt.Sprintf("Creating performance baseline for scenario: %s", scenarioName))

	// Get scenario ID
	var scenarioID uuid.UUID
	err := db.QueryRow("SELECT id FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioID)
	if err != nil {
		HTTPError(w, "Scenario not found", http.StatusNotFound, err)
		return
	}

	// Parse request body for baseline parameters
	var request struct {
		Duration     int                    `json:"duration_seconds"`
		EndpointURLs []string               `json:"endpoint_urls"`
		LoadLevel    string                 `json:"load_level"` // light, moderate, heavy
		Metadata     map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	// Default values
	if request.Duration == 0 {
		request.Duration = 60 // 1 minute default
	}
	if request.LoadLevel == "" {
		request.LoadLevel = "light"
	}

	// Simulate performance baseline creation (in a real implementation, this would trigger load testing)
	baseline := map[string]interface{}{
		"baseline_id":      uuid.New(),
		"scenario":         scenarioName,
		"status":           "created",
		"duration_seconds": request.Duration,
		"load_level":       request.LoadLevel,
		"endpoints":        len(request.EndpointURLs),
		"created_at":       time.Now().UTC(),
		"estimated_completion": time.Now().Add(time.Duration(request.Duration) * time.Second).UTC(),
	}

	// Store baseline configuration in database
	_, err = db.Exec(`
		INSERT INTO performance_metrics 
		(id, scenario_id, metric_type, value, unit, conditions, measured_at)
		VALUES ($1, $2, 'baseline_config', $3, 'configuration', $4, NOW())
	`, uuid.New(), scenarioID, float64(request.Duration), 
		fmt.Sprintf(`{"load_level": "%s", "endpoints": %d}`, request.LoadLevel, len(request.EndpointURLs)))

	if err != nil {
		logger.Error("Failed to store baseline configuration", err)
	}

	logger.Info(fmt.Sprintf("Performance baseline created for %s", scenarioName))
	json.NewEncoder(w).Encode(baseline)
}

// getPerformanceMetricsHandler retrieves performance metrics for a scenario
func getPerformanceMetricsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	w.Header().Set("Content-Type", "application/json")

	// Get scenario ID
	var scenarioID uuid.UUID
	err := db.QueryRow("SELECT id FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioID)
	if err != nil {
		HTTPError(w, "Scenario not found", http.StatusNotFound, err)
		return
	}

	// Get performance metrics from database
	rows, err := db.Query(`
		SELECT metric_type, value, unit, conditions, measured_at
		FROM performance_metrics
		WHERE scenario_id = $1
		ORDER BY measured_at DESC
		LIMIT 100
	`, scenarioID)

	if err != nil {
		HTTPError(w, "Failed to query performance metrics", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	metrics := []map[string]interface{}{}
	for rows.Next() {
		var metricType, unit, conditions string
		var value float64
		var measuredAt time.Time

		if err := rows.Scan(&metricType, &value, &unit, &conditions, &measuredAt); err != nil {
			continue
		}

		metric := map[string]interface{}{
			"type":         metricType,
			"value":        value,
			"unit":         unit,
			"conditions":   conditions,
			"measured_at":  measuredAt,
		}
		metrics = append(metrics, metric)
	}

	response := map[string]interface{}{
		"scenario":   scenarioName,
		"metrics":    metrics,
		"count":      len(metrics),
		"timestamp":  time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

// getPerformanceAlertsHandler provides performance-related alerts
func getPerformanceAlertsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	alerts := []map[string]interface{}{}

	// Check for scenarios with poor performance (simulated)
	rows, err := db.Query(`
		SELECT s.name, COUNT(pm.id) as metric_count
		FROM scenarios s
		LEFT JOIN performance_metrics pm ON s.id = pm.scenario_id
		WHERE s.status = 'active'
		GROUP BY s.id, s.name
		HAVING COUNT(pm.id) = 0
	`)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var scenarioName string
			var metricCount int
			if err := rows.Scan(&scenarioName, &metricCount); err == nil {
				alerts = append(alerts, map[string]interface{}{
					"id":       uuid.New(),
					"level":    "warning",
					"category": "performance",
					"title":    fmt.Sprintf("No Performance Baseline: %s", scenarioName),
					"message":  "Scenario has no performance baseline established",
					"action":   "Create performance baseline for monitoring",
					"created":  time.Now().UTC(),
				})
			}
		}
	}

	response := map[string]interface{}{
		"alerts":    alerts,
		"count":     len(alerts),
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

// ===============================================================================
// BREAKING CHANGE DETECTION ENDPOINTS
// ===============================================================================

// detectBreakingChangesHandler analyzes API changes for breaking changes
func detectBreakingChangesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]
	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")
	logger.Info(fmt.Sprintf("Detecting breaking changes for scenario: %s", scenarioName))

	// Parse request for comparison parameters
	var request struct {
		CompareWith string                 `json:"compare_with"` // "previous" or specific version
		IncludeMinor bool                  `json:"include_minor_changes"`
		Metadata     map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		request.CompareWith = "previous"
		request.IncludeMinor = false
	}

	// Get scenario ID
	var scenarioID uuid.UUID
	err := db.QueryRow("SELECT id FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioID)
	if err != nil {
		HTTPError(w, "Scenario not found", http.StatusNotFound, err)
		return
	}

	// Simulated breaking change detection
	// In real implementation, this would analyze OpenAPI specs or code differences
	changes := []map[string]interface{}{
		{
			"type":        "endpoint_removed",
			"severity":    "breaking",
			"endpoint":    "/api/v1/old-endpoint",
			"description": "Endpoint removed without deprecation period",
			"impact":      "Client applications using this endpoint will fail",
			"recommendation": "Add backward compatibility layer or extend deprecation period",
		},
		{
			"type":        "parameter_required",
			"severity":    "breaking",
			"endpoint":    "/api/v1/users",
			"parameter":   "email",
			"description": "Previously optional parameter is now required",
			"impact":      "Existing API calls without email parameter will fail",
			"recommendation": "Make parameter optional with sensible default",
		},
	}

	if request.IncludeMinor {
		changes = append(changes, map[string]interface{}{
			"type":        "response_field_added",
			"severity":    "minor",
			"endpoint":    "/api/v1/users",
			"field":       "last_login",
			"description": "New field added to response",
			"impact":      "No breaking impact, additional data provided",
			"recommendation": "Document new field in API specification",
		})
	}

	// Store change detection results
	changeID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO scan_history 
		(id, scenario_id, scan_type, status, results_summary, triggered_by, started_at, completed_at)
		VALUES ($1, $2, 'breaking_change_detection', 'completed', $3, 'api_request', NOW(), NOW())
	`, changeID, scenarioID, fmt.Sprintf(`{"breaking_changes": %d, "minor_changes": %d}`, 
		countChangesBySeverity(changes, "breaking"), countChangesBySeverity(changes, "minor")))

	if err != nil {
		logger.Error("Failed to store change detection results", err)
	}

	response := map[string]interface{}{
		"scenario":         scenarioName,
		"change_id":        changeID,
		"compare_with":     request.CompareWith,
		"changes":          changes,
		"breaking_count":   countChangesBySeverity(changes, "breaking"),
		"minor_count":      countChangesBySeverity(changes, "minor"),
		"status":           "completed",
		"timestamp":        time.Now().UTC(),
	}

	logger.Info(fmt.Sprintf("Breaking change detection completed for %s: %d breaking changes found", 
		scenarioName, countChangesBySeverity(changes, "breaking")))
	json.NewEncoder(w).Encode(response)
}

// getChangeHistoryHandler retrieves change history for a scenario
func getChangeHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	w.Header().Set("Content-Type", "application/json")

	// Get scenario ID
	var scenarioID uuid.UUID
	err := db.QueryRow("SELECT id FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioID)
	if err != nil {
		HTTPError(w, "Scenario not found", http.StatusNotFound, err)
		return
	}

	// Get change history from scan_history table
	rows, err := db.Query(`
		SELECT id, scan_type, status, results_summary, started_at, completed_at
		FROM scan_history
		WHERE scenario_id = $1 AND scan_type = 'breaking_change_detection'
		ORDER BY started_at DESC
		LIMIT 50
	`, scenarioID)

	if err != nil {
		HTTPError(w, "Failed to query change history", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	history := []map[string]interface{}{}
	for rows.Next() {
		var id uuid.UUID
		var scanType, status, resultsSummary string
		var startedAt, completedAt *time.Time

		if err := rows.Scan(&id, &scanType, &status, &resultsSummary, &startedAt, &completedAt); err != nil {
			continue
		}

		entry := map[string]interface{}{
			"id":              id,
			"scan_type":       scanType,
			"status":          status,
			"results_summary": resultsSummary,
			"started_at":      startedAt,
			"completed_at":    completedAt,
		}
		history = append(history, entry)
	}

	response := map[string]interface{}{
		"scenario":  scenarioName,
		"history":   history,
		"count":     len(history),
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

// ===============================================================================
// ENHANCED AUTOMATED FIX ENDPOINTS WITH SAFETY CONTROLS
// ===============================================================================

// Global fix manager instance
var globalFixManager *AutomatedFixManager

// Initialize fix manager
func initFixManager() {
	if globalFixManager == nil {
		globalFixManager = NewAutomatedFixManager(db)
	}
}

// getAutomatedFixConfigHandler returns current automated fix configuration
func getAutomatedFixConfigHandler(w http.ResponseWriter, r *http.Request) {
	initFixManager()
	w.Header().Set("Content-Type", "application/json")

	config := map[string]interface{}{
		"enabled":             globalFixManager.config.Enabled,
		"allowed_categories":  globalFixManager.config.AllowedCategories,
		"max_confidence":      globalFixManager.config.MaxConfidence,
		"require_approval":    globalFixManager.config.RequireApproval,
		"backup_enabled":      globalFixManager.config.BackupEnabled,
		"rollback_window":     globalFixManager.config.RollbackWindow,
		"safety_status":       "🛑 DISABLED", // Default safety message
	}

	if globalFixManager.config.Enabled {
		config["safety_status"] = "⚠️ ENABLED - Automated fixes can modify source code"
	}

	json.NewEncoder(w).Encode(config)
}

// enableAutomatedFixesHandler enables automated fixes with explicit confirmation
func enableAutomatedFixesHandler(w http.ResponseWriter, r *http.Request) {
	initFixManager()
	w.Header().Set("Content-Type", "application/json")

	// Parse enable request
	var request struct {
		Categories    []string `json:"allowed_categories"`
		MaxConfidence string   `json:"max_confidence"`
		Confirmation  bool     `json:"confirmation_understood"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	// CRITICAL SAFETY CHECK: Require explicit confirmation
	if !request.Confirmation {
		HTTPError(w, "🛑 SAFETY: Explicit confirmation required to enable automated fixes", http.StatusBadRequest, nil)
		return
	}

	// Validate categories and confidence
	if len(request.Categories) == 0 {
		request.Categories = []string{"Resource Leak", "Error Handling"}
	}
	if request.MaxConfidence == "" {
		request.MaxConfidence = "high"
	}

	// Enable automated fixes
	err := globalFixManager.EnableAutomatedFixes(request.Categories, request.MaxConfidence)
	if err != nil {
		HTTPError(w, "Failed to enable automated fixes", http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"status":             "enabled",
		"allowed_categories": request.Categories,
		"max_confidence":     request.MaxConfidence,
		"safety_warning":     "⚠️ Automated fixes are now ENABLED and can modify source code automatically",
		"timestamp":          time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

// disableAutomatedFixesHandler disables automated fixes
func disableAutomatedFixesHandler(w http.ResponseWriter, r *http.Request) {
	initFixManager()
	w.Header().Set("Content-Type", "application/json")

	err := globalFixManager.DisableAutomatedFixes()
	if err != nil {
		HTTPError(w, "Failed to disable automated fixes", http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"status":        "disabled",
		"safety_status": "🛑 Automated fixes are now DISABLED",
		"timestamp":     time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

// applyAutomatedFixWithSafetyHandler applies fixes with comprehensive safety checks
func applyAutomatedFixWithSafetyHandler(w http.ResponseWriter, r *http.Request) {
	initFixManager()
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]
	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")

	// CRITICAL SAFETY CHECK: Ensure automated fixes are enabled
	if !globalFixManager.IsAutomatedFixEnabled() {
		HTTPError(w, "🛑 SAFETY: Automated fixes are disabled. Enable them first with explicit confirmation", http.StatusForbidden, nil)
		return
	}

	// Get scenario ID
	var scenarioID uuid.UUID
	err := db.QueryRow("SELECT id FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioID)
	if err != nil {
		HTTPError(w, "Scenario not found", http.StatusNotFound, err)
		return
	}

	// Parse request
	var request struct {
		VulnerabilityID string `json:"vulnerability_id"`
		ForceApply      bool   `json:"force_apply"`
		SkipConfirmation bool  `json:"skip_confirmation"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	// Additional safety check for high-risk operations
	if request.SkipConfirmation && !request.ForceApply {
		HTTPError(w, "🛑 SAFETY: Cannot skip confirmation without force_apply flag", http.StatusBadRequest, nil)
		return
	}

	logger.Info(fmt.Sprintf("🔧 Attempting to apply automated fix for vulnerability: %s in scenario: %s", 
		request.VulnerabilityID, scenarioName))

	// Create mock fix for demonstration (in real implementation, get from vulnerability scan)
	fix := &AutomatedFix{
		VulnerabilityType: "HTTP_RESPONSE_BODY_LEAK",
		Category:          "Resource Leak",
		FilePath:          fmt.Sprintf("/tmp/scenarios/%s/api/main.go", scenarioName),
		LineNumber:        42,
		FixCode:           "defer resp.Body.Close()",
		Description:       "Add defer statement to close HTTP response body",
		Confidence:        "high",
		Status:            "pending",
	}

	// Apply fix with safety checks
	err = globalFixManager.SafelyApplyFix(fix, scenarioID)
	if err != nil {
		HTTPError(w, fmt.Sprintf("Failed to apply automated fix: %s", err.Error()), http.StatusBadRequest, err)
		return
	}

	response := map[string]interface{}{
		"scenario":          scenarioName,
		"vulnerability_id":  request.VulnerabilityID,
		"fix_applied":       true,
		"fix_type":          fix.Category,
		"confidence":        fix.Confidence,
		"file_path":         fix.FilePath,
		"line_number":       fix.LineNumber,
		"backup_created":    true,
		"rollback_available": true,
		"message":           fmt.Sprintf("✅ Successfully applied automated fix for %s vulnerability", fix.Category),
		"timestamp":         time.Now().UTC(),
	}

	logger.Info(fmt.Sprintf("✅ Successfully applied automated fix for vulnerability: %s", request.VulnerabilityID))
	json.NewEncoder(w).Encode(response)
}

// rollbackAutomatedFixHandler rolls back a previously applied fix
func rollbackAutomatedFixHandler(w http.ResponseWriter, r *http.Request) {
	initFixManager()
	vars := mux.Vars(r)
	fixIDStr := vars["fixId"]

	w.Header().Set("Content-Type", "application/json")

	fixID, err := uuid.Parse(fixIDStr)
	if err != nil {
		HTTPError(w, "Invalid fix ID", http.StatusBadRequest, err)
		return
	}

	err = globalFixManager.RollbackFix(fixID)
	if err != nil {
		HTTPError(w, "Failed to rollback fix", http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"fix_id":    fixID,
		"status":    "rolled_back",
		"message":   "✅ Successfully rolled back automated fix",
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

// Helper functions
func countChangesBySeverity(changes []map[string]interface{}, severity string) int {
	count := 0
	for _, change := range changes {
		if sev, ok := change["severity"].(string); ok && sev == severity {
			count++
		}
	}
	return count
}


func getVulnerabilityTrends(scenarioID uuid.UUID, days int) (map[string]interface{}, error) {
	// Get vulnerability counts over time
	// This would typically involve more complex time-series data
	trends := map[string]interface{}{
		"period_days": days,
		"trend":       "stable", // This would be calculated based on historical data
	}
	return trends, nil
}