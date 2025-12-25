package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ToolDefinition describes a tool available to the AI assistant
type ToolDefinition struct {
	Type     string `json:"type"` // "function"
	Function struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"function"`
}

// ScenarioPort retrieves the port for a scenario service
func getScenarioPort(scenarioName, portName string) (string, error) {
	cmd := exec.Command("vrooli", "scenario", "port", scenarioName, portName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get port for %s/%s: %w", scenarioName, portName, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// AvailableTools returns the list of tools available to the assistant
func (s *Server) AvailableTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Function: struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Parameters  map[string]interface{} `json:"parameters"`
			}{
				Name:        "spawn_coding_agent",
				Description: "Spawn a coding agent (Claude Code, Codex, or OpenCode) to execute a software engineering task. Use this when the user wants to: write code, fix bugs, implement features, refactor code, run tests, or make changes to a codebase.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"task": map[string]interface{}{
							"type":        "string",
							"description": "Clear description of the coding task to perform",
						},
						"runner_type": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"claude-code", "codex", "opencode"},
							"default":     "claude-code",
							"description": "Which coding agent to use. claude-code is recommended for most tasks.",
						},
						"workspace_path": map[string]interface{}{
							"type":        "string",
							"description": "Optional: Path to the workspace/repository to work in. Defaults to current Vrooli repo.",
						},
						"timeout_minutes": map[string]interface{}{
							"type":        "integer",
							"default":     30,
							"description": "Maximum time for the agent to run (in minutes)",
						},
					},
					"required": []string{"task"},
				},
			},
		},
		{
			Type: "function",
			Function: struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Parameters  map[string]interface{} `json:"parameters"`
			}{
				Name:        "check_agent_status",
				Description: "Check the status of a running or completed coding agent run",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"run_id": map[string]interface{}{
							"type":        "string",
							"description": "The ID of the agent run to check",
						},
					},
					"required": []string{"run_id"},
				},
			},
		},
		{
			Type: "function",
			Function: struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Parameters  map[string]interface{} `json:"parameters"`
			}{
				Name:        "stop_agent",
				Description: "Stop a running coding agent",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"run_id": map[string]interface{}{
							"type":        "string",
							"description": "The ID of the agent run to stop",
						},
					},
					"required": []string{"run_id"},
				},
			},
		},
		{
			Type: "function",
			Function: struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Parameters  map[string]interface{} `json:"parameters"`
			}{
				Name:        "list_active_agents",
				Description: "List all active coding agent runs",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Parameters  map[string]interface{} `json:"parameters"`
			}{
				Name:        "get_agent_diff",
				Description: "Get the code changes (diff) from a completed agent run",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"run_id": map[string]interface{}{
							"type":        "string",
							"description": "The ID of the agent run to get diff from",
						},
					},
					"required": []string{"run_id"},
				},
			},
		},
		{
			Type: "function",
			Function: struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Parameters  map[string]interface{} `json:"parameters"`
			}{
				Name:        "approve_agent_changes",
				Description: "Approve and apply the code changes from an agent run to the main workspace",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"run_id": map[string]interface{}{
							"type":        "string",
							"description": "The ID of the agent run to approve",
						},
					},
					"required": []string{"run_id"},
				},
			},
		},
	}
}

// ToolExecutor handles execution of tool calls
type ToolExecutor struct {
	server *Server
}

func NewToolExecutor(s *Server) *ToolExecutor {
	return &ToolExecutor{server: s}
}

// ExecuteTool runs a tool and returns the result
func (e *ToolExecutor) ExecuteTool(ctx context.Context, chatID, toolCallID, toolName, arguments string) (*ToolCallRecord, error) {
	record := &ToolCallRecord{
		ID:        toolCallID,
		ChatID:    chatID,
		ToolName:  toolName,
		Arguments: arguments,
		Status:    "running",
		StartedAt: time.Now(),
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		record.Status = "failed"
		record.ErrorMessage = fmt.Sprintf("Invalid arguments: %v", err)
		record.CompletedAt = time.Now()
		return record, err
	}

	var result interface{}
	var err error

	switch toolName {
	case "spawn_coding_agent":
		result, err = e.spawnCodingAgent(ctx, args)
		record.ScenarioName = "agent-manager"
		if res, ok := result.(map[string]interface{}); ok {
			if runID, ok := res["run_id"].(string); ok {
				record.ExternalRunID = runID
			}
		}
	case "check_agent_status":
		result, err = e.checkAgentStatus(ctx, args)
		record.ScenarioName = "agent-manager"
	case "stop_agent":
		result, err = e.stopAgent(ctx, args)
		record.ScenarioName = "agent-manager"
	case "list_active_agents":
		result, err = e.listActiveAgents(ctx)
		record.ScenarioName = "agent-manager"
	case "get_agent_diff":
		result, err = e.getAgentDiff(ctx, args)
		record.ScenarioName = "agent-manager"
	case "approve_agent_changes":
		result, err = e.approveAgentChanges(ctx, args)
		record.ScenarioName = "agent-manager"
	default:
		err = fmt.Errorf("unknown tool: %s", toolName)
	}

	record.CompletedAt = time.Now()
	if err != nil {
		record.Status = "failed"
		record.ErrorMessage = err.Error()
		resultJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		record.Result = string(resultJSON)
	} else {
		record.Status = "completed"
		resultJSON, _ := json.Marshal(result)
		record.Result = string(resultJSON)
	}

	return record, err
}

// getAgentManagerURL returns the URL for the agent-manager API
func (e *ToolExecutor) getAgentManagerURL() (string, error) {
	// Try environment variable first (set by lifecycle system when agent-manager is a dependency)
	if url := os.Getenv("AGENT_MANAGER_API_URL"); url != "" {
		return url, nil
	}

	// Fall back to port discovery
	port, err := getScenarioPort("agent-manager", "API_PORT")
	if err != nil {
		return "", fmt.Errorf("agent-manager not available: %w", err)
	}
	return fmt.Sprintf("http://localhost:%s", port), nil
}

// spawnCodingAgent creates a new coding agent run via agent-manager
func (e *ToolExecutor) spawnCodingAgent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := e.getAgentManagerURL()
	if err != nil {
		return nil, err
	}

	task, _ := args["task"].(string)
	if task == "" {
		return nil, fmt.Errorf("task is required")
	}

	runnerType := "claude-code"
	if rt, ok := args["runner_type"].(string); ok && rt != "" {
		runnerType = rt
	}

	workspacePath := os.Getenv("VROOLI_ROOT")
	if workspacePath == "" {
		workspacePath = os.Getenv("HOME") + "/Vrooli"
	}
	if wp, ok := args["workspace_path"].(string); ok && wp != "" {
		workspacePath = wp
	}

	timeoutMinutes := 30
	if tm, ok := args["timeout_minutes"].(float64); ok {
		timeoutMinutes = int(tm)
	}

	reqBody := map[string]interface{}{
		"task": map[string]interface{}{
			"name":        "Chat-initiated task",
			"description": task,
		},
		"runner_type":     runnerType,
		"workspace_path":  workspacePath,
		"timeout_seconds": timeoutMinutes * 60,
		"auto_approve":    false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/runs", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"run_id":  result["id"],
		"status":  result["status"],
		"message": fmt.Sprintf("Coding agent spawned successfully. Run ID: %v", result["id"]),
	}, nil
}

// checkAgentStatus gets the status of a coding agent run
func (e *ToolExecutor) checkAgentStatus(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := e.getAgentManagerURL()
	if err != nil {
		return nil, err
	}

	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/runs/"+runID, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// stopAgent stops a running coding agent
func (e *ToolExecutor) stopAgent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := e.getAgentManagerURL()
	if err != nil {
		return nil, err
	}

	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/runs/"+runID+"/stop", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent run %s stopped", runID),
	}, nil
}

// listActiveAgents returns a list of active agent runs
func (e *ToolExecutor) listActiveAgents(ctx context.Context) (interface{}, error) {
	baseURL, err := e.getAgentManagerURL()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/runs?status=running", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// getAgentDiff gets the code changes from an agent run
func (e *ToolExecutor) getAgentDiff(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := e.getAgentManagerURL()
	if err != nil {
		return nil, err
	}

	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/runs/"+runID+"/diff", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// approveAgentChanges approves and applies changes from an agent run
func (e *ToolExecutor) approveAgentChanges(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := e.getAgentManagerURL()
	if err != nil {
		return nil, err
	}

	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/runs/"+runID+"/approve", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Changes from run %s approved and applied", runID),
	}, nil
}

// SaveToolCallRecord persists a tool call record to the database
func (s *Server) SaveToolCallRecord(ctx context.Context, messageID string, record *ToolCallRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tool_calls (id, message_id, chat_id, tool_name, arguments, result, status, scenario_name, external_run_id, started_at, completed_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			result = EXCLUDED.result,
			status = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at,
			error_message = EXCLUDED.error_message
	`, record.ID, messageID, record.ChatID, record.ToolName, record.Arguments, record.Result, record.Status, record.ScenarioName, record.ExternalRunID, record.StartedAt, record.CompletedAt, record.ErrorMessage)
	return err
}

// GetToolCallsForMessage retrieves tool calls associated with a message
func (s *Server) GetToolCallsForMessage(ctx context.Context, messageID string) ([]ToolCallRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, message_id, chat_id, tool_name, arguments, result, status, scenario_name, external_run_id, started_at, completed_at, error_message
		FROM tool_calls WHERE message_id = $1 ORDER BY started_at ASC
	`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ToolCallRecord
	for rows.Next() {
		var r ToolCallRecord
		var completedAt sql.NullTime
		var result, scenarioName, externalRunID, errorMessage sql.NullString
		if err := rows.Scan(&r.ID, &r.MessageID, &r.ChatID, &r.ToolName, &r.Arguments, &result, &r.Status, &scenarioName, &externalRunID, &r.StartedAt, &completedAt, &errorMessage); err != nil {
			continue
		}
		if result.Valid {
			r.Result = result.String
		}
		if scenarioName.Valid {
			r.ScenarioName = scenarioName.String
		}
		if externalRunID.Valid {
			r.ExternalRunID = externalRunID.String
		}
		if completedAt.Valid {
			r.CompletedAt = completedAt.Time
		}
		if errorMessage.Valid {
			r.ErrorMessage = errorMessage.String
		}
		records = append(records, r)
	}
	return records, nil
}
