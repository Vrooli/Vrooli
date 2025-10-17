package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// getStatsHandler returns dashboard statistics
func (s *Server) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Count issues by status
	var totalIssues, openIssues, inProgress, completedToday int

	allIssues, _ := s.getAllIssues("", "", "", "", 0)
	totalIssues = len(allIssues)

	today := time.Now().UTC().Format("2006-01-02")

	for _, issue := range allIssues {
		switch issue.Status {
		case "open":
			openIssues++
		case "active":
			inProgress++
		case "completed":
			if strings.HasPrefix(issue.Metadata.ResolvedAt, today) {
				completedToday++
			}
		}
	}

	// Count by app
	appCounts := make(map[string]int)
	for _, issue := range allIssues {
		appCounts[issue.AppID]++
	}

	// Convert to top apps list
	type appCount struct {
		AppName    string `json:"app_name"`
		IssueCount int    `json:"issue_count"`
	}
	var topApps []appCount
	for appID, count := range appCounts {
		topApps = append(topApps, appCount{AppName: appID, IssueCount: count})
	}

	// Sort by issue count
	sort.Slice(topApps, func(i, j int) bool {
		return topApps[i].IssueCount > topApps[j].IssueCount
	})

	// Limit to top 5
	if len(topApps) > 5 {
		topApps = topApps[:5]
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"stats": map[string]interface{}{
				"total_issues":         totalIssues,
				"open_issues":          openIssues,
				"in_progress":          inProgress,
				"completed_today":      completedToday,
				"avg_resolution_hours": 24.5, // TODO: Calculate from resolved issues
				"top_apps":             topApps,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getAgentsHandler returns available agents
func (s *Server) getAgentsHandler(w http.ResponseWriter, r *http.Request) {
	// Single unified agent exposed for the simplified workflow
	agents := []Agent{
		{
			ID:             "unified-resolver",
			Name:           "unified-resolver",
			DisplayName:    "Unified Issue Resolver",
			Description:    "Single-pass agent that triages, investigates, and proposes fixes",
			Capabilities:   []string{"triage", "investigate", "fix", "test"},
			IsActive:       true,
			SuccessRate:    88.4,
			TotalRuns:      173,
			SuccessfulRuns: 153,
		},
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"agents": agents,
			"count":  len(agents),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// triggerFixGenerationHandler returns deprecation notice
func (s *Server) triggerFixGenerationHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Fix generation now runs automatically as part of the unified /investigate workflow. Pass auto_resolve=false to /investigate for investigation-only runs.", http.StatusGone)
}

// getAppsHandler returns a list of applications with issue counts
func (s *Server) getAppsHandler(w http.ResponseWriter, r *http.Request) {
	// Count issues per app
	allIssues, _ := s.getAllIssues("", "", "", "", 0)
	appStats := make(map[string]struct {
		total int
		open  int
	})

	for _, issue := range allIssues {
		stats := appStats[issue.AppID]
		stats.total++
		if issue.Status == "open" || issue.Status == "active" {
			stats.open++
		}
		appStats[issue.AppID] = stats
	}

	var apps []App
	for appID, stats := range appStats {
		apps = append(apps, App{
			ID:          appID,
			Name:        appID,
			DisplayName: strings.Title(strings.ReplaceAll(appID, "-", " ")),
			Type:        "scenario",
			Status:      "active",
			TotalIssues: stats.total,
			OpenIssues:  stats.open,
		})
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"apps":  apps,
			"count": len(apps),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getAgentSettingsHandler returns the current agent backend settings
func (s *Server) getAgentSettingsHandler(w http.ResponseWriter, r *http.Request) {
	settingsPath := filepath.Join(s.config.ScenarioRoot, "initialization/configuration/agent-settings.json")

	// Load settings from file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		LogErrorErr("Failed to read agent settings", err, "path", settingsPath)
		http.Error(w, "Failed to load agent settings", http.StatusInternalServerError)
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		LogErrorErr("Failed to parse agent settings", err)
		http.Error(w, "Failed to parse agent settings", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Data:    settings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateAgentSettingsHandler updates agent backend settings
func (s *Server) updateAgentSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider        string  `json:"provider"`
		AutoFallback    bool    `json:"auto_fallback"`
		TimeoutSeconds  *int    `json:"timeout_seconds"`  // Optional timeout update
		MaxTurns        *int    `json:"max_turns"`        // Optional max turns update
		AllowedTools    *string `json:"allowed_tools"`    // Optional allowed tools update
		SkipPermissions *bool   `json:"skip_permissions"` // Optional skip permissions update
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate provider if provided
	if req.Provider != "" {
		validProviders := map[string]bool{
			"codex":       true,
			"claude-code": true,
		}

		if !validProviders[req.Provider] {
			http.Error(w, "Invalid provider. Must be 'codex' or 'claude-code'", http.StatusBadRequest)
			return
		}
	}

	settingsPath := filepath.Join(s.config.ScenarioRoot, "initialization/configuration/agent-settings.json")

	// Load current settings
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		LogErrorErr("Failed to read agent settings", err, "path", settingsPath)
		http.Error(w, "Failed to load agent settings", http.StatusInternalServerError)
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		LogErrorErr("Failed to parse agent settings", err)
		http.Error(w, "Failed to parse agent settings", http.StatusInternalServerError)
		return
	}

	agentBackendMap, _ := settings["agent_backend"].(map[string]interface{})
	if agentBackendMap == nil {
		agentBackendMap = map[string]interface{}{}
		settings["agent_backend"] = agentBackendMap
	}

	// Ensure fallback order exists for new configurations
	if _, ok := agentBackendMap["fallback_order"]; !ok {
		agentBackendMap["fallback_order"] = []string{"codex", "claude-code"}
	}

	// Determine current provider and apply updates
	currentProvider := "claude-code"
	if provider, ok := agentBackendMap["provider"].(string); ok && provider != "" {
		currentProvider = provider
	}

	if req.Provider != "" {
		agentBackendMap["provider"] = req.Provider
		currentProvider = req.Provider
	}

	agentBackendMap["auto_fallback"] = req.AutoFallback
	if req.SkipPermissions != nil {
		agentBackendMap["skip_permissions"] = *req.SkipPermissions
	}

	targetProvider := currentProvider

	// Update provider operation settings if provided
	if req.TimeoutSeconds != nil || req.MaxTurns != nil || req.AllowedTools != nil {
		if providers, ok := settings["providers"].(map[string]interface{}); ok {
			if providerConfig, ok := providers[targetProvider].(map[string]interface{}); ok {
				if operations, ok := providerConfig["operations"].(map[string]interface{}); ok {
					for _, opKey := range []string{"investigate", "fix"} {
						opMap, ok := operations[opKey].(map[string]interface{})
						if !ok {
							continue
						}

						if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
							opMap["timeout_seconds"] = *req.TimeoutSeconds
							LogInfo(
								"Updated agent timeout",
								"provider", targetProvider,
								"operation", opKey,
								"timeout_seconds", *req.TimeoutSeconds,
								"timeout_minutes", *req.TimeoutSeconds/60,
							)
						}
						if req.MaxTurns != nil && *req.MaxTurns > 0 {
							opMap["max_turns"] = *req.MaxTurns
							LogInfo("Updated agent max turns", "provider", targetProvider, "operation", opKey, "max_turns", *req.MaxTurns)
						}
						if req.AllowedTools != nil {
							trimmed := strings.TrimSpace(*req.AllowedTools)
							opMap["allowed_tools"] = trimmed
							LogInfo("Updated agent allowed tools", "provider", targetProvider, "operation", opKey, "allowed_tools", trimmed)
						}
					}
				}
			}
		}
	}

	// Write back to file with proper formatting
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		LogErrorErr("Failed to marshal agent settings", err)
		http.Error(w, "Failed to save agent settings", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(settingsPath, updatedData, 0o644); err != nil {
		LogErrorErr("Failed to write agent settings", err, "path", settingsPath)
		http.Error(w, "Failed to save agent settings", http.StatusInternalServerError)
		return
	}

	// CRITICAL: Reload settings into memory so changes take effect immediately
	ReloadAgentSettings()

	LogInfo("Agent settings updated", "provider", targetProvider, "auto_fallback", req.AutoFallback)

	response := ApiResponse{
		Success: true,
		Message: "Agent settings updated successfully",
		Data: map[string]interface{}{
			"agent_backend": settings["agent_backend"],
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
