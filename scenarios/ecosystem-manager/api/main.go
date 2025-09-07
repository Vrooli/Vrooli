package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

// TaskItem represents a unified task in the ecosystem
type TaskItem struct {
	ID                string                 `json:"id" yaml:"id"`
	Title             string                 `json:"title" yaml:"title"`
	Type              string                 `json:"type" yaml:"type"`                           // resource | scenario
	Operation         string                 `json:"operation" yaml:"operation"`                 // generator | improver
	Category          string                 `json:"category" yaml:"category"`
	Priority          string                 `json:"priority" yaml:"priority"`
	EffortEstimate    string                 `json:"effort_estimate" yaml:"effort_estimate"`
	Urgency           string                 `json:"urgency" yaml:"urgency"`
	ImpactScore       int                    `json:"impact_score" yaml:"impact_score"`
	Requirements      map[string]interface{} `json:"requirements" yaml:"requirements"`
	Dependencies      []string               `json:"dependencies" yaml:"dependencies"`
	Blocks            []string               `json:"blocks" yaml:"blocks"`
	RelatedScenarios  []string               `json:"related_scenarios" yaml:"related_scenarios"`
	RelatedResources  []string               `json:"related_resources" yaml:"related_resources"`
	AssignedResources map[string]bool        `json:"assigned_resources" yaml:"assigned_resources"`
	Status            string                 `json:"status" yaml:"status"`
	ProgressPercent   int                    `json:"progress_percentage" yaml:"progress_percentage"`
	CurrentPhase      string                 `json:"current_phase" yaml:"current_phase"`
	StartedAt         string                 `json:"started_at" yaml:"started_at"`
	CompletedAt       string                 `json:"completed_at" yaml:"completed_at"`
	EstimatedComplete string                 `json:"estimated_completion" yaml:"estimated_completion"`
	ValidationCriteria []string              `json:"validation_criteria" yaml:"validation_criteria"`
	CreatedBy         string                 `json:"created_by" yaml:"created_by"`
	CreatedAt         string                 `json:"created_at" yaml:"created_at"`
	UpdatedAt         string                 `json:"updated_at" yaml:"updated_at"`
	Tags              []string               `json:"tags" yaml:"tags"`
	Notes             string                 `json:"notes" yaml:"notes"`
	Results           map[string]interface{} `json:"results" yaml:"results"`
}

// OperationConfig represents configuration for each operation type
type OperationConfig struct {
	Name               string            `json:"name" yaml:"name"`
	Type               string            `json:"type" yaml:"type"`
	Target             string            `json:"target" yaml:"target"`
	Description        string            `json:"description" yaml:"description"`
	AdditionalSections []string          `json:"additional_sections" yaml:"additional_sections"`
	Variables          map[string]interface{} `json:"variables" yaml:"variables"`
	EffortAllocation   map[string]string `json:"effort_allocation" yaml:"effort_allocation"`
	SuccessCriteria    []string          `json:"success_criteria" yaml:"success_criteria"`
	Principles         []string          `json:"principles" yaml:"principles"`
}

// PromptsConfig represents the unified prompts configuration
type PromptsConfig struct {
	Name         string                     `json:"name" yaml:"name"`
	Type         string                     `json:"type" yaml:"type"`
	Target       string                     `json:"target" yaml:"target"`
	Description  string                     `json:"description" yaml:"description"`
	BaseSections []string                   `json:"base_sections" yaml:"base_sections"`
	Operations   map[string]OperationConfig `json:"operations" yaml:"operations"`
	GlobalConfig map[string]interface{}     `json:"global_config" yaml:"global_config"`
}

var (
	queueDir      = "../queue"
	promptsConfig PromptsConfig
	queueProcessor *QueueProcessor
)

// QueueProcessor manages automated queue processing
type QueueProcessor struct {
	mu              sync.Mutex
	isRunning       bool
	stopChannel     chan bool
	processInterval time.Duration
}

// ClaudeCodeRequest represents a request to the Claude Code resource
type ClaudeCodeRequest struct {
	Prompt  string                 `json:"prompt"`
	Context map[string]interface{} `json:"context"`
}

// ClaudeCodeResponse represents a response from Claude Code
type ClaudeCodeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// ResourceInfo represents information about a discovered resource
type ResourceInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Port        int    `json:"port,omitempty"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Healthy     bool   `json:"healthy"`
	Version     string `json:"version,omitempty"`
}

// ScenarioInfo represents information about a discovered scenario
type ScenarioInfo struct {
	Name            string `json:"name"`
	Path            string `json:"path"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	PRDComplete     int    `json:"prd_completion_percentage"`
	Healthy         bool   `json:"healthy"`
	P0Requirements  int    `json:"p0_requirements"`
	P0Completed     int    `json:"p0_completed"`
}

// PRDStatus represents the status of a scenario's PRD
type PRDStatus struct {
	CompletionPercentage int      `json:"completion_percentage"`
	P0Requirements       int      `json:"p0_requirements"`
	P0Completed          int      `json:"p0_completed"`
	P1Requirements       int      `json:"p1_requirements"`
	P1Completed          int      `json:"p1_completed"`
	MissingRequirements  []string `json:"missing_requirements"`
}

// ServiceConfig represents a resource's service.json configuration
type ServiceConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Port        int    `json:"port,omitempty"`
	Category    string `json:"category,omitempty"`
	Version     string `json:"version,omitempty"`
}

func init() {
	// Load prompts configuration
	configFile := "../prompts/sections.yaml"
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read prompts config: %v", err)
	}
	
	if err := yaml.Unmarshal(data, &promptsConfig); err != nil {
		log.Fatalf("Failed to parse prompts config: %v", err)
	}
	
	log.Println("Loaded prompts configuration for", len(promptsConfig.Operations), "operations")
}

// Smart prompt selection based on task type and operation
func selectPromptAssembly(taskType, operation string) (OperationConfig, error) {
	operationKey := fmt.Sprintf("%s-%s", taskType, operation)
	config, exists := promptsConfig.Operations[operationKey]
	if !exists {
		return OperationConfig{}, fmt.Errorf("no configuration found for operation: %s", operationKey)
	}
	return config, nil
}

// Generate prompt sections list for a task
func generatePromptSections(task TaskItem) ([]string, error) {
	operationConfig, err := selectPromptAssembly(task.Type, task.Operation)
	if err != nil {
		return nil, err
	}
	
	// Combine base sections with operation-specific sections
	allSections := append(promptsConfig.BaseSections, operationConfig.AdditionalSections...)
	
	return allSections, nil
}

// assemblePrompt reads and concatenates prompt sections into a full prompt
func assemblePrompt(sections []string) (string, error) {
	var promptBuilder strings.Builder
	
	promptBuilder.WriteString("# Ecosystem Manager Task Execution\n\n")
	promptBuilder.WriteString("You are executing a task for the Vrooli Ecosystem Manager.\n\n")
	promptBuilder.WriteString("---\n\n")
	
	for i, section := range sections {
		// Convert section path to file path
		// Handle both relative and absolute section paths
		filePath := filepath.Join("..", "prompts", section + ".md")
		
		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Try without adding .md extension (in case it's already included)
			filePath = filepath.Join("..", "prompts", section)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				log.Printf("Warning: Section file not found: %s", section)
				continue // Skip missing sections with warning
			}
		}
		
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read section %s: %v", section, err)
		}
		
		// Add section header
		promptBuilder.WriteString(fmt.Sprintf("## Section %d: %s\n\n", i+1, section))
		promptBuilder.WriteString(string(content))
		promptBuilder.WriteString("\n\n---\n\n")
	}
	
	return promptBuilder.String(), nil
}

// assemblePromptForTask generates a complete prompt for a specific task
func assemblePromptForTask(task TaskItem) (string, error) {
	sections, err := generatePromptSections(task)
	if err != nil {
		return "", fmt.Errorf("failed to generate sections: %v", err)
	}
	
	prompt, err := assemblePrompt(sections)
	if err != nil {
		return "", fmt.Errorf("failed to assemble prompt: %v", err)
	}
	
	// Add task-specific context to the prompt
	var taskContext strings.Builder
	taskContext.WriteString("\n\n## Task Context\n\n")
	taskContext.WriteString(fmt.Sprintf("**Task ID**: %s\n", task.ID))
	taskContext.WriteString(fmt.Sprintf("**Title**: %s\n", task.Title))
	taskContext.WriteString(fmt.Sprintf("**Type**: %s\n", task.Type))
	taskContext.WriteString(fmt.Sprintf("**Operation**: %s\n", task.Operation))
	taskContext.WriteString(fmt.Sprintf("**Category**: %s\n", task.Category))
	taskContext.WriteString(fmt.Sprintf("**Priority**: %s\n", task.Priority))
	
	if task.Requirements != nil && len(task.Requirements) > 0 {
		taskContext.WriteString("\n### Requirements\n")
		requirementsJSON, _ := json.MarshalIndent(task.Requirements, "", "  ")
		taskContext.WriteString(string(requirementsJSON))
	}
	
	if len(task.Notes) > 0 {
		taskContext.WriteString(fmt.Sprintf("\n### Notes\n%s\n", task.Notes))
	}
	
	return prompt + taskContext.String(), nil
}

// Queue management functions
func getQueueItems(status string) ([]TaskItem, error) {
	queuePath := filepath.Join(queueDir, status)
	
	files, err := filepath.Glob(filepath.Join(queuePath, "*.yaml"))
	if err != nil {
		return nil, err
	}
	
	var items []TaskItem
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		var item TaskItem
		if err := yaml.Unmarshal(data, &item); err != nil {
			continue
		}
		
		items = append(items, item)
	}
	
	return items, nil
}

func saveQueueItem(item TaskItem, status string) error {
	queuePath := filepath.Join(queueDir, status)
	filename := fmt.Sprintf("%s.yaml", item.ID)
	filePath := filepath.Join(queuePath, filename)
	
	data, err := yaml.Marshal(item)
	if err != nil {
		return err
	}
	
	return os.WriteFile(filePath, data, 0644)
}

func moveQueueItem(itemID, fromStatus, toStatus string) error {
	fromPath := filepath.Join(queueDir, fromStatus, fmt.Sprintf("%s.yaml", itemID))
	toPath := filepath.Join(queueDir, toStatus, fmt.Sprintf("%s.yaml", itemID))
	
	return os.Rename(fromPath, toPath)
}

// moveTask moves a task between queue states and updates timestamps
func moveTask(taskID, fromStatus, toStatus string) error {
	// Read the task
	fromPath := filepath.Join(queueDir, fromStatus, fmt.Sprintf("%s.yaml", taskID))
	data, err := os.ReadFile(fromPath)
	if err != nil {
		return fmt.Errorf("failed to read task: %v", err)
	}
	
	var task TaskItem
	if err := yaml.Unmarshal(data, &task); err != nil {
		return fmt.Errorf("failed to unmarshal task: %v", err)
	}
	
	// Update timestamps based on status change
	now := time.Now().Format(time.RFC3339)
	task.UpdatedAt = now
	
	switch toStatus {
	case "in-progress":
		task.StartedAt = now
		task.Status = "in-progress"
		task.CurrentPhase = "initialization"
	case "completed":
		task.CompletedAt = now
		task.Status = "completed"
		task.ProgressPercent = 100
	case "failed":
		task.CompletedAt = now
		task.Status = "failed"
	}
	
	// Save updated task to new location
	if err := saveQueueItem(task, toStatus); err != nil {
		return fmt.Errorf("failed to save task to %s: %v", toStatus, err)
	}
	
	// Remove from old location
	if err := os.Remove(fromPath); err != nil {
		return fmt.Errorf("failed to remove task from %s: %v", fromStatus, err)
	}
	
	log.Printf("Moved task %s from %s to %s", taskID, fromStatus, toStatus)
	return nil
}

// getTaskByID finds a task by ID across all queue statuses
func getTaskByID(taskID string) (*TaskItem, string, error) {
	statuses := []string{"pending", "in-progress", "completed", "failed"}
	
	for _, status := range statuses {
		filePath := filepath.Join(queueDir, status, fmt.Sprintf("%s.yaml", taskID))
		if _, err := os.Stat(filePath); err == nil {
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			
			var task TaskItem
			if err := yaml.Unmarshal(data, &task); err != nil {
				continue
			}
			
			return &task, status, nil
		}
	}
	
	return nil, "", fmt.Errorf("task not found: %s", taskID)
}

// Queue Processing Functions

// NewQueueProcessor creates a new queue processor
func NewQueueProcessor(interval time.Duration) *QueueProcessor {
	return &QueueProcessor{
		processInterval: interval,
		stopChannel:     make(chan bool),
	}
}

// Start begins the queue processing loop
func (qp *QueueProcessor) Start() {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	
	if qp.isRunning {
		log.Println("Queue processor already running")
		return
	}
	
	qp.isRunning = true
	go qp.processLoop()
	log.Println("Queue processor started")
}

// Stop halts the queue processing loop
func (qp *QueueProcessor) Stop() {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	
	if !qp.isRunning {
		return
	}
	
	qp.stopChannel <- true
	qp.isRunning = false
	log.Println("Queue processor stopped")
}

// processLoop is the main queue processing loop
func (qp *QueueProcessor) processLoop() {
	ticker := time.NewTicker(qp.processInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			qp.processQueue()
		case <-qp.stopChannel:
			return
		}
	}
}

// processQueue processes pending tasks in the queue
func (qp *QueueProcessor) processQueue() {
	// Check if there are already tasks in progress
	inProgressTasks, err := getQueueItems("in-progress")
	if err != nil {
		log.Printf("Error checking in-progress tasks: %v", err)
		return
	}
	
	// Limit concurrent tasks (configurable, defaulting to 1 for now)
	maxConcurrent := 1
	if len(inProgressTasks) >= maxConcurrent {
		log.Printf("Queue processor: %d tasks already in progress, skipping", len(inProgressTasks))
		return
	}
	
	// Get pending tasks
	pendingTasks, err := getQueueItems("pending")
	if err != nil {
		log.Printf("Error getting pending tasks: %v", err)
		return
	}
	
	if len(pendingTasks) == 0 {
		return // No tasks to process
	}
	
	// Sort tasks by priority (critical > high > medium > low)
	priorityOrder := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}
	
	// Find highest priority task
	var selectedTask *TaskItem
	highestPriority := 0
	
	for i, task := range pendingTasks {
		priority := priorityOrder[task.Priority]
		if priority > highestPriority {
			highestPriority = priority
			selectedTask = &pendingTasks[i]
		}
	}
	
	if selectedTask == nil {
		return
	}
	
	log.Printf("Processing task: %s - %s", selectedTask.ID, selectedTask.Title)
	
	// Move task to in-progress
	if err := moveTask(selectedTask.ID, "pending", "in-progress"); err != nil {
		log.Printf("Failed to move task to in-progress: %v", err)
		return
	}
	
	// Process the task asynchronously
	go qp.executeTask(*selectedTask)
}

// executeTask executes a single task
func (qp *QueueProcessor) executeTask(task TaskItem) {
	log.Printf("Executing task %s: %s", task.ID, task.Title)
	
	// Generate the full prompt for the task
	prompt, err := assemblePromptForTask(task)
	if err != nil {
		log.Printf("Failed to assemble prompt for task %s: %v", task.ID, err)
		qp.handleTaskFailure(task, fmt.Sprintf("Prompt assembly failed: %v", err))
		return
	}
	
	// Update task progress
	task.CurrentPhase = "prompt_assembled"
	task.ProgressPercent = 25
	saveQueueItem(task, "in-progress")
	
	// Call Claude Code resource
	result, err := callClaudeCode(prompt, task)
	if err != nil {
		log.Printf("Failed to execute task %s with Claude Code: %v", task.ID, err)
		qp.handleTaskFailure(task, fmt.Sprintf("Claude Code execution failed: %v", err))
		return
	}
	
	// Process the result
	if result.Success {
		log.Printf("Task %s completed successfully", task.ID)
		
		// Update task with results
		task.Results = map[string]interface{}{
			"success": true,
			"message": result.Message,
			"output":  result.Output,
		}
		task.ProgressPercent = 100
		task.CurrentPhase = "completed"
		saveQueueItem(task, "in-progress")
		
		// Move to completed
		if err := moveTask(task.ID, "in-progress", "completed"); err != nil {
			log.Printf("Failed to move task %s to completed: %v", task.ID, err)
		}
	} else {
		log.Printf("Task %s failed: %s", task.ID, result.Error)
		qp.handleTaskFailure(task, result.Error)
	}
}

// handleTaskFailure handles a failed task
func (qp *QueueProcessor) handleTaskFailure(task TaskItem, errorMsg string) {
	task.Results = map[string]interface{}{
		"success": false,
		"error":   errorMsg,
	}
	task.CurrentPhase = "failed"
	saveQueueItem(task, "in-progress")
	
	if err := moveTask(task.ID, "in-progress", "failed"); err != nil {
		log.Printf("Failed to move task %s to failed: %v", task.ID, err)
	}
}

// discoverResources scans the filesystem for available resources
func discoverResources() ([]ResourceInfo, error) {
	resourcesPath := "../../../resources"
	var resources []ResourceInfo
	
	// Check if resources directory exists
	if _, err := os.Stat(resourcesPath); os.IsNotExist(err) {
		return resources, nil
	}
	
	// Scan for directories with service.json files
	pattern := filepath.Join(resourcesPath, "*/service.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan resources: %v", err)
	}
	
	for _, match := range matches {
		// Extract resource name from path
		resourceDir := filepath.Dir(match)
		resourceName := filepath.Base(resourceDir)
		
		// Read service.json
		var serviceConfig ServiceConfig
		data, err := os.ReadFile(match)
		if err != nil {
			log.Printf("Warning: could not read service.json for %s: %v", resourceName, err)
			continue
		}
		
		if err := json.Unmarshal(data, &serviceConfig); err != nil {
			log.Printf("Warning: could not parse service.json for %s: %v", resourceName, err)
			continue
		}
		
		// Check resource health
		healthy := checkResourceHealth(resourceName, resourceDir)
		
		resource := ResourceInfo{
			Name:        resourceName,
			Path:        resourceDir,
			Port:        serviceConfig.Port,
			Category:    serviceConfig.Category,
			Description: serviceConfig.Description,
			Version:     serviceConfig.Version,
			Healthy:     healthy,
		}
		
		resources = append(resources, resource)
	}
	
	return resources, nil
}

// discoverScenarios scans the filesystem for available scenarios
func discoverScenarios() ([]ScenarioInfo, error) {
	scenariosPath := "../../../scenarios"
	var scenarios []ScenarioInfo
	
	// Check if scenarios directory exists
	if _, err := os.Stat(scenariosPath); os.IsNotExist(err) {
		return scenarios, nil
	}
	
	// Scan for directories with PRD.md files
	pattern := filepath.Join(scenariosPath, "*/PRD.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan scenarios: %v", err)
	}
	
	for _, match := range matches {
		// Extract scenario name from path
		scenarioDir := filepath.Dir(match)
		scenarioName := filepath.Base(scenarioDir)
		
		// Skip self (ecosystem-manager)
		if scenarioName == "ecosystem-manager" {
			continue
		}
		
		// Get PRD status
		prdStatus := getScenarioPRDStatus(scenarioName, match)
		
		// Check scenario health
		healthy := checkScenarioHealth(scenarioName, scenarioDir)
		
		// Try to extract description from PRD
		description := extractDescriptionFromPRD(match)
		
		scenario := ScenarioInfo{
			Name:            scenarioName,
			Path:            scenarioDir,
			Category:        inferScenarioCategory(scenarioName, description),
			Description:     description,
			PRDComplete:     prdStatus.CompletionPercentage,
			Healthy:         healthy,
			P0Requirements:  prdStatus.P0Requirements,
			P0Completed:     prdStatus.P0Completed,
		}
		
		scenarios = append(scenarios, scenario)
	}
		
	return scenarios, nil
}

// checkResourceHealth checks if a resource is healthy
func checkResourceHealth(resourceName, resourceDir string) bool {
	// Try to find and execute health check script
	healthScript := filepath.Join(resourceDir, "lib", "health.sh")
	if _, err := os.Stat(healthScript); os.IsNotExist(err) {
		// No health script found, assume healthy if service.json exists
		return true
	}
	
	// TODO: Actually execute the health script
	// For now, just return true to avoid blocking
	return true
}

// checkScenarioHealth checks if a scenario is healthy
func checkScenarioHealth(scenarioName, scenarioDir string) bool {
	// Check if scenario has basic structure
	requiredFiles := []string{"PRD.md", "README.md"}
	
	for _, file := range requiredFiles {
		filePath := filepath.Join(scenarioDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return false
		}
	}
	
	return true
}

// getScenarioPRDStatus parses a PRD.md file to get completion status
func getScenarioPRDStatus(scenarioName, prdPath string) PRDStatus {
	status := PRDStatus{}
	
	// Read PRD file
	data, err := os.ReadFile(prdPath)
	if err != nil {
		log.Printf("Warning: could not read PRD for %s: %v", scenarioName, err)
		return status
	}
	
	content := string(data)
	lines := strings.Split(content, "\n")
	
	// Parse checkboxes to count requirements
	totalRequirements := 0
	completedRequirements := 0
	p0Requirements := 0
	p0Completed := 0
	p1Requirements := 0
	p1Completed := 0
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Look for checkbox patterns
		if strings.Contains(trimmed, "- [ ]") || strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
			totalRequirements++
			
			// Check if completed
			if strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
				completedRequirements++
			}
			
			// Check priority by looking at context
			contextLines := ""
			for j := max(0, i-3); j <= min(len(lines)-1, i+1); j++ {
				contextLines += strings.ToLower(lines[j]) + " "
			}
			
			if strings.Contains(contextLines, "p0") || strings.Contains(contextLines, "priority 0") {
				p0Requirements++
				if strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
					p0Completed++
				}
			} else if strings.Contains(contextLines, "p1") || strings.Contains(contextLines, "priority 1") {
				p1Requirements++
				if strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
					p1Completed++
				}
			}
		}
	}
	
	// Calculate completion percentage
	completionPercentage := 0
	if totalRequirements > 0 {
		completionPercentage = (completedRequirements * 100) / totalRequirements
	}
	
	status.CompletionPercentage = completionPercentage
	status.P0Requirements = p0Requirements
	status.P0Completed = p0Completed
	status.P1Requirements = p1Requirements
	status.P1Completed = p1Completed
	
	return status
}

// extractDescriptionFromPRD extracts description from PRD file
func extractDescriptionFromPRD(prdPath string) string {
	data, err := os.ReadFile(prdPath)
	if err != nil {
		return "No description available"
	}
	
	content := string(data)
	lines := strings.Split(content, "\n")
	
	// Look for overview or description section
	for i, line := range lines {
		trimmed := strings.TrimSpace(strings.ToLower(line))
		if strings.Contains(trimmed, "## overview") || strings.Contains(trimmed, "## description") {
			// Get the next few non-empty lines
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine != "" && !strings.HasPrefix(nextLine, "#") {
					return nextLine
				}
			}
		}
	}
	
	return "No description available"
}

// inferScenarioCategory attempts to categorize a scenario based on name and description
func inferScenarioCategory(name, description string) string {
	lower := strings.ToLower(name + " " + description)
	
	if strings.Contains(lower, "ai") || strings.Contains(lower, "ml") || strings.Contains(lower, "llm") {
		return "ai-tools"
	}
	if strings.Contains(lower, "business") || strings.Contains(lower, "invoice") || strings.Contains(lower, "finance") {
		return "business"
	}
	if strings.Contains(lower, "automat") || strings.Contains(lower, "workflow") {
		return "automation"
	}
	if strings.Contains(lower, "personal") || strings.Contains(lower, "life") {
		return "personal"
	}
	if strings.Contains(lower, "game") || strings.Contains(lower, "entertainment") {
		return "entertainment"
	}
	
	return "productivity" // default
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// callClaudeCode calls the Claude Code resource (stub implementation)
func callClaudeCode(prompt string, task TaskItem) (*ClaudeCodeResponse, error) {
	// This is a stub implementation
	// In production, this would make an actual HTTP request to the Claude Code resource
	
	claudeCodeURL := os.Getenv("CLAUDE_CODE_URL")
	if claudeCodeURL == "" {
		// Default to local Claude Code instance
		claudeCodeURL = "http://localhost:27182/api/execute"
	}
	
	// For now, return a simulated response
	// TODO: Implement actual HTTP request to Claude Code
	log.Printf("Would send prompt to Claude Code at %s (length: %d characters)", claudeCodeURL, len(prompt))
	log.Printf("Task context: ID=%s, Type=%s, Operation=%s, Category=%s", task.ID, task.Type, task.Operation, task.Category)
	
	// Simulate processing time
	time.Sleep(2 * time.Second)
	
	// Return simulated success for testing
	return &ClaudeCodeResponse{
		Success: true,
		Message: "Task processed successfully (simulated)",
		Output:  fmt.Sprintf("Simulated output for task %s", task.ID),
	}, nil
	
	// Actual implementation would be:
	/*
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}
	
	resp, err := http.Post(claudeCodeURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call Claude Code: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	var response ClaudeCodeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	
	return &response, nil
	*/
}

// API Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "healthy",
		"service": "ecosystem-manager",
		"version": "1.0.0",
		"supported_operations": []string{"resource-generator", "resource-improver", "scenario-generator", "scenario-improver"},
		"timestamp": time.Now().Unix(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	
	taskType := r.URL.Query().Get("type")     // filter by resource/scenario
	operation := r.URL.Query().Get("operation") // filter by generator/improver
	category := r.URL.Query().Get("category")   // filter by category
	
	items, err := getQueueItems(status)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get tasks: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Apply filters
	var filteredItems []TaskItem
	for _, item := range items {
		if taskType != "" && item.Type != taskType {
			continue
		}
		if operation != "" && item.Operation != operation {
			continue
		}
		if category != "" && item.Category != category {
			continue
		}
		filteredItems = append(filteredItems, item)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredItems)
}

func createTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task TaskItem
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	// Validate task type and operation
	validTypes := []string{"resource", "scenario"}
	validOperations := []string{"generator", "improver"}
	
	if !contains(validTypes, task.Type) {
		http.Error(w, fmt.Sprintf("Invalid type: %s. Must be one of: %v", task.Type, validTypes), http.StatusBadRequest)
		return
	}
	
	if !contains(validOperations, task.Operation) {
		http.Error(w, fmt.Sprintf("Invalid operation: %s. Must be one of: %v", task.Operation, validOperations), http.StatusBadRequest)
		return
	}
	
	// Validate that we have configuration for this operation
	_, err := selectPromptAssembly(task.Type, task.Operation)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unsupported operation combination: %v", err), http.StatusBadRequest)
		return
	}
	
	// Set defaults
	if task.ID == "" {
		timestamp := time.Now().Format("20060102-150405")
		task.ID = fmt.Sprintf("%s-%s-%s", task.Type, task.Operation, timestamp)
	}
	
	if task.Status == "" {
		task.Status = "pending"
	}
	
	if task.CreatedAt == "" {
		task.CreatedAt = time.Now().Format(time.RFC3339)
	}
	
	task.UpdatedAt = time.Now().Format(time.RFC3339)
	
	// Save to pending queue
	if err := saveQueueItem(task, "pending"); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]
	
	// Search across all queue statuses
	statuses := []string{"pending", "in-progress", "completed", "failed"}
	
	for _, status := range statuses {
		filePath := filepath.Join(queueDir, status, fmt.Sprintf("%s.yaml", taskID))
		if _, err := os.Stat(filePath); err == nil {
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			
			var task TaskItem
			if err := yaml.Unmarshal(data, &task); err != nil {
				continue
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)
			return
		}
	}
	
	http.Error(w, "Task not found", http.StatusNotFound)
}

func updateTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]
	
	var update struct {
		Status          string `json:"status"`
		ProgressPercent int    `json:"progress_percentage"`
		CurrentPhase    string `json:"current_phase"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	// Find current task location
	statuses := []string{"pending", "in-progress", "completed", "failed"}
	var currentStatus string
	var task TaskItem
	
	for _, status := range statuses {
		filePath := filepath.Join(queueDir, status, fmt.Sprintf("%s.yaml", taskID))
		if _, err := os.Stat(filePath); err == nil {
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			
			if err := yaml.Unmarshal(data, &task); err != nil {
				continue
			}
			
			currentStatus = status
			break
		}
	}
	
	if currentStatus == "" {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	// Update task
	if update.Status != "" && update.Status != currentStatus {
		task.Status = update.Status
		
		// Move file to new status directory
		if err := moveQueueItem(taskID, currentStatus, update.Status); err != nil {
			http.Error(w, fmt.Sprintf("Failed to move task: %v", err), http.StatusInternalServerError)
			return
		}
	}
	
	if update.ProgressPercent > 0 {
		task.ProgressPercent = update.ProgressPercent
	}
	
	if update.CurrentPhase != "" {
		task.CurrentPhase = update.CurrentPhase
	}
	
	task.UpdatedAt = time.Now().Format(time.RFC3339)
	
	// Save updated task
	if err := saveQueueItem(task, task.Status); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func getOperationsHandler(w http.ResponseWriter, r *http.Request) {
	operations := make(map[string]interface{})
	
	for key, config := range promptsConfig.Operations {
		operations[key] = map[string]interface{}{
			"name":        config.Name,
			"type":        config.Type,
			"target":      config.Target,
			"description": config.Description,
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(operations)
}

// getResourcesHandler returns all discovered resources
func getResourcesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	resources, err := discoverResources()
	if err != nil {
		log.Printf("Error discovering resources: %v", err)
		http.Error(w, fmt.Sprintf("Failed to discover resources: %v", err), http.StatusInternalServerError)
		return
	}
	
	if err := json.NewEncoder(w).Encode(resources); err != nil {
		log.Printf("Error encoding resources response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// getScenariosHandler returns all discovered scenarios
func getScenariosHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	scenarios, err := discoverScenarios()
	if err != nil {
		log.Printf("Error discovering scenarios: %v", err)
		http.Error(w, fmt.Sprintf("Failed to discover scenarios: %v", err), http.StatusInternalServerError)
		return
	}
	
	if err := json.NewEncoder(w).Encode(scenarios); err != nil {
		log.Printf("Error encoding scenarios response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// getResourceStatusHandler returns detailed status for a specific resource
func getResourceStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	vars := mux.Vars(r)
	resourceName := vars["name"]
	
	if resourceName == "" {
		http.Error(w, "Resource name is required", http.StatusBadRequest)
		return
	}
	
	// Find the resource
	resources, err := discoverResources()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to discover resources: %v", err), http.StatusInternalServerError)
		return
	}
	
	for _, resource := range resources {
		if resource.Name == resourceName {
			if err := json.NewEncoder(w).Encode(resource); err != nil {
				log.Printf("Error encoding resource status response: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}
	
	http.Error(w, "Resource not found", http.StatusNotFound)
}

// getScenarioStatusHandler returns detailed status for a specific scenario
func getScenarioStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	
	if scenarioName == "" {
		http.Error(w, "Scenario name is required", http.StatusBadRequest)
		return
	}
	
	// Find the scenario
	scenarios, err := discoverScenarios()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to discover scenarios: %v", err), http.StatusInternalServerError)
		return
	}
	
	for _, scenario := range scenarios {
		if scenario.Name == scenarioName {
			if err := json.NewEncoder(w).Encode(scenario); err != nil {
				log.Printf("Error encoding scenario status response: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}
	
	http.Error(w, "Scenario not found", http.StatusNotFound)
}

func getTaskPromptHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]
	
	// Find task
	statuses := []string{"pending", "in-progress", "completed", "failed"}
	var task TaskItem
	
	for _, status := range statuses {
		filePath := filepath.Join(queueDir, status, fmt.Sprintf("%s.yaml", taskID))
		if _, err := os.Stat(filePath); err == nil {
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			
			if err := yaml.Unmarshal(data, &task); err != nil {
				continue
			}
			break
		}
	}
	
	if task.ID == "" {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	// Generate prompt sections
	sections, err := generatePromptSections(task)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate prompt: %v", err), http.StatusInternalServerError)
		return
	}
	
	operationConfig, _ := selectPromptAssembly(task.Type, task.Operation)
	
	response := map[string]interface{}{
		"task_id":           task.ID,
		"operation":         fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt_sections":   sections,
		"operation_config":  operationConfig,
		"task_details":      task,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getAssembledPromptHandler returns the fully assembled prompt for a task
func getAssembledPromptHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]
	
	// Find task
	task, status, err := getTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	// Generate the full assembled prompt
	prompt, err := assemblePromptForTask(*task)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assemble prompt: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Get operation config for metadata
	operationConfig, _ := selectPromptAssembly(task.Type, task.Operation)
	
	response := map[string]interface{}{
		"task_id":          task.ID,
		"operation":        fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt":           prompt,
		"prompt_length":    len(prompt),
		"operation_config": operationConfig,
		"task_status":      status,
		"task_details":     task,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func enableCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		handler.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// Initialize queue processor
	queueProcessorEnabled := os.Getenv("QUEUE_PROCESSOR_ENABLED")
	if queueProcessorEnabled == "" {
		queueProcessorEnabled = "true" // Default to enabled
	}
	
	if queueProcessorEnabled == "true" {
		// Create and start queue processor with 30-second interval
		queueProcessor = NewQueueProcessor(30 * time.Second)
		queueProcessor.Start()
		log.Println("Queue processor enabled and started")
		
		// Handle graceful shutdown
		defer func() {
			if queueProcessor != nil {
				queueProcessor.Stop()
				log.Println("Queue processor stopped")
			}
		}()
	} else {
		log.Println("Queue processor disabled")
	}
	
	r := mux.NewRouter()
	
	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")
	
	// Task management
	r.HandleFunc("/api/tasks", getTasksHandler).Methods("GET")
	r.HandleFunc("/api/tasks", createTaskHandler).Methods("POST")
	r.HandleFunc("/api/tasks/{id}", getTaskHandler).Methods("GET")
	r.HandleFunc("/api/tasks/{id}/status", updateTaskStatusHandler).Methods("PUT")
	r.HandleFunc("/api/tasks/{id}/prompt", getTaskPromptHandler).Methods("GET")
	r.HandleFunc("/api/tasks/{id}/prompt/assembled", getAssembledPromptHandler).Methods("GET")
	
	// Configuration
	r.HandleFunc("/api/operations", getOperationsHandler).Methods("GET")
	
	// Discovery endpoints
	r.HandleFunc("/api/resources", getResourcesHandler).Methods("GET")
	r.HandleFunc("/api/scenarios", getScenariosHandler).Methods("GET")
	r.HandleFunc("/api/resources/{name}/status", getResourceStatusHandler).Methods("GET")
	r.HandleFunc("/api/scenarios/{name}/status", getScenarioStatusHandler).Methods("GET")
	
	// Static file serving for UI
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./ui/")))
	
	handler := enableCORS(r)
	
	log.Printf("Ecosystem Manager API starting on port %s", port)
	log.Printf("Supporting operations: %v", getOperationNames())
	log.Printf("Queue processor: %s", queueProcessorEnabled)
	
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func getOperationNames() []string {
	var names []string
	for key := range promptsConfig.Operations {
		names = append(names, key)
	}
	return names
}