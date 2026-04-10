// Package server provides the ServerAdapter for tool execution.
//
// This file implements the bridge between the Server and the IssueOperations
// interface required by the tool executor.
package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/services"
	"app-issue-tracker-api/internal/toolexecution"
)

// Verify ServerAdapter implements IssueOperations
var _ toolexecution.IssueOperations = (*ServerAdapter)(nil)

// ServerAdapter adapts the Server to the IssueOperations interface.
type ServerAdapter struct {
	server *Server
}

// NewServerAdapter creates a new ServerAdapter.
func NewServerAdapter(server *Server) *ServerAdapter {
	return &ServerAdapter{server: server}
}

// -----------------------------------------------------------------------------
// Issue Discovery
// -----------------------------------------------------------------------------

// ListIssues returns issues matching the given filters.
func (a *ServerAdapter) ListIssues(status, priority, issueType, targetID, targetType string, limit int) ([]interface{}, error) {
	issues, err := a.server.issues.ListIssues(status, priority, issueType, targetID, targetType, limit)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(issues))
	for i, issue := range issues {
		result[i] = issue
	}
	return result, nil
}

// GetIssue returns a single issue by ID.
func (a *ServerAdapter) GetIssue(issueID string) (interface{}, error) {
	issue, _, _, err := a.server.issues.LoadIssueWithStatus(issueID)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

// SearchIssues searches for issues by keyword.
func (a *ServerAdapter) SearchIssues(query string, limit int) ([]interface{}, error) {
	issues, err := a.server.issues.SearchIssues(query, limit)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(issues))
	for i, issue := range issues {
		result[i] = issue
	}
	return result, nil
}

// GetStatistics returns issue statistics.
func (a *ServerAdapter) GetStatistics() (map[string]interface{}, error) {
	issues, err := a.server.getAllIssues("", "", "", "", 0)
	if err != nil {
		return nil, err
	}

	analytics := newIssueAnalytics(issues, time.Now())
	totalIssues, openIssues, inProgress, completedToday, avgResolutionHours := analytics.totals()
	topApps := analytics.topApps(5)
	manualFailures, autoFailures, failureReasons := analytics.failureMetrics()
	avgCompletedAge := analytics.avgCompletedAgeHours()

	return map[string]interface{}{
		"total_issues":              totalIssues,
		"open_issues":               openIssues,
		"in_progress":               inProgress,
		"completed_today":           completedToday,
		"avg_resolution_hours":      avgResolutionHours,
		"avg_completed_age_hours":   avgCompletedAge,
		"top_apps":                  topApps,
		"manual_failures":           manualFailures,
		"auto_failures":             autoFailures,
		"failure_reasons_breakdown": failureReasons,
	}, nil
}

// ListApplications returns all applications with issues.
func (a *ServerAdapter) ListApplications() ([]interface{}, error) {
	issues, err := a.server.getAllIssues("", "", "", "", 0)
	if err != nil {
		return nil, err
	}

	// Build app statistics
	appStats := make(map[string]int)
	for _, issue := range issues {
		for _, target := range issue.Targets {
			key := fmt.Sprintf("%s:%s", target.Type, target.ID)
			appStats[key]++
		}
	}

	result := make([]interface{}, 0, len(appStats))
	for app, count := range appStats {
		result = append(result, map[string]interface{}{
			"id":          app,
			"issue_count": count,
		})
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// Issue Management
// -----------------------------------------------------------------------------

// CreateIssue creates a new issue.
func (a *ServerAdapter) CreateIssue(req interface{}) (interface{}, string, error) {
	// Convert the generic request to CreateIssueRequest
	data, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	var createReq issuespkg.CreateIssueRequest
	if err := json.Unmarshal(data, &createReq); err != nil {
		return nil, "", fmt.Errorf("failed to parse request: %w", err)
	}

	issue, storagePath, err := a.server.issues.CreateIssue(&createReq)
	if err != nil {
		return nil, "", err
	}

	// Publish event
	a.server.hub.Publish(NewEvent(EventIssueCreated, IssueEventData{Issue: issue}))

	return issue, storagePath, nil
}

// UpdateIssue updates an existing issue.
func (a *ServerAdapter) UpdateIssue(issueID string, updates map[string]interface{}) (interface{}, error) {
	// Convert updates to UpdateIssueRequest
	data, err := json.Marshal(updates)
	if err != nil {
		return nil, err
	}

	var updateReq issuespkg.UpdateIssueRequest
	if err := json.Unmarshal(data, &updateReq); err != nil {
		return nil, err
	}

	issue, _, err := a.server.issues.UpdateIssue(issueID, &updateReq)
	if err != nil {
		return nil, err
	}

	// Publish event
	a.server.hub.Publish(NewEvent(EventIssueUpdated, IssueEventData{Issue: issue}))

	return issue, nil
}

// TransitionIssue changes an issue's status.
func (a *ServerAdapter) TransitionIssue(issueID, newStatus, reason string) (interface{}, error) {
	// Create an update request with just the status change
	updateReq := &issuespkg.UpdateIssueRequest{
		Status: &newStatus,
	}

	// If marking as failed, add failure reason
	if newStatus == "failed" && reason != "" {
		markedAsFailed := true
		updateReq.ManualReview = &struct {
			MarkedAsFailed *bool   `json:"marked_as_failed"`
			FailureReason  *string `json:"failure_reason"`
			ReviewedBy     *string `json:"reviewed_by"`
			ReviewedAt     *string `json:"reviewed_at"`
			ReviewNotes    *string `json:"review_notes"`
			OriginalStatus *string `json:"original_status"`
		}{
			MarkedAsFailed: &markedAsFailed,
			FailureReason:  &reason,
		}
	}

	issue, _, err := a.server.issues.UpdateIssue(issueID, updateReq)
	if err != nil {
		return nil, err
	}

	// Publish event
	a.server.hub.Publish(NewEvent(EventIssueStatusChanged, IssueEventData{Issue: issue}))

	return issue, nil
}

// AddArtifacts adds artifacts to an issue.
func (a *ServerAdapter) AddArtifacts(issueID string, artifacts []interface{}) error {
	// Load the issue
	issue, _, issueDir, err := a.server.issues.LoadIssueWithStatus(issueID)
	if err != nil {
		return err
	}

	// Convert artifacts to ArtifactPayload
	var payloads []issuespkg.ArtifactPayload
	for _, art := range artifacts {
		data, err := json.Marshal(art)
		if err != nil {
			continue
		}
		var payload issuespkg.ArtifactPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}
		payloads = append(payloads, payload)
	}

	// Store artifacts
	return a.server.issues.StoreArtifacts(issue, issueDir, payloads, false)
}

// -----------------------------------------------------------------------------
// Investigation
// -----------------------------------------------------------------------------

// TriggerInvestigation triggers an AI investigation.
func (a *ServerAdapter) TriggerInvestigation(issueID, agentID string, autoResolve bool) (string, error) {
	// Trigger investigation via the investigation service
	// The TriggerInvestigation method returns only an error, not a runID
	err := a.server.investigations.TriggerInvestigation(issueID, agentID, autoResolve)
	if err != nil {
		return "", err
	}

	// Return the issue ID as the run ID for tracking
	return issueID, nil
}

// GetInvestigationStatus gets the current status of an investigation.
func (a *ServerAdapter) GetInvestigationStatus(issueID string) (map[string]interface{}, error) {
	// Check if currently running
	processes := a.server.processor.RunningProcesses()
	for _, proc := range processes {
		if proc.IssueID == issueID {
			return map[string]interface{}{
				"issue_id":   proc.IssueID,
				"agent_id":   proc.AgentID,
				"run_id":     proc.RunID,
				"status":     "running",
				"start_time": proc.StartTime, // StartTime is already a string
			}, nil
		}
	}

	// Not running - check issue status
	issue, _, _, err := a.server.issues.LoadIssueWithStatus(issueID)
	if err != nil {
		return nil, err
	}

	// Map issue status to investigation status
	status := "completed"
	if issue.Status == "failed" {
		status = "failed"
	} else if issue.Status == "open" || issue.Status == "active" {
		status = "active"
	}

	result := map[string]interface{}{
		"issue_id": issueID,
		"status":   status,
	}

	// Add investigation results if available
	if issue.Investigation.CompletedAt != "" {
		result["completed_at"] = issue.Investigation.CompletedAt
	}
	if issue.Investigation.RootCause != "" {
		result["root_cause"] = issue.Investigation.RootCause
	}
	if issue.Investigation.SuggestedFix != "" {
		result["suggested_fix"] = issue.Investigation.SuggestedFix
	}

	return result, nil
}

// PreviewInvestigationPrompt previews the prompt for an investigation.
func (a *ServerAdapter) PreviewInvestigationPrompt(issueID, agentID string) (string, error) {
	// Load the issue
	issue, _, issueDir, err := a.server.issues.LoadIssueWithStatus(issueID)
	if err != nil {
		return "", err
	}

	// Build the prompt using the investigation service's method
	timestamp := time.Now().UTC().Format(time.RFC3339)
	prompt := a.server.investigations.buildInvestigationPrompt(issue, issueDir, agentID, a.server.config.ScenarioRoot, timestamp)
	return prompt, nil
}

// GetAgentConversation gets the conversation for an investigation.
func (a *ServerAdapter) GetAgentConversation(issueID string, maxEvents int) (interface{}, error) {
	conversation, err := a.server.content.AgentConversation(issueID, maxEvents)
	if err != nil {
		return nil, err
	}
	return conversation, nil
}

// ListRunningInvestigations lists all running investigations.
func (a *ServerAdapter) ListRunningInvestigations() ([]interface{}, error) {
	processes := a.server.processor.RunningProcesses()
	result := make([]interface{}, len(processes))
	for i, proc := range processes {
		result[i] = map[string]interface{}{
			"issue_id":   proc.IssueID,
			"agent_id":   proc.AgentID,
			"run_id":     proc.RunID,
			"status":     proc.Status,
			"start_time": proc.StartTime, // StartTime is already a string
		}
	}
	return result, nil
}

// StopInvestigation stops a running investigation.
func (a *ServerAdapter) StopInvestigation(issueID string) (bool, error) {
	stopped := a.server.processor.CancelRunningProcess(issueID, "stopped via tool execution")
	return stopped, nil
}

// IsIssueRunning checks if an issue has a running investigation.
func (a *ServerAdapter) IsIssueRunning(issueID string) bool {
	return a.server.processor.IsRunning(issueID)
}

// -----------------------------------------------------------------------------
// Agent/Automation
// -----------------------------------------------------------------------------

// ListAgents lists available agents.
func (a *ServerAdapter) ListAgents() ([]interface{}, error) {
	settings := GetAgentSettings()

	// Calculate stats from issues
	issues, err := a.server.getAllIssues("", "", "", "", 0)
	if err != nil {
		return nil, err
	}

	// Calculate success metrics from real data
	totalRuns := 0
	successfulRuns := 0
	for _, issue := range issues {
		if issue.Investigation.CompletedAt != "" {
			totalRuns++
			if issue.Status == StatusCompleted {
				successfulRuns++
			}
		}
	}

	successRate := 0.0
	if totalRuns > 0 {
		successRate = float64(successfulRuns) / float64(totalRuns) * 100
	}

	// Return the configured agent as the available agent
	agent := map[string]interface{}{
		"id":           "unified-resolver",
		"name":         "Unified Resolver",
		"runner_type":  settings.RunnerType,
		"total_runs":   totalRuns,
		"success_rate": successRate,
		"capabilities": []string{"investigation", "fix-suggestion", "root-cause-analysis"},
	}

	return []interface{}{agent}, nil
}

// GetAgentSettings gets the current agent settings.
func (a *ServerAdapter) GetAgentSettings() (map[string]interface{}, error) {
	settings, err := LoadAgentSettings(a.server.config.ScenarioRoot)
	if err != nil {
		return nil, err
	}
	// Convert to map
	data, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateAgentSettings updates agent settings.
func (a *ServerAdapter) UpdateAgentSettings(updates map[string]interface{}) error {
	settingsPath := filepath.Join(a.server.config.ScenarioRoot, "initialization/configuration/agent-settings.json")

	// Load current file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("failed to read agent settings: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse agent settings: %w", err)
	}

	agentManagerMap, _ := settings["agent_manager"].(map[string]interface{})
	if agentManagerMap == nil {
		agentManagerMap = map[string]interface{}{}
		settings["agent_manager"] = agentManagerMap
	}

	// Apply updates
	if runnerType, ok := updates["runner_type"].(string); ok && runnerType != "" {
		agentManagerMap["runner_type"] = runnerType
	}
	if timeout, ok := updates["timeout_seconds"].(float64); ok && timeout > 0 {
		agentManagerMap["timeout_seconds"] = int(timeout)
	}
	if maxTurns, ok := updates["max_turns"].(float64); ok && maxTurns > 0 {
		agentManagerMap["max_turns"] = int(maxTurns)
	}
	if skipPerms, ok := updates["skip_permissions"].(bool); ok {
		agentManagerMap["skip_permissions"] = skipPerms
	}

	// Write back
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, updatedData, 0o644); err != nil {
		return fmt.Errorf("failed to write agent settings: %w", err)
	}

	// Reload settings so in-memory cache is updated
	if err := ReloadAgentSettings(); err != nil {
		logging.LogWarn("Failed to reload agent settings after update", "error", err)
	}

	return nil
}

// GetProcessorState gets the automation processor state.
func (a *ServerAdapter) GetProcessorState() (map[string]interface{}, error) {
	state := a.server.processor.CurrentState()
	// Convert to map
	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// Reporting
// -----------------------------------------------------------------------------

// ExportIssues exports issues in the specified format.
func (a *ServerAdapter) ExportIssues(format, status, priority, issueType, targetID string) (interface{}, error) {
	// Get issues matching filters
	issues, err := a.server.issues.ListIssues(status, priority, issueType, targetID, "", 1000)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return issues, nil
	case "csv":
		return exportToCSV(issues), nil
	case "markdown":
		return exportToMarkdown(issues), nil
	default:
		return issues, nil
	}
}

// AnalyzeFailures analyzes failed issues.
func (a *ServerAdapter) AnalyzeFailures(since *time.Time, targetID string) (map[string]interface{}, error) {
	// Get failed issues
	issues, err := a.server.issues.ListIssues("failed", "", "", targetID, "", 1000)
	if err != nil {
		return nil, err
	}

	// Filter by since date if provided
	var filtered []issuespkg.Issue
	for _, issue := range issues {
		if since != nil {
			// Parse issue created time and filter
			if issue.Metadata.CreatedAt != "" {
				createdTime, err := time.Parse(time.RFC3339, issue.Metadata.CreatedAt)
				if err == nil && createdTime.Before(*since) {
					continue
				}
			}
		}
		filtered = append(filtered, issue)
	}

	// Analyze failures
	analysis := map[string]interface{}{
		"total_failures": len(filtered),
		"breakdown":      analyzeFailureReasons(filtered),
		"analyzed_at":    time.Now().UTC().Format(time.RFC3339),
	}

	if targetID != "" {
		analysis["target_id"] = targetID
	}

	return analysis, nil
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

func exportToCSV(issues []issuespkg.Issue) string {
	// Simple CSV export
	csv := "ID,Title,Status,Priority,Type,CreatedAt\n"
	for _, issue := range issues {
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			issue.ID, issue.Title, issue.Status, issue.Priority, issue.Type, issue.Reporter.Timestamp)
	}
	return csv
}

func exportToMarkdown(issues []issuespkg.Issue) string {
	// Simple Markdown export
	md := "# Issues Report\n\n"
	for _, issue := range issues {
		md += fmt.Sprintf("## %s\n\n", issue.Title)
		md += fmt.Sprintf("- **ID:** %s\n", issue.ID)
		md += fmt.Sprintf("- **Status:** %s\n", issue.Status)
		md += fmt.Sprintf("- **Priority:** %s\n", issue.Priority)
		md += fmt.Sprintf("- **Type:** %s\n\n", issue.Type)
		if issue.Description != "" {
			md += fmt.Sprintf("### Description\n\n%s\n\n", issue.Description)
		}
	}
	return md
}

func analyzeFailureReasons(issues []issuespkg.Issue) map[string]int {
	reasons := make(map[string]int)
	for _, issue := range issues {
		reason := "unknown"
		if issue.ManualReview.MarkedAsFailed && issue.ManualReview.FailureReason != "" {
			reason = issue.ManualReview.FailureReason
		}
		reasons[reason]++
	}
	return reasons
}

func fieldsToArgs(fields map[string]interface{}) []any {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}

// Ensure services.ErrIssueNotFound is recognized
var _ = services.ErrIssueNotFound
