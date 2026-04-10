// Package toolregistry provides tool definitions for app-issue-tracker.
//
// This file defines tools for managing AI agent configuration and automation.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// AgentToolProvider provides agent configuration tools.
type AgentToolProvider struct{}

// NewAgentToolProvider creates a new AgentToolProvider.
func NewAgentToolProvider() *AgentToolProvider {
	return &AgentToolProvider{}
}

// Name returns the provider identifier.
func (p *AgentToolProvider) Name() string {
	return "issue-tracker-agent"
}

// Categories returns the tool categories for agent tools.
func (p *AgentToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "agent_automation",
			Name:         "Agent & Automation",
			Description:  "Tools for configuring AI agents and automation settings",
			Icon:         "cpu",
			DisplayOrder: 4,
		},
	}
}

// Tools returns the agent configuration tool definitions.
func (p *AgentToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.listAgentsTool(),
		p.getAgentSettingsTool(),
		p.updateAgentSettingsTool(),
		p.getProcessorStateTool(),
	}
}

// listAgentsTool returns the agent listing tool.
func (p *AgentToolProvider) listAgentsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_agents",
		Description: "List all available AI agents with their capabilities and performance metrics. Returns agent ID, name, runner type, success rate, and total runs.",
		Category:    "agent_automation",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"agents", "list", "configuration"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all available agents",
					map[string]interface{}{},
				),
			},
		},
	}
}

// getAgentSettingsTool returns the agent settings retrieval tool.
func (p *AgentToolProvider) getAgentSettingsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_agent_settings",
		Description: "Get current agent profile configuration including runner type, timeout, max turns, and allowed tools.",
		Category:    "agent_automation",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"settings", "configuration", "agents"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get current agent settings",
					map[string]interface{}{},
				),
			},
		},
	}
}

// updateAgentSettingsTool returns the agent settings update tool.
func (p *AgentToolProvider) updateAgentSettingsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "update_agent_settings",
		Description: "Update agent profile configuration. Changes affect future investigations.",
		Category:    "agent_automation",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"runner_type": {
					Type:        "string",
					Enum:        []string{"claude-code", "codex", "opencode"},
					Description: "AI runner to use for investigations",
				},
				"timeout_seconds": {
					Type:        "integer",
					Description: "Maximum time for a single investigation (in seconds)",
				},
				"max_turns": {
					Type:        "integer",
					Description: "Maximum conversation turns per investigation",
				},
				"allowed_tools": {
					Type:        "array",
					Description: "List of tool names the agent is allowed to use",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"skip_permissions": {
					Type:        "boolean",
					Description: "Allow agent to execute tools without approval prompts",
				},
			},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // Configuration changes should be reviewed
			TimeoutSeconds:     10,
			RateLimitPerMinute: 10,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"settings", "configuration", "agents"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Switch to codex runner",
					map[string]interface{}{
						"runner_type": "codex",
					},
				),
				NewToolExample(
					"Increase timeout and max turns",
					map[string]interface{}{
						"timeout_seconds": 3600,
						"max_turns":       50,
					},
				),
				NewToolExample(
					"Enable auto-approval for tools",
					map[string]interface{}{
						"skip_permissions": true,
					},
				),
			},
		},
	}
}

// getProcessorStateTool returns the automation processor state tool.
func (p *AgentToolProvider) getProcessorStateTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_processor_state",
		Description: "Get the current state of the automation processor including active status, concurrent slots, refresh interval, and currently running investigations.",
		Category:    "agent_automation",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"processor", "automation", "status"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get automation processor state",
					map[string]interface{}{},
				),
			},
		},
	}
}
