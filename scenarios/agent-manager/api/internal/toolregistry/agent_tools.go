// Package toolregistry provides the tool discovery service for agent-manager.
//
// This file defines the agent-manager specific tools that are exposed
// via the Tool Discovery Protocol.
package toolregistry

import (
	"context"

	"agent-manager/internal/domain"
)

// AgentToolProvider provides the core agent-manager tools.
// These tools enable spawning, monitoring, and managing coding agents.
type AgentToolProvider struct{}

// NewAgentToolProvider creates a new AgentToolProvider.
func NewAgentToolProvider() *AgentToolProvider {
	return &AgentToolProvider{}
}

// Name returns the provider identifier.
func (p *AgentToolProvider) Name() string {
	return "agent-manager-core"
}

// Categories returns the tool categories for agent-manager.
func (p *AgentToolProvider) Categories(_ context.Context) []domain.ToolCategory {
	return []domain.ToolCategory{
		{
			ID:          "agent_lifecycle",
			Name:        "Agent Lifecycle",
			Description: "Tools for spawning, stopping, and managing coding agent runs",
			Icon:        "play",
		},
		{
			ID:          "agent_status",
			Name:        "Agent Status",
			Description: "Tools for checking agent status and progress",
			Icon:        "info",
		},
		{
			ID:          "agent_results",
			Name:        "Agent Results",
			Description: "Tools for retrieving and managing agent output",
			Icon:        "file-diff",
		},
	}
}

// Tools returns the tool definitions for agent-manager.
func (p *AgentToolProvider) Tools(_ context.Context) []domain.ToolDefinition {
	return []domain.ToolDefinition{
		p.spawnCodingAgentTool(),
		p.checkAgentStatusTool(),
		p.stopAgentTool(),
		p.listActiveAgentsTool(),
		p.getAgentDiffTool(),
		p.approveAgentChangesTool(),
	}
}

func (p *AgentToolProvider) spawnCodingAgentTool() domain.ToolDefinition {
	thirtyInt := 30

	return domain.ToolDefinition{
		Name:        "spawn_coding_agent",
		Description: "Spawn a coding agent (Claude Code, Codex, or OpenCode) to execute a software engineering task. Use this when the user wants to: write code, fix bugs, implement features, refactor code, run tests, or make changes to a codebase.",
		Category:    "agent_lifecycle",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ParameterSchema{
				"task": {
					Type:        "string",
					Description: "Clear description of the coding task to perform. Be specific about what needs to be done.",
				},
				"runner_type": {
					Type:        "string",
					Enum:        []string{"claude-code", "codex", "opencode"},
					Default:     "claude-code",
					Description: "Which coding agent to use. claude-code is recommended for most tasks as it has the best code understanding.",
				},
				"workspace_path": {
					Type:        "string",
					Description: "Path to the workspace/repository to work in. Defaults to the current Vrooli repository if not specified.",
				},
				"timeout_minutes": {
					Type:        "integer",
					Default:     thirtyInt,
					Description: "Maximum time for the agent to run (in minutes). Longer tasks may need higher timeouts.",
				},
				"profile_key": {
					Type:        "string",
					Description: "Optional profile key to use for the agent configuration. Profiles define runner settings, permissions, and behavior.",
				},
			},
			Required: []string{"task"},
		},
		Metadata: domain.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 10,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         false,
			Tags:               []string{"coding", "agent", "async"},
			Examples: []domain.ToolExample{
				{
					Description: "Fix a bug in the authentication module",
					Input: map[string]interface{}{
						"task":        "Fix the login bug where users are logged out after 5 minutes. The session timeout should be 24 hours.",
						"runner_type": "claude-code",
					},
				},
				{
					Description: "Add a new feature with custom timeout",
					Input: map[string]interface{}{
						"task":            "Implement a dark mode toggle in the settings page",
						"runner_type":     "claude-code",
						"timeout_minutes": 45,
					},
				},
			},
		},
	}
}

func (p *AgentToolProvider) checkAgentStatusTool() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "check_agent_status",
		Description: "Check the current status of a running or completed coding agent run. Returns the run status, progress, and any output available so far.",
		Category:    "agent_status",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ParameterSchema{
				"run_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The ID of the agent run to check. This is returned when spawning an agent.",
				},
			},
			Required: []string{"run_id"},
		},
		Metadata: domain.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"status", "monitoring"},
			Examples: []domain.ToolExample{
				{
					Description: "Check status of a running agent",
					Input: map[string]interface{}{
						"run_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
			},
		},
	}
}

func (p *AgentToolProvider) stopAgentTool() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "stop_agent",
		Description: "Stop a running coding agent. Use this when you need to cancel an agent that is taking too long or going in the wrong direction.",
		Category:    "agent_lifecycle",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ParameterSchema{
				"run_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The ID of the agent run to stop.",
				},
			},
			Required: []string{"run_id"},
		},
		Metadata: domain.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"lifecycle", "cancellation"},
			Examples: []domain.ToolExample{
				{
					Description: "Stop a runaway agent",
					Input: map[string]interface{}{
						"run_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
			},
		},
	}
}

func (p *AgentToolProvider) listActiveAgentsTool() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "list_active_agents",
		Description: "List all currently active (running or pending) coding agent runs. Useful for seeing what agents are currently working.",
		Category:    "agent_status",
		Parameters: domain.ToolParameters{
			Type:       "object",
			Properties: map[string]domain.ParameterSchema{},
		},
		Metadata: domain.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"status", "monitoring", "list"},
			Examples: []domain.ToolExample{
				{
					Description: "List all active agents",
					Input:       map[string]interface{}{},
				},
			},
		},
	}
}

func (p *AgentToolProvider) getAgentDiffTool() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "get_agent_diff",
		Description: "Get the code changes (diff) from a completed agent run. Shows what files were modified and what changes were made.",
		Category:    "agent_results",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ParameterSchema{
				"run_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The ID of the agent run to get the diff from.",
				},
			},
			Required: []string{"run_id"},
		},
		Metadata: domain.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"diff", "results", "code-review"},
			Examples: []domain.ToolExample{
				{
					Description: "Get changes from a completed run",
					Input: map[string]interface{}{
						"run_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
			},
		},
	}
}

func (p *AgentToolProvider) approveAgentChangesTool() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "approve_agent_changes",
		Description: "Approve and apply the code changes from a completed agent run to the main workspace. This merges the agent's changes into the working directory.",
		Category:    "agent_results",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ParameterSchema{
				"run_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The ID of the agent run to approve.",
				},
				"files": {
					Type:        "array",
					Description: "Optional list of specific files to approve. If not provided, all changes are approved.",
					Items: &domain.ParameterSchema{
						Type: "string",
					},
				},
			},
			Required: []string{"run_id"},
		},
		Metadata: domain.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // This modifies the filesystem
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         false,
			Tags:               []string{"approval", "merge", "filesystem"},
			Examples: []domain.ToolExample{
				{
					Description: "Approve all changes from an agent",
					Input: map[string]interface{}{
						"run_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				{
					Description: "Approve only specific files",
					Input: map[string]interface{}{
						"run_id": "550e8400-e29b-41d4-a716-446655440000",
						"files":  []string{"src/auth/login.go", "src/auth/session.go"},
					},
				},
			},
		},
	}
}
