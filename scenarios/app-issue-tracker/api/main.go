package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Port      string
	QdrantURL string
	IssuesDir string
}

type Server struct {
	config *Config
}

type Issue struct {
	ID          string `yaml:"id" json:"id"`
	Title       string `yaml:"title" json:"title"`
	Description string `yaml:"description" json:"description"`
	Type        string `yaml:"type" json:"type"`
	Priority    string `yaml:"priority" json:"priority"`
	AppID       string `yaml:"app_id" json:"app_id"`
	Status      string `yaml:"status" json:"status"`

	Reporter struct {
		Name      string `yaml:"name" json:"name"`
		Email     string `yaml:"email" json:"email"`
		UserID    string `yaml:"user_id,omitempty" json:"user_id,omitempty"`
		Timestamp string `yaml:"timestamp" json:"timestamp"`
	} `yaml:"reporter" json:"reporter"`

	ErrorContext struct {
		ErrorMessage       string            `yaml:"error_message,omitempty" json:"error_message,omitempty"`
		ErrorLogs          string            `yaml:"error_logs,omitempty" json:"error_logs,omitempty"`
		StackTrace         string            `yaml:"stack_trace,omitempty" json:"stack_trace,omitempty"`
		AffectedFiles      []string          `yaml:"affected_files,omitempty" json:"affected_files,omitempty"`
		AffectedComponents []string          `yaml:"affected_components,omitempty" json:"affected_components,omitempty"`
		EnvironmentInfo    map[string]string `yaml:"environment_info,omitempty" json:"environment_info,omitempty"`
	} `yaml:"error_context,omitempty" json:"error_context,omitempty"`

	Investigation struct {
		AgentID                      string   `yaml:"agent_id,omitempty" json:"agent_id,omitempty"`
		StartedAt                    string   `yaml:"started_at,omitempty" json:"started_at,omitempty"`
		CompletedAt                  string   `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`
		Report                       string   `yaml:"report,omitempty" json:"report,omitempty"`
		RootCause                    string   `yaml:"root_cause,omitempty" json:"root_cause,omitempty"`
		SuggestedFix                 string   `yaml:"suggested_fix,omitempty" json:"suggested_fix,omitempty"`
		ConfidenceScore              *int     `yaml:"confidence_score,omitempty" json:"confidence_score,omitempty"`
		InvestigationDurationMinutes *int     `yaml:"investigation_duration_minutes,omitempty" json:"investigation_duration_minutes,omitempty"`
		TokensUsed                   *int     `yaml:"tokens_used,omitempty" json:"tokens_used,omitempty"`
		CostEstimate                 *float64 `yaml:"cost_estimate,omitempty" json:"cost_estimate,omitempty"`
	} `yaml:"investigation,omitempty" json:"investigation,omitempty"`

	Fix struct {
		SuggestedFix       string `yaml:"suggested_fix,omitempty" json:"suggested_fix,omitempty"`
		ImplementationPlan string `yaml:"implementation_plan,omitempty" json:"implementation_plan,omitempty"`
		Applied            bool   `yaml:"applied" json:"applied"`
		AppliedAt          string `yaml:"applied_at,omitempty" json:"applied_at,omitempty"`
		CommitHash         string `yaml:"commit_hash,omitempty" json:"commit_hash,omitempty"`
		PrURL              string `yaml:"pr_url,omitempty" json:"pr_url,omitempty"`
		VerificationStatus string `yaml:"verification_status,omitempty" json:"verification_status,omitempty"`
		RollbackPlan       string `yaml:"rollback_plan,omitempty" json:"rollback_plan,omitempty"`
		FixDurationMinutes *int   `yaml:"fix_duration_minutes,omitempty" json:"fix_duration_minutes,omitempty"`
	} `yaml:"fix,omitempty" json:"fix,omitempty"`

	Metadata struct {
		CreatedAt  string            `yaml:"created_at" json:"created_at"`
		UpdatedAt  string            `yaml:"updated_at" json:"updated_at"`
		ResolvedAt string            `yaml:"resolved_at,omitempty" json:"resolved_at,omitempty"`
		Tags       []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
		Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
		Watchers   []string          `yaml:"watchers,omitempty" json:"watchers,omitempty"`
	} `yaml:"metadata" json:"metadata"`

	Notes string `yaml:"notes,omitempty" json:"notes,omitempty"`
}

type Agent struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Description    string   `json:"description"`
	Capabilities   []string `json:"capabilities"`
	IsActive       bool     `json:"is_active"`
	SuccessRate    float64  `json:"success_rate"`
	TotalRuns      int      `json:"total_runs"`
	SuccessfulRuns int      `json:"successful_runs"`
}

type App struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	TotalIssues int    `json:"total_issues"`
	OpenIssues  int    `json:"open_issues"`
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func loadConfig() *Config {
	// Default to the actual scenario issues directory if not specified
	defaultIssuesDir := filepath.Join(getVrooliRoot(), "scenarios/app-issue-tracker/issues")
	if _, err := os.Stat("./issues"); err == nil {
		// If local issues directory exists, use it
		defaultIssuesDir = "./issues"
	}

	return &Config{
		Port:      getEnv("API_PORT", getEnv("PORT", "")),
		QdrantURL: getEnv("QDRANT_URL", "http://localhost:6333"),
		IssuesDir: getEnv("ISSUES_DIR", defaultIssuesDir),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// File operations for issues
func (s *Server) loadIssueFromFile(filePath string) (*Issue, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var issue Issue
	err = yaml.Unmarshal(data, &issue)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML: %v", err)
	}

	return &issue, nil
}

func (s *Server) saveIssueToFile(issue *Issue, folder string) error {
	// Generate filename based on priority and title
	priorityNum := s.getPriorityNumber(issue.Priority)
	safeTitle := strings.ToLower(issue.Title)
	safeTitle = strings.ReplaceAll(safeTitle, " ", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "_", "-")
	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range safeTitle {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	safeTitle = result.String()
	// Remove consecutive hyphens and trim
	safeTitle = strings.Trim(strings.ReplaceAll(safeTitle, "--", "-"), "-")

	filename := fmt.Sprintf("%03d-%s.yaml", priorityNum, safeTitle)
	filePath := filepath.Join(s.config.IssuesDir, folder, filename)

	// Ensure timestamps are set
	now := time.Now().UTC().Format(time.RFC3339)
	if issue.Metadata.CreatedAt == "" {
		issue.Metadata.CreatedAt = now
	}
	issue.Metadata.UpdatedAt = now
	issue.Status = folder

	data, err := yaml.Marshal(issue)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %v", err)
	}

	return ioutil.WriteFile(filePath, data, 0644)
}

func (s *Server) getPriorityNumber(priority string) int {
	switch strings.ToLower(priority) {
	case "critical":
		return s.getNextAvailableNumber(1, 99)
	case "high":
		return s.getNextAvailableNumber(100, 199)
	case "medium":
		return s.getNextAvailableNumber(200, 499)
	case "low":
		return s.getNextAvailableNumber(500, 999)
	default:
		return 200 // Default to medium
	}
}

func (s *Server) getNextAvailableNumber(start, end int) int {
	used := make(map[int]bool)

	// Check all folders for used numbers
	folders := []string{"open", "investigating", "in-progress", "fixed", "closed", "failed"}
	for _, folder := range folders {
		folderPath := filepath.Join(s.config.IssuesDir, folder)
		files, _ := filepath.Glob(filepath.Join(folderPath, "*.yaml"))

		for _, file := range files {
			filename := filepath.Base(file)
			if len(filename) >= 3 && filename[3] == '-' {
				if num, err := strconv.Atoi(filename[:3]); err == nil {
					used[num] = true
				}
			}
		}
	}

	// Find next available number in range
	for i := start; i <= end; i++ {
		if !used[i] {
			return i
		}
	}

	// If range is full, use the end number
	return end
}

func (s *Server) loadIssuesFromFolder(folder string) ([]Issue, error) {
	folderPath := filepath.Join(s.config.IssuesDir, folder)
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return []Issue{}, nil
	}

	files, err := filepath.Glob(filepath.Join(folderPath, "*.yaml"))
	if err != nil {
		return nil, err
	}

	var issues []Issue
	for _, file := range files {
		issue, err := s.loadIssueFromFile(file)
		if err != nil {
			log.Printf("Warning: Could not load issue from %s: %v", file, err)
			continue
		}
		issues = append(issues, *issue)
	}

	// Sort by filename (priority)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Metadata.CreatedAt > issues[j].Metadata.CreatedAt
	})

	return issues, nil
}

func (s *Server) findIssueFile(issueID string) (string, string, error) {
	folders := []string{"open", "investigating", "in-progress", "fixed", "closed", "failed"}

	for _, folder := range folders {
		folderPath := filepath.Join(s.config.IssuesDir, folder)
		files, _ := filepath.Glob(filepath.Join(folderPath, "*.yaml"))

		for _, file := range files {
			issue, err := s.loadIssueFromFile(file)
			if err != nil {
				continue
			}
			if issue.ID == issueID {
				return file, folder, nil
			}
		}
	}

	return "", "", fmt.Errorf("issue not found: %s", issueID)
}

func (s *Server) moveIssue(issueID, toFolder string) error {
	filePath, currentFolder, err := s.findIssueFile(issueID)
	if err != nil {
		return err
	}

	if currentFolder == toFolder {
		return nil // Already in target folder
	}

	// Load issue
	issue, err := s.loadIssueFromFile(filePath)
	if err != nil {
		return err
	}

	// Update status and timestamps
	issue.Status = toFolder
	now := time.Now().UTC().Format(time.RFC3339)
	issue.Metadata.UpdatedAt = now

	// Add state-specific timestamps
	switch toFolder {
	case "investigating":
		if issue.Investigation.StartedAt == "" {
			issue.Investigation.StartedAt = now
		}
	case "fixed":
		if issue.Metadata.ResolvedAt == "" {
			issue.Metadata.ResolvedAt = now
		}
	}

	// Save to new location
	err = s.saveIssueToFile(issue, toFolder)
	if err != nil {
		return err
	}

	// Remove from old location
	return os.Remove(filePath)
}

func (s *Server) getAllIssues(statusFilter, priorityFilter, typeFilter string, limit int) ([]Issue, error) {
	var allIssues []Issue

	folders := []string{"open", "investigating", "in-progress", "fixed", "closed", "failed"}
	if statusFilter != "" {
		folders = []string{statusFilter}
	}

	for _, folder := range folders {
		folderIssues, err := s.loadIssuesFromFolder(folder)
		if err != nil {
			log.Printf("Warning: Could not load issues from %s: %v", folder, err)
			continue
		}
		allIssues = append(allIssues, folderIssues...)
	}

	// Apply filters
	var filteredIssues []Issue
	for _, issue := range allIssues {
		if priorityFilter != "" && issue.Priority != priorityFilter {
			continue
		}
		if typeFilter != "" && issue.Type != typeFilter {
			continue
		}
		filteredIssues = append(filteredIssues, issue)
	}

	// Sort by creation date (newest first)
	sort.Slice(filteredIssues, func(i, j int) bool {
		return filteredIssues[i].Metadata.CreatedAt > filteredIssues[j].Metadata.CreatedAt
	})

	// Apply limit
	if limit > 0 && len(filteredIssues) > limit {
		filteredIssues = filteredIssues[:limit]
	}

	return filteredIssues, nil
}

// API Handlers

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := ApiResponse{
		Success: true,
		Message: "App Issue Tracker API is healthy",
		Data: map[string]interface{}{
			"timestamp":  time.Now().UTC(),
			"version":    "2.0.0-file-based",
			"storage":    "file-based-yaml",
			"issues_dir": s.config.IssuesDir,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getIssuesHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	issueType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}

	issues, err := s.getAllIssues(status, priority, issueType, limit)
	if err != nil {
		log.Printf("Error getting issues: %v", err)
		http.Error(w, "Failed to load issues", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"issues": issues,
			"count":  len(issues),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) createIssueHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title         string            `json:"title"`
		Description   string            `json:"description"`
		Type          string            `json:"type"`
		Priority      string            `json:"priority"`
		AppID         string            `json:"app_id"`
		ErrorMessage  string            `json:"error_message"`
		StackTrace    string            `json:"stack_trace"`
		Tags          []string          `json:"tags"`
		ReporterName  string            `json:"reporter_name"`
		ReporterEmail string            `json:"reporter_email"`
		Environment   map[string]string `json:"environment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	// Default values
	if req.Type == "" {
		req.Type = "bug"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.AppID == "" {
		req.AppID = "unknown"
	}
	if req.Description == "" {
		req.Description = req.Title
	}

	// Create issue
	issue := Issue{
		ID:          fmt.Sprintf("issue-%s", uuid.New().String()[:8]),
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Priority:    req.Priority,
		AppID:       req.AppID,
		Status:      "open",
	}

	// Set reporter info
	issue.Reporter.Name = req.ReporterName
	issue.Reporter.Email = req.ReporterEmail
	issue.Reporter.Timestamp = time.Now().UTC().Format(time.RFC3339)

	// Set error context
	issue.ErrorContext.ErrorMessage = req.ErrorMessage
	issue.ErrorContext.StackTrace = req.StackTrace
	issue.ErrorContext.EnvironmentInfo = req.Environment

	// Set metadata
	now := time.Now().UTC().Format(time.RFC3339)
	issue.Metadata.CreatedAt = now
	issue.Metadata.UpdatedAt = now
	issue.Metadata.Tags = req.Tags

	// Save to file
	err := s.saveIssueToFile(&issue, "open")
	if err != nil {
		log.Printf("Error saving issue: %v", err)
		http.Error(w, "Failed to create issue", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Message: "Issue created successfully",
		Data: map[string]interface{}{
			"issue_id": issue.ID,
			"filename": fmt.Sprintf("%03d-%s.yaml", s.getPriorityNumber(issue.Priority),
				strings.ReplaceAll(strings.ToLower(issue.Title), " ", "-")),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) searchIssuesHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}

	var results []Issue
	queryLower := strings.ToLower(query)

	// Search through all issues in all folders
	folders := []string{"open", "investigating", "in-progress", "fixed", "closed", "failed"}
	for _, folder := range folders {
		issues, err := s.loadIssuesFromFolder(folder)
		if err != nil {
			continue
		}

		for _, issue := range issues {
			// Simple text search
			searchText := strings.ToLower(fmt.Sprintf("%s %s %s %s",
				issue.Title, issue.Description,
				issue.ErrorContext.ErrorMessage,
				strings.Join(issue.Metadata.Tags, " ")))

			if strings.Contains(searchText, queryLower) {
				results = append(results, issue)
			}
		}
	}

	// Sort by relevance (title matches first)
	sort.Slice(results, func(i, j int) bool {
		iTitleMatch := strings.Contains(strings.ToLower(results[i].Title), queryLower)
		jTitleMatch := strings.Contains(strings.ToLower(results[j].Title), queryLower)

		if iTitleMatch != jTitleMatch {
			return iTitleMatch
		}

		return results[i].Metadata.CreatedAt > results[j].Metadata.CreatedAt
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"results": results,
			"count":   len(results),
			"query":   query,
			"method":  "text_search",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) triggerInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID  string `json:"issue_id"`
		AgentID  string `json:"agent_id"`
		Priority string `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.IssueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	// Find issue file
	filePath, currentFolder, err := s.findIssueFile(req.IssueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	// Load issue
	issue, err := s.loadIssueFromFile(filePath)
	if err != nil {
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	// Update investigation details
	now := time.Now().UTC().Format(time.RFC3339)
	issue.Investigation.AgentID = req.AgentID
	issue.Investigation.StartedAt = now
	issue.Metadata.UpdatedAt = now

	// Move to investigating folder if not already there
	if currentFolder != "investigating" {
		err = s.moveIssue(req.IssueID, "investigating")
		if err != nil {
			log.Printf("Error moving issue: %v", err)
			http.Error(w, "Failed to update issue status", http.StatusInternalServerError)
			return
		}
	} else {
		// Just update in place
		err = s.saveIssueToFile(issue, "investigating")
		if err != nil {
			http.Error(w, "Failed to update issue", http.StatusInternalServerError)
			return
		}
	}

	// Trigger direct investigation via script
	runID := fmt.Sprintf("run_%d", time.Now().Unix())
	investigationID := fmt.Sprintf("inv_%d", time.Now().Unix())

	// Execute investigation script in background
	go func() {
		// Prepare command with proper arguments
		scriptPath := filepath.Join(filepath.Dir(s.config.IssuesDir), "scripts", "claude-investigator.sh")
		projectPath := filepath.Dir(s.config.IssuesDir)

		// Default agent if not specified
		agentID := req.AgentID
		if agentID == "" {
			agentID = "deep-investigator"
		}

		// Create a prompt template based on issue details
		promptTemplate := fmt.Sprintf("Investigate issue: %s", issue.Title)
		if issue.ErrorContext.ErrorMessage != "" {
			promptTemplate += fmt.Sprintf(". Error: %s", issue.ErrorContext.ErrorMessage)
		}

		cmd := exec.Command("bash", scriptPath, "investigate", req.IssueID, agentID, projectPath, promptTemplate)
		cmd.Dir = filepath.Dir(s.config.IssuesDir)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Investigation script failed: %v\nOutput: %s", err, output)
			// Try to update issue status to failed
			s.moveIssue(req.IssueID, "failed")
			return
		}

		log.Printf("Investigation completed for issue %s", req.IssueID)

		// Parse the JSON output from the investigation script
		var result struct {
			IssueID             string   `json:"issue_id"`
			InvestigationReport string   `json:"investigation_report"`
			RootCause           string   `json:"root_cause"`
			SuggestedFix        string   `json:"suggested_fix"`
			ConfidenceScore     int      `json:"confidence_score"`
			AffectedFiles       []string `json:"affected_files"`
			Status              string   `json:"status"`
		}

		if err := json.Unmarshal(output, &result); err != nil {
			log.Printf("Failed to parse investigation result: %v", err)
			return
		}

		// Update the issue with investigation results
		filePath, _, err := s.findIssueFile(req.IssueID)
		if err != nil {
			log.Printf("Failed to find issue file: %v", err)
			return
		}

		issue, err := s.loadIssueFromFile(filePath)
		if err != nil {
			log.Printf("Failed to load issue: %v", err)
			return
		}

		// Update investigation fields
		completedAt := time.Now().UTC().Format(time.RFC3339)
		issue.Investigation.CompletedAt = completedAt
		issue.Investigation.Report = result.InvestigationReport
		issue.Investigation.RootCause = result.RootCause
		issue.Investigation.SuggestedFix = result.SuggestedFix
		confidenceScore := result.ConfidenceScore
		issue.Investigation.ConfidenceScore = &confidenceScore

		// Calculate duration
		if issue.Investigation.StartedAt != "" {
			if startTime, err := time.Parse(time.RFC3339, issue.Investigation.StartedAt); err == nil {
				duration := int(time.Since(startTime).Minutes())
				issue.Investigation.InvestigationDurationMinutes = &duration
			}
		}

		// Update affected files in error context
		if len(result.AffectedFiles) > 0 {
			issue.ErrorContext.AffectedFiles = result.AffectedFiles
		}

		// Save the updated issue
		if err := s.saveIssueToFile(issue, "investigating"); err != nil {
			log.Printf("Failed to save investigation results: %v", err)
		} else {
			log.Printf("Investigation results saved for issue %s", req.IssueID)
		}
	}()

	response := ApiResponse{
		Success: true,
		Message: "Investigation triggered successfully",
		Data: map[string]interface{}{
			"run_id":           runID,
			"investigation_id": investigationID,
			"issue_id":         req.IssueID,
			"agent_id":         req.AgentID,
			"status":           "queued",
			"workflow":         "file-based",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Count issues by status
	var totalIssues, openIssues, inProgress, fixedToday int

	allIssues, _ := s.getAllIssues("", "", "", 0)
	totalIssues = len(allIssues)

	today := time.Now().UTC().Format("2006-01-02")

	for _, issue := range allIssues {
		switch issue.Status {
		case "open", "investigating":
			openIssues++
		case "in-progress":
			inProgress++
		case "fixed":
			if strings.HasPrefix(issue.Metadata.ResolvedAt, today) {
				fixedToday++
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
				"fixed_today":          fixedToday,
				"avg_resolution_hours": 24.5, // TODO: Calculate from resolved issues
				"top_apps":             topApps,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getAgentsHandler(w http.ResponseWriter, r *http.Request) {
	// Mock agents for now - in file-based system, these could also be YAML files
	agents := []Agent{
		{
			ID:             "deep-investigator",
			Name:           "deep-investigator",
			DisplayName:    "Deep Code Investigator",
			Description:    "Performs thorough investigation of issues",
			Capabilities:   []string{"investigate", "analyze", "debug"},
			IsActive:       true,
			SuccessRate:    85.5,
			TotalRuns:      47,
			SuccessfulRuns: 40,
		},
		{
			ID:             "auto-fixer",
			Name:           "auto-fixer",
			DisplayName:    "Automated Fix Generator",
			Description:    "Generates and validates fixes for identified issues",
			Capabilities:   []string{"fix", "test", "validate"},
			IsActive:       true,
			SuccessRate:    72.3,
			TotalRuns:      22,
			SuccessfulRuns: 16,
		},
		{
			ID:             "quick-analyzer",
			Name:           "quick-analyzer",
			DisplayName:    "Quick Triage Analyzer",
			Description:    "Performs rapid initial assessment",
			Capabilities:   []string{"triage", "categorize"},
			IsActive:       true,
			SuccessRate:    90.1,
			TotalRuns:      128,
			SuccessfulRuns: 115,
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

func (s *Server) triggerFixGenerationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID       string `json:"issue_id"`
		AutoApply     bool   `json:"auto_apply"`
		BackupEnabled bool   `json:"backup_enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.IssueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	// Find issue file
	filePath, currentFolder, err := s.findIssueFile(req.IssueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	// Load issue to check if investigation is complete
	issue, err := s.loadIssueFromFile(filePath)
	if err != nil {
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	// Check if investigation exists
	if issue.Investigation.Report == "" {
		http.Error(w, "Issue must be investigated before generating fixes", http.StatusBadRequest)
		return
	}

	// Generate fix in background
	runID := fmt.Sprintf("fix_run_%d", time.Now().Unix())
	fixID := fmt.Sprintf("fix_%d", time.Now().Unix())

	go func() {
		// Prepare command
		scriptPath := filepath.Join(filepath.Dir(s.config.IssuesDir), "scripts", "claude-fix-generator.sh")
		projectPath := filepath.Dir(s.config.IssuesDir)

		// Set POSTGRES_PASSWORD if not set (use a default for file-based mode)
		if os.Getenv("POSTGRES_PASSWORD") == "" {
			os.Setenv("POSTGRES_PASSWORD", "unused-in-file-mode")
		}

		autoApplyStr := "false"
		if req.AutoApply {
			autoApplyStr = "true"
		}

		backupStr := "true"
		if !req.BackupEnabled {
			backupStr = "false"
		}

		cmd := exec.Command("bash", scriptPath, "generate", req.IssueID, projectPath, autoApplyStr, backupStr)
		cmd.Dir = filepath.Dir(s.config.IssuesDir)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Fix generation script failed: %v\nOutput: %s", err, output)
			return
		}

		log.Printf("Fix generation completed for issue %s", req.IssueID)

		// Parse the JSON output
		var result struct {
			IssueID             string `json:"issue_id"`
			FixGenerationStatus string `json:"fix_generation_status"`
			FixReport           string `json:"fix_report"`
			FixSummary          string `json:"fix_summary"`
			ImplementationPlan  string `json:"implementation_steps"`
			RollbackPlan        string `json:"rollback_plan"`
			RiskLevel           string `json:"risk_level"`
			AutoApplyResult     string `json:"auto_apply_result"`
		}

		if err := json.Unmarshal(output, &result); err != nil {
			log.Printf("Failed to parse fix generation result: %v", err)
			return
		}

		// Update issue with fix information
		issue, err := s.loadIssueFromFile(filePath)
		if err != nil {
			log.Printf("Failed to reload issue: %v", err)
			return
		}

		// Update fix fields
		issue.Fix.SuggestedFix = result.FixSummary
		issue.Fix.ImplementationPlan = result.ImplementationPlan
		if req.AutoApply && result.AutoApplyResult == "success" {
			issue.Fix.Applied = true
			issue.Fix.AppliedAt = time.Now().UTC().Format(time.RFC3339)
		}

		// Move to in-progress if not already there
		if currentFolder != "in-progress" && currentFolder != "fixed" {
			s.moveIssue(req.IssueID, "in-progress")
		} else {
			s.saveIssueToFile(issue, currentFolder)
		}

		log.Printf("Fix information saved for issue %s", req.IssueID)
	}()

	response := ApiResponse{
		Success: true,
		Message: "Fix generation triggered successfully",
		Data: map[string]interface{}{
			"run_id":         runID,
			"fix_id":         fixID,
			"issue_id":       req.IssueID,
			"auto_apply":     req.AutoApply,
			"backup_enabled": req.BackupEnabled,
			"status":         "queued",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getAppsHandler(w http.ResponseWriter, r *http.Request) {
	// Count issues per app
	allIssues, _ := s.getAllIssues("", "", "", 0)
	appStats := make(map[string]struct {
		total int
		open  int
	})

	for _, issue := range allIssues {
		stats := appStats[issue.AppID]
		stats.total++
		if issue.Status == "open" || issue.Status == "investigating" || issue.Status == "in-progress" {
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	ex, _ := os.Executable()
	return filepath.Dir(filepath.Dir(ex))
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start app-issue-tracker

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	config := loadConfig()

	// Ensure issues directory structure exists
	folders := []string{"open", "investigating", "in-progress", "fixed", "closed", "failed", "templates"}
	for _, folder := range folders {
		folderPath := filepath.Join(config.IssuesDir, folder)
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			log.Fatalf("Failed to create folder %s: %v", folder, err)
		}
	}

	log.Printf("Using file-based storage at: %s", config.IssuesDir)

	server := &Server{config: config}

	// Setup routes
	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", server.healthHandler).Methods("GET")

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/issues", server.getIssuesHandler).Methods("GET")
	api.HandleFunc("/issues", server.createIssueHandler).Methods("POST")
	api.HandleFunc("/issues/search", server.searchIssuesHandler).Methods("GET")
	// api.HandleFunc("/issues/search/similar", server.vectorSearchHandler).Methods("POST") // TODO: Implement vector search
	api.HandleFunc("/agents", server.getAgentsHandler).Methods("GET")
	api.HandleFunc("/apps", server.getAppsHandler).Methods("GET")
	api.HandleFunc("/investigate", server.triggerInvestigationHandler).Methods("POST")
	api.HandleFunc("/generate-fix", server.triggerFixGenerationHandler).Methods("POST")
	api.HandleFunc("/stats", server.getStatsHandler).Methods("GET")

	// Apply CORS middleware
	handler := corsMiddleware(r)

	log.Printf("Starting File-Based App Issue Tracker API on port %s", config.Port)
	log.Printf("Health check: http://localhost:%s/health", config.Port)
	log.Printf("API base URL: http://localhost:%s/api", config.Port)
	log.Printf("Issues directory: %s", config.IssuesDir)

	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
