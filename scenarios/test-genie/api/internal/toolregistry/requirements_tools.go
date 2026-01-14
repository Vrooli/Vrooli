// Package toolregistry provides the tool discovery service for test-genie.
//
// This file defines the requirements management tools for improving
// and syncing scenario requirements.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// RequirementsToolProvider provides the requirements management tools.
type RequirementsToolProvider struct{}

// NewRequirementsToolProvider creates a new RequirementsToolProvider.
func NewRequirementsToolProvider() *RequirementsToolProvider {
	return &RequirementsToolProvider{}
}

// Name returns the provider identifier.
func (p *RequirementsToolProvider) Name() string {
	return "test-genie-requirements"
}

// Categories returns the tool categories for requirements management.
func (p *RequirementsToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "requirements",
			Name:        "Requirements",
			Description: "Tools for managing and improving scenario requirements documentation",
			Icon:        "document",
		},
	}
}

// Tools returns the tool definitions for requirements management.
func (p *RequirementsToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.improveRequirementsTool(),
		p.getRequirementsImproveTool(),
		p.listRequirementsImprovesTool(),
		p.getActiveRequirementsImproveTool(),
		p.stopRequirementsImproveTool(),
		p.syncRequirementsTool(),
	}
}

func (p *RequirementsToolProvider) improveRequirementsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "improve_requirements",
		Description: "Spawn an AI agent to analyze and improve the requirements documentation for a scenario. The agent will suggest improvements based on the codebase and tests.",
		Category:    "requirements",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario whose requirements to improve",
				},
				"focus_areas": {
					Type:        "array",
					Description: "Optional specific areas to focus on (e.g., 'api', 'security', 'performance')",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"additional_context": {
					Type:        "string",
					Description: "Optional additional context or guidance for the agent",
				},
			},
			Required: []string{"scenario"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // This modifies documentation
			TimeoutSeconds:     60,
			RateLimitPerMinute: 5,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"requirements", "agent", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Improve requirements for a scenario",
					map[string]interface{}{
						"scenario": "agent-manager",
					},
				),
				NewToolExample(
					"Focus on specific areas",
					map[string]interface{}{
						"scenario":    "test-genie",
						"focus_areas": []string{"api", "testing"},
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "get_requirements_improve",
					OperationIdField:       "improve_id",
					StatusToolIdParam:      "improve_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 1800, // 30 minutes
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"completed", "success"},
					FailureValues: []string{"failed", "error", "stopped"},
					PendingValues: []string{"running", "pending", "starting"},
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:        "stop_requirements_improve",
					CancelToolIdParam: "improve_id",
				},
			},
		},
	}
}

func (p *RequirementsToolProvider) getRequirementsImproveTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_requirements_improve",
		Description: "Get the status and details of a requirements improvement operation by ID.",
		Category:    "requirements",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"improve_id": {
					Type:        "string",
					Description: "The ID of the improvement operation to retrieve",
				},
			},
			Required: []string{"improve_id"},
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
					"Check improvement status",
					map[string]interface{}{
						"improve_id": "improve-550e8400-e29b-41d4",
					},
				),
			},
		},
	}
}

func (p *RequirementsToolProvider) listRequirementsImprovesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_requirements_improves",
		Description: "List requirements improvement attempts for a scenario.",
		Category:    "requirements",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario to list improvements for",
				},
				"limit": {
					Type:        "integer",
					Default:     IntValue(10),
					Description: "Maximum number of improvements to return",
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
					"List recent improvements",
					map[string]interface{}{
						"scenario": "agent-manager",
						"limit":    5,
					},
				),
			},
		},
	}
}

func (p *RequirementsToolProvider) getActiveRequirementsImproveTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_active_requirements_improve",
		Description: "Get the currently active requirements improvement for a scenario, if one exists.",
		Category:    "requirements",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario to check",
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
					"Check for active improvement",
					map[string]interface{}{
						"scenario": "test-genie",
					},
				),
			},
		},
	}
}

func (p *RequirementsToolProvider) stopRequirementsImproveTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "stop_requirements_improve",
		Description: "Stop a running requirements improvement operation.",
		Category:    "requirements",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"improve_id": {
					Type:        "string",
					Description: "The ID of the improvement operation to stop",
				},
			},
			Required: []string{"improve_id"},
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
					"Stop a running improvement",
					map[string]interface{}{
						"improve_id": "improve-550e8400-e29b-41d4",
					},
				),
			},
		},
	}
}

func (p *RequirementsToolProvider) syncRequirementsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "sync_requirements",
		Description: "Synchronize requirements documentation from the scenario directory. This updates the cached requirements to match the current state on disk.",
		Category:    "requirements",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario to sync requirements for",
				},
			},
			Required: []string{"scenario"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"sync", "requirements"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Sync requirements from disk",
					map[string]interface{}{
						"scenario": "agent-manager",
					},
				),
			},
		},
	}
}
