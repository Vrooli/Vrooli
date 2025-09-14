package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "time"
)

type Agent struct {
    ID           string                 `json:"id"`
    Name         string                 `json:"name"`
    Type         string                 `json:"type"`
    Status       string                 `json:"status"`
    Description  string                 `json:"description"`
    LastHeartbeat time.Time            `json:"last_heartbeat"`
    Capabilities []string              `json:"capabilities"`
    Metrics      map[string]interface{} `json:"metrics"`
}

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

type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

type APIServer struct {
    agents map[string]*Agent
}

func main() {
	port := getEnv("API_PORT", getEnv("PORT", ""))

    // API endpoints
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/api/agents", agentsHandler)
    http.HandleFunc("/api/agents/status", statusHandler)
    http.HandleFunc("/api/agents/orchestrate", orchestrateHandler)

    log.Printf("Agent Dashboard API starting on port %s", port)
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
    response := APIResponse{
        Success: true,
        Data: map[string]string{
            "status":  "healthy",
            "service": "agent-dashboard-api",
            "version": "1.0.0",
        },
    }
    jsonResponse(w, response, http.StatusOK)
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
        // Return list of agents
        agents := getAgents()
        response := APIResponse{
            Success: true,
            Data:    agents,
        }
        jsonResponse(w, response, http.StatusOK)
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

func orchestrateHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var request map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        errorResponse(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Simulate orchestration response
    response := APIResponse{
        Success: true,
        Data: map[string]interface{}{
            "orchestration_id": fmt.Sprintf("orch-%d", time.Now().Unix()),
            "status":          "queued",
            "message":         "Orchestration plan created",
        },
    }
    jsonResponse(w, response, http.StatusOK)
}

// List of all resources that support agent tracking
var supportedResources = []string{
    "codex", "claude-code", "cline", "ollama", "agent-s2", "autogen-studio",
    "autogpt", "crewai", "gemini", "langchain", "litellm", "openrouter",
    "whisper", "comfyui", "pandas-ai", "parlant", "huginn", "opencode",
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
            Description:  fmt.Sprintf("%s agent running: %s", strings.Title(resourceName), resourceAgent.Command),
            LastHeartbeat: lastSeen,
            Capabilities: getResourceCapabilities(resourceName),
            Metrics: map[string]interface{}{
                "pid":       resourceAgent.PID,
                "start_time": startTime.Format("2006-01-02 15:04:05"),
                "uptime":    getUptimeString(startTime),
                "command":   resourceAgent.Command,
            },
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
