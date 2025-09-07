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

// Core data structures for resource improvement
type ResourceReport struct {
	ID           int                    `json:"id,omitempty"`
	ResourceName string                 `json:"resource_name"`
	IssueType    string                 `json:"issue_type"`
	Description  string                 `json:"description"`
	Context      map[string]interface{} `json:"context"`
	Severity     string                 `json:"severity,omitempty"`
	Timestamp    time.Time              `json:"timestamp,omitempty"`
}

type ImprovementSuggestion struct {
	ID              int      `json:"id"`
	ResourceReportID int     `json:"resource_report_id"`
	Suggestions     []string `json:"suggestions"`
	Confidence      float64  `json:"confidence"`
	Priority        string   `json:"priority"`
	EstimatedTime   string   `json:"estimated_time"`
}

type ResourceMetrics struct {
	ResourceName     string                 `json:"resource_name"`
	V2ComplianceScore float64               `json:"v2_compliance_score"`
	HealthReliability float64               `json:"health_reliability"`
	CLICoverage      float64               `json:"cli_coverage"`
	DocCompleteness  float64               `json:"doc_completeness"`
	Status           string                 `json:"status"`
	Recommendations  []string              `json:"recommendations"`
	Timestamp        time.Time              `json:"timestamp"`
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
	QueueItemID   string                 `json:"queue_item_id"`
	Success       bool                   `json:"success"`
	Changes       []string               `json:"changes"`
	TestResults   []string               `json:"test_results"`
	Metrics       map[string]interface{} `json:"metrics"`
	CompletedAt   time.Time              `json:"completed_at"`
	FailureReason string                 `json:"failure_reason,omitempty"`
}

type ValidationResult struct {
	Passed    bool             `json:"passed"`
	Gates     []ValidationGate `json:"gates"`
	Summary   string           `json:"summary"`
	Timestamp time.Time        `json:"timestamp"`
}

type ValidationGate struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Details string `json:"details,omitempty"`
}

type HealthStatus struct {
	Status    string           `json:"status"`
	Resources []ResourceHealth `json:"resources"`
	Message   string           `json:"message"`
	Timestamp time.Time        `json:"timestamp"`
}

type ResourceHealth struct {
	Name              string  `json:"name"`
	Status            string  `json:"status"`
	Health            string  `json:"health"`
	V2ComplianceScore float64 `json:"v2_compliance_score,omitempty"`
	HealthReliability float64 `json:"health_reliability,omitempty"`
	IssueCount        int     `json:"issue_count,omitempty"`
}

// Global configuration
var (
	queueDir       = getEnv("QUEUE_DIR", "../queue")
	claudeCodePath string // Cached path to resource-claude-code binary
)

// Claude Code client for AI analysis and implementation
func callClaudeForAnalysis(prompt string) (string, error) {
	if claudeCodePath == "" {
		return "", fmt.Errorf("resource-claude-code not found in PATH")
	}
	
	// Retry logic with exponential backoff
	maxRetries := 3
	baseDelay := 5 * time.Second
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1))
			log.Printf("Retrying Claude analysis after %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}
		
		var stdout, stderr bytes.Buffer
		
		// Run with timeout for analysis tasks (3 minutes)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		
		cmd := exec.CommandContext(ctx, claudeCodePath, "run", "--prompt", prompt, "--max-turns", "5")
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		
		err := cmd.Run()
		cancel()
		
		if err == nil {
			return stdout.String(), nil
		}
		
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Claude analysis timed out after 3 minutes (attempt %d/%d)", attempt+1, maxRetries)
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

// Resource analysis (analyzes resources for improvement opportunities)
func analyzeResource(report ResourceReport) (*ImprovementSuggestion, error) {
	prompt := fmt.Sprintf(`Analyze this resource improvement request for '%s':

Issue Type: %s
Description: %s
Context: %v

This is for a Vrooli resource that should meet v2.0 contract requirements. Provide:
1. Root cause analysis of the issue
2. 3 specific improvement suggestions prioritizing v2.0 compliance, health checks, and CLI standardization
3. Confidence level (0-1)
4. Priority (critical/high/medium/low)
5. Estimated implementation time

Focus on:
- v2.0 contract compliance (lib/ directory structure, lifecycle hooks)
- Health check robustness (timeouts, retries, detailed errors)
- CLI standardization (consistent commands, JSON output, help)
- Documentation completeness

Format as JSON with fields: analysis, suggestions[], confidence, priority, estimated_time`, 
		report.ResourceName, report.IssueType, report.Description, report.Context)
	
	response, err := callClaudeForAnalysis(prompt)
	if err != nil {
		return nil, err
	}
	
	// Parse AI response and create improvement suggestion
	suggestion := &ImprovementSuggestion{
		ResourceReportID: report.ID,
		Suggestions:      parseImprovementSuggestions(response),
		Confidence:       parseConfidence(response),
		Priority:         parsePriority(response),
		EstimatedTime:    parseEstimatedTime(response),
	}
	
	return suggestion, nil
}

// Resource metrics analysis
func analyzeResourceMetrics(resourceName string) (*ResourceMetrics, error) {
	// Get current resource status and compliance
	metrics := &ResourceMetrics{
		ResourceName: resourceName,
		Timestamp:    time.Now(),
	}
	
	// Check v2.0 contract compliance
	v2Score := checkV2Compliance(resourceName)
	metrics.V2ComplianceScore = v2Score
	
	// Check health reliability
	healthScore := checkHealthReliability(resourceName)
	metrics.HealthReliability = healthScore
	
	// Check CLI coverage
	cliScore := checkCLICoverage(resourceName)
	metrics.CLICoverage = cliScore
	
	// Check documentation completeness
	docScore := checkDocumentationCompleteness(resourceName)
	metrics.DocCompleteness = docScore
	
	// Get AI recommendations for improvements
	prompt := fmt.Sprintf(`Analyze resource improvement opportunities for '%s':
v2.0 Compliance Score: %.2f%%
Health Check Reliability: %.2f%%
CLI Coverage: %.2f%%
Documentation Completeness: %.2f%%

Provide specific improvement recommendations focusing on the lowest scores.
Priority order: v2.0 compliance > health checks > CLI > documentation.

Format as JSON array of specific, actionable recommendations.`, 
		resourceName, v2Score, healthScore, cliScore, docScore)
	
	response, err := callClaudeForAnalysis(prompt)
	if err != nil {
		return nil, err
	}
	
	metrics.Recommendations = parseRecommendations(response)
	metrics.Status = determineResourceStatus(metrics)
	
	return metrics, nil
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
	
	// Search Vrooli memory for relevant improvement patterns
	memoryContext, err := searchVrooliMemory(item.Target, item.Type)
	if err != nil {
		log.Printf("Warning: Memory search failed: %v", err)
		memoryContext = "No previous context found"
	}
	
	// Build comprehensive prompt for Claude Code
	prompt := fmt.Sprintf(`You are implementing a Vrooli resource improvement.

Task: %s
Target Resource: %s
Type: %s
Description: %s

Memory Context from Previous Improvements:
%s

Resource Improvement Focus Areas:
1. **v2.0 Contract Compliance**: Ensure lib/ directory structure, lifecycle hooks, health checks
2. **Health Check Robustness**: Add timeouts, retries, detailed error messages
3. **CLI Standardization**: Consistent commands (status, health, logs, content, help)
4. **Documentation**: Usage examples, environment variables, troubleshooting

Please:
1. Navigate to the resource directory at /home/matthalloran8/Vrooli/resources/%s
2. Analyze current v2.0 compliance and identify gaps
3. Implement the specific improvement described above
4. Test health checks and CLI commands
5. Update documentation if needed
6. Report back with:
   - List of files modified
   - v2.0 compliance improvements made
   - Tests run and results
   - Whether the improvement was successful

IMPORTANT: Focus on making the resource more reliable and easier to use.
Prioritize v2.0 contract compliance and robust health checking.`, 
		item.Title, item.Target, item.Type, item.Description, memoryContext, item.Target)
	
	// Call Claude Code CLI to do the actual implementation
	log.Printf("Delegating implementation to resource-claude-code...")
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
	// Comprehensive validation for resource improvements
	gates := []ValidationGate{
		{Name: "Implementation", Passed: result.Success, Details: "Claude Code execution status"},
		{Name: "Tests", Passed: len(result.TestResults) == 0 || !strings.Contains(strings.Join(result.TestResults, " "), "FAIL"), Details: strings.Join(result.TestResults, "; ")},
	}
	
	// Load the original queue item to get target info for additional validation
	item, err := loadQueueItemByID(result.QueueItemID)
	if err == nil && item != nil {
		// Add resource-specific validation
		v2ComplianceGate := ValidationGate{
			Name:    "V2 Compliance",
			Passed:  checkV2Compliance(item.Target) > 80.0, // 80% minimum
			Details: fmt.Sprintf("v2.0 contract compliance check for %s", item.Target),
		}
		gates = append(gates, v2ComplianceGate)
		
		// Health check validation
		healthGate := ValidationGate{
			Name:    "Health Check",
			Passed:  checkHealthReliability(item.Target) > 90.0, // 90% minimum
			Details: fmt.Sprintf("Health check reliability for %s", item.Target),
		}
		gates = append(gates, healthGate)
	}
	
	allPassed := result.Success
	for _, gate := range gates {
		if !gate.Passed {
			allPassed = false
		}
	}
	
	return &ValidationResult{
		Passed:    allPassed,
		Gates:     gates,
		Summary:   fmt.Sprintf("Improvement %s: %d/%d gates passed", result.QueueItemID, countPassedGates(gates), len(gates)),
		Timestamp: time.Now(),
	}, nil
}

// HTTP handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	resources := getAvailableResources()
	
	status := HealthStatus{
		Status:    "healthy",
		Resources: resources,
		Message:   "Resource Improver is running",
		Timestamp: time.Now(),
	}
	
	// Check if any resources have issues
	for _, resource := range resources {
		if resource.Health == "unhealthy" || resource.V2ComplianceScore < 80.0 {
			status.Status = "degraded"
			status.Message = "Some resources need improvement"
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

func analyzeResourceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var report ResourceReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Analyze resource and generate improvement suggestions
	suggestion, err := analyzeResource(report)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Convert report to improvement queue item automatically
	improvements := resourceReportToImprovements([]ResourceReport{report})
	for _, improvement := range improvements {
		saveQueueItem(&improvement, "pending")
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"analysis":     suggestion,
		"improvements": improvements,
	})
}

func listResourcesHandler(w http.ResponseWriter, r *http.Request) {
	resources := getAvailableResources()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"resources": resources,
		"count":     len(resources),
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

func resourceMetricsHandler(w http.ResponseWriter, r *http.Request) {
	resourceName := r.URL.Query().Get("resource")
	if resourceName == "" {
		http.Error(w, "resource parameter required", http.StatusBadRequest)
		return
	}
	
	metrics, err := analyzeResourceMetrics(resourceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Convert metrics issues to improvement queue items automatically
	if metrics.V2ComplianceScore < 90.0 || metrics.HealthReliability < 95.0 {
		improvements := metricsToImprovements(metrics)
		for _, improvement := range improvements {
			saveQueueItem(&improvement, "pending")
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics": metrics,
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
		Type        string                 `json:"type"`        // v2-compliance, health-check, cli-enhancement, documentation
		Target      string                 `json:"target"`      // resource name
		Priority    string                 `json:"priority"`    // critical, high, medium, low
		Estimates   map[string]interface{} `json:"estimates"`   // optional
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
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
		http.Error(w, "Failed to save improvement: "+err.Error(), http.StatusInternalServerError)
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

// Router for /api/resources/{name}/* endpoints
func resourcesRouter(w http.ResponseWriter, r *http.Request) {
	// Extract resource name and action from path
	path := strings.TrimPrefix(r.URL.Path, "/api/resources/")
	parts := strings.Split(path, "/")
	
	if len(parts) < 2 {
		http.Error(w, "Invalid resource path", http.StatusBadRequest)
		return
	}
	
	resourceName := parts[0]
	action := parts[1]
	
	switch action {
	case "analyze":
		handleResourceAnalyze(w, r, resourceName)
	case "status":
		handleResourceStatus(w, r, resourceName)
	default:
		http.Error(w, "Unknown action", http.StatusNotFound)
	}
}

// GET /api/resources/{name}/analyze - UI expects resource metrics analysis
func handleResourceAnalyze(w http.ResponseWriter, r *http.Request, resourceName string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get resource metrics analysis
	metrics, err := analyzeResourceMetrics(resourceName)
	if err != nil {
		log.Printf("Failed to analyze resource %s: %v", resourceName, err)
		http.Error(w, fmt.Sprintf("Failed to analyze resource '%s': %v", resourceName, err), http.StatusInternalServerError)
		return
	}
	
	// Format response for UI expectations
	response := map[string]interface{}{
		"analysis": map[string]interface{}{
			"v2_compliance_score": metrics.V2ComplianceScore,
			"health_reliability":  metrics.HealthReliability,
			"cli_coverage":       metrics.CLICoverage,
			"doc_completeness":   metrics.DocCompleteness,
			"recommendations":    metrics.Recommendations,
			"status":            metrics.Status,
		},
		"resource_name": resourceName,
		"timestamp":     metrics.Timestamp,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /api/resources/{name}/status - UI expects resource status and health
func handleResourceStatus(w http.ResponseWriter, r *http.Request, resourceName string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Check resource health and status
	v2Score := checkV2Compliance(resourceName)
	healthScore := checkHealthReliability(resourceName)
	cliScore := checkCLICoverage(resourceName)
	
	// Determine status
	status := "running"
	health := "healthy"
	
	if v2Score < 80.0 || healthScore < 90.0 {
		health = "warning"
	}
	if v2Score < 50.0 || healthScore < 70.0 {
		health = "unhealthy"
		status = "degraded"
	}
	
	// Build response
	response := map[string]interface{}{
		"status": status,
		"health": health,
		"details": map[string]interface{}{
			"v2_compliance":     fmt.Sprintf("%.1f%%", v2Score),
			"health_reliability": fmt.Sprintf("%.1f%%", healthScore),
			"cli_coverage":      fmt.Sprintf("%.1f%%", cliScore),
			"last_checked":      time.Now().Format(time.RFC3339),
		},
		"resource_name": resourceName,
		"timestamp":     time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /api/reports - UI expects to submit resource issue reports
func reportsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse the report request
	var reportReq struct {
		ResourceName string                 `json:"resource_name"`
		IssueType    string                 `json:"issue_type"`
		Description  string                 `json:"description"`
		Context      map[string]interface{} `json:"context"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&reportReq); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if reportReq.ResourceName == "" || reportReq.Description == "" {
		http.Error(w, "resource_name and description are required", http.StatusBadRequest)
		return
	}
	
	// Create resource report
	report := ResourceReport{
		ResourceName: reportReq.ResourceName,
		IssueType:    reportReq.IssueType,
		Description:  reportReq.Description,
		Context:      reportReq.Context,
		Severity:     "medium", // Default severity
		Timestamp:    time.Now(),
	}
	
	// Analyze and create improvement suggestions
	suggestion, err := analyzeResource(report)
	if err != nil {
		log.Printf("Failed to analyze resource report for %s: %v", reportReq.ResourceName, err)
		// Provide graceful degradation - create improvement anyway
		suggestion = &ImprovementSuggestion{
			Suggestions:   []string{fmt.Sprintf("Review %s resource for %s issues", reportReq.ResourceName, reportReq.IssueType)},
			Confidence:    0.5,
			Priority:      "medium",
			EstimatedTime: "2 hours",
		}
		log.Printf("Using fallback improvement suggestion for %s", reportReq.ResourceName)
	}
	
	// Convert to improvement queue items automatically
	improvements := resourceReportToImprovements([]ResourceReport{report})
	savedCount := 0
	for _, improvement := range improvements {
		if err := saveQueueItem(&improvement, "pending"); err != nil {
			log.Printf("Failed to save improvement queue item %s: %v", improvement.ID, err)
		} else {
			savedCount++
			log.Printf("Saved improvement queue item: %s", improvement.ID)
		}
	}
	
	// Return response with report ID
	reportID := fmt.Sprintf("report-%s-%d", reportReq.ResourceName, time.Now().UnixNano())
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       reportID,
		"status":   "received",
		"message":  fmt.Sprintf("Issue report received for %s", reportReq.ResourceName),
		"analysis": suggestion,
		"queue_items_created": savedCount,
		"total_improvements": len(improvements),
	})
}

func main() {
	// Initialize binary paths
	initBinaryPaths()
	
	port := getEnv("API_PORT", getEnv("PORT", ""))
	if port == "" {
		log.Fatal("No port specified. Please set API_PORT or PORT environment variable.")
	}
	
	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Setup routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/improvement/start", startImprovementHandler)
	http.HandleFunc("/api/improvement/submit", submitImprovementHandler)
	http.HandleFunc("/api/resource/analyze", analyzeResourceHandler)
	http.HandleFunc("/api/resources/list", listResourcesHandler)
	http.HandleFunc("/api/queue/status", queueStatusHandler)
	http.HandleFunc("/api/resource/metrics", resourceMetricsHandler)
	
	// UI-expected endpoints (path parameter versions)
	http.HandleFunc("/api/resources/", resourcesRouter) // Handle /api/resources/{name}/analyze and /api/resources/{name}/status
	http.HandleFunc("/api/reports", reportsHandler) // Alias for resource report submission
	
	// Start HTTP server in goroutine
	server := &http.Server{Addr: ":" + port}
	go func() {
		log.Printf("Resource Improver API starting on port %s", port)
		log.Printf("Using resource-claude-code for all AI analysis and implementation")
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
	
	log.Println("Resource Improver shut down successfully")
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function implementations (continuing from scenario-improver pattern)
func parseImprovementSuggestions(response string) []string {
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
			clean := regexp.MustCompile(`^[0-9]+\.|^[-*]\s`).ReplaceAllString(line, "")
			clean = strings.TrimSpace(clean)
			if clean != "" {
				suggestions = append(suggestions, clean)
			}
		}
	}
	
	// Ensure we have at least one suggestion
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Review resource for v2.0 compliance and health check improvements")
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
	
	// Default confidence
	return 0.7
}

func parsePriority(response string) string {
	lowerResp := strings.ToLower(response)
	
	if strings.Contains(lowerResp, "critical") || strings.Contains(lowerResp, "urgent") {
		return "critical"
	}
	if strings.Contains(lowerResp, "high priority") || strings.Contains(lowerResp, "important") {
		return "high"
	}
	if strings.Contains(lowerResp, "low priority") || strings.Contains(lowerResp, "minor") {
		return "low"
	}
	
	return "medium"
}

func parseEstimatedTime(response string) string {
	lowerResp := strings.ToLower(response)
	
	// Look for time patterns
	timePatterns := []string{
		`([0-9]+)\s*(hour|hr)s?`,
		`([0-9]+)\s*(minute|min)s?`,
		`([0-9]+)\s*(day)s?`,
	}
	
	for _, pattern := range timePatterns {
		if matches := regexp.MustCompile(pattern).FindStringSubmatch(lowerResp); len(matches) > 2 {
			return matches[1] + " " + matches[2] + "s"
		}
	}
	
	// Default estimate
	return "2 hours"
}

// Call Claude Code CLI to execute improvements
func callClaudeCode(prompt string) (string, error) {
	if claudeCodePath == "" {
		return "", fmt.Errorf("resource-claude-code not found in PATH")
	}
	
	// Set working directory to resources folder for context
	resourcesDir := getEnv("RESOURCES_DIR", filepath.Join(os.Getenv("HOME"), "Vrooli/resources"))
	
	log.Printf("Executing resource-claude-code with prompt length: %d characters", len(prompt))
	
	// Retry logic with exponential backoff
	maxRetries := 3
	baseDelay := 10 * time.Second
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1))
			log.Printf("Retrying Claude Code execution after %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}
		
		var stdout, stderr bytes.Buffer
		
		// Run with timeout (15 minutes for complex improvements)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		
		cmd := exec.CommandContext(ctx, claudeCodePath, "run", "--prompt", prompt, "--max-turns", "10")
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Dir = resourcesDir
		
		err := cmd.Run()
		cancel()
		
		if err == nil {
			return stdout.String(), nil
		}
		
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Claude Code execution timed out after 15 minutes (attempt %d/%d)", attempt+1, maxRetries)
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
	
	// Look for failure indicators
	lowerResponse := strings.ToLower(response)
	if strings.Contains(lowerResponse, "error:") || 
	   strings.Contains(lowerResponse, "failed") ||
	   strings.Contains(lowerResponse, "could not") ||
	   strings.Contains(lowerResponse, "unable to") {
		result.Success = false
	}
	
	// Extract files modified
	if strings.Contains(response, "modified") || strings.Contains(response, "updated") {
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			if strings.Contains(line, "lib/") || strings.Contains(line, ".sh") || 
			   strings.Contains(line, ".md") || strings.Contains(line, ".json") {
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
	
	// Add metrics
	result.Metrics["files_modified"] = len(result.Changes)
	result.Metrics["response_length"] = len(response)
	
	return result
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
}

// Additional helper functions for resource improvement logic will be added in queue.go