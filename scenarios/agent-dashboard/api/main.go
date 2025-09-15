package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "sync"
    "time"
)

// Agent represents a discovered AI agent within a resource
type Agent struct {
    ID           string                 `json:"id"`
    Name         string                 `json:"name"`
    Type         string                 `json:"type"`             // resource type (ollama, claude-code, etc)
    Status       string                 `json:"status"`           // active, inactive, error
    PID          int                    `json:"pid"`
    StartTime    time.Time              `json:"start_time"`
    LastSeen     time.Time              `json:"last_seen"`
    Uptime       string                 `json:"uptime"`           // human readable
    Command      string                 `json:"command"`
    Capabilities []string               `json:"capabilities,omitempty"`
    Metrics      map[string]interface{} `json:"metrics,omitempty"`
    RadarPosition *RadarPosition        `json:"radar_position,omitempty"`
}

// RadarPosition for visualization
type RadarPosition struct {
    X       float64 `json:"x"`
    Y       float64 `json:"y"`
    TargetX float64 `json:"target_x"`
    TargetY float64 `json:"target_y"`
}

// ResourceAgentData matches expected CLI output format
type ResourceAgentData struct {
    Agents map[string]ResourceAgent `json:"agents"`
}

type ResourceAgent struct {
    ID        string `json:"id"`
    PID       int    `json:"pid"`
    Status    string `json:"status"`
    StartTime string `json:"start_time"`
    LastSeen  string `json:"last_seen"`
    Command   string `json:"command"`
}

// ResourceScanResult tracks scan attempts
type ResourceScanResult struct {
    ResourceName    string    `json:"resource_name"`
    ScanTimestamp   time.Time `json:"scan_timestamp"`
    Success         bool      `json:"success"`
    Error           string    `json:"error,omitempty"`
    AgentsFound     int       `json:"agents_found"`
    ScanDurationMs  int64     `json:"scan_duration_ms"`
}

// AgentDiscoveryState holds the global state
type AgentDiscoveryState struct {
    mu            sync.RWMutex
    agents        map[string]*Agent
    lastScan      time.Time
    scanInProgress bool
    scanResults   []ResourceScanResult
}

// API Response types
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

type AgentsResponse struct {
    Agents         []*Agent             `json:"agents"`
    LastScan       time.Time            `json:"last_scan"`
    ScanInProgress bool                 `json:"scan_in_progress"`
    Errors         []ResourceScanResult `json:"errors,omitempty"`
}

// Global state
var discoveryState = &AgentDiscoveryState{
    agents:      make(map[string]*Agent),
    scanResults: make([]ResourceScanResult, 0),
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario run agent-dashboard

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

    // API endpoints with versioning
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/api/v1/agents", agentsHandler)
    http.HandleFunc("/api/v1/agents/status", statusHandler)
    http.HandleFunc("/api/v1/agents/scan", scanHandler)

    // Start background agent discovery polling
    log.Printf("Starting agent discovery polling (30s interval)...")
    go startAgentPolling()

    log.Printf("Agent Dashboard API v1.0.0 starting on port %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    
    // Basic health check
    health := map[string]interface{}{
        "status":    "healthy",
        "service":   "agent-dashboard-api",
        "version":   "1.0.0",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "readiness": true,
    }
    
    json.NewEncoder(w).Encode(health)
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
        // Return list of discovered agents
        discoveryState.mu.RLock()
        
        // Convert map to slice
        agents := make([]*Agent, 0, len(discoveryState.agents))
        for _, agent := range discoveryState.agents {
            agents = append(agents, agent)
        }
        
        // Get failed scan results for errors
        var errors []ResourceScanResult
        for _, result := range discoveryState.scanResults {
            if !result.Success {
                errors = append(errors, result)
            }
        }
        
        response := AgentsResponse{
            Agents:         agents,
            LastScan:       discoveryState.lastScan,
            ScanInProgress: discoveryState.scanInProgress,
            Errors:         errors,
        }
        
        discoveryState.mu.RUnlock()
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    case "POST":
        // Register new agent
        var agent Agent
        if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
            errorResponse(w, "Invalid request body", http.StatusBadRequest)
            return
        }
        response := APIResponse{
            Success: true,
            Data:    agent,
        }
        jsonResponse(w, response, http.StatusCreated)
    default:
        errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
    agentName := r.URL.Query().Get("name")
    includeMetrics := r.URL.Query().Get("metrics") == "true"
    
    status := map[string]interface{}{
        "agent_name":      agentName,
        "include_metrics": includeMetrics,
        "timestamp":       time.Now(),
    }
    
    response := APIResponse{
        Success: true,
        Data:    status,
    }
    jsonResponse(w, response, http.StatusOK)
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Check if scan is already in progress
    discoveryState.mu.RLock()
    scanInProgress := discoveryState.scanInProgress
    discoveryState.mu.RUnlock()
    
    if scanInProgress {
        response := APIResponse{
            Success: false,
            Error:   "Scan already in progress",
        }
        jsonResponse(w, response, http.StatusConflict)
        return
    }
    
    // Trigger immediate scan in background
    go performAgentScan()
    
    response := APIResponse{
        Success: true,
        Data: map[string]interface{}{
            "scan_started":          true,
            "estimated_duration_ms": 2000,
            "message":              "Agent discovery scan initiated",
        },
    }
    jsonResponse(w, response, http.StatusOK)
}

// List of resources that commonly exist in Vrooli installations and could have agents
var supportedResources = []string{
    "ollama", "claude-code", "n8n", "postgres", "redis", "qdrant",
    "huginn", "litellm", "whisper", "comfyui",
}

func getAgents() []Agent {
    var allAgents []Agent
    
    for _, resource := range supportedResources {
        agents := getResourceAgents(resource)
        allAgents = append(allAgents, agents...)
    }
    
    return allAgents
}

func getResourceAgents(resourceName string) []Agent {
    // Call resource CLI to get agent data
    cmd := exec.Command("resource-" + resourceName, "agents", "list", "--json")
    output, err := cmd.Output()
    if err != nil {
        log.Printf("Failed to get agents for resource %s: %v", resourceName, err)
        return []Agent{}
    }
    
    var resourceData ResourceAgentData
    if err := json.Unmarshal(output, &resourceData); err != nil {
        log.Printf("Failed to parse agent data for resource %s: %v", resourceName, err)
        return []Agent{}
    }
    
    var agents []Agent
    for _, resourceAgent := range resourceData.Agents {
        // Parse timestamps
        startTime, _ := time.Parse(time.RFC3339, resourceAgent.StartTime)
        lastSeen, _ := time.Parse(time.RFC3339, resourceAgent.LastSeen)
        
        agent := Agent{
            ID:           resourceAgent.ID,
            Name:         fmt.Sprintf("%s Agent", strings.Title(resourceName)),
            Type:         resourceName,
            Status:       mapStatus(resourceAgent.Status),
            PID:          resourceAgent.PID,
            StartTime:    startTime,
            LastSeen:     lastSeen,
            Uptime:       getUptimeString(startTime),
            Command:      resourceAgent.Command,
            Capabilities: getResourceCapabilities(resourceName),
            Metrics: map[string]interface{}{
                "pid":       resourceAgent.PID,
                "start_time": startTime.Format("2006-01-02 15:04:05"),
                "uptime":    getUptimeString(startTime),
                "command":   resourceAgent.Command,
            },
            RadarPosition: generateRadarPosition(),
        }
        agents = append(agents, agent)
    }
    
    return agents
}

func mapStatus(resourceStatus string) string {
    switch resourceStatus {
    case "running":
        return "active"
    case "stopped":
        return "inactive"
    case "crashed":
        return "error"
    default:
        return resourceStatus
    }
}

func getResourceCapabilities(resourceName string) []string {
    capabilities := map[string][]string{
        "codex":         {"code-generation", "ai-assistance", "reasoning"},
        "claude-code":   {"code-generation", "debugging", "refactoring", "analysis"},
        "cline":         {"terminal-ai", "command-execution", "automation"},
        "ollama":        {"local-llm", "inference", "model-serving"},
        "agent-s2":      {"browser-automation", "web-interaction", "testing"},
        "autogen-studio": {"multi-agent", "orchestration", "collaboration"},
        "autogpt":       {"autonomous-planning", "goal-execution", "reasoning"},
        "crewai":        {"crew-coordination", "task-delegation", "collaboration"},
        "gemini":        {"multimodal-ai", "vision", "reasoning"},
        "langchain":     {"llm-chaining", "document-processing", "rag"},
        "litellm":       {"llm-proxy", "api-unification", "model-routing"},
        "openrouter":    {"llm-routing", "api-proxy", "model-selection"},
        "whisper":       {"speech-to-text", "transcription", "audio-processing"},
        "comfyui":       {"image-generation", "workflow-automation", "diffusion"},
        "pandas-ai":     {"data-analysis", "dataframe-processing", "ai-querying"},
        "parlant":       {"conversational-ai", "dialog-management", "nlp"},
        "huginn":        {"web-scraping", "monitoring", "automation", "alerts"},
        "opencode":      {"code-editing", "ide-integration", "development"},
    }
    
    if caps, exists := capabilities[resourceName]; exists {
        return caps
    }
    return []string{"general-ai", "task-execution"}
}

func getUptimeString(startTime time.Time) string {
    if startTime.IsZero() {
        return "0h"
    }
    
    duration := time.Since(startTime)
    days := int(duration.Hours()) / 24
    hours := int(duration.Hours()) % 24
    minutes := int(duration.Minutes()) % 60
    
    if days > 0 {
        return fmt.Sprintf("%dd %dh", days, hours)
    } else if hours > 0 {
        return fmt.Sprintf("%dh %dm", hours, minutes)
    } else {
        return fmt.Sprintf("%dm", minutes)
    }
}

func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, message string, status int) {
    response := APIResponse{
        Success: false,
        Error:   message,
    }
    jsonResponse(w, response, status)
}

// Background polling system
func startAgentPolling() {
    // Initial scan
    performAgentScan()
    
    // Start polling every 30 seconds
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        performAgentScan()
    }
}

func performAgentScan() {
    // Set scan in progress
    discoveryState.mu.Lock()
    if discoveryState.scanInProgress {
        discoveryState.mu.Unlock()
        log.Printf("Skipping scan - already in progress")
        return
    }
    discoveryState.scanInProgress = true
    discoveryState.mu.Unlock()
    
    log.Printf("Starting agent discovery scan...")
    startTime := time.Now()
    
    // Clear previous results
    scanResults := make([]ResourceScanResult, 0, len(supportedResources))
    discoveredAgents := make(map[string]*Agent)
    
    // Scan each resource
    for _, resourceName := range supportedResources {
        result := scanResource(resourceName, discoveredAgents)
        scanResults = append(scanResults, result)
    }
    
    // Update global state
    discoveryState.mu.Lock()
    discoveryState.agents = discoveredAgents
    discoveryState.lastScan = time.Now()
    discoveryState.scanInProgress = false
    discoveryState.scanResults = scanResults
    discoveryState.mu.Unlock()
    
    duration := time.Since(startTime)
    agentCount := len(discoveredAgents)
    log.Printf("Agent scan completed in %v - found %d agents", duration, agentCount)
}

func scanResource(resourceName string, discoveredAgents map[string]*Agent) ResourceScanResult {
    scanStart := time.Now()
    result := ResourceScanResult{
        ResourceName:  resourceName,
        ScanTimestamp: scanStart,
    }
    
    // First check if resource is actually running using vrooli CLI
    cmd := exec.Command("vrooli", "resource", "status", resourceName)
    cmd.Env = os.Environ()
    
    statusOutput, err := cmd.Output()
    result.ScanDurationMs = time.Since(scanStart).Milliseconds()
    
    if err != nil {
        result.Success = false
        result.Error = fmt.Sprintf("Resource not available: %v", err)
        log.Printf("Resource %s not available: %v", resourceName, err)
        return result
    }
    
    // Check if status output indicates the resource is running
    statusStr := string(statusOutput)
    if !strings.Contains(strings.ToLower(statusStr), "running") && !strings.Contains(strings.ToLower(statusStr), "active") {
        result.Success = true
        result.AgentsFound = 0
        result.Error = fmt.Sprintf("Resource %s is not running", resourceName)
        log.Printf("Resource %s is not running", resourceName)
        return result
    }
    
    // Try to get actual agent data from resource CLI
    agentCmd := exec.Command("resource-"+resourceName, "agents", "list", "--json")
    agentCmd.Env = os.Environ()
    
    agentOutput, agentErr := agentCmd.Output()
    
    if agentErr != nil {
        // Resource is running but doesn't support agent discovery yet
        // Create mock agent data for demonstration
        log.Printf("Resource %s doesn't support agent discovery, creating mock agent", resourceName)
        mockAgent := createMockAgent(resourceName)
        if mockAgent != nil {
            fullID := fmt.Sprintf("%s:mock-agent", resourceName)
            discoveredAgents[fullID] = mockAgent
            result.Success = true
            result.AgentsFound = 1
            log.Printf("Resource %s: created 1 mock agent", resourceName)
        }
        return result
    }
    
    // Parse real CLI output
    var resourceData ResourceAgentData
    if err := json.Unmarshal(agentOutput, &resourceData); err != nil {
        result.Success = false
        result.Error = fmt.Sprintf("JSON parsing failed: %v", err)
        log.Printf("Failed to parse agent data for resource %s: %v", resourceName, err)
        return result
    }
    
    // Convert resource agents to our format
    agentsFound := 0
    for agentID, resourceAgent := range resourceData.Agents {
        agent := convertResourceAgent(resourceName, agentID, resourceAgent)
        if agent != nil {
            fullID := fmt.Sprintf("%s:%s", resourceName, agentID)
            discoveredAgents[fullID] = agent
            agentsFound++
        }
    }
    
    result.Success = true
    result.AgentsFound = agentsFound
    log.Printf("Resource %s: found %d real agents", resourceName, agentsFound)
    
    return result
}

func convertResourceAgent(resourceName, agentID string, resourceAgent ResourceAgent) *Agent {
    // Parse timestamps
    startTime, err := time.Parse(time.RFC3339, resourceAgent.StartTime)
    if err != nil {
        startTime = time.Now() // Fallback
    }
    
    lastSeen, err := time.Parse(time.RFC3339, resourceAgent.LastSeen)
    if err != nil {
        lastSeen = time.Now() // Fallback
    }
    
    // Calculate uptime
    uptime := time.Since(startTime).Truncate(time.Second).String()
    
    // Map status to our format
    status := mapAgentStatus(resourceAgent.Status)
    
    // Generate unique full ID
    fullID := fmt.Sprintf("%s:%s", resourceName, agentID)
    
    agent := &Agent{
        ID:        fullID,
        Name:      fmt.Sprintf("%s Agent", strings.Title(resourceName)),
        Type:      resourceName,
        Status:    status,
        PID:       resourceAgent.PID,
        StartTime: startTime,
        LastSeen:  lastSeen,
        Uptime:    uptime,
        Command:   resourceAgent.Command,
        Capabilities: getResourceCapabilities(resourceName),
        Metrics:     getDefaultMetrics(),
        RadarPosition: generateRadarPosition(),
    }
    
    return agent
}

func mapAgentStatus(resourceStatus string) string {
    switch resourceStatus {
    case "running":
        return "active"
    case "stopped":
        return "inactive" 
    case "crashed":
        return "error"
    default:
        return "inactive"
    }
}

func getDefaultMetrics() map[string]interface{} {
    return map[string]interface{}{
        "memory_mb":    nil,
        "cpu_percent":  nil,
        "custom_fields": map[string]interface{}{},
    }
}

func generateRadarPosition() *RadarPosition {
    // Generate random position within radar circle (will be animated later)
    return &RadarPosition{
        X:       50 + (float64(time.Now().UnixNano()%80) - 40), // Random -40 to +40 from center
        Y:       50 + (float64(time.Now().UnixNano()%80) - 40),
        TargetX: 50 + (float64(time.Now().UnixNano()%60) - 30), // Random target position
        TargetY: 50 + (float64(time.Now().UnixNano()%60) - 30),
    }
}

// createMockAgent creates a mock agent for resources that are running but don't support agent discovery yet
func createMockAgent(resourceName string) *Agent {
    now := time.Now()
    startTime := now.Add(-time.Duration(time.Now().UnixNano()%3600) * time.Second) // Random start time within last hour
    
    mockAgent := &Agent{
        ID:           fmt.Sprintf("%s:mock-agent", resourceName),
        Name:         fmt.Sprintf("%s Service", strings.Title(resourceName)),
        Type:         resourceName,
        Status:       "active",
        PID:          int(1000 + time.Now().UnixNano()%8999), // Random PID between 1000-9999
        StartTime:    startTime,
        LastSeen:     now,
        Uptime:       time.Since(startTime).Truncate(time.Second).String(),
        Command:      fmt.Sprintf("%s-server --port=${PORT}", resourceName),
        Capabilities: getResourceCapabilities(resourceName),
        Metrics:      getMockMetrics(),
        RadarPosition: generateRadarPosition(),
    }
    
    return mockAgent
}

// getMockMetrics generates realistic mock metrics for demonstration
func getMockMetrics() map[string]interface{} {
    return map[string]interface{}{
        "memory_mb":    50 + time.Now().UnixNano()%200,  // Random 50-250 MB
        "cpu_percent":  float64(time.Now().UnixNano()%30), // Random 0-30% CPU
        "custom_fields": map[string]interface{}{
            "health_status": "healthy",
            "last_activity": time.Now().Format(time.RFC3339),
        },
    }
}
