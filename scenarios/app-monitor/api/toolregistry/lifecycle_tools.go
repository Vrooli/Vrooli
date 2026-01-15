// Package toolregistry provides tool definitions for app-monitor.
//
// This file defines state-modifying app lifecycle tools.
// These tools can start, stop, and restart applications.
// State-modifying tools require human approval before execution.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// LifecycleToolProvider provides app lifecycle management tools.
type LifecycleToolProvider struct{}

// NewLifecycleToolProvider creates a new LifecycleToolProvider.
func NewLifecycleToolProvider() *LifecycleToolProvider {
	return &LifecycleToolProvider{}
}

// Name returns the provider identifier.
func (p *LifecycleToolProvider) Name() string {
	return "app-monitor-lifecycle"
}

// Categories returns the tool categories for lifecycle tools.
func (p *LifecycleToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "app_lifecycle",
			Name:         "App Lifecycle",
			Description:  "Tools for starting, stopping, and restarting applications",
			Icon:         "play",
			DisplayOrder: 2,
		},
	}
}

// Tools returns the lifecycle tool definitions.
func (p *LifecycleToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.startAppTool(),
		p.stopAppTool(),
		p.restartAppTool(),
	}
}

// startAppTool returns the app start tool.
func (p *LifecycleToolProvider) startAppTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "start_app",
		Description: "Start a stopped application. This runs the scenario's startup sequence using the Vrooli CLI. The app must exist and be in a stopped state.",
		Category:    "app_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // State-modifying: starts processes
			TimeoutSeconds:     120,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"lifecycle", "start"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Start the agent-inbox scenario",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// stopAppTool returns the app stop tool.
func (p *LifecycleToolProvider) stopAppTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "stop_app",
		Description: "Stop a running application. This gracefully stops the scenario's processes using the Vrooli CLI. The app must be in a running state.",
		Category:    "app_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // State-modifying: stops processes
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"lifecycle", "stop"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Stop the agent-inbox scenario",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// restartAppTool returns the app restart tool.
func (p *LifecycleToolProvider) restartAppTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "restart_app",
		Description: "Restart a running application. This stops and then starts the scenario. Useful for applying configuration changes or recovering from issues.",
		Category:    "app_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // State-modifying: restarts processes
			TimeoutSeconds:     180,
			RateLimitPerMinute: 5,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"lifecycle", "restart"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Restart the agent-inbox scenario",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}
