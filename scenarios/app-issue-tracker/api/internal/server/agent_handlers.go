package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"app-issue-tracker-api/internal/agents"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/handlers"
)

const maxAgentSettingsPayloadBytes int64 = 256 << 10 // 256 KiB

// getStatsHandler returns dashboard statistics
func (s *Server) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	issues, err := s.getAllIssues("", "", "", "", 0)
	if err != nil {
		logging.LogErrorErr("Failed to load issues for stats", err)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load stats")
		return
	}

	analytics := newIssueAnalytics(issues, time.Now())
	totalIssues, openIssues, inProgress, completedToday, avgResolutionHours := analytics.totals()
	topApps := analytics.topApps(5)
	manualFailures, autoFailures, failureReasons := analytics.failureMetrics()
	avgCompletedAge := analytics.avgCompletedAgeHours()

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"stats": map[string]interface{}{
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
			},
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write stats response", err)
	}
}

// getAgentsHandler returns available agents (current agent-manager runner configuration).
func (s *Server) getAgentsHandler(w http.ResponseWriter, r *http.Request) {
	settings := GetAgentSettings()

	// Calculate actual stats from issues
	issues, err := s.getAllIssues("", "", "", "", 0)
	if err != nil {
		logging.LogErrorErr("Failed to load issues for agent stats", err)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load agent stats")
		return
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

	// Return the current runner as the active agent
	agentList := []Agent{
		{
			ID:             agents.UnifiedResolverID, // Keep for backwards compatibility
			Name:           settings.RunnerType,
			DisplayName:    formatRunnerDisplayName(settings.RunnerType),
			Description:    "AI agent that triages, investigates, and proposes fixes for issues",
			Capabilities:   []string{"triage", "investigate", "fix", "test"},
			IsActive:       true,
			SuccessRate:    successRate,
			TotalRuns:      totalRuns,
			SuccessfulRuns: successfulRuns,
		},
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"agents": agentList,
			"count":  len(agentList),
			"runner": settings.RunnerType,
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write agents response", err)
	}
}

// formatRunnerDisplayName converts runner ID to human-readable name
func formatRunnerDisplayName(runnerType string) string {
	switch runnerType {
	case "codex":
		return "Codex Runner"
	case "claude-code":
		return "Claude Code Runner"
	case "opencode":
		return "OpenCode Runner"
	default:
		// Title case the runner name
		caser := cases.Title(language.English)
		return caser.String(strings.ReplaceAll(runnerType, "-", " ")) + " Runner"
	}
}

// getAppsHandler returns a list of applications with issue counts
func (s *Server) getAppsHandler(w http.ResponseWriter, r *http.Request) {
	issues, err := s.getAllIssues("", "", "", "", 0)
	if err != nil {
		logging.LogErrorErr("Failed to load issues for app summary", err)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load apps")
		return
	}

	analytics := newIssueAnalytics(issues, time.Now())
	summaries := analytics.appSummaries()

	caser := cases.Title(language.English)
	apps := make([]App, 0, len(summaries))
	for appID, stats := range summaries {
		displayName := caser.String(strings.ReplaceAll(appID, "-", " "))
		apps = append(apps, App{
			ID:          appID,
			Name:        appID,
			DisplayName: displayName,
			Type:        "scenario",
			Status:      "active",
			TotalIssues: stats.total,
			OpenIssues:  stats.open,
		})
	}

	sort.Slice(apps, func(i, j int) bool {
		if apps[i].TotalIssues == apps[j].TotalIssues {
			return apps[i].ID < apps[j].ID
		}
		return apps[i].TotalIssues > apps[j].TotalIssues
	})

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"apps":  apps,
			"count": len(apps),
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write apps response", err)
	}
}

// getAgentSettingsHandler returns the current agent-manager settings with constraints
func (s *Server) getAgentSettingsHandler(w http.ResponseWriter, r *http.Request) {
	settingsPath := filepath.Join(s.config.ScenarioRoot, "initialization/configuration/agent-settings.json")

	// Load settings from file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		logging.LogErrorErr("Failed to read agent settings", err, "path", settingsPath)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load agent settings")
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		logging.LogErrorErr("Failed to parse agent settings", err)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to parse agent settings")
		return
	}

	// Include constraints for UI components (sliders, etc.)
	constraints := GetSettingsConstraints()

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"settings":    settings["agent_manager"],
			"constraints": constraints,
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write agent settings response", err)
	}
}

// getIssueStatusesHandler returns the list of supported issue lifecycle statuses
func (s *Server) getIssueStatusesHandler(w http.ResponseWriter, r *http.Request) {
	statuses := ValidIssueStatuses()
	responsePayload := make([]map[string]string, 0, len(statuses))
	for _, status := range statuses {
		responsePayload = append(responsePayload, map[string]string{
			"id":    status,
			"label": formatStatusLabel(status),
		})
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"statuses": responsePayload,
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write issue statuses response", err)
	}
}

// updateAgentSettingsHandler updates agent-manager settings
func (s *Server) updateAgentSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RunnerType      string  `json:"runner_type"`
		TimeoutSeconds  *int    `json:"timeout_seconds"`  // Optional timeout update
		MaxTurns        *int    `json:"max_turns"`        // Optional max turns update
		AllowedTools    *string `json:"allowed_tools"`    // Optional allowed tools update
		SkipPermissions *bool   `json:"skip_permissions"` // Optional skip permissions update
	}

	if err := handlers.DecodeJSON(r, &req, maxAgentSettingsPayloadBytes); err != nil {
		handlers.WriteDecodeError(w, err)
		return
	}

	// Validate runner_type if provided
	if req.RunnerType != "" {
		validRunner := map[string]bool{
			"claude-code": true,
			"codex":       true,
			"opencode":    true,
		}
		if !validRunner[req.RunnerType] {
			handlers.WriteError(w, http.StatusBadRequest, "Invalid runner_type. Must be 'claude-code', 'codex', or 'opencode'")
			return
		}
	}

	// Validate timeout_seconds
	if req.TimeoutSeconds != nil {
		if *req.TimeoutSeconds < MinTimeoutSeconds || *req.TimeoutSeconds > MaxTimeoutSeconds {
			handlers.WriteError(w, http.StatusBadRequest,
				fmt.Sprintf("timeout_seconds must be between %d and %d", MinTimeoutSeconds, MaxTimeoutSeconds))
			return
		}
	}

	// Validate max_turns
	if req.MaxTurns != nil {
		if *req.MaxTurns < MinMaxTurns || *req.MaxTurns > MaxMaxTurns {
			handlers.WriteError(w, http.StatusBadRequest,
				fmt.Sprintf("max_turns must be between %d and %d", MinMaxTurns, MaxMaxTurns))
			return
		}
	}

	settingsPath := filepath.Join(s.config.ScenarioRoot, "initialization/configuration/agent-settings.json")

	// Load current settings
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		logging.LogErrorErr("Failed to read agent settings", err, "path", settingsPath)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load agent settings")
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		logging.LogErrorErr("Failed to parse agent settings", err)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to parse agent settings")
		return
	}

	agentManagerMap, _ := settings["agent_manager"].(map[string]interface{})
	if agentManagerMap == nil {
		agentManagerMap = map[string]interface{}{}
		settings["agent_manager"] = agentManagerMap
	}

	if req.RunnerType != "" {
		agentManagerMap["runner_type"] = req.RunnerType
	}
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
		agentManagerMap["timeout_seconds"] = *req.TimeoutSeconds
	}
	if req.MaxTurns != nil && *req.MaxTurns > 0 {
		agentManagerMap["max_turns"] = *req.MaxTurns
	}
	if req.AllowedTools != nil {
		trimmed := strings.TrimSpace(*req.AllowedTools)
		agentManagerMap["allowed_tools"] = trimmed
	}
	if req.SkipPermissions != nil {
		agentManagerMap["skip_permissions"] = *req.SkipPermissions
	}

	// Write back to file with proper formatting
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		logging.LogErrorErr("Failed to marshal agent settings", err)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to save agent settings")
		return
	}

	if err := os.WriteFile(settingsPath, updatedData, 0o644); err != nil {
		logging.LogErrorErr("Failed to write agent settings", err, "path", settingsPath)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to save agent settings")
		return
	}

	// Reload settings into memory so changes take effect immediately
	if err := ReloadAgentSettings(); err != nil {
		logging.LogErrorErr("Failed to reload agent settings", err)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to reload agent settings")
		return
	}

	if err := s.updateAgentManagerProfile(); err != nil {
		logging.LogWarn("Failed to update agent-manager profile", "error", err)
	}

	logging.LogInfo("Agent settings updated", "runner_type", req.RunnerType)

	response := ApiResponse{
		Success: true,
		Message: "Agent settings updated successfully",
		Data: map[string]interface{}{
			"agent_manager": settings["agent_manager"],
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write agent settings update response", err)
	}
}

func formatStatusLabel(status string) string {
	parts := strings.Split(status, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}
