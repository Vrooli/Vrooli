package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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

const (
	metadataFilename = "metadata.yaml"
	artifactsDirName = "artifacts"
)

var validIssueStatuses = map[string]struct{}{
	"open":          {},
	"investigating": {},
	"in-progress":   {},
	"fixed":         {},
	"closed":        {},
	"failed":        {},
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func normalizeStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	var result []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for k, v := range values {
		clone[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(clone) == 0 {
		return nil
	}
	return clone
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

	Attachments []Attachment `yaml:"attachments,omitempty" json:"attachments,omitempty"`

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

type Attachment struct {
	Name string `yaml:"name" json:"name"`
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
	Path string `yaml:"path" json:"path"`
	Size int64  `yaml:"size,omitempty" json:"size,omitempty"`
}

type UpdateIssueRequest struct {
	Title       *string            `json:"title"`
	Description *string            `json:"description"`
	Type        *string            `json:"type"`
	Priority    *string            `json:"priority"`
	AppID       *string            `json:"app_id"`
	Status      *string            `json:"status"`
	Tags        *[]string          `json:"tags"`
	Labels      *map[string]string `json:"labels"`
	Watchers    *[]string          `json:"watchers"`
	Notes       *string            `json:"notes"`
	ResolvedAt  *string            `json:"resolved_at"`

	Reporter *struct {
		Name      *string `json:"name"`
		Email     *string `json:"email"`
		UserID    *string `json:"user_id"`
		Timestamp *string `json:"timestamp"`
	} `json:"reporter"`

	ErrorContext *struct {
		ErrorMessage       *string            `json:"error_message"`
		ErrorLogs          *string            `json:"error_logs"`
		StackTrace         *string            `json:"stack_trace"`
		AffectedFiles      *[]string          `json:"affected_files"`
		AffectedComponents *[]string          `json:"affected_components"`
		EnvironmentInfo    *map[string]string `json:"environment_info"`
	} `json:"error_context"`

	Investigation *struct {
		AgentID                      *string  `json:"agent_id"`
		StartedAt                    *string  `json:"started_at"`
		CompletedAt                  *string  `json:"completed_at"`
		Report                       *string  `json:"report"`
		RootCause                    *string  `json:"root_cause"`
		SuggestedFix                 *string  `json:"suggested_fix"`
		ConfidenceScore              *int     `json:"confidence_score"`
		InvestigationDurationMinutes *int     `json:"investigation_duration_minutes"`
		TokensUsed                   *int     `json:"tokens_used"`
		CostEstimate                 *float64 `json:"cost_estimate"`
	} `json:"investigation"`

	Fix *struct {
		SuggestedFix       *string `json:"suggested_fix"`
		ImplementationPlan *string `json:"implementation_plan"`
		Applied            *bool   `json:"applied"`
		AppliedAt          *string `json:"applied_at"`
		CommitHash         *string `json:"commit_hash"`
		PrURL              *string `json:"pr_url"`
		VerificationStatus *string `json:"verification_status"`
		RollbackPlan       *string `json:"rollback_plan"`
		FixDurationMinutes *int    `json:"fix_duration_minutes"`
	} `json:"fix"`
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
	defaultIssuesDir := filepath.Join(getVrooliRoot(), "scenarios/app-issue-tracker/data/issues")
	if _, err := os.Stat("./data/issues"); err == nil {
		// If local issues directory exists, use it
		defaultIssuesDir = "./data/issues"
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
func (s *Server) issueDir(folder, issueID string) string {
	return filepath.Join(s.config.IssuesDir, folder, issueID)
}

func (s *Server) metadataPath(issueDir string) string {
	return filepath.Join(issueDir, metadataFilename)
}

func (s *Server) loadIssueFromDir(issueDir string) (*Issue, error) {
	metadata := s.metadataPath(issueDir)
	data, err := ioutil.ReadFile(metadata)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := yaml.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %v", err)
	}

	if issue.ID == "" {
		issue.ID = filepath.Base(issueDir)
	}

	return &issue, nil
}

func (s *Server) writeIssueMetadata(issueDir string, issue *Issue) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if issue.Metadata.CreatedAt == "" {
		issue.Metadata.CreatedAt = now
	}
	issue.Metadata.UpdatedAt = now

	data, err := yaml.Marshal(issue)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %v", err)
	}

	return os.WriteFile(s.metadataPath(issueDir), data, 0644)
}

func (s *Server) saveIssue(issue *Issue, folder string) (string, error) {
	issue.Status = folder
	issueDir := s.issueDir(folder, issue.ID)
	if err := os.MkdirAll(issueDir, 0755); err != nil {
		return "", err
	}

	if err := s.writeIssueMetadata(issueDir, issue); err != nil {
		return "", err
	}

	return issueDir, nil
}

func (s *Server) loadIssuesFromFolder(folder string) ([]Issue, error) {
	folderPath := filepath.Join(s.config.IssuesDir, folder)
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Issue{}, nil
		}
		return nil, err
	}

	var issues []Issue
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		issueDir := filepath.Join(folderPath, entry.Name())
		issue, err := s.loadIssueFromDir(issueDir)
		if err != nil {
			log.Printf("Warning: could not load issue from %s: %v", issueDir, err)
			continue
		}
		issue.Status = folder
		issues = append(issues, *issue)
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Metadata.CreatedAt > issues[j].Metadata.CreatedAt
	})

	return issues, nil
}

func (s *Server) findIssueDirectory(issueID string) (string, string, error) {
	folders := []string{"open", "investigating", "in-progress", "fixed", "closed", "failed"}

	for _, folder := range folders {
		directDir := s.issueDir(folder, issueID)
		if _, err := os.Stat(s.metadataPath(directDir)); err == nil {
			return directDir, folder, nil
		}
	}

	for _, folder := range folders {
		folderPath := filepath.Join(s.config.IssuesDir, folder)
		entries, err := os.ReadDir(folderPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", "", err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			issueDir := filepath.Join(folderPath, entry.Name())
			issue, err := s.loadIssueFromDir(issueDir)
			if err != nil {
				continue
			}
			if issue.ID == issueID {
				return issueDir, folder, nil
			}
		}
	}

	return "", "", fmt.Errorf("issue not found: %s", issueID)
}

func (s *Server) moveIssue(issueID, toFolder string) error {
	issueDir, currentFolder, err := s.findIssueDirectory(issueID)
	if err != nil {
		return err
	}

	if currentFolder == toFolder {
		return nil
	}

	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	issue.Status = toFolder
	issue.Metadata.UpdatedAt = now

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

	targetDir := s.issueDir(toFolder, issue.ID)
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return err
	}

	if err := os.Rename(issueDir, targetDir); err != nil {
		return err
	}

	return s.writeIssueMetadata(targetDir, issue)
}

func decodeBase64Payload(data string) ([]byte, error) {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return nil, fmt.Errorf("empty payload")
	}
	if idx := strings.Index(trimmed, ","); idx != -1 && strings.Contains(trimmed[:idx], "base64") {
		trimmed = trimmed[idx+1:]
	}
	trimmed = strings.ReplaceAll(trimmed, "\n", "")
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	return base64.StdEncoding.DecodeString(trimmed)
}

func extensionFromContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/png":
		return "png"
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "application/json":
		return "json"
	case "text/plain":
		return "txt"
	default:
		return ""
	}
}

func looksLikeJSON(payload string) bool {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return false
	}
	first := trimmed[0]
	last := trimmed[len(trimmed)-1]
	return (first == '{' && last == '}') || (first == '[' && last == ']')
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
		Title                 string            `json:"title"`
		Description           string            `json:"description"`
		Type                  string            `json:"type"`
		Priority              string            `json:"priority"`
		AppID                 string            `json:"app_id"`
		ErrorMessage          string            `json:"error_message"`
		StackTrace            string            `json:"stack_trace"`
		Tags                  []string          `json:"tags"`
		ReporterName          string            `json:"reporter_name"`
		ReporterEmail         string            `json:"reporter_email"`
		Environment           map[string]string `json:"environment"`
		AppLogs               string            `json:"app_logs"`
		ConsoleLogs           string            `json:"console_logs"`
		NetworkLogs           string            `json:"network_logs"`
		ScreenshotData        string            `json:"screenshot_data"`
		ScreenshotContentType string            `json:"screenshot_content_type"`
		ScreenshotFilename    string            `json:"screenshot_filename"`
		Attachments           []struct {
			Name        string `json:"name"`
			Content     string `json:"content"`
			Encoding    string `json:"encoding"`
			ContentType string `json:"content_type"`
		} `json:"attachments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

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

	issue := Issue{
		ID:          fmt.Sprintf("issue-%s", uuid.New().String()[:8]),
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Priority:    req.Priority,
		AppID:       req.AppID,
		Status:      "open",
	}

	issue.Reporter.Name = req.ReporterName
	issue.Reporter.Email = req.ReporterEmail
	issue.Reporter.Timestamp = time.Now().UTC().Format(time.RFC3339)

	issue.ErrorContext.ErrorMessage = req.ErrorMessage
	issue.ErrorContext.StackTrace = req.StackTrace
	issue.ErrorContext.EnvironmentInfo = req.Environment

	issue.Metadata.Tags = req.Tags

	issueDir := s.issueDir("open", issue.ID)
	if err := os.MkdirAll(issueDir, 0755); err != nil {
		log.Printf("Error preparing issue directory: %v", err)
		http.Error(w, "Failed to prepare storage", http.StatusInternalServerError)
		return
	}

	var attachments []Attachment
	var artifactsDir string
	ensureArtifactsDir := func() (string, error) {
		if artifactsDir != "" {
			return artifactsDir, nil
		}
		dir := filepath.Join(issueDir, artifactsDirName)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
		artifactsDir = dir
		return dir, nil
	}

	addTextArtifact := func(label, fileName, content, contentType string) error {
		if strings.TrimSpace(content) == "" {
			return nil
		}
		dir, err := ensureArtifactsDir()
		if err != nil {
			return err
		}
		data := []byte(content)
		relPath := filepath.Join(artifactsDirName, fileName)
		if err := os.WriteFile(filepath.Join(dir, fileName), data, 0644); err != nil {
			return err
		}
		attachments = append(attachments, Attachment{
			Name: label,
			Type: contentType,
			Path: relPath,
			Size: int64(len(data)),
		})
		return nil
	}

	if err := addTextArtifact("Application Logs", "app-logs.txt", req.AppLogs, "text/plain"); err != nil {
		log.Printf("Error saving app logs: %v", err)
		http.Error(w, "Failed to store app logs", http.StatusInternalServerError)
		return
	}
	if err := addTextArtifact("Console Logs", "console-logs.txt", req.ConsoleLogs, "text/plain"); err != nil {
		log.Printf("Error saving console logs: %v", err)
		http.Error(w, "Failed to store console logs", http.StatusInternalServerError)
		return
	}
	networkContentType := "application/json"
	networkFileName := "network-requests.json"
	if !looksLikeJSON(req.NetworkLogs) {
		networkContentType = "text/plain"
		networkFileName = "network-requests.txt"
	}
	if err := addTextArtifact("Network Requests", networkFileName, req.NetworkLogs, networkContentType); err != nil {
		log.Printf("Error saving network requests: %v", err)
		http.Error(w, "Failed to store network requests", http.StatusInternalServerError)
		return
	}

	if strings.TrimSpace(req.ScreenshotData) != "" {
		data, err := decodeBase64Payload(req.ScreenshotData)
		if err != nil {
			log.Printf("Error decoding screenshot: %v", err)
			http.Error(w, "Invalid screenshot data", http.StatusBadRequest)
			return
		}
		dir, err := ensureArtifactsDir()
		if err != nil {
			log.Printf("Error preparing artifacts directory: %v", err)
			http.Error(w, "Failed to store screenshot", http.StatusInternalServerError)
			return
		}
		filename := strings.TrimSpace(req.ScreenshotFilename)
		if filename == "" {
			ext := extensionFromContentType(req.ScreenshotContentType)
			if ext == "" {
				ext = "png"
			}
			filename = fmt.Sprintf("screenshot.%s", ext)
		}
		filename = filepath.Base(filename)
		relPath := filepath.Join(artifactsDirName, filename)
		if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
			log.Printf("Error writing screenshot: %v", err)
			http.Error(w, "Failed to store screenshot", http.StatusInternalServerError)
			return
		}
		attachments = append(attachments, Attachment{
			Name: "Screenshot",
			Type: req.ScreenshotContentType,
			Path: relPath,
			Size: int64(len(data)),
		})
	}

	for _, payload := range req.Attachments {
		content := strings.TrimSpace(payload.Content)
		name := strings.TrimSpace(payload.Name)
		if content == "" || name == "" {
			continue
		}
		var data []byte
		var err error
		switch strings.ToLower(payload.Encoding) {
		case "", "plain", "text":
			data = []byte(content)
		case "base64":
			data, err = decodeBase64Payload(content)
		default:
			err = fmt.Errorf("unsupported encoding: %s", payload.Encoding)
		}
		if err != nil {
			log.Printf("Error decoding attachment %s: %v", payload.Name, err)
			http.Error(w, "Failed to decode attachment", http.StatusBadRequest)
			return
		}
		dir, err := ensureArtifactsDir()
		if err != nil {
			log.Printf("Error preparing artifacts directory: %v", err)
			http.Error(w, "Failed to store attachment", http.StatusInternalServerError)
			return
		}
		filename := filepath.Base(name)
		if filepath.Ext(filename) == "" {
			ext := extensionFromContentType(payload.ContentType)
			if ext != "" {
				filename = fmt.Sprintf("%s.%s", filename, ext)
			}
		}
		relPath := filepath.Join(artifactsDirName, filename)
		if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
			log.Printf("Error writing attachment %s: %v", filename, err)
			http.Error(w, "Failed to store attachment", http.StatusInternalServerError)
			return
		}
		attachments = append(attachments, Attachment{
			Name: filename,
			Type: payload.ContentType,
			Path: relPath,
			Size: int64(len(data)),
		})
	}

	if len(attachments) > 0 {
		issue.Attachments = attachments
	}

	if _, err := s.saveIssue(&issue, "open"); err != nil {
		log.Printf("Error saving issue: %v", err)
		http.Error(w, "Failed to create issue", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Message: "Issue created successfully",
		Data: map[string]interface{}{
			"issue_id":     issue.ID,
			"storage_path": filepath.Join("open", issue.ID),
			"attachments":  issue.Attachments,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	issueDir, statusFolder, err := s.findIssueDirectory(issueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		log.Printf("Error loading issue %s: %v", issueID, err)
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	issue.Status = statusFolder

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"issue": issue,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) updateIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	issueDir, currentFolder, err := s.findIssueDirectory(issueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		log.Printf("Error loading issue %s: %v", issueID, err)
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	issue.Status = currentFolder

	var req UpdateIssueRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		log.Printf("Invalid update payload for issue %s: %v", issueID, err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	targetStatus := currentFolder
	if req.Status != nil {
		normalized := strings.ToLower(strings.TrimSpace(*req.Status))
		if normalized == "" {
			http.Error(w, "Status cannot be empty", http.StatusBadRequest)
			return
		}
		if _, ok := validIssueStatuses[normalized]; !ok {
			http.Error(w, fmt.Sprintf("Invalid status: %s", normalized), http.StatusBadRequest)
			return
		}
		targetStatus = normalized
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			http.Error(w, "Title cannot be empty", http.StatusBadRequest)
			return
		}
		issue.Title = title
	}
	if req.Description != nil {
		issue.Description = strings.TrimSpace(*req.Description)
	}
	if req.Type != nil {
		issue.Type = strings.TrimSpace(*req.Type)
	}
	if req.Priority != nil {
		priority := strings.TrimSpace(*req.Priority)
		if priority != "" {
			issue.Priority = priority
		}
	}
	if req.AppID != nil {
		appID := strings.TrimSpace(*req.AppID)
		if appID != "" {
			issue.AppID = appID
		}
	}
	if req.Tags != nil {
		if *req.Tags == nil {
			issue.Metadata.Tags = nil
		} else {
			issue.Metadata.Tags = normalizeStringSlice(*req.Tags)
		}
	}
	if req.Labels != nil {
		if *req.Labels == nil {
			issue.Metadata.Labels = nil
		} else {
			issue.Metadata.Labels = cloneStringMap(*req.Labels)
		}
	}
	if req.Watchers != nil {
		if *req.Watchers == nil {
			issue.Metadata.Watchers = nil
		} else {
			issue.Metadata.Watchers = normalizeStringSlice(*req.Watchers)
		}
	}
	if req.Notes != nil {
		issue.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.ResolvedAt != nil {
		issue.Metadata.ResolvedAt = strings.TrimSpace(*req.ResolvedAt)
	}

	if req.Reporter != nil {
		if req.Reporter.Name != nil {
			issue.Reporter.Name = strings.TrimSpace(*req.Reporter.Name)
		}
		if req.Reporter.Email != nil {
			issue.Reporter.Email = strings.TrimSpace(*req.Reporter.Email)
		}
		if req.Reporter.UserID != nil {
			issue.Reporter.UserID = strings.TrimSpace(*req.Reporter.UserID)
		}
		if req.Reporter.Timestamp != nil {
			issue.Reporter.Timestamp = strings.TrimSpace(*req.Reporter.Timestamp)
		}
	}

	if req.ErrorContext != nil {
		if req.ErrorContext.ErrorMessage != nil {
			issue.ErrorContext.ErrorMessage = strings.TrimSpace(*req.ErrorContext.ErrorMessage)
		}
		if req.ErrorContext.ErrorLogs != nil {
			issue.ErrorContext.ErrorLogs = *req.ErrorContext.ErrorLogs
		}
		if req.ErrorContext.StackTrace != nil {
			issue.ErrorContext.StackTrace = *req.ErrorContext.StackTrace
		}
		if req.ErrorContext.AffectedFiles != nil {
			if *req.ErrorContext.AffectedFiles == nil {
				issue.ErrorContext.AffectedFiles = nil
			} else {
				issue.ErrorContext.AffectedFiles = normalizeStringSlice(*req.ErrorContext.AffectedFiles)
			}
		}
		if req.ErrorContext.AffectedComponents != nil {
			if *req.ErrorContext.AffectedComponents == nil {
				issue.ErrorContext.AffectedComponents = nil
			} else {
				issue.ErrorContext.AffectedComponents = normalizeStringSlice(*req.ErrorContext.AffectedComponents)
			}
		}
		if req.ErrorContext.EnvironmentInfo != nil {
			if *req.ErrorContext.EnvironmentInfo == nil {
				issue.ErrorContext.EnvironmentInfo = nil
			} else {
				envCopy := make(map[string]string, len(*req.ErrorContext.EnvironmentInfo))
				for k, v := range *req.ErrorContext.EnvironmentInfo {
					envCopy[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
				if len(envCopy) == 0 {
					issue.ErrorContext.EnvironmentInfo = nil
				} else {
					issue.ErrorContext.EnvironmentInfo = envCopy
				}
			}
		}
	}

	if req.Investigation != nil {
		if req.Investigation.AgentID != nil {
			issue.Investigation.AgentID = strings.TrimSpace(*req.Investigation.AgentID)
		}
		if req.Investigation.StartedAt != nil {
			issue.Investigation.StartedAt = strings.TrimSpace(*req.Investigation.StartedAt)
		}
		if req.Investigation.CompletedAt != nil {
			issue.Investigation.CompletedAt = strings.TrimSpace(*req.Investigation.CompletedAt)
		}
		if req.Investigation.Report != nil {
			issue.Investigation.Report = strings.TrimSpace(*req.Investigation.Report)
		}
		if req.Investigation.RootCause != nil {
			issue.Investigation.RootCause = strings.TrimSpace(*req.Investigation.RootCause)
		}
		if req.Investigation.SuggestedFix != nil {
			issue.Investigation.SuggestedFix = strings.TrimSpace(*req.Investigation.SuggestedFix)
		}
		if req.Investigation.ConfidenceScore != nil {
			issue.Investigation.ConfidenceScore = req.Investigation.ConfidenceScore
		}
		if req.Investigation.InvestigationDurationMinutes != nil {
			issue.Investigation.InvestigationDurationMinutes = req.Investigation.InvestigationDurationMinutes
		}
		if req.Investigation.TokensUsed != nil {
			issue.Investigation.TokensUsed = req.Investigation.TokensUsed
		}
		if req.Investigation.CostEstimate != nil {
			issue.Investigation.CostEstimate = req.Investigation.CostEstimate
		}
	}

	if req.Fix != nil {
		if req.Fix.SuggestedFix != nil {
			issue.Fix.SuggestedFix = strings.TrimSpace(*req.Fix.SuggestedFix)
		}
		if req.Fix.ImplementationPlan != nil {
			issue.Fix.ImplementationPlan = strings.TrimSpace(*req.Fix.ImplementationPlan)
		}
		if req.Fix.Applied != nil {
			issue.Fix.Applied = *req.Fix.Applied
		}
		if req.Fix.AppliedAt != nil {
			issue.Fix.AppliedAt = strings.TrimSpace(*req.Fix.AppliedAt)
		}
		if req.Fix.CommitHash != nil {
			issue.Fix.CommitHash = strings.TrimSpace(*req.Fix.CommitHash)
		}
		if req.Fix.PrURL != nil {
			issue.Fix.PrURL = strings.TrimSpace(*req.Fix.PrURL)
		}
		if req.Fix.VerificationStatus != nil {
			issue.Fix.VerificationStatus = strings.TrimSpace(*req.Fix.VerificationStatus)
		}
		if req.Fix.RollbackPlan != nil {
			issue.Fix.RollbackPlan = strings.TrimSpace(*req.Fix.RollbackPlan)
		}
		if req.Fix.FixDurationMinutes != nil {
			issue.Fix.FixDurationMinutes = req.Fix.FixDurationMinutes
		}
	}

	if targetStatus != currentFolder {
		now := time.Now().UTC().Format(time.RFC3339)
		if targetStatus == "investigating" && strings.TrimSpace(issue.Investigation.StartedAt) == "" {
			issue.Investigation.StartedAt = now
		}
		if targetStatus == "fixed" && strings.TrimSpace(issue.Metadata.ResolvedAt) == "" {
			issue.Metadata.ResolvedAt = now
		}

		targetDir := s.issueDir(targetStatus, issue.ID)
		if _, statErr := os.Stat(targetDir); statErr == nil {
			http.Error(w, "Issue already exists in target status", http.StatusConflict)
			return
		} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			log.Printf("Failed to stat target directory for issue %s: %v", issueID, statErr)
			http.Error(w, "Failed to move issue", http.StatusInternalServerError)
			return
		}
		if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
			log.Printf("Failed to prepare target directory for issue %s: %v", issueID, err)
			http.Error(w, "Failed to move issue", http.StatusInternalServerError)
			return
		}
		if err := os.Rename(issueDir, targetDir); err != nil {
			log.Printf("Failed to move issue %s from %s to %s: %v", issueID, issueDir, targetDir, err)
			http.Error(w, "Failed to move issue", http.StatusInternalServerError)
			return
		}
		issueDir = targetDir
		currentFolder = targetStatus
	}

	issue.Status = currentFolder

	if err := s.writeIssueMetadata(issueDir, issue); err != nil {
		log.Printf("Failed to persist updated issue %s: %v", issueID, err)
		http.Error(w, "Failed to save issue", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Message: "Issue updated successfully",
		Data: map[string]interface{}{
			"issue": issue,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) deleteIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	issueDir, _, err := s.findIssueDirectory(issueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	if err := os.RemoveAll(issueDir); err != nil {
		log.Printf("Failed to delete issue %s: %v", issueID, err)
		http.Error(w, "Failed to delete issue", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Message: "Issue deleted successfully",
		Data: map[string]interface{}{
			"issue_id": issueID,
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

	// Locate issue directory
	issueDir, currentFolder, err := s.findIssueDirectory(req.IssueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	// Load issue metadata
	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	log.Printf("Fix generation requested for issue %s (status: %s)", req.IssueID, currentFolder)

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
		if err := s.writeIssueMetadata(issueDir, issue); err != nil {
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
		updatedDir, _, err := s.findIssueDirectory(req.IssueID)
		if err != nil {
			log.Printf("Failed to find issue file: %v", err)
			return
		}

		issue, err := s.loadIssueFromDir(updatedDir)
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
		if err := s.writeIssueMetadata(updatedDir, issue); err != nil {
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

	issueDir, currentFolder, err := s.findIssueDirectory(req.IssueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	log.Printf("Fix generation requested for issue %s (status: %s)", req.IssueID, currentFolder)

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

		issueDir, currentFolder, err := s.findIssueDirectory(req.IssueID)
		if err != nil {
			log.Printf("Failed to locate issue directory: %v", err)
			return
		}

		issue, err := s.loadIssueFromDir(issueDir)
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

		// Ensure issue sits in in-progress unless already fixed
		if currentFolder != "in-progress" && currentFolder != "fixed" {
			if err := s.moveIssue(req.IssueID, "in-progress"); err != nil {
				log.Printf("Failed to move issue %s to in-progress: %v", req.IssueID, err)
				return
			}
			issueDir, currentFolder, err = s.findIssueDirectory(req.IssueID)
			if err != nil {
				log.Printf("Failed to refresh issue directory after move: %v", err)
				return
			}
		}

		if err := s.writeIssueMetadata(issueDir, issue); err != nil {
			log.Printf("Failed to persist fix details: %v", err)
			return
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
	api.HandleFunc("/issues/{id}", server.getIssueHandler).Methods("GET")
	api.HandleFunc("/issues/{id}", server.updateIssueHandler).Methods("PUT", "PATCH")
	api.HandleFunc("/issues/{id}", server.deleteIssueHandler).Methods("DELETE")
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
