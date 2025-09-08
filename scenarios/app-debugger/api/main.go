package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
)

type DebugRequest struct {
    AppName   string `json:"app_name"`
    DebugType string `json:"debug_type"`
}

type ErrorReport struct {
    AppName      string `json:"app_name"`
    ErrorMessage string `json:"error_message"`
    StackTrace   string `json:"stack_trace"`
    Context      map[string]interface{} `json:"context"`
}

type HealthStatus struct {
    Status  string   `json:"status"`
    Apps    []string `json:"apps"`
    Message string   `json:"message"`
}

var debugManager *DebugManager

func init() {
    debugManager = NewDebugManager()
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    status := HealthStatus{
        Status:  "healthy",
        Apps:    []string{},
        Message: "App Debugger API is running",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(status)
}

func debugHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var req DebugRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Start debug session using the debug manager
    session, err := debugManager.StartDebugSession(req.AppName, req.DebugType)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(session)
}

func reportErrorHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var report ErrorReport
    if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    response := map[string]interface{}{
        "status": "received",
        "app_name": report.AppName,
        "error_type": "analyzed",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func listAppsHandler(w http.ResponseWriter, r *http.Request) {
    // Mock implementation - would query actual running apps
    apps := []map[string]interface{}{
        {"name": "app-monitor", "status": "running", "health": "healthy"},
        {"name": "prompt-manager", "status": "running", "health": "warning"},
        {"name": "idea-generator", "status": "stopped", "health": "unknown"},
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(apps)
}

func main() {
	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
    
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/api/debug", debugHandler)
    http.HandleFunc("/api/report-error", reportErrorHandler)
    http.HandleFunc("/api/apps", listAppsHandler)
    
    log.Printf("App Debugger API starting on port %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}

// getEnv removed to prevent hardcoded defaults
