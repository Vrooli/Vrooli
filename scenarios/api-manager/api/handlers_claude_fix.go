package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ClaudeFixRequest represents a request to fix issues via Claude
type ClaudeFixRequest struct {
	ScenarioName string   `json:"scenario_name"`
	FixType      string   `json:"fix_type"` // "standards" or "vulnerabilities"
	IssueIDs     []string `json:"issue_ids,omitempty"` // Optional: specific issues to fix
}

// ClaudeFixResponse represents the response from a fix attempt
type ClaudeFixResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	FixID     string    `json:"fix_id"`
	StartedAt time.Time `json:"started_at"`
	Output    string    `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// triggerClaudeFixHandler handles requests to trigger Claude agent fixes
func triggerClaudeFixHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()
	logger.Info("Triggering Claude fix")
	
	// Parse request body
	var req ClaudeFixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate fix type
	if req.FixType != "standards" && req.FixType != "vulnerabilities" {
		http.Error(w, "Invalid fix_type. Must be 'standards' or 'vulnerabilities'", http.StatusBadRequest)
		return
	}
	
	// Validate scenario name
	if req.ScenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}
	
	// Load the appropriate prompt template
	// IMPORTANT: Always read fresh from disk - never cache
	// Prompts are located at the scenario root level, not in the API directory
	promptFile := filepath.Join("..", "prompts", fmt.Sprintf("%s-fix.txt", 
		map[string]string{
			"standards":       "standards-compliance",
			"vulnerabilities": "vulnerability",
		}[req.FixType]))
	
	promptTemplate, err := os.ReadFile(promptFile)
	if err != nil {
		logger.Error(fmt.Sprintf("Error reading prompt file %s", promptFile), err)
		http.Error(w, "Failed to load prompt template", http.StatusInternalServerError)
		return
	}
	
	// Build the prompt with actual data
	prompt := buildFixPrompt(string(promptTemplate), req)
	
	// Execute Claude Code agent
	response := executeClaudeCodeAgent(prompt, req.ScenarioName)
	
	// Return response
	w.Header().Set("Content-Type", "application/json")
	if response.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(response)
}

// buildFixPrompt replaces template variables with actual data
func buildFixPrompt(template string, req ClaudeFixRequest) string {
	prompt := strings.ReplaceAll(template, "{{SCENARIO_NAME}}", req.ScenarioName)
	
	if req.FixType == "standards" {
		// Get standards violations for the scenario
		violations := standardsStore.GetViolations(req.ScenarioName)
		
		// Build summary
		summary := fmt.Sprintf("Found %d standards violations in %s", len(violations), req.ScenarioName)
		prompt = strings.ReplaceAll(prompt, "{{VIOLATIONS_SUMMARY}}", summary)
		
		// Build detailed violations list
		var details strings.Builder
		for _, v := range violations {
			if len(req.IssueIDs) > 0 {
				// Check if this violation should be included
				found := false
				for _, id := range req.IssueIDs {
					if v.ID == id {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			
			details.WriteString(fmt.Sprintf("\n## %s (%s)\n", v.Title, v.Severity))
			details.WriteString(fmt.Sprintf("- File: %s:%d\n", v.FilePath, v.LineNumber))
			details.WriteString(fmt.Sprintf("- Type: %s\n", v.Type))
			details.WriteString(fmt.Sprintf("- Standard: %s\n", v.Standard))
			details.WriteString(fmt.Sprintf("- Description: %s\n", v.Description))
			details.WriteString(fmt.Sprintf("- Recommendation: %s\n", v.Recommendation))
			if v.CodeSnippet != "" {
				details.WriteString(fmt.Sprintf("- Code:\n```\n%s\n```\n", v.CodeSnippet))
			}
		}
		prompt = strings.ReplaceAll(prompt, "{{DETAILED_VIOLATIONS}}", details.String())
		
	} else if req.FixType == "vulnerabilities" {
		// Get vulnerabilities for the scenario
		vulnerabilities := vulnStore.GetVulnerabilities(req.ScenarioName)
		
		// Build summary
		criticalCount := 0
		highCount := 0
		for _, v := range vulnerabilities {
			if v.Severity == "critical" {
				criticalCount++
			} else if v.Severity == "high" {
				highCount++
			}
		}
		summary := fmt.Sprintf("Found %d vulnerabilities (%d critical, %d high) in %s", 
			len(vulnerabilities), criticalCount, highCount, req.ScenarioName)
		prompt = strings.ReplaceAll(prompt, "{{VULNERABILITIES_SUMMARY}}", summary)
		
		// Build detailed vulnerabilities list
		var details strings.Builder
		for _, v := range vulnerabilities {
			if len(req.IssueIDs) > 0 {
				// Check if this vulnerability should be included
				found := false
				for _, id := range req.IssueIDs {
					if v.ID == id {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			
			details.WriteString(fmt.Sprintf("\n## %s (%s)\n", v.Title, v.Severity))
			details.WriteString(fmt.Sprintf("- File: %s:%d\n", v.FilePath, v.LineNumber))
			details.WriteString(fmt.Sprintf("- Type: %s\n", v.Type))
			details.WriteString(fmt.Sprintf("- Description: %s\n", v.Description))
			details.WriteString(fmt.Sprintf("- Recommendation: %s\n", v.Recommendation))
			if v.CodeSnippet != "" {
				details.WriteString(fmt.Sprintf("- Code:\n```\n%s\n```\n", v.CodeSnippet))
			}
		}
		prompt = strings.ReplaceAll(prompt, "{{DETAILED_VULNERABILITIES}}", details.String())
	}
	
	return prompt
}

// executeClaudeCodeAgent executes the resource-claude-code CLI with the given prompt
func executeClaudeCodeAgent(prompt string, scenarioName string) ClaudeFixResponse {
	logger := NewLogger()
	fixID := fmt.Sprintf("api-manager-agent-%s-%d", scenarioName, time.Now().Unix())
	startTime := time.Now()
	
	// Set timeout (5 minutes for fixes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	// Get Vrooli root directory
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = os.Getenv("HOME") + "/Vrooli"
	}
	
	// Navigate to the scenario directory for the fix
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", scenarioName)
	
	// Setup logging for the agent
	logDir := filepath.Join(vrooliRoot, ".vrooli", "logs", "scenarios", "api-manager")
	os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", fixID))
	
	// Register agent in the registry
	registry := GetAgentRegistry()
	agent := &Agent{
		ID:        fixID,
		Status:    "running",
		StartTime: startTime,
		Command:   fmt.Sprintf("resource-claude-code run - (scenario: %s)", scenarioName),
		Type:      "claude-fix",
		Scenario:  scenarioName,
		LogFile:   logFile,
	}
	
	// Register agent first
	if err := registry.RegisterAgent(agent); err != nil {
		logger.Error(fmt.Sprintf("Failed to register agent %s", fixID), err)
		// Continue anyway, but log the error
	}
	
	// Cleanup function to update agent status and remove from registry
	defer func() {
		if agent.Status == "running" {
			agent.Status = "stopped"
		}
		registry.UpdateAgent(fixID, agent)
		
		// Remove from registry after a delay to allow for log viewing
		go func() {
			time.Sleep(5 * time.Minute)
			registry.RemoveAgent(fixID)
		}()
	}()
	
	// Execute resource-claude-code with stdin input
	cmd := exec.CommandContext(ctx, "resource-claude-code", "run", "-")
	cmd.Dir = scenarioDir // Run in the scenario directory
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		"MAX_TURNS=30",              // Allow more turns for complex fixes
		"TIMEOUT=300",               // 5 minutes
		"SKIP_PERMISSIONS=yes",      // Skip permission prompts for automated fixes
	)
	
	// Set up pipes
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		agent.Status = "crashed"
		return ClaudeFixResponse{
			Success:   false,
			FixID:     fixID,
			StartedAt: startTime,
			Error:     fmt.Sprintf("Failed to create stdin pipe: %v", err),
		}
	}
	
	// Setup logging to file and capture output
	var stdout, stderr bytes.Buffer
	
	// Create log file for agent output
	logFileHandle, err := os.Create(logFile)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create log file %s", logFile), err)
		// Fallback to stdout/stderr buffers only
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	} else {
		defer logFileHandle.Close()
		
		// Write both to log file and capture in buffers
		cmd.Stdout = io.MultiWriter(&stdout, logFileHandle)
		cmd.Stderr = io.MultiWriter(&stderr, logFileHandle)
		
		// Write initial log entry
		fmt.Fprintf(logFileHandle, "[%s] Starting Claude fix for scenario: %s\n", time.Now().Format(time.RFC3339), scenarioName)
		fmt.Fprintf(logFileHandle, "[%s] Agent ID: %s\n", time.Now().Format(time.RFC3339), fixID)
		fmt.Fprintf(logFileHandle, "[%s] Command: resource-claude-code run -\n", time.Now().Format(time.RFC3339))
		fmt.Fprintf(logFileHandle, "[%s] Working directory: %s\n", time.Now().Format(time.RFC3339), scenarioDir)
		fmt.Fprintf(logFileHandle, "=====================================\n")
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		agent.Status = "crashed"
		return ClaudeFixResponse{
			Success:   false,
			FixID:     fixID,
			StartedAt: startTime,
			Error:     fmt.Sprintf("Failed to start Claude Code: %v", err),
		}
	}
	
	// Update agent with PID
	agent.PID = cmd.Process.Pid
	registry.UpdateAgent(fixID, agent)
	
	// Send prompt via stdin
	go func() {
		defer stdinPipe.Close()
		if _, err := io.WriteString(stdinPipe, prompt); err != nil {
			logger.Error("Error writing prompt to stdin", err)
		}
	}()
	
	// Wait for completion
	err = cmd.Wait()
	
	// Prepare response
	response := ClaudeFixResponse{
		FixID:     fixID,
		StartedAt: startTime,
		Output:    stdout.String(),
	}
	
	if err != nil {
		agent.Status = "crashed"
		response.Success = false
		response.Error = fmt.Sprintf("Claude Code execution failed: %v\nStderr: %s", err, stderr.String())
		response.Message = "Fix attempt failed"
		
		// Log to file if available
		if logFileHandle != nil {
			fmt.Fprintf(logFileHandle, "\n[%s] FAILED: %v\n", time.Now().Format(time.RFC3339), err)
			fmt.Fprintf(logFileHandle, "[%s] Stderr: %s\n", time.Now().Format(time.RFC3339), stderr.String())
		}
	} else {
		agent.Status = "stopped"  // Completed successfully
		response.Success = true
		response.Message = fmt.Sprintf("Successfully completed fixes for %s", scenarioName)
		
		// Log successful completion
		logger.Info(fmt.Sprintf("Claude fix completed successfully for %s (agent ID: %s)", scenarioName, fixID))
		
		// Log to file if available
		if logFileHandle != nil {
			fmt.Fprintf(logFileHandle, "\n[%s] SUCCESS: Fix completed successfully\n", time.Now().Format(time.RFC3339))
		}
	}
	
	return response
}

// getClaudeFixStatusHandler checks the status of a Claude fix (placeholder for future enhancement)
func getClaudeFixStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fixID := vars["fixId"]
	
	// For now, return a simple response
	// In the future, this could track running fixes
	response := map[string]interface{}{
		"fix_id": fixID,
		"status": "completed",
		"message": "Fix status tracking not yet implemented",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}