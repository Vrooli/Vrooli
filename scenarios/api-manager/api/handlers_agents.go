package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

// getAgentsHandler handles GET /api/v1/agents
func getAgentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	registry := GetAgentRegistry()
	agents := registry.ListAgents()
	
	// Return in the standard format expected by agent-dashboard
	response := map[string]interface{}{
		"agents": agents,
	}
	
	json.NewEncoder(w).Encode(response)
}

// stopAgentHandler handles POST /api/v1/agents/{agentId}/stop
func stopAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentId"]
	
	w.Header().Set("Content-Type", "application/json")
	
	registry := GetAgentRegistry()
	
	// Check if agent exists
	agent, exists := registry.GetAgent(agentID)
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Agent %s not found", agentID),
		})
		return
	}
	
	// Attempt to stop the agent
	err := registry.StopAgent(agentID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to stop agent %s: %v", agentID, err),
		})
		return
	}
	
	// Log the action
	logger := NewLogger()
	logger.Info(fmt.Sprintf("Agent %s stopped via API request", agentID))
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent %s stopped successfully", agentID),
		"agent_id": agentID,
		"previous_status": agent.Status,
	})
}

// getAgentLogsHandler handles GET /api/v1/agents/{agentId}/logs
func getAgentLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentId"]
	
	registry := GetAgentRegistry()
	
	// Check if agent exists
	agent, exists := registry.GetAgent(agentID)
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Agent %s not found", agentID),
		})
		return
	}
	
	// Determine log file path
	var logPath string
	if agent.LogFile != "" {
		logPath = agent.LogFile
	} else {
		// Default log location based on configuration
		appRoot := os.Getenv("APP_ROOT")
		if appRoot == "" {
			appRoot = os.Getenv("VROOLI_ROOT")
			if appRoot == "" {
				appRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
			}
		}
		logDir := filepath.Join(appRoot, ".vrooli", "logs", "scenarios", "api-manager")
		logPath = filepath.Join(logDir, fmt.Sprintf("%s.log", agentID))
	}
	
	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "No logs available for agent %s\nExpected log file: %s\n", agentID, logPath)
		return
	}
	
	// Read and return log file contents
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Failed to read log file: %v", err),
		})
		return
	}
	
	// Return logs as plain text
	w.Header().Set("Content-Type", "text/plain")
	w.Write(logContent)
}