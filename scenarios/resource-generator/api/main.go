package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

// Core data structures
type Resource struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Template    string                 `json:"template"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type QueueItem struct {
	ID           string                 `json:"id" yaml:"id"`
	Title        string                 `json:"title" yaml:"title"`
	Description  string                 `json:"description" yaml:"description"`
	Type         string                 `json:"type" yaml:"type"`
	Template     string                 `json:"template" yaml:"template"`
	ResourceName string                 `json:"resource_name" yaml:"resource_name"`
	Priority     string                 `json:"priority" yaml:"priority"`
	Requirements map[string]interface{} `json:"requirements" yaml:"requirements"`
	Status       string                 `json:"status" yaml:"status"`
	CreatedBy    string                 `json:"created_by" yaml:"created_by"`
	CreatedAt    time.Time              `json:"created_at" yaml:"created_at"`
	AttemptCount int                    `json:"attempt_count" yaml:"attempt_count"`
}

type GenerationResult struct {
	QueueItemID   string                 `json:"queue_item_id"`
	ResourceName  string                 `json:"resource_name"`
	Success       bool                   `json:"success"`
	FilesCreated  []string               `json:"files_created"`
	PortAllocated int                    `json:"port_allocated"`
	TestResults   []string               `json:"test_results"`
	Metrics       map[string]interface{} `json:"metrics"`
	CompletedAt   time.Time              `json:"completed_at"`
	FailureReason string                 `json:"failure_reason,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type HealthStatus struct {
	Status           string    `json:"status"`
	QueueStatus      string    `json:"queue_status"`
	PendingItems     int       `json:"pending_items"`
	ProcessingActive bool      `json:"processing_active"`
	LastProcessed    time.Time `json:"last_processed,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
}

// Global configuration
var (
	queueDir          = getEnv("QUEUE_DIR", "../queue")
	claudeCodePath    string // Cached path to resource-claude-code binary
	processingActive  = false
	processingMutex   sync.Mutex
	lastProcessedTime time.Time
	shutdownChan      = make(chan os.Signal, 1)
)

// ==================== PROMPT LOADING ====================

// Load prompt file and process {{INCLUDE}} directives
func loadPromptWithIncludes(promptPath string) (string, error) {
	content, err := ioutil.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s: %v", promptPath, err)
	}
	
	// Process {{INCLUDE}} directives
	includePattern := regexp.MustCompile(`\{\{INCLUDE:\s*([^\}]+)\}\}`)
	processed := includePattern.ReplaceAllStringFunc(string(content), func(match string) string {
		// Extract the file path from the match
		parts := strings.Split(match, ":")
		if len(parts) < 2 {
			log.Printf("Invalid include directive: %s", match)
			return match
		}
		
		// Clean up the path
		includePath := strings.TrimSpace(strings.TrimSuffix(parts[1], "}}"))
		
		// Handle absolute vs relative paths
		if !filepath.IsAbs(includePath) {
			// If relative, make it relative to prompt directory
			includePath = filepath.Join(filepath.Dir(promptPath), includePath)
		} else {
			// Convert absolute path to be relative to home directory
			includePath = filepath.Join(os.Getenv("HOME"), "Vrooli", includePath)
		}
		
		// Read the included file
		includeContent, err := ioutil.ReadFile(includePath)
		if err != nil {
			log.Printf("Failed to read include file %s: %v", includePath, err)
			return fmt.Sprintf("<!-- Failed to include %s: %v -->", includePath, err)
		}
		
		return string(includeContent)
	})
	
	return processed, nil
}

// Load and prepare the main generation prompt
func prepareGenerationPrompt(item *QueueItem) (string, error) {
	// Load the main prompt template with includes
	promptPath := filepath.Join(filepath.Dir(queueDir), "prompts", "main-prompt.md")
	promptTemplate, err := loadPromptWithIncludes(promptPath)
	if err != nil {
		// Fallback to a basic prompt if loading fails
		log.Printf("Failed to load prompt template: %v. Using fallback prompt.", err)
		return prepareFallbackPrompt(item), nil
	}
	
	// Add specific resource details to the prompt
	additionalContext := fmt.Sprintf(`

## SPECIFIC RESOURCE REQUEST

### Resource Details
- **Name**: %s
- **Template**: %s  
- **Type**: %s
- **Description**: %s
- **Priority**: %s
- **Requirements**: %v

### Implementation Instructions
1. Create complete resource at /home/matthalloran8/Vrooli/resources/%s/
2. Use template: %s as the base pattern
3. Ensure full v2.0 contract compliance
4. Allocate appropriate ports based on resource type
5. Generate comprehensive documentation
6. Create integration tests
7. Set up CLI commands

### Expected Deliverables
- Complete directory structure with all required files
- Working health checks and lifecycle hooks
- CLI integration script
- Comprehensive README.md
- Integration test suite
- Port allocation registration

Please proceed with generating this resource now. Focus on production-ready quality and seamless integration with the Vrooli ecosystem.`,
		item.ResourceName,
		item.Template,
		item.Type,
		item.Description,
		item.Priority,
		item.Requirements,
		item.ResourceName,
		item.Template,
	)
	
	return promptTemplate + additionalContext, nil
}

// Fallback prompt if template loading fails
func prepareFallbackPrompt(item *QueueItem) string {
	return fmt.Sprintf(`You are creating a new Vrooli resource with full v2.0 contract compliance.

RESOURCE DETAILS:
Name: %s
Template: %s
Type: %s
Description: %s

REQUIREMENTS:
%v

CRITICAL REQUIREMENTS:
1. Create the complete resource directory structure at /home/matthalloran8/Vrooli/resources/%s/
2. Implement v2.0 contract compliance:
   - lib/ directory with health checks and lifecycle hooks
   - .vrooli/service.json with proper configuration
   - CLI integration script
   - README.md with comprehensive documentation
3. Allocate an appropriate port from the correct range:
   - 20000-29999: Core resources (databases, queues)
   - 30000-39999: Application scenarios
   - 40000-49999: Dynamic allocation
4. Create initialization scripts if needed
5. Implement health check endpoints
6. Add error handling and logging
7. Generate integration tests

Please implement the complete resource now. Create all necessary files and directories.
Report back with:
- List of all files created
- Port allocated (if applicable)
- Any issues encountered
- Whether the resource is ready for use

Focus on creating a production-ready resource that other scenarios can immediately use.`,
		item.ResourceName, item.Template, item.Type, item.Description,
		item.Requirements, item.ResourceName)
}

// ==================== CLAUDE CODE INTEGRATION ====================

// Call Claude Code CLI to execute resource generation
func callClaudeCode(prompt string) (string, error) {
	if claudeCodePath == "" {
		return "", fmt.Errorf("resource-claude-code not found in PATH")
	}
	
	// Set working directory to resources folder for context
	resourcesDir := getEnv("RESOURCES_DIR", filepath.Join(os.Getenv("HOME"), "Vrooli/resources"))
	
	log.Printf("Executing Claude Code for resource generation (prompt length: %d characters)", len(prompt))
	
	// Retry logic with exponential backoff
	maxRetries := 3
	baseDelay := 10 * time.Second // Longer delay for complex generation tasks
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1)) // Exponential backoff: 10s, 20s, 40s
			log.Printf("Retrying Claude Code execution after %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}
		
		var stdout, stderr bytes.Buffer
		
		// Run with longer timeout for resource generation (15 minutes)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		
		cmd := exec.CommandContext(ctx, claudeCodePath, "run", "--prompt", prompt, "--max-turns", "15")
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
			return "", fmt.Errorf("Claude Code execution failed after %d attempts: %v", maxRetries, err)
		}
	}
	
	return "", fmt.Errorf("Claude Code execution failed unexpectedly")
}

// Parse Claude Code's response to extract generation results
func parseClaudeCodeResponse(response string, queueItem *QueueItem) *GenerationResult {
	result := &GenerationResult{
		QueueItemID:  queueItem.ID,
		ResourceName: queueItem.ResourceName,
		Success:      true, // Trust Claude Code by default
		FilesCreated: []string{},
		TestResults:  []string{},
		Metrics:      map[string]interface{}{},
	}
	
	// Simple check for critical failures only
	lowerResponse := strings.ToLower(response)
	
	// Only mark as failed for explicit critical errors
	if strings.Contains(lowerResponse, "critical error") || 
	   strings.Contains(lowerResponse, "fatal:") ||
	   strings.Contains(lowerResponse, "cannot proceed") ||
	   strings.Contains(lowerResponse, "failed to create resource") {
		result.Success = false
		
		// Extract failure reason if present
		if idx := strings.Index(lowerResponse, "error:"); idx >= 0 {
			endIdx := strings.Index(response[idx:], "\n")
			if endIdx > 0 {
				result.FailureReason = strings.TrimSpace(response[idx:idx+endIdx])
			} else {
				result.FailureReason = "Resource generation encountered an error"
			}
		}
	}
	
	// Trust Claude Code's output - it will report what it did
	// Look for resource directory creation confirmation
	if strings.Contains(response, "/resources/"+queueItem.ResourceName) {
		result.FilesCreated = append(result.FilesCreated, 
			fmt.Sprintf("Resource directory: /home/matthalloran8/Vrooli/resources/%s", queueItem.ResourceName))
	}
	
	// Check for port allocation mentions
	portPattern := regexp.MustCompile(`(?i)port[:\s]+(\d{4,5})`)
	if matches := portPattern.FindStringSubmatch(response); len(matches) > 1 {
		if port, err := strconv.Atoi(matches[1]); err == nil {
			result.PortAllocated = port
		}
	}
	
	// Add basic metrics
	result.Metrics["response_length"] = len(response)
	result.Metrics["template_used"] = queueItem.Template
	result.Metrics["resource_name"] = queueItem.ResourceName
	
	// Log a summary of Claude's response for debugging
	if len(response) > 500 {
		log.Printf("Claude Code response summary (first 500 chars): %s...", response[:500])
	} else {
		log.Printf("Claude Code response: %s", response)
	}
	
	return result
}

// ==================== QUEUE PROCESSING ====================

// Load a queue item from YAML file
func loadQueueItem(filePath string) (*QueueItem, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var item QueueItem
	if err := yaml.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	
	// Set defaults if not specified
	if item.Status == "" {
		item.Status = "pending"
	}
	if item.Priority == "" {
		item.Priority = "medium"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	
	return &item, nil
}

// Save queue item to YAML file
func saveQueueItem(item *QueueItem, dir string) error {
	data, err := yaml.Marshal(item)
	if err != nil {
		return err
	}
	
	filename := fmt.Sprintf("%s.yaml", item.ID)
	filePath := filepath.Join(dir, filename)
	
	return ioutil.WriteFile(filePath, data, 0644)
}

// Select next resource to generate from queue
func selectNextGeneration() (*QueueItem, string, error) {
	pendingDir := filepath.Join(queueDir, "pending")
	
	files, err := filepath.Glob(filepath.Join(pendingDir, "*.yaml"))
	if err != nil {
		return nil, "", err
	}
	
	if len(files) == 0 {
		return nil, "", nil // No pending items
	}
	
	// Load all queue items
	var items []QueueItem
	fileMap := make(map[string]string) // Map item ID to file path
	
	for _, file := range files {
		item, err := loadQueueItem(file)
		if err != nil {
			log.Printf("Failed to load queue item %s: %v", file, err)
			continue
		}
		items = append(items, *item)
		fileMap[item.ID] = file
	}
	
	if len(items) == 0 {
		return nil, "", nil
	}
	
	// Simple priority selection (in production, would use more sophisticated algorithm)
	// Priority order: critical > high > medium > low
	priorityOrder := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}
	
	var selected *QueueItem
	highestPriority := 0
	
	for i := range items {
		priority := priorityOrder[items[i].Priority]
		if priority > highestPriority {
			highestPriority = priority
			selected = &items[i]
		}
	}
	
	if selected == nil && len(items) > 0 {
		// Fallback to first item if no priority match
		selected = &items[0]
	}
	
	return selected, fileMap[selected.ID], nil
}

// Process a resource generation request
func processResourceGeneration(item *QueueItem) (*GenerationResult, error) {
	log.Printf("Processing resource generation: %s (Template: %s)", item.ResourceName, item.Template)
	
	// Prepare prompt using the prompt loading system
	prompt, err := prepareGenerationPrompt(item)
	if err != nil {
		log.Printf("Warning: Failed to prepare prompt from template: %v", err)
		// Continue with fallback prompt
	}
	
	// Call Claude Code CLI to generate the resource
	log.Printf("Delegating resource generation to Claude Code...")
	response, err := callClaudeCode(prompt)
	if err != nil {
		return &GenerationResult{
			QueueItemID:   item.ID,
			ResourceName:  item.ResourceName,
			Success:       false,
			FailureReason: fmt.Sprintf("Claude Code execution failed: %v", err),
			CompletedAt:   time.Now(),
		}, nil
	}
	
	// Parse Claude's response to extract results
	result := parseClaudeCodeResponse(response, item)
	result.CompletedAt = time.Now()
	
	// Validate that resource was actually created
	resourcePath := filepath.Join(os.Getenv("HOME"), "Vrooli/resources", item.ResourceName)
	if _, err := os.Stat(resourcePath); os.IsNotExist(err) {
		result.Success = false
		if result.FailureReason == "" {
			result.FailureReason = "Resource directory was not created"
		}
	}
	
	return result, nil
}

// Process queue continuously
func processQueueContinuously() {
	log.Println("Starting queue processor...")
	
	for {
		select {
		case <-shutdownChan:
			log.Println("Shutting down queue processor...")
			return
		default:
			// Check if we should process
			processingMutex.Lock()
			if !processingActive {
				processingMutex.Unlock()
				time.Sleep(10 * time.Second)
				continue
			}
			processingMutex.Unlock()
			
			// Select next item
			item, filePath, err := selectNextGeneration()
			if err != nil {
				log.Printf("Error selecting next generation: %v", err)
				time.Sleep(30 * time.Second)
				continue
			}
			
			if item == nil {
				// No items to process
				time.Sleep(30 * time.Second)
				continue
			}
			
			// Move to in-progress
			inProgressDir := filepath.Join(queueDir, "in-progress")
			newPath := filepath.Join(inProgressDir, filepath.Base(filePath))
			if err := os.Rename(filePath, newPath); err != nil {
				log.Printf("Failed to move item to in-progress: %v", err)
				continue
			}
			
			// Process the generation
			result, err := processResourceGeneration(item)
			if err != nil {
				log.Printf("Error processing generation: %v", err)
				// Move to failed
				item.Status = "failed"
				item.AttemptCount++
				failedDir := filepath.Join(queueDir, "failed")
				saveQueueItem(item, failedDir)
				os.Remove(newPath)
				continue
			}
			
			// Move to completed or failed based on result
			if result.Success {
				item.Status = "completed"
				completedDir := filepath.Join(queueDir, "completed")
				saveQueueItem(item, completedDir)
				log.Printf("Successfully generated resource: %s", item.ResourceName)
			} else {
				item.Status = "failed"
				item.AttemptCount++
				failedDir := filepath.Join(queueDir, "failed")
				saveQueueItem(item, failedDir)
				log.Printf("Failed to generate resource: %s - %s", item.ResourceName, result.FailureReason)
			}
			
			// Remove from in-progress
			os.Remove(newPath)
			lastProcessedTime = time.Now()
			
			// Brief pause before next item
			time.Sleep(5 * time.Second)
		}
	}
}

// ==================== HTTP HANDLERS ====================

func healthHandler(w http.ResponseWriter, r *http.Request) {
	pendingCount := 0
	pendingDir := filepath.Join(queueDir, "pending")
	if files, err := filepath.Glob(filepath.Join(pendingDir, "*.yaml")); err == nil {
		pendingCount = len(files)
	}
	
	processingMutex.Lock()
	active := processingActive
	processingMutex.Unlock()
	
	status := HealthStatus{
		Status:           "healthy",
		QueueStatus:      "ready",
		PendingItems:     pendingCount,
		ProcessingActive: active,
		LastProcessed:    lastProcessedTime,
		Timestamp:        time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    status,
	})
}

func queueHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case "GET":
		// Return queue status
		var pending, inProgress, completed, failed []string
		
		// Count items in each queue directory
		dirs := map[string]*[]string{
			"pending":     &pending,
			"in-progress": &inProgress,
			"completed":   &completed,
			"failed":      &failed,
		}
		
		for dir, list := range dirs {
			path := filepath.Join(queueDir, dir)
			if files, err := filepath.Glob(filepath.Join(path, "*.yaml")); err == nil {
				for _, f := range files {
					*list = append(*list, filepath.Base(f))
				}
			}
		}
		
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"pending":     pending,
				"in_progress": inProgress,
				"completed":   completed,
				"failed":      failed,
			},
		})
		
	case "POST":
		// Add new item to queue
		var item QueueItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "Invalid request body",
			})
			return
		}
		
		// Set defaults
		if item.ID == "" {
			item.ID = fmt.Sprintf("gen-%d", time.Now().Unix())
		}
		item.Status = "pending"
		item.CreatedAt = time.Now()
		
		// Save to pending queue
		pendingDir := filepath.Join(queueDir, "pending")
		if err := saveQueueItem(&item, pendingDir); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to save queue item: %v", err),
			})
			return
		}
		
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    item,
		})
		
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
	}
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}
	
	// Create queue item from request
	item := QueueItem{
		ID:           fmt.Sprintf("gen-%d", time.Now().Unix()),
		Title:        fmt.Sprintf("Generate %v", request["name"]),
		ResourceName: fmt.Sprintf("%v", request["name"]),
		Template:     fmt.Sprintf("%v", request["template"]),
		Type:         fmt.Sprintf("%v", request["type"]),
		Priority:     "high",
		Status:       "pending",
		CreatedAt:    time.Now(),
		Requirements: request,
	}
	
	// Save to queue
	pendingDir := filepath.Join(queueDir, "pending")
	if err := saveQueueItem(&item, pendingDir); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to queue generation: %v", err),
		})
		return
	}
	
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message":      "Resource generation queued",
			"id":           item.ID,
			"status":       "pending",
			"resourceName": item.ResourceName,
		},
	})
}

func templatesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	templates := []map[string]interface{}{
		{
			"id":          "ai-powered",
			"name":        "AI/ML Resource",
			"description": "Resource with model management, inference API, and GPU support",
			"category":    "ai",
		},
		{
			"id":          "data-processing",
			"name":        "Data Processing Resource",
			"description": "ETL pipelines, batch/stream processing, validation",
			"category":    "data",
		},
		{
			"id":          "workflow-automation",
			"name":        "Workflow Automation",
			"description": "Workflow engine, event-driven architecture, scheduling",
			"category":    "automation",
		},
		{
			"id":          "monitoring",
			"name":        "Monitoring & Observability",
			"description": "Metrics, logs, alerts, dashboards",
			"category":    "monitoring",
		},
		{
			"id":          "communication",
			"name":        "Communication Platform",
			"description": "Messaging, real-time features, protocol bridges",
			"category":    "communication",
		},
		{
			"id":          "security",
			"name":        "Security Resource",
			"description": "Scanning, authentication, secrets management",
			"category":    "security",
		},
	}
	
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    templates,
	})
}

func startProcessingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	processingMutex.Lock()
	processingActive = true
	processingMutex.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Queue processing started",
		},
	})
}

func stopProcessingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	processingMutex.Lock()
	processingActive = false
	processingMutex.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Queue processing stopped",
		},
	})
}

// ==================== UTILITIES ====================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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

// ==================== MAIN ====================

func main() {
	// Initialize
	initBinaryPaths()
	
	// Handle graceful shutdown
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	
	// Start queue processor in background
	processingActive = true // Auto-start processing
	go processQueueContinuously()
	
	// Set up HTTP routes
	port := getEnv("API_PORT", getEnv("PORT", ""))
	if port == "" {
		log.Fatal("No port specified. Set API_PORT or PORT environment variable")
	}
	
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/queue", queueHandler)
	http.HandleFunc("/api/resources/generate", generateHandler)
	http.HandleFunc("/api/templates", templatesHandler)
	http.HandleFunc("/api/processing/start", startProcessingHandler)
	http.HandleFunc("/api/processing/stop", stopProcessingHandler)
	
	log.Printf("Resource Generator API starting on port %s", port)
	log.Printf("Queue processing is ACTIVE")
	
	// Start server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}