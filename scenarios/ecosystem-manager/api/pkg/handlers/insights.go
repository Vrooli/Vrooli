package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ecosystem-manager/api/pkg/insights"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/gorilla/mux"
)

// InsightHandlers contains handlers for insight-related endpoints
type InsightHandlers struct {
	processor    ProcessorAPI
	scenarioRoot string // Root directory for scenarios
}

// NewInsightHandlers creates a new InsightHandlers instance
func NewInsightHandlers(processor ProcessorAPI, scenarioRoot string) *InsightHandlers {
	return &InsightHandlers{
		processor:    processor,
		scenarioRoot: scenarioRoot,
	}
}

// GetTaskInsightsHandler retrieves all insight reports for a task
// GET /api/tasks/{id}/insights
func (h *InsightHandlers) GetTaskInsightsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	reports, err := h.processor.LoadInsightReports(taskID)
	if err != nil {
		log.Printf("ERROR: Failed to load insight reports for task %s: %v", taskID, err)
		systemlog.Errorf("Failed to load insight reports for task %s: %v", taskID, err)
		writeError(w, fmt.Sprintf("Failed to load insight reports: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"task_id": taskID,
		"reports": reports,
		"count":   len(reports),
	}, http.StatusOK)
}

// GetInsightReportHandler retrieves a specific insight report
// GET /api/tasks/{id}/insights/{report_id}
func (h *InsightHandlers) GetInsightReportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]
	reportID := vars["report_id"]

	report, err := h.processor.LoadInsightReport(taskID, reportID)
	if err != nil {
		if err.Error() == "insight report not found" {
			writeError(w, "Insight report not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to load insight report %s for task %s: %v", reportID, taskID, err)
		systemlog.Errorf("Failed to load insight report %s for task %s: %v", reportID, taskID, err)
		writeError(w, fmt.Sprintf("Failed to load insight report: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, report, http.StatusOK)
}

// GenerateInsightReportHandler triggers insight generation for a task
// POST /api/tasks/{id}/insights/generate
func (h *InsightHandlers) GenerateInsightReportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Parse request body for options
	var options struct {
		Limit        int    `json:"limit"`
		StatusFilter string `json:"status_filter"` // e.g., "failed,timeout"
	}

	if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
		// Use defaults if no body provided
		options.Limit = 10
		options.StatusFilter = "failed,timeout"
	}

	// Validate limit
	if options.Limit <= 0 {
		options.Limit = 10
	}
	if options.Limit > 50 {
		options.Limit = 50 // Cap at 50 to avoid overwhelming the LLM
	}

	// Load execution history
	history, err := h.processor.LoadExecutionHistory(taskID)
	if err != nil {
		log.Printf("ERROR: Failed to load execution history for insight generation (task %s): %v", taskID, err)
		systemlog.Errorf("Failed to load execution history for insight generation (task %s): %v", taskID, err)
		writeError(w, fmt.Sprintf("Failed to load execution history: %v", err), http.StatusInternalServerError)
		return
	}

	if len(history) == 0 {
		writeError(w, "No execution history available to analyze", http.StatusBadRequest)
		return
	}

	// Filter executions
	filtered := filterExecutions(history, options.StatusFilter, options.Limit)

	if len(filtered) == 0 {
		writeError(w, fmt.Sprintf("No executions matching filter: %s", options.StatusFilter), http.StatusBadRequest)
		return
	}

	// Generate insight report using the workflow
	report, err := h.processor.GenerateInsightReportForTask(taskID, options.Limit, options.StatusFilter)
	if err != nil {
		log.Printf("ERROR: Failed to generate insight report for task %s: %v", taskID, err)
		systemlog.Errorf("Failed to generate insight report for task %s: %v", taskID, err)
		writeError(w, fmt.Sprintf("Failed to generate insight report: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully generated insight report %s for task %s (%d patterns, %d suggestions)",
		report.ID, taskID, len(report.Patterns), len(report.Suggestions))

	writeJSON(w, map[string]any{
		"message":   "Insight report generated successfully",
		"report_id": report.ID,
		"report":    *report,
	}, http.StatusCreated)
}

// UpdateSuggestionStatusHandler updates the status of a suggestion
// POST /api/tasks/{id}/insights/{report_id}/suggestions/{suggestion_id}/status
func (h *InsightHandlers) UpdateSuggestionStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]
	reportID := vars["report_id"]
	suggestionID := vars["suggestion_id"]

	var req struct {
		Status string `json:"status"` // pending, applied, rejected, superseded
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		"pending":    true,
		"applied":    true,
		"rejected":   true,
		"superseded": true,
	}
	if !validStatuses[req.Status] {
		writeError(w, "Invalid status. Must be: pending, applied, rejected, or superseded", http.StatusBadRequest)
		return
	}

	// Update suggestion status
	if err := h.processor.UpdateSuggestionStatus(taskID, reportID, suggestionID, req.Status); err != nil {
		log.Printf("ERROR: Failed to update suggestion status (task %s, report %s, suggestion %s): %v",
			taskID, reportID, suggestionID, err)
		systemlog.Errorf("Failed to update suggestion status: %v", err)
		writeError(w, fmt.Sprintf("Failed to update suggestion status: %v", err), http.StatusInternalServerError)
		return
	}

	systemlog.Infof("Updated suggestion %s status to %s (task %s, report %s)",
		suggestionID, req.Status, taskID, reportID)

	writeJSON(w, map[string]any{
		"message":       "Suggestion status updated successfully",
		"task_id":       taskID,
		"report_id":     reportID,
		"suggestion_id": suggestionID,
		"status":        req.Status,
	}, http.StatusOK)
}

// GetSystemInsightsHandler retrieves system-wide insights
// GET /api/insights/system
func (h *InsightHandlers) GetSystemInsightsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query params
	sinceDays := parseIntParam(r, "since_days", 7) // Default to last 7 days

	sinceTime := time.Now().AddDate(0, 0, -sinceDays)

	// Load all insight reports since the time window
	reports, err := h.processor.LoadAllInsightReports(sinceTime)
	if err != nil {
		log.Printf("ERROR: Failed to load system insights: %v", err)
		systemlog.Errorf("Failed to load system insights: %v", err)
		writeError(w, fmt.Sprintf("Failed to load system insights: %v", err), http.StatusInternalServerError)
		return
	}

	// Aggregate stats across all reports
	tasksByID := make(map[string]bool)
	totalExecutions := 0
	patternsByType := make(map[string]int)
	suggestionsByType := make(map[string]int)

	for _, report := range reports {
		tasksByID[report.TaskID] = true
		totalExecutions += report.ExecutionCount

		for _, pattern := range report.Patterns {
			patternsByType[pattern.Type]++
		}

		for _, suggestion := range report.Suggestions {
			suggestionsByType[suggestion.Type]++
		}
	}

	writeJSON(w, map[string]any{
		"time_window": map[string]any{
			"since": sinceTime,
			"days":  sinceDays,
		},
		"summary": map[string]any{
			"total_reports":       len(reports),
			"unique_tasks":        len(tasksByID),
			"total_executions":    totalExecutions,
			"patterns_by_type":    patternsByType,
			"suggestions_by_type": suggestionsByType,
		},
		"reports": reports,
	}, http.StatusOK)
}

// GenerateSystemInsightsHandler triggers system-wide insight generation
// POST /api/insights/system/generate
func (h *InsightHandlers) GenerateSystemInsightsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query params
	sinceDays := parseIntParam(r, "since_days", 7) // Default to last 7 days

	sinceTime := time.Now().AddDate(0, 0, -sinceDays)

	// Generate system insight report
	report, err := h.processor.GenerateSystemInsightReport(sinceTime)
	if err != nil {
		log.Printf("ERROR: Failed to generate system insight report: %v", err)
		systemlog.Errorf("Failed to generate system insight report: %v", err)
		writeError(w, fmt.Sprintf("Failed to generate system insight report: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully generated system insight report %s (%d tasks, %d cross-task patterns, %d suggestions)",
		report.ID, report.TaskCount, len(report.CrossTaskPatterns), len(report.SystemSuggestions))

	writeJSON(w, map[string]any{
		"message":   "System insight report generated successfully",
		"report_id": report.ID,
		"report":    *report,
	}, http.StatusCreated)
}

// ApplySuggestionHandler applies a suggestion to the codebase
// POST /api/tasks/{id}/insights/{report_id}/suggestions/{suggestion_id}/apply
func (h *InsightHandlers) ApplySuggestionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]
	reportID := vars["report_id"]
	suggestionID := vars["suggestion_id"]

	// Parse request body for options
	var req struct {
		DryRun bool `json:"dry_run"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to dry run if no body provided
		req.DryRun = true
	}

	// Load the insight report
	report, err := h.processor.LoadInsightReport(taskID, reportID)
	if err != nil {
		log.Printf("ERROR: Failed to load insight report for apply (task %s, report %s): %v", taskID, reportID, err)
		writeError(w, fmt.Sprintf("Failed to load insight report: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the suggestion (need to work with queue.Suggestion type)
	var suggestion *queue.Suggestion
	for i := range report.Suggestions {
		if report.Suggestions[i].ID == suggestionID {
			suggestion = &report.Suggestions[i]
			break
		}
	}

	if suggestion == nil {
		writeError(w, "Suggestion not found", http.StatusNotFound)
		return
	}

	// Check if already applied
	if suggestion.Status == "applied" {
		writeError(w, "Suggestion has already been applied", http.StatusBadRequest)
		return
	}

	// Determine scenario root
	scenarioRoot := h.scenarioRoot
	if scenarioRoot == "" {
		scenarioRoot = "/home/matthalloran8/Vrooli/scenarios"
	}

	// Extract scenario name from task ID
	scenarioName := extractScenarioName(taskID)
	if scenarioName != "" {
		scenarioRoot = filepath.Join(scenarioRoot, scenarioName)
	}

	// Create applier
	applier := insights.NewSuggestionApplier(scenarioRoot)

	// Apply the suggestion
	result, err := applier.ApplySuggestion(*suggestion, req.DryRun)
	if err != nil {
		log.Printf("ERROR: Failed to apply suggestion %s (task %s, report %s): %v",
			suggestionID, taskID, reportID, err)
		systemlog.Errorf("Failed to apply suggestion %s: %v", suggestionID, err)
		writeError(w, fmt.Sprintf("Failed to apply suggestion: %v", err), http.StatusInternalServerError)
		return
	}

	// Update suggestion status if not dry run
	if !req.DryRun && result.Success {
		if err := h.processor.UpdateSuggestionStatus(taskID, reportID, suggestionID, "applied"); err != nil {
			log.Printf("Warning: Failed to update suggestion status after apply: %v", err)
		}
	}

	statusCode := http.StatusOK
	if !req.DryRun && result.Success {
		statusCode = http.StatusCreated
	}

	writeJSON(w, map[string]any{
		"success":       result.Success,
		"message":       result.Message,
		"files_changed": result.FilesChanged,
		"backup_path":   result.BackupPath,
		"dry_run":       req.DryRun,
		"error":         result.Error,
	}, statusCode)
}

// extractScenarioName attempts to extract scenario name from task ID
func extractScenarioName(taskID string) string {
	if !hasPrefix(taskID, "scenario-") {
		return ""
	}

	taskID = trimPrefix(taskID, "scenario-improver-")
	taskID = trimPrefix(taskID, "scenario-generator-")

	parts := splitBy(taskID, "-")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if len(lastPart) >= 8 && isNumeric(lastPart) {
			parts = parts[:len(parts)-1]
		}
		if len(parts) > 0 {
			lastPart = parts[len(parts)-1]
			if len(lastPart) == 6 && isNumeric(lastPart) {
				parts = parts[:len(parts)-1]
			}
		}
	}

	return joinBy(parts, "-")
}

// Helper functions
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func trimPrefix(s, prefix string) string {
	if hasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func splitBy(s, sep string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
			i += len(sep) - 1
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func joinBy(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
