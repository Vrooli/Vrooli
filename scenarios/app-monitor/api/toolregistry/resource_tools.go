// Package toolregistry provides tool definitions for app-monitor.
//
// This file defines resource management tools for Vrooli resources
// like PostgreSQL, Redis, Ollama, Qdrant, etc.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// ResourceToolProvider provides resource management tools.
type ResourceToolProvider struct{}

// NewResourceToolProvider creates a new ResourceToolProvider.
func NewResourceToolProvider() *ResourceToolProvider {
	return &ResourceToolProvider{}
}

// Name returns the provider identifier.
func (p *ResourceToolProvider) Name() string {
	return "app-monitor-resources"
}

// Categories returns the tool categories for resource tools.
func (p *ResourceToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "resources",
			Name:         "Resource Management",
			Description:  "Tools for managing Vrooli resources (databases, caches, AI models, etc.)",
			Icon:         "database",
			DisplayOrder: 7,
		},
	}
}

// Tools returns the resource management tool definitions.
func (p *ResourceToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.getResourceTool(),
		p.startResourceTool(),
		p.stopResourceTool(),
	}
}

// getResourceTool returns the resource details tool.
func (p *ResourceToolProvider) getResourceTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_resource",
		Description: "Get detailed information about a specific Vrooli resource including status, configuration, and health.",
		Category:    "resources",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"resource_id": {
					Type:        "string",
					Description: "The resource identifier (e.g., 'postgres', 'redis', 'ollama', 'qdrant')",
				},
			},
			Required: []string{"resource_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"resources", "details"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get PostgreSQL resource details",
					map[string]interface{}{
						"resource_id": "postgres",
					},
				),
				NewToolExample(
					"Get Ollama resource details",
					map[string]interface{}{
						"resource_id": "ollama",
					},
				),
			},
		},
	}
}

// startResourceTool returns the resource start tool.
func (p *ResourceToolProvider) startResourceTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "start_resource",
		Description: "Start a stopped Vrooli resource. This initializes the resource and makes it available for scenarios to use.",
		Category:    "resources",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"resource_id": {
					Type:        "string",
					Description: "The resource identifier (e.g., 'postgres', 'redis', 'ollama')",
				},
			},
			Required: []string{"resource_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // State-modifying: starts system resource
			TimeoutSeconds:     120,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"resources", "start", "lifecycle"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Start PostgreSQL",
					map[string]interface{}{
						"resource_id": "postgres",
					},
				),
			},
		},
	}
}

// stopResourceTool returns the resource stop tool.
func (p *ResourceToolProvider) stopResourceTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "stop_resource",
		Description: "Stop a running Vrooli resource. Warning: This may affect scenarios that depend on this resource.",
		Category:    "resources",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"resource_id": {
					Type:        "string",
					Description: "The resource identifier (e.g., 'postgres', 'redis', 'ollama')",
				},
			},
			Required: []string{"resource_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // State-modifying: stops system resource
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"resources", "stop", "lifecycle"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Stop Redis",
					map[string]interface{}{
						"resource_id": "redis",
					},
				),
			},
		},
	}
}
