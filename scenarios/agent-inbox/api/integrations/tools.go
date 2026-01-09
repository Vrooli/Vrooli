package integrations

// AvailableTools returns the list of tools available to the AI assistant.
// These tools enable the AI to dispatch work to specialized scenarios.
//
// Deprecated: Use services.ToolRegistry.GetToolsForOpenAI() instead.
// This function is kept for backward compatibility but will be removed
// in a future version. The dynamic ToolRegistry fetches tools from
// all configured scenarios at runtime.
func AvailableTools() []ToolDefinition {
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
