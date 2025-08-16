package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
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

type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8100"
    }

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

func getAgents() []Agent {
    // In production, this would query the database
    return []Agent{
        {
            ID:            "agent-001",
            Name:          "Huginn Scraper",
            Type:          "huginn",
            Status:        "active",
            Description:   "Web scraping and monitoring",
            LastHeartbeat: time.Now(),
            Capabilities:  []string{"scraping", "monitoring"},
            Metrics: map[string]interface{}{
                "cpu":    23,
                "memory": 512,
            },
        },
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