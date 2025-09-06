package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Core data structures
type ErrorReport struct {
	ID           int                    `json:"id,omitempty"`
	AppName      string                 `json:"app_name"`
	ErrorMessage string                 `json:"error_message"`
	StackTrace   string                 `json:"stack_trace"`
	Context      map[string]interface{} `json:"context"`
	Severity     string                 `json:"severity,omitempty"`
	Timestamp    time.Time              `json:"timestamp,omitempty"`
}

type FixSuggestion struct {
	ID          int      `json:"id"`
	ErrorID     int      `json:"error_id"`
	Suggestions []string `json:"suggestions"`
	Confidence  float64  `json:"confidence"`
	Priority    string   `json:"priority"`
	EstimatedTime string `json:"estimated_time"`
}

type PerformanceMetrics struct {
	AppName       string                 `json:"app_name"`
	CPUUsage      float64               `json:"cpu_usage"`
	MemoryUsage   float64               `json:"memory_usage"`
	DiskUsage     float64               `json:"disk_usage"`
	NetworkIO     map[string]interface{} `json:"network_io"`
	Status        string                 `json:"status"`
	Recommendations []string            `json:"recommendations"`
	Timestamp     time.Time              `json:"timestamp"`
}

type QueueItem struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Target      string                 `json:"target"`
	Priority    string                 `json:"priority"`
	Estimates   map[string]interface{} `json:"priority_estimates"`
	Status      string                 `json:"status"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	AttemptCount int                   `json:"attempt_count"`
}

type ImprovementResult struct {
	QueueItemID string    `json:"queue_item_id"`
	Success     bool      `json:"success"`
	Changes     []string  `json:"changes"`
	TestResults []string  `json:"test_results"`
	Metrics     map[string]interface{} `json:"metrics"`
	CompletedAt time.Time `json:"completed_at"`
	FailureReason string  `json:"failure_reason,omitempty"`
}

type ValidationResult struct {
	Passed     bool                   `json:"passed"`
	Gates      []ValidationGate       `json:"gates"`
	Summary    string                 `json:"summary"`
	Timestamp  time.Time              `json:"timestamp"`
}

type ValidationGate struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Details string `json:"details,omitempty"`
}

type DebugRequest struct {
	AppName         string `json:"app_name"`
	DebugType       string `json:"debug_type"`
	IncludeLogs     bool   `json:"include_logs"`
	IncludePerformance bool `json:"include_performance"`
	IncludeErrors   bool   `json:"include_errors"`
}

type HealthStatus struct {
	Status    string                 `json:"status"`
	Scenarios []ScenarioHealth       `json:"scenarios"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
}

type ScenarioHealth struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	Health     string  `json:"health"`
	CPUUsage   float64 `json:"cpu_usage,omitempty"`
	MemoryUsage float64 `json:"memory_usage,omitempty"`
	ErrorCount int     `json:"error_count,omitempty"`
}

// Global configuration
var (
	queueDir  = getEnv("QUEUE_DIR", "../queue")
	claudeCodePath string // Cached path to resource-claude-code binary
)

// Claude Code client for AI analysis
func callClaudeForAnalysis(prompt string) (string, error) {
	// Use Claude Code for all AI analysis tasks
	// This is a simplified version that doesn't need file modifications
	if claudeCodePath == "" {
		return "", fmt.Errorf("resource-claude-code not found in PATH")
	}
	
	// Retry logic with exponential backoff
	maxRetries := 3
	baseDelay := 5 * time.Second
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1)) // Exponential backoff: 5s, 10s, 20s
			log.Printf("Retrying Claude analysis after %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}
		
		var stdout, stderr bytes.Buffer
		
		// Run with shorter timeout for analysis tasks (2 minutes)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		
		cmd := exec.CommandContext(ctx, claudeCodePath, "run", "--prompt", prompt, "--max-turns", "3")
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		
		err := cmd.Run()
		cancel()
		
		if err == nil {
			return stdout.String(), nil
		}
		
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Claude analysis timed out after 2 minutes (attempt %d/%d)", attempt+1, maxRetries)
			if attempt == maxRetries-1 {
				return "", fmt.Errorf("Claude analysis timed out after %d attempts", maxRetries)
			}
			continue
		}
		
		log.Printf("Claude analysis failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
		if attempt == maxRetries-1 {
			return "", fmt.Errorf("Claude analysis failed after %d attempts: %v", maxRetries, err)
		}
	}
	
	return "", fmt.Errorf("Claude analysis failed unexpectedly")
}

// Error analysis (replaces error-analyzer.json)
func analyzeError(report ErrorReport) (*FixSuggestion, error) {
	prompt := fmt.Sprintf(`Analyze this error from scenario '%s':

Error: %s
Stack Trace: %s
Context: %v

Provide:
1. Root cause analysis
2. 3 specific fix suggestions 
3. Confidence level (0-1)
4. Priority (critical/high/medium/low)
5. Estimated fix time

Format as JSON with fields: analysis, suggestions[], confidence, priority, estimated_time`, 
		report.AppName, report.ErrorMessage, report.StackTrace, report.Context)
	
	response, err := callClaudeForAnalysis(prompt)
	if err != nil {
		return nil, err
	}
	
	// Parse AI response and create fix suggestion
	suggestion := &FixSuggestion{
		ErrorID:     report.ID,
		Suggestions: parseFixSuggestions(response),
		Confidence:  parseConfidence(response),
		Priority:    parsePriority(response),
		EstimatedTime: parseEstimatedTime(response),
	}
	
	return suggestion, nil
}

// Performance profiling (replaces performance-profiler.json)
func profilePerformance(appName string) (*PerformanceMetrics, error) {
	// Simulate getting system metrics - in real implementation, would use system calls
	metrics := &PerformanceMetrics{
		AppName:     appName,
		CPUUsage:    getCurrentCPUUsage(appName),
		MemoryUsage: getCurrentMemoryUsage(appName),
		DiskUsage:   getCurrentDiskUsage(appName),
		NetworkIO:   getCurrentNetworkIO(appName),
		Timestamp:   time.Now(),
	}
	
	// Get AI recommendations for performance optimization
	prompt := fmt.Sprintf(`Analyze performance metrics for scenario '%s':
CPU Usage: %.2f%%
Memory Usage: %.2f%%
Disk Usage: %.2f%%

Provide specific optimization recommendations as a JSON array.`, 
		appName, metrics.CPUUsage, metrics.MemoryUsage, metrics.DiskUsage)
	
	response, err := callClaudeForAnalysis(prompt)
	if err != nil {
		return nil, err
	}
	
	metrics.Recommendations = parseRecommendations(response)
	metrics.Status = determinePerformanceStatus(metrics)
	
	return metrics, nil
}

// Log monitoring (replaces log-monitor.json)
func monitorLogs(appName string) ([]ErrorReport, error) {
	// Look for logs in multiple possible locations
	logPaths := []string{
		fmt.Sprintf("/var/log/vrooli/%s.log", appName),
		fmt.Sprintf("../logs/%s.log", appName),
		fmt.Sprintf("/tmp/vrooli/%s.log", appName),
		fmt.Sprintf("../../scenarios/%s/logs/app.log", appName),
	}
	
	var logPath string
	for _, path := range logPaths {
		if _, err := os.Stat(path); err == nil {
			logPath = path
			break
		}
	}
	
	// Read recent logs and identify errors
	errors := []ErrorReport{}
	
	// Check if we found a valid log path
	if logPath != "" && logExists(logPath) {
		recentErrors := parseLogForErrors(logPath)
		for _, err := range recentErrors {
			errors = append(errors, ErrorReport{
				AppName:      appName,
				ErrorMessage: err.Message,
				StackTrace:   err.Stack,
				Context:      err.Context,
				Severity:     err.Severity,
				Timestamp:    err.Timestamp,
			})
		}
	}
	
	return errors, nil
}

// Queue processing engine
func selectNextImprovement() (*QueueItem, error) {
	pendingDir := filepath.Join(queueDir, "pending")
	
	files, err := filepath.Glob(filepath.Join(pendingDir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	
	if len(files) == 0 {
		return nil, nil // No pending items
	}
	
	// Load all queue items and calculate priority scores
	var items []QueueItem
	for _, file := range files {
		item, err := loadQueueItem(file)
		if err != nil {
			continue
		}
		items = append(items, *item)
	}
	
	if len(items) == 0 {
		return nil, nil
	}
	
	// Sort by priority score (higher is better)
	sort.Slice(items, func(i, j int) bool {
		return calculatePriorityScore(&items[i]) > calculatePriorityScore(&items[j])
	})
	
	selected := &items[0]
	
	// Move to in-progress
	err = moveQueueItem(selected.ID, "pending", "in-progress")
	if err != nil {
		return nil, err
	}
	
	selected.Status = "in-progress"
	return selected, nil
}

func processImprovement(item *QueueItem) (*ImprovementResult, error) {
	log.Printf("Processing improvement: %s for target: %s", item.Title, item.Target)
	
	// Search Vrooli memory for relevant patterns
	memoryContext, err := searchVrooliMemory(item.Target, item.Type)
	if err != nil {
		log.Printf("Warning: Memory search failed: %v", err)
		memoryContext = "No previous context found"
	}
	
	// Build comprehensive prompt for Claude Code
	prompt := fmt.Sprintf(`You are implementing a Vrooli scenario improvement. 

Task: %s
Target Scenario: %s
Type: %s
Description: %s

Memory Context from Previous Improvements:
%s

Please:
1. Navigate to the target scenario directory at /home/matthalloran8/Vrooli/scenarios/%s
2. Analyze the current implementation
3. Implement the improvement described above
4. Run any existing tests to validate your changes
5. Update documentation if needed
6. Report back with:
   - A list of files modified
   - Tests run and their results
   - Whether the improvement was successful

Focus on small, incremental improvements that enhance functionality and maintainability.
If you encounter any errors or issues, document them clearly.

IMPORTANT: Make actual code changes, don't just analyze or plan.`, 
		item.Title, item.Target, item.Type, item.Description, memoryContext, item.Target)
	
	// Call Claude Code CLI to do the actual implementation
	log.Printf("Delegating implementation to Claude Code...")
	response, err := callClaudeCode(prompt)
	if err != nil {
		return &ImprovementResult{
			QueueItemID:   item.ID,
			Success:       false,
			FailureReason: fmt.Sprintf("Claude Code execution failed: %v", err),
			CompletedAt:   time.Now(),
		}, nil
	}
	
	// Parse Claude's response to extract results
	result := parseClaudeCodeResponse(response, item.ID)
	result.CompletedAt = time.Now()
	
	return result, nil
}

func validateImprovement(result *ImprovementResult) (*ValidationResult, error) {
	// Simple validation - trust Claude Code's assessment
	// Claude already ran tests and validated during implementation
	
	gates := []ValidationGate{
		{Name: "Implementation", Passed: result.Success, Details: "Claude Code execution status"},
		{Name: "Tests", Passed: len(result.TestResults) == 0 || !strings.Contains(strings.Join(result.TestResults, " "), "FAIL"), Details: strings.Join(result.TestResults, "; ")},
	}
	
	allPassed := result.Success
	
	return &ValidationResult{
		Passed:    allPassed,
		Gates:     gates,
		Summary:   fmt.Sprintf("Improvement %s: %d/%d gates passed", result.QueueItemID, countPassedGates(gates), len(gates)),
		Timestamp: time.Now(),
	}, nil
}

// Debug→Improve pipeline connectors
func errorsToImprovements(errors []ErrorReport) []QueueItem {
	var improvements []QueueItem
	
	for _, err := range errors {
		if err.Severity == "critical" || err.Severity == "high" {
			improvement := QueueItem{
				ID:          fmt.Sprintf("error-%d-%d", err.ID, time.Now().UnixNano()),
				Title:       fmt.Sprintf("Fix %s error in %s", err.Severity, err.AppName),
				Description: fmt.Sprintf("Error: %s\nStack: %s", err.ErrorMessage, err.StackTrace),
				Type:        "fix",
				Target:      err.AppName,
				Priority:    err.Severity,
				CreatedBy:   "error-monitor",
				CreatedAt:   time.Now(),
			}
			improvements = append(improvements, improvement)
		}
	}
	
	return improvements
}

func performanceToImprovements(metrics *PerformanceMetrics) []QueueItem {
	var improvements []QueueItem
	
	if metrics.CPUUsage > 80 || metrics.MemoryUsage > 85 {
		improvement := QueueItem{
			ID:          fmt.Sprintf("perf-%s-%d", metrics.AppName, time.Now().UnixNano()),
			Title:       fmt.Sprintf("Optimize performance for %s", metrics.AppName),
			Description: fmt.Sprintf("High resource usage detected. CPU: %.2f%%, Memory: %.2f%%", metrics.CPUUsage, metrics.MemoryUsage),
			Type:        "optimization",
			Target:      metrics.AppName,
			Priority:    "high",
			CreatedBy:   "performance-monitor",
			CreatedAt:   time.Now(),
		}
		improvements = append(improvements, improvement)
	}
	
	return improvements
}

// HTTP handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	scenarios := getRunningScenarios()
	
	status := HealthStatus{
		Status:    "healthy",
		Scenarios: scenarios,
		Message:   "Scenario Improver is running",
		Timestamp: time.Now(),
	}
	
	// Check if any scenarios have issues
	for _, scenario := range scenarios {
		if scenario.Health == "unhealthy" {
			status.Status = "degraded"
			status.Message = "Some scenarios have health issues"
			break
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func startImprovementHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Select next improvement from queue
	item, err := selectNextImprovement()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	if item == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "no_work",
			"message": "No pending improvements in queue",
		})
		return
	}
	
	// Process the improvement
	go func() {
		result, err := processImprovement(item)
		if err != nil {
			log.Printf("Improvement processing failed: %v", err)
			moveQueueItem(item.ID, "in-progress", "failed")
			return
		}
		
		// Validate the improvement
		validation, err := validateImprovement(result)
		if err != nil || !validation.Passed {
			log.Printf("Improvement validation failed: %v", err)
			moveQueueItem(item.ID, "in-progress", "failed")
			return
		}
		
		// Success - move to completed
		moveQueueItem(item.ID, "in-progress", "completed")
		log.Printf("Improvement completed successfully: %s", item.Title)
		
		// Update Qdrant memory with results
		updateVrooliMemory(result)
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "started",
		"item":   item,
	})
}

func analyzeErrorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var report ErrorReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Analyze error and generate suggestions
	suggestion, err := analyzeError(report)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Convert error to improvement queue item automatically
	improvements := errorsToImprovements([]ErrorReport{report})
	for _, improvement := range improvements {
		saveQueueItem(&improvement, "pending")
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"analysis":     suggestion,
		"improvements": improvements,
	})
}

func listScenariosHandler(w http.ResponseWriter, r *http.Request) {
	scenarios := getRunningScenarios()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
	})
}

func queueStatusHandler(w http.ResponseWriter, r *http.Request) {
	pending := countQueueItems("pending")
	inProgress := countQueueItems("in-progress")
	completed := countQueueItems("completed")
	failed := countQueueItems("failed")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pending":     pending,
		"in_progress": inProgress,
		"completed":   completed,
		"failed":      failed,
		"total":       pending + inProgress + completed + failed,
	})
}

func profilePerformanceHandler(w http.ResponseWriter, r *http.Request) {
	appName := r.URL.Query().Get("app")
	if appName == "" {
		http.Error(w, "app parameter required", http.StatusBadRequest)
		return
	}
	
	metrics, err := profilePerformance(appName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Convert performance issues to improvement queue items automatically
	improvements := performanceToImprovements(metrics)
	for _, improvement := range improvements {
		saveQueueItem(&improvement, "pending")
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics":      metrics,
		"improvements": improvements,
	})
}

// New endpoint to submit improvements to the queue
func submitImprovementHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse improvement request
	var req struct {
		Title       string                 `json:"title"`
		Description string                 `json:"description"`
		Type        string                 `json:"type"`        // fix, optimization, feature, documentation
		Target      string                 `json:"target"`      // scenario name
		Priority    string                 `json:"priority"`    // critical, high, medium, low
		Estimates   map[string]interface{} `json:"estimates"`   // optional
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: " + err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if req.Title == "" || req.Target == "" {
		http.Error(w, "title and target are required fields", http.StatusBadRequest)
		return
	}
	
	// Set defaults
	if req.Type == "" {
		req.Type = "improvement"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.Estimates == nil {
		req.Estimates = map[string]interface{}{
			"impact":       5,
			"urgency":      "medium",
			"success_prob": 0.7,
			"resource_cost": "moderate",
		}
	}
	
	// Create queue item
	item := QueueItem{
		ID:          fmt.Sprintf("%s-%d", req.Target, time.Now().UnixNano()),
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Target:      req.Target,
		Priority:    req.Priority,
		Estimates:   req.Estimates,
		Status:      "pending",
		CreatedBy:   "api-submission",
		CreatedAt:   time.Now(),
		AttemptCount: 0,
	}
	
	// Save to pending queue
	if err := saveQueueItem(&item, "pending"); err != nil {
		http.Error(w, "Failed to save improvement: " + err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Log the event
	logQueueEvent("submitted", item.ID, fmt.Sprintf("New improvement: %s", req.Title))
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "created",
		"item_id": item.ID,
		"message": fmt.Sprintf("Improvement '%s' added to queue", req.Title),
	})
}

// Handler for /api/logs/{appName} endpoint
func logsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract app name from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/logs/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "App name is required", http.StatusBadRequest)
		return
	}
	
	appName := pathParts[0]
	
	// Monitor logs for the specified app
	errors, err := monitorLogs(appName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch logs: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Convert errors to log format that UI expects
	var logs []string
	for _, error := range errors {
		logEntry := fmt.Sprintf("[%s] %s: %s", 
			error.Timestamp.Format("15:04:05"), 
			strings.ToUpper(error.Severity), 
			error.ErrorMessage)
		if error.StackTrace != "" {
			logEntry += fmt.Sprintf("\n  Stack: %s", error.StackTrace)
		}
		logs = append(logs, logEntry)
	}
	
	// If no errors found, add a default message
	if len(logs) == 0 {
		logs = []string{
			fmt.Sprintf("[%s] INFO: No recent errors found for %s", 
				time.Now().Format("15:04:05"), appName),
			fmt.Sprintf("[%s] INFO: Log monitoring is active", 
				time.Now().Format("15:04:05")),
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": logs,
		"app_name": appName,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Handler for /api/fixes/{appName} endpoint  
func fixesHandler(w http.ResponseWriter, r *http.Request) {
	// Extract app name from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/fixes/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "App name is required", http.StatusBadRequest)
		return
	}
	
	appName := pathParts[0]
	
	// Get recent errors for the app
	errors, err := monitorLogs(appName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze app: %v", err), http.StatusInternalServerError)
		return
	}
	
	var suggestions []string
	
	// Generate suggestions based on recent errors
	if len(errors) > 0 {
		// Take the most recent/severe error and analyze it
		mostSevereError := errors[0]
		for _, error := range errors {
			if error.Severity == "critical" || error.Severity == "high" {
				mostSevereError = error
				break
			}
		}
		
		// Create error report for analysis
		report := ErrorReport{
			AppName:      appName,
			ErrorMessage: mostSevereError.ErrorMessage,
			StackTrace:   mostSevereError.StackTrace,
			Context:      mostSevereError.Context,
			Severity:     mostSevereError.Severity,
			Timestamp:    mostSevereError.Timestamp,
		}
		
		// Get AI suggestions
		fixSuggestion, err := analyzeError(report)
		if err == nil && fixSuggestion != nil {
			suggestions = fixSuggestion.Suggestions
		}
	}
	
	// Add general improvement suggestions if no specific errors
	if len(suggestions) == 0 {
		suggestions = []string{
			"Review recent logs for any warning patterns",
			"Check system resource usage and optimize if needed", 
			"Verify all dependencies are up to date",
			"Run comprehensive tests to identify potential issues",
			"Consider adding additional monitoring and alerting",
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": suggestions,
		"app_name": appName,
		"error_count": len(errors),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Handler for /api/debug endpoint
func debugHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req DebugRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: " + err.Error(), http.StatusBadRequest)
		return
	}
	
	if req.AppName == "" {
		http.Error(w, "app_name is required", http.StatusBadRequest)
		return
	}
	
	// Perform debug session based on debug type
	var debugData map[string]interface{}
	var debugMessage string
	
	switch req.DebugType {
	case "errors":
		if req.IncludeErrors {
			errors, err := monitorLogs(req.AppName)
			if err != nil {
				debugMessage = fmt.Sprintf("Failed to fetch errors: %v", err)
			} else {
				debugData = map[string]interface{}{
					"errors": errors,
				}
				debugMessage = fmt.Sprintf("Found %d errors for %s", len(errors), req.AppName)
			}
		}
		
	case "performance":
		if req.IncludePerformance {
			metrics, err := profilePerformance(req.AppName)
			if err != nil {
				debugMessage = fmt.Sprintf("Failed to profile performance: %v", err)
			} else {
				debugData = map[string]interface{}{
					"metrics": map[string]interface{}{
						"cpu_usage":    fmt.Sprintf("%.2f%%", metrics.CPUUsage),
						"memory_usage": fmt.Sprintf("%.2f%%", metrics.MemoryUsage),
						"disk_usage":   fmt.Sprintf("%.2f%%", metrics.DiskUsage),
						"network_io":   metrics.NetworkIO,
						"status":       metrics.Status,
						"recommendations": metrics.Recommendations,
					},
				}
				debugMessage = fmt.Sprintf("Performance profiling completed for %s", req.AppName)
			}
		}
		
	case "logs":
		if req.IncludeLogs {
			errors, err := monitorLogs(req.AppName)
			if err != nil {
				debugMessage = fmt.Sprintf("Failed to fetch logs: %v", err)
			} else {
				var logs []string
				for _, error := range errors {
					logs = append(logs, fmt.Sprintf("[%s] %s: %s", 
						error.Timestamp.Format("15:04:05"), 
						error.Severity, 
						error.ErrorMessage))
				}
				debugData = map[string]interface{}{
					"logs": logs,
				}
				debugMessage = fmt.Sprintf("Retrieved %d log entries for %s", len(logs), req.AppName)
			}
		}
		
	case "fix":
		// Generate fix suggestions
		errors, err := monitorLogs(req.AppName)
		if err == nil && len(errors) > 0 {
			// Analyze the first error for suggestions
			report := ErrorReport{
				AppName:      req.AppName,
				ErrorMessage: errors[0].ErrorMessage,
				StackTrace:   errors[0].StackTrace,
				Context:      errors[0].Context,
				Severity:     errors[0].Severity,
			}
			
			fixSuggestion, err := analyzeError(report)
			if err == nil && fixSuggestion != nil {
				debugData = map[string]interface{}{
					"suggestions": fixSuggestion.Suggestions,
				}
				debugMessage = fmt.Sprintf("Generated %d fix suggestions for %s", len(fixSuggestion.Suggestions), req.AppName)
			} else {
				debugMessage = fmt.Sprintf("Failed to generate suggestions: %v", err)
			}
		} else {
			debugData = map[string]interface{}{
				"suggestions": []string{"No recent errors found - system appears healthy"},
			}
			debugMessage = fmt.Sprintf("No issues detected for %s", req.AppName)
		}
		
	default:
		debugMessage = fmt.Sprintf("Unknown debug type: %s", req.DebugType)
	}
	
	// Default response if no data was set
	if debugData == nil {
		debugData = map[string]interface{}{
			"message": "Debug session completed but no data available",
		}
	}
	
	if debugMessage == "" {
		debugMessage = fmt.Sprintf("Debug session completed for %s", req.AppName)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "completed",
		"message": debugMessage,
		"data":    debugData,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func main() {
	// Initialize binary paths
	initBinaryPaths()
	
	port := getEnv("API_PORT", getEnv("PORT", ""))
	if port == "" {
		log.Fatal("No port specified")
	}
	
	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Setup routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/improvement/start", startImprovementHandler)
	http.HandleFunc("/api/improvement/submit", submitImprovementHandler)  // New endpoint
	http.HandleFunc("/api/error/analyze", analyzeErrorHandler)
	http.HandleFunc("/api/scenarios/list", listScenariosHandler)
	http.HandleFunc("/api/queue/status", queueStatusHandler)
	http.HandleFunc("/api/performance/profile", profilePerformanceHandler)
	http.HandleFunc("/api/logs/", logsHandler)
	http.HandleFunc("/api/fixes/", fixesHandler)
	http.HandleFunc("/api/debug", debugHandler)
	
	// Start background services
	go startLogMonitor()
	go startPerformanceMonitor()
	
	// Start HTTP server in goroutine
	server := &http.Server{Addr: ":" + port}
	go func() {
		log.Printf("Scenario Improver API starting on port %s", port)
		log.Printf("Using Claude Code for all AI analysis and implementation")
		log.Printf("Submit improvements via POST /api/improvement/submit")
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	
	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down gracefully...")
	
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	
	// Move any in-progress items back to pending
	inProgressDir := filepath.Join(queueDir, "in-progress")
	if files, err := filepath.Glob(filepath.Join(inProgressDir, "*.yaml")); err == nil {
		for _, file := range files {
			filename := filepath.Base(file)
			id := strings.TrimSuffix(filename, ".yaml")
			if err := moveQueueItem(id, "in-progress", "pending"); err == nil {
				log.Printf("Moved in-progress item %s back to pending", id)
			}
		}
	}
	
	log.Println("Scenario Improver shut down successfully")
}

// Utility functions (implementations would be added here)
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function implementations
func parseFixSuggestions(response string) []string {
	// Try to parse JSON response first
	var jsonResp map[string]interface{}
	if err := json.Unmarshal([]byte(response), &jsonResp); err == nil {
		if suggestions, ok := jsonResp["suggestions"].([]interface{}); ok {
			var result []string
			for _, suggestion := range suggestions {
				if str, ok := suggestion.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	
	// Fallback to text parsing - look for numbered lists
	lines := strings.Split(response, "\n")
	var suggestions []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for numbered items (1. 2. 3.) or bullet points
		if matched, _ := regexp.MatchString(`^[0-9]+\.|^[-*]\s`, line); matched {
			// Clean up the suggestion text
			clean := regexp.MustCompile(`^[0-9]+\.|^[-*]\s`).ReplaceAllString(line, "")
			clean = strings.TrimSpace(clean)
			if clean != "" {
				suggestions = append(suggestions, clean)
			}
		}
	}
	
	// If no structured suggestions found, extract sentences
	if len(suggestions) == 0 {
		sentences := strings.Split(response, ".")
		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if len(sentence) > 20 && len(sentence) < 200 {
				suggestions = append(suggestions, sentence+".")
				if len(suggestions) >= 3 {
					break
				}
			}
		}
	}
	
	// Ensure we have at least one suggestion
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Review the error and check system logs for more details")
	}
	
	return suggestions
}

func parseConfidence(response string) float64 {
	// Try JSON parsing first
	var jsonResp map[string]interface{}
	if err := json.Unmarshal([]byte(response), &jsonResp); err == nil {
		if confidence, ok := jsonResp["confidence"].(float64); ok {
			return confidence
		}
	}
	
	// Look for confidence mentions in text
	lowerResp := strings.ToLower(response)
	
	// Look for percentage patterns
	if matches := regexp.MustCompile(`confidence.*?([0-9]+)%`).FindStringSubmatch(lowerResp); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val / 100.0
		}
	}
	
	// Look for decimal patterns
	if matches := regexp.MustCompile(`confidence.*?([0-9]+\.[0-9]+)`).FindStringSubmatch(lowerResp); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}
	
	// Look for qualitative confidence indicators
	if strings.Contains(lowerResp, "very confident") || strings.Contains(lowerResp, "highly confident") {
		return 0.9
	}
	if strings.Contains(lowerResp, "confident") {
		return 0.8
	}
	if strings.Contains(lowerResp, "somewhat confident") || strings.Contains(lowerResp, "moderately confident") {
		return 0.6
	}
	if strings.Contains(lowerResp, "low confidence") || strings.Contains(lowerResp, "uncertain") {
		return 0.3
	}
	
	// Default confidence based on response quality
	if len(response) > 200 && strings.Contains(response, "suggest") {
		return 0.7
	}
	return 0.5
}

func parsePriority(response string) string {
	// Try JSON parsing first
	var jsonResp map[string]interface{}
	if err := json.Unmarshal([]byte(response), &jsonResp); err == nil {
		if priority, ok := jsonResp["priority"].(string); ok {
			return strings.ToLower(priority)
		}
	}
	
	// Look for priority keywords in text
	lowerResp := strings.ToLower(response)
	
	// Check for critical/urgent indicators
	if strings.Contains(lowerResp, "critical") || strings.Contains(lowerResp, "urgent") || 
	   strings.Contains(lowerResp, "immediate") || strings.Contains(lowerResp, "emergency") {
		return "critical"
	}
	
	// Check for high priority indicators
	if strings.Contains(lowerResp, "high priority") || strings.Contains(lowerResp, "important") || 
	   strings.Contains(lowerResp, "severe") {
		return "high"
	}
	
	// Check for low priority indicators
	if strings.Contains(lowerResp, "low priority") || strings.Contains(lowerResp, "minor") || 
	   strings.Contains(lowerResp, "cosmetic") || strings.Contains(lowerResp, "nice to have") {
		return "low"
	}
	
	// Default to medium priority
	return "medium"
}

func parseEstimatedTime(response string) string {
	// Try JSON parsing first
	var jsonResp map[string]interface{}
	if err := json.Unmarshal([]byte(response), &jsonResp); err == nil {
		if estimatedTime, ok := jsonResp["estimated_time"].(string); ok {
			return estimatedTime
		}
	}
	
	// Look for time estimates in text
	lowerResp := strings.ToLower(response)
	
	// Look for specific time patterns
	timePatterns := []string{
		`([0-9]+)\s*(hour|hr)s?`,
		`([0-9]+)\s*(minute|min)s?`,
		`([0-9]+)\s*(day)s?`,
		`([0-9]+)\s*(week)s?`,
	}
	
	for _, pattern := range timePatterns {
		if matches := regexp.MustCompile(pattern).FindStringSubmatch(lowerResp); len(matches) > 2 {
			return matches[1] + " " + matches[2] + "s"
		}
	}
	
	// Look for qualitative time estimates
	if strings.Contains(lowerResp, "quick") || strings.Contains(lowerResp, "fast") || strings.Contains(lowerResp, "simple") {
		return "30 minutes"
	}
	if strings.Contains(lowerResp, "complex") || strings.Contains(lowerResp, "difficult") || strings.Contains(lowerResp, "challenging") {
		return "4 hours"
	}
	if strings.Contains(lowerResp, "major") || strings.Contains(lowerResp, "significant") {
		return "1 day"
	}
	
	// Default estimate based on response complexity
	if len(response) > 500 {
		return "2 hours"
	}
	return "1 hour"
}

func getCurrentCPUUsage(appName string) float64 {
	// Use system-monitor (required dependency)
	cmd := exec.Command("system-monitor", "metrics", "--json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("ERROR: system-monitor metrics failed: %v", err)
		return 0.0
	}
	
	// Parse JSON response from system-monitor
	var metrics map[string]interface{}
	if err := json.Unmarshal(output, &metrics); err != nil {
		log.Printf("ERROR: Failed to parse system-monitor output: %v", err)
		return 0.0
	}
	
	if cpuUsage, ok := metrics["cpu_usage"].(float64); ok {
		return cpuUsage
	}
	
	log.Printf("WARNING: cpu_usage not found in system-monitor output")
	return 0.0
}

func getCurrentMemoryUsage(appName string) float64 {
	// Use system-monitor (required dependency)
	cmd := exec.Command("system-monitor", "metrics", "--json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("ERROR: system-monitor metrics failed: %v", err)
		return 0.0
	}
	
	// Parse JSON response from system-monitor
	var metrics map[string]interface{}
	if err := json.Unmarshal(output, &metrics); err != nil {
		log.Printf("ERROR: Failed to parse system-monitor output: %v", err)
		return 0.0
	}
	
	if memUsage, ok := metrics["memory_usage"].(float64); ok {
		return memUsage
	}
	
	log.Printf("WARNING: memory_usage not found in system-monitor output")
	return 0.0
}

func getCurrentDiskUsage(appName string) float64 {
	// Use system-monitor (required dependency)
	cmd := exec.Command("system-monitor", "metrics", "--json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("ERROR: system-monitor metrics failed: %v", err)
		return 0.0
	}
	
	// Parse JSON response from system-monitor
	var metrics map[string]interface{}
	if err := json.Unmarshal(output, &metrics); err != nil {
		log.Printf("ERROR: Failed to parse system-monitor output: %v", err)
		return 0.0
	}
	
	if diskUsage, ok := metrics["disk_usage"].(float64); ok {
		return diskUsage
	}
	
	log.Printf("WARNING: disk_usage not found in system-monitor output")
	return 0.0
}

func getCurrentNetworkIO(appName string) map[string]interface{} {
	// Use system-monitor (required dependency)
	cmd := exec.Command("system-monitor", "metrics", "--json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("ERROR: system-monitor metrics failed: %v", err)
		return map[string]interface{}{
			"bytes_in":  0,
			"bytes_out": 0,
		}
	}
	
	// Parse JSON response from system-monitor
	var metrics map[string]interface{}
	if err := json.Unmarshal(output, &metrics); err != nil {
		log.Printf("ERROR: Failed to parse system-monitor output: %v", err)
		return map[string]interface{}{
			"bytes_in":  0,
			"bytes_out": 0,
		}
	}
	
	// Extract network IO from metrics
	if networkIO, ok := metrics["network_io"].(map[string]interface{}); ok {
		return networkIO
	}
	
	log.Printf("WARNING: network_io not found in system-monitor output")
	return map[string]interface{}{
		"bytes_in":  0,
		"bytes_out": 0,
	}
}

func parseRecommendations(response string) []string {
	// Try to extract recommendations from Claude's response
	lines := strings.Split(response, "\n")
	var recommendations []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for recommendation patterns
		if strings.Contains(strings.ToLower(line), "recommend") ||
		   strings.Contains(strings.ToLower(line), "suggest") ||
		   strings.Contains(strings.ToLower(line), "optimize") ||
		   strings.Contains(strings.ToLower(line), "improve") {
			if len(line) > 10 && len(line) < 200 {
				recommendations = append(recommendations, line)
			}
		}
	}
	
	// Fallback if no recommendations found
	if len(recommendations) == 0 {
		recommendations = []string{"Review resource usage patterns", "Consider optimization opportunities"}
	}
	
	return recommendations
}

func determinePerformanceStatus(metrics *PerformanceMetrics) string {
	if metrics.CPUUsage > 80 || metrics.MemoryUsage > 85 {
		return "warning"
	}
	return "healthy"
}

func logExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}


func parseLogForErrors(logPath string) []struct {
	Message   string
	Stack     string
	Context   map[string]interface{}
	Severity  string
	Timestamp time.Time
} {
	type LogError = struct {
		Message   string
		Stack     string
		Context   map[string]interface{}
		Severity  string
		Timestamp time.Time
	}
	
	var errors []LogError
	
	// Read the last 100 lines of the log file
	cmd := exec.Command("tail", "-100", logPath)
	output, err := cmd.Output()
	if err != nil {
		return []LogError{}
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Look for common error patterns
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "exception") || 
		   strings.Contains(lowerLine, "panic") || strings.Contains(lowerLine, "fatal") {
			
			// Determine severity
			severity := "low"
			if strings.Contains(lowerLine, "fatal") || strings.Contains(lowerLine, "panic") {
				severity = "critical"
			} else if strings.Contains(lowerLine, "error") {
				severity = "high"
			} else if strings.Contains(lowerLine, "exception") {
				severity = "medium"
			}
			
			// Extract timestamp (assume first part of line is timestamp)
			timestamp := time.Now()
			parts := strings.Fields(line)
			if len(parts) > 0 {
				// Try to parse timestamp from first part
				if parsedTime, err := time.Parse("2006-01-02T15:04:05", parts[0]); err == nil {
					timestamp = parsedTime
				} else if parsedTime, err := time.Parse("2006/01/02 15:04:05", parts[0]+" "+parts[1]); err == nil {
					timestamp = parsedTime
				}
			}
			
			errors = append(errors, LogError{
				Message:   line,
				Stack:     "", // Would need more sophisticated parsing for stack traces
				Context:   map[string]interface{}{"log_file": logPath},
				Severity:  severity,
				Timestamp: timestamp,
			})
			
			// Limit to recent errors
			if len(errors) >= 10 {
				break
			}
		}
	}
	
	return errors
}


func searchVrooliMemory(target, typ string) (string, error) {
	// Try to use resource-qdrant CLI if available
	cmd := exec.Command("resource-qdrant", "search", "--collection", "improvements", "--query", fmt.Sprintf("%s %s", target, typ), "--limit", "5")
	output, err := cmd.Output()
	if err == nil {
		// Parse the search results and extract relevant context
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		var context []string
		for _, line := range lines {
			if strings.Contains(line, "text:") {
				// Extract text content
				if idx := strings.Index(line, "text:"); idx >= 0 {
					text := strings.TrimSpace(line[idx+5:])
					if text != "" {
						context = append(context, text)
					}
				}
			}
		}
		if len(context) > 0 {
			return strings.Join(context, "\n\n"), nil
		}
	}
	
	// Fallback: Search local improvement history files
	historyDir := "../queue/completed"
	files, err := filepath.Glob(filepath.Join(historyDir, "*.yaml"))
	if err != nil {
		return "No relevant memory context found", nil
	}
	
	var relevantContext []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		// Check if file content is relevant to target/type
		contentStr := strings.ToLower(string(content))
		if strings.Contains(contentStr, strings.ToLower(target)) || strings.Contains(contentStr, strings.ToLower(typ)) {
			// Extract relevant lines
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "description") && strings.Contains(line, ":") {
					if parts := strings.SplitN(line, ":", 2); len(parts) > 1 {
						relevantContext = append(relevantContext, strings.TrimSpace(parts[1]))
					}
				}
			}
			if len(relevantContext) >= 3 {
				break
			}
		}
	}
	
	if len(relevantContext) > 0 {
		return "Previous related improvements:\n" + strings.Join(relevantContext, "\n"), nil
	}
	
	return "No relevant memory context found", nil
}

// All implementation delegated to Claude Code - deprecated functions removed

func runIntegrationTests(result *ImprovementResult) bool {
	// Try to run tests for the target scenario
	if result == nil || result.QueueItemID == "" {
		return false
	}
	
	// Load the original queue item to get target info
	item, err := loadQueueItemByID(result.QueueItemID)
	if err != nil || item == nil {
		return false
	}
	
	// Use lifecycle.sh to run the "test" lifecycle event for the target scenario
	scenarioDir := fmt.Sprintf("../../scenarios/%s", item.Target)
	lifecycleScript := filepath.Join(scenarioDir, "lifecycle.sh")
	
	// Check if lifecycle.sh exists in the target scenario
	if _, err := os.Stat(lifecycleScript); err == nil {
		// Run the test lifecycle event
		cmd := exec.Command("bash", lifecycleScript, "test")
		cmd.Dir = scenarioDir
		
		// Set a reasonable timeout for tests (5 minutes)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		var stdout, stderr bytes.Buffer
		cmdWithTimeout := exec.CommandContext(ctx, "bash", lifecycleScript, "test")
		cmdWithTimeout.Dir = scenarioDir
		cmdWithTimeout.Stdout = &stdout
		cmdWithTimeout.Stderr = &stderr
		
		err := cmdWithTimeout.Run()
		if err == nil {
			log.Printf("Tests passed for scenario %s", item.Target)
			return true
		}
		
		// Log test failure details
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Tests timed out for scenario %s after 5 minutes", item.Target)
		} else {
			log.Printf("Tests failed for scenario %s: %v\nStdout: %s\nStderr: %s", 
				item.Target, err, stdout.String(), stderr.String())
		}
		return false
	}
	
	// Fallback: Check if scenario is healthy using CLI
	cmd := exec.Command("vrooli", "scenario", item.Target, "health")
	output, err := cmd.Output()
	if err == nil {
		// Check if output indicates healthy status
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(strings.ToLower(outputStr), "healthy") ||
		   strings.Contains(strings.ToLower(outputStr), "running") ||
		   strings.Contains(strings.ToLower(outputStr), "ok") {
			log.Printf("Scenario %s health check passed (no lifecycle.sh found)", item.Target)
			return true
		}
	}
	
	log.Printf("Unable to verify scenario %s - no lifecycle.sh and health check failed", item.Target)
	return false // Conservative: assume failure if we can't verify
}

func checkPerformanceImpact(result *ImprovementResult) bool {
	// Get current system performance
	cpuUsage := getCurrentCPUUsage("")
	memoryUsage := getCurrentMemoryUsage("")
	
	// Check if performance is within acceptable limits
	// Allow up to 80% CPU and 90% memory during improvements
	if cpuUsage > 80.0 {
		log.Printf("Performance impact warning: High CPU usage %.2f%%", cpuUsage)
		return false
	}
	
	if memoryUsage > 90.0 {
		log.Printf("Performance impact warning: High memory usage %.2f%%", memoryUsage)
		return false
	}
	
	// Check if improvement metrics show degradation
	if result != nil && result.Metrics != nil {
		if duration, ok := result.Metrics["execution_time_ms"]; ok {
			if durationFloat, ok := duration.(float64); ok && durationFloat > 30000 { // 30 seconds
				log.Printf("Performance impact warning: Long execution time %.2f ms", durationFloat)
				return false
			}
		}
	}
	
	return true
}

func runSecurityChecks(result *ImprovementResult) bool {
	// Basic security checks for improvements
	if result == nil {
		return false
	}
	
	// Check if any changes involve sensitive operations
	for _, change := range result.Changes {
		lowerChange := strings.ToLower(change)
		
		// Flag potentially dangerous operations
		dangerousPatterns := []string{
			"rm -rf", "chmod 777", "sudo", "password", "secret", "token",
			"exec(", "eval(", "system(", "shell_exec",
		}
		
		for _, pattern := range dangerousPatterns {
			if strings.Contains(lowerChange, pattern) {
				log.Printf("Security warning: Potentially dangerous operation detected: %s", pattern)
				return false
			}
		}
	}
	
	// Check if improvement involves network access patterns
	for _, change := range result.Changes {
		if strings.Contains(change, "http://") && !strings.Contains(change, "localhost") {
			log.Printf("Security warning: External HTTP access detected")
			// Allow but log warning
		}
	}
	
	return true
}

// Documentation and cross-scenario validation handled by Claude Code

func generateValidationSummary(gates []ValidationGate) string {
	passed := 0
	for _, gate := range gates {
		if gate.Passed {
			passed++
		}
	}
	return fmt.Sprintf("%d/%d validation gates passed", passed, len(gates))
}

func getRunningScenarios() []ScenarioHealth {
	var scenarios []ScenarioHealth
	
	// Use vrooli CLI to list scenarios
	cmd := exec.Command("vrooli", "scenario", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		// If vrooli CLI fails, try alternative method
		cmd = exec.Command("vrooli", "scenario", "list")
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Failed to list scenarios via CLI: %v", err)
			return scenarios
		}
		
		// Parse non-JSON output
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			// Extract scenario name from line
			parts := strings.Fields(line)
			if len(parts) > 0 {
				name := parts[0]
				
				// Get health status for this scenario
				healthCmd := exec.Command("vrooli", "scenario", name, "health")
				healthOutput, _ := healthCmd.Output()
				healthStr := strings.TrimSpace(string(healthOutput))
				
				health := "unknown"
				status := "unknown"
				if strings.Contains(strings.ToLower(healthStr), "healthy") {
					health = "healthy"
					status = "running"
				} else if strings.Contains(strings.ToLower(healthStr), "warning") {
					health = "warning"
					status = "running"
				} else if strings.Contains(strings.ToLower(healthStr), "error") {
					health = "unhealthy"
					status = "running"
				} else if strings.Contains(strings.ToLower(healthStr), "stopped") {
					status = "stopped"
				}
				
				scenarios = append(scenarios, ScenarioHealth{
					Name:   name,
					Status: status,
					Health: health,
				})
			}
		}
		return scenarios
	}
	
	// Parse JSON output from vrooli CLI
	var cliScenarios []map[string]interface{}
	if err := json.Unmarshal(output, &cliScenarios); err != nil {
		log.Printf("Failed to parse JSON from vrooli CLI: %v", err)
		return scenarios
	}
	
	for _, s := range cliScenarios {
		scenario := ScenarioHealth{
			Name:   getStringFromMap(s, "name", "unknown"),
			Status: getStringFromMap(s, "status", "unknown"),
			Health: getStringFromMap(s, "health", "unknown"),
		}
		
		// Get resource usage if available
		if cpu, ok := s["cpu_usage"].(float64); ok {
			scenario.CPUUsage = cpu
		}
		if mem, ok := s["memory_usage"].(float64); ok {
			scenario.MemoryUsage = mem
		}
		if errors, ok := s["error_count"].(float64); ok {
			scenario.ErrorCount = int(errors)
		}
		
		scenarios = append(scenarios, scenario)
	}
	
	return scenarios
}

// Helper function to extract string from map
func getStringFromMap(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}


func updateVrooliMemory(result *ImprovementResult) {
	// Prepare memory entry for storage
	memoryEntry := map[string]interface{}{
		"queue_item_id": result.QueueItemID,
		"success":       result.Success,
		"changes":       strings.Join(result.Changes, "; "),
		"test_results":  strings.Join(result.TestResults, "; "),
		"completed_at":  result.CompletedAt.Format(time.RFC3339),
	}
	
	// If failure, include failure reason
	if !result.Success {
		memoryEntry["failure_reason"] = result.FailureReason
	}
	
	// Try to store in Qdrant using CLI
	memoryJSON, _ := json.Marshal(memoryEntry)
	cmd := exec.Command("resource-qdrant", "upsert", "--collection", "improvements", "--payload", string(memoryJSON))
	if err := cmd.Run(); err != nil {
		// Fallback: Store locally in a memory file
		memoryFile := "../data/improvement_memory.jsonl"
		file, err := os.OpenFile(memoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to store memory: %v", err)
			return
		}
		defer file.Close()
		
		// Add timestamp and write as JSONL
		memoryEntry["stored_at"] = time.Now().Format(time.RFC3339)
		memoryJSON, _ = json.Marshal(memoryEntry)
		file.WriteString(string(memoryJSON) + "\n")
	}
	
	log.Printf("Memory updated for improvement %s (success: %v)", result.QueueItemID, result.Success)
}

func startLogMonitor() {
	// Background log monitoring with proper implementation
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			// Check if we should monitor logs
			if getEnv("ENABLE_LOG_MONITORING", "false") != "true" {
				continue
			}
			
			// Get list of running scenarios
			scenarios := getRunningScenarios()
			for _, scenario := range scenarios {
				if scenario.Status == "running" {
					// Check for errors in scenario logs
					errors, err := monitorLogs(scenario.Name)
					if err != nil {
						log.Printf("Failed to monitor logs for %s: %v", scenario.Name, err)
						continue
					}
					
					// Convert critical errors to improvement tasks
					if len(errors) > 0 {
						improvements := errorsToImprovements(errors)
						for _, imp := range improvements {
							saveQueueItem(&imp, "pending")
							log.Printf("Created improvement task from error: %s", imp.Title)
						}
					}
				}
			}
		}
	}()
}

func startPerformanceMonitor() {
	// Background performance monitoring with proper implementation
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			// Check if we should monitor performance
			if getEnv("ENABLE_PERFORMANCE_MONITORING", "false") != "true" {
				continue
			}
			
			// Get list of running scenarios
			scenarios := getRunningScenarios()
			for _, scenario := range scenarios {
				if scenario.Status == "running" {
					// Profile performance for high-usage scenarios
					if scenario.CPUUsage > 70 || scenario.MemoryUsage > 70 {
						metrics, err := profilePerformance(scenario.Name)
						if err != nil {
							log.Printf("Failed to profile %s: %v", scenario.Name, err)
							continue
						}
						
						// Convert performance issues to improvements
						improvements := performanceToImprovements(metrics)
						for _, imp := range improvements {
							saveQueueItem(&imp, "pending")
							log.Printf("Created performance improvement task: %s", imp.Title)
						}
					}
				}
			}
		}
	}()
}

// Load queue item by ID from any status directory
func loadQueueItemByID(id string) (*QueueItem, error) {
	statuses := []string{"pending", "in-progress", "completed", "failed"}
	
	for _, status := range statuses {
		files, err := filepath.Glob(filepath.Join(queueDir, status, fmt.Sprintf("%s*.yaml", id)))
		if err != nil {
			continue
		}
		
		for _, file := range files {
			item, err := loadQueueItem(file)
			if err == nil && item.ID == id {
				return item, nil
			}
		}
	}
	
	return nil, fmt.Errorf("queue item %s not found", id)
}

// ==================== CLAUDE CODE INTEGRATION ====================
// New functions for delegating work to resource-claude-code CLI

// Call Claude Code CLI to execute improvements
func callClaudeCode(prompt string) (string, error) {
	if claudeCodePath == "" {
		return "", fmt.Errorf("resource-claude-code not found in PATH")
	}
	
	// Set working directory to scenarios folder for context
	scenariosDir := getEnv("SCENARIOS_DIR", filepath.Join(os.Getenv("HOME"), "Vrooli/scenarios"))
	
	log.Printf("Executing Claude Code with prompt length: %d characters", len(prompt))
	
	// Retry logic with exponential backoff
	maxRetries := 3
	baseDelay := 10 * time.Second // Longer initial delay for complex tasks
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1)) // Exponential backoff: 10s, 20s, 40s
			log.Printf("Retrying Claude Code execution after %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}
		
		var stdout, stderr bytes.Buffer
		
		// Run with timeout (10 minutes for complex improvements)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		
		cmd := exec.CommandContext(ctx, claudeCodePath, "run", "--prompt", prompt, "--max-turns", "10")
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Dir = scenariosDir
		
		err := cmd.Run()
		cancel()
		
		if err == nil {
			return stdout.String(), nil
		}
		
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Claude Code execution timed out after 10 minutes (attempt %d/%d)", attempt+1, maxRetries)
			if attempt == maxRetries-1 {
				return "", fmt.Errorf("Claude Code execution timed out after %d attempts", maxRetries)
			}
			continue
		}
		
		log.Printf("Claude Code execution failed (attempt %d/%d): %v\nStderr: %s", attempt+1, maxRetries, err, stderr.String())
		if attempt == maxRetries-1 {
			return "", fmt.Errorf("Claude Code execution failed after %d attempts: %v\nStderr: %s", maxRetries, err, stderr.String())
		}
	}
	
	return "", fmt.Errorf("Claude Code execution failed unexpectedly")
}

// Parse Claude Code's response to extract improvement results
func parseClaudeCodeResponse(response string, queueItemID string) *ImprovementResult {
	result := &ImprovementResult{
		QueueItemID: queueItemID,
		Success:     true, // Default to success unless we find failure indicators
		Changes:     []string{},
		TestResults: []string{},
		Metrics:     map[string]interface{}{},
	}
	
	// Look for common failure indicators in response
	lowerResponse := strings.ToLower(response)
	if strings.Contains(lowerResponse, "error:") || 
	   strings.Contains(lowerResponse, "failed") ||
	   strings.Contains(lowerResponse, "could not") ||
	   strings.Contains(lowerResponse, "unable to") {
		result.Success = false
	}
	
	// Extract files modified (look for common patterns)
	if strings.Contains(response, "modified") || strings.Contains(response, "updated") || strings.Contains(response, "changed") {
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			if strings.Contains(line, ".go") || strings.Contains(line, ".js") || 
			   strings.Contains(line, ".ts") || strings.Contains(line, ".py") ||
			   strings.Contains(line, ".md") || strings.Contains(line, ".yaml") {
				result.Changes = append(result.Changes, strings.TrimSpace(line))
			}
		}
	}
	
	// Extract test results
	if strings.Contains(response, "test") || strings.Contains(response, "Test") {
		if strings.Contains(response, "PASS") || strings.Contains(response, "passed") {
			result.TestResults = append(result.TestResults, "Tests passed")
		}
		if strings.Contains(response, "FAIL") || strings.Contains(response, "failed") {
			result.TestResults = append(result.TestResults, "Some tests failed")
			result.Success = false
		}
	}
	
	// Set failure reason if not successful
	if !result.Success && result.FailureReason == "" {
		// Extract error message if present
		if idx := strings.Index(lowerResponse, "error:"); idx >= 0 {
			endIdx := strings.Index(response[idx:], "\n")
			if endIdx > 0 {
				result.FailureReason = strings.TrimSpace(response[idx:idx+endIdx])
			} else {
				result.FailureReason = "Improvement failed - check logs for details"
			}
		} else {
			result.FailureReason = "Improvement failed - check logs for details"
		}
	}
	
	// Add basic metrics
	result.Metrics["files_modified"] = len(result.Changes)
	result.Metrics["response_length"] = len(response)
	
	return result
}

// Helper function to count passed gates
func countPassedGates(gates []ValidationGate) int {
	count := 0
	for _, gate := range gates {
		if gate.Passed {
			count++
		}
	}
	return count
}

// Initialize binary paths at startup
func initBinaryPaths() {
	// Find resource-claude-code binary (required)
	if path, err := exec.LookPath("resource-claude-code"); err == nil {
		claudeCodePath = path
		log.Printf("Found resource-claude-code at: %s", claudeCodePath)
	} else {
		log.Fatal("CRITICAL: resource-claude-code not found in PATH - this is a required dependency")
	}
	
	// Verify system-monitor is available (required)
	if _, err := exec.LookPath("system-monitor"); err != nil {
		log.Fatal("CRITICAL: system-monitor not found in PATH - this is a required dependency")
	}
	log.Printf("Found system-monitor in PATH")
	
	// Check for optional resource-qdrant
	if _, err := exec.LookPath("resource-qdrant"); err == nil {
		log.Printf("Found resource-qdrant in PATH (memory storage available)")
	} else {
		log.Printf("WARNING: resource-qdrant not found - memory features will be limited")
	}
}