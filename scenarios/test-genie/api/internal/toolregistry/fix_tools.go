// Package toolregistry provides the tool discovery service for test-genie.
//
// This file defines the fix operation tools for spawning and managing
// agents that fix failing tests.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// FixToolProvider provides the fix operation tools.
type FixToolProvider struct{}

// NewFixToolProvider creates a new FixToolProvider.
func NewFixToolProvider() *FixToolProvider {
	return &FixToolProvider{}
}

// Name returns the provider identifier.
func (p *FixToolProvider) Name() string {
	return "test-genie-fix"
}

// Categories returns the tool categories for fix operations.
func (p *FixToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "fix_operations",
			Name:        "Fix Operations",
			Description: "Tools for spawning agents to automatically fix failing tests",
			Icon:        "wrench",
		},
	}
}

// Tools returns the tool definitions for fix operations.
func (p *FixToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.spawnFixTool(),
		p.getFixTool(),
		p.listFixesTool(),
		p.getActiveFixTool(),
		p.stopFixTool(),
	}
}

func (p *FixToolProvider) spawnFixTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "spawn_fix",
		Description: "Spawn an AI agent to automatically fix failing tests for a scenario. The agent will analyze test failures and attempt to fix the underlying code.",
		Category:    "fix_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario with failing tests to fix",
				},
				"phase": {
					Type:        "string",
					Description: "Optional specific phase to focus on (e.g., 'unit', 'integration')",
				},
				"error_context": {
					Type:        "string",
					Description: "Optional error message or context to help the agent understand what to fix",
				},
				"focus_files": {
					Type:        "array",
					Description: "Optional list of specific files to focus on",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
			},
			Required: []string{"scenario"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // This modifies code
			TimeoutSeconds:     60,
			RateLimitPerMinute: 5,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"fix", "agent", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Fix failing unit tests",
					map[string]interface{}{
						"scenario": "agent-manager",
						"phase":    "unit",
					},
				),
				NewToolExample(
					"Fix with error context",
					map[string]interface{}{
						"scenario":      "test-genie",
						"error_context": "nil pointer dereference in service.go",
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "get_fix",
					OperationIdField:       "fix_id",
					StatusToolIdParam:      "fix_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 1800, // 30 minutes for fix agents
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"completed", "success"},
					FailureValues: []string{"failed", "error", "stopped"},
					PendingValues: []string{"running", "pending", "starting"},
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:        "stop_fix",
					CancelToolIdParam: "fix_id",
				},
			},
		},
	}
}

func (p *FixToolProvider) getFixTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_fix",
		Description: "Get the status and details of a fix operation by ID. Returns the agent's progress, any changes made, and current status.",
		Category:    "fix_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"fix_id": {
					Type:        "string",
					Description: "The ID of the fix operation to retrieve",
				},
			},
			Required: []string{"fix_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"status", "monitoring"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check fix status",
					map[string]interface{}{
						"fix_id": "fix-550e8400-e29b-41d4",
					},
				),
			},
		},
	}
}

func (p *FixToolProvider) listFixesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_fixes",
		Description: "List fix attempts for a scenario. Shows recent fix operations and their outcomes.",
		Category:    "fix_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario to list fixes for",
				},
				"limit": {
					Type:        "integer",
					Default:     IntValue(10),
					Description: "Maximum number of fixes to return",
				},
			},
			Required: []string{"scenario"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"list", "history"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List recent fixes for a scenario",
					map[string]interface{}{
						"scenario": "agent-manager",
						"limit":    5,
					},
				),
			},
		},
	}
}

func (p *FixToolProvider) getActiveFixTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_active_fix",
		Description: "Get the currently active fix operation for a scenario, if one exists. Only one fix can be active per scenario at a time.",
		Category:    "fix_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario to check for active fix",
				},
			},
			Required: []string{"scenario"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"status", "active"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check if scenario has active fix",
					map[string]interface{}{
						"scenario": "test-genie",
					},
				),
			},
		},
	}
}

func (p *FixToolProvider) stopFixTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "stop_fix",
		Description: "Stop a running fix operation. Cancels the agent and cleans up resources.",
		Category:    "fix_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"fix_id": {
					Type:        "string",
					Description: "The ID of the fix operation to stop",
				},
			},
			Required: []string{"fix_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"lifecycle", "cancellation"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Stop a running fix",
					map[string]interface{}{
						"fix_id": "fix-550e8400-e29b-41d4",
					},
				),
			},
		},
	}
}
