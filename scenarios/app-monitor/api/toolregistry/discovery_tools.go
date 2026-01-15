// Package toolregistry provides tool definitions for app-monitor.
//
// This file defines read-only discovery tools for listing and inspecting apps.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// DiscoveryToolProvider provides app discovery and status tools.
type DiscoveryToolProvider struct{}

// NewDiscoveryToolProvider creates a new DiscoveryToolProvider.
func NewDiscoveryToolProvider() *DiscoveryToolProvider {
	return &DiscoveryToolProvider{}
}

// Name returns the provider identifier.
func (p *DiscoveryToolProvider) Name() string {
	return "app-monitor-discovery"
}

// Categories returns the tool categories for discovery tools.
func (p *DiscoveryToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "app_discovery",
			Name:         "App Discovery",
			Description:  "Tools for discovering and inspecting monitored applications",
			Icon:         "search",
			DisplayOrder: 1,
		},
	}
}

// Tools returns the discovery tool definitions.
func (p *DiscoveryToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.listAppsTool(),
		p.getAppTool(),
		p.getSystemStatusTool(),
		p.getAppSummaryTool(),
		p.listResourcesTool(),
	}
}

// listAppsTool returns the app listing tool.
func (p *DiscoveryToolProvider) listAppsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_apps",
		Description: "List all monitored applications in the Vrooli ecosystem. Returns app names, statuses, health indicators, port mappings, and basic metadata. Use this to get an overview of what's running.",
		Category:    "app_discovery",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
			Required:   []string{},
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
			Tags:               []string{"discovery", "apps", "list"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all running applications",
					map[string]interface{}{},
				),
			},
		},
	}
}

// getAppTool returns the single app retrieval tool.
func (p *DiscoveryToolProvider) getAppTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_app",
		Description: "Get detailed information about a specific application including its status, dependencies, tech stack, port mappings, environment, and completeness score.",
		Category:    "app_discovery",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name or UUID)",
				},
			},
			Required: []string{"app_id"},
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
			Tags:               []string{"discovery", "apps", "details"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get details for agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// getSystemStatusTool returns the system status tool.
func (p *DiscoveryToolProvider) getSystemStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_system_status",
		Description: "Get complete system status snapshot including all running apps, resources, and overall health of the Vrooli ecosystem.",
		Category:    "app_discovery",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
			Required:   []string{},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"discovery", "system", "status"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get complete system status",
					map[string]interface{}{},
				),
			},
		},
	}
}

// getAppSummaryTool returns the fast app summary tool.
func (p *DiscoveryToolProvider) getAppSummaryTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_app_summary",
		Description: "Get a fast-loading summary of all apps. Returns basic info with background hydration of detailed data. Ideal for dashboards and quick overviews.",
		Category:    "app_discovery",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
			Required:   []string{},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"discovery", "apps", "summary"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get app summary for dashboard",
					map[string]interface{}{},
				),
			},
		},
	}
}

// listResourcesTool returns the resource listing tool.
func (p *DiscoveryToolProvider) listResourcesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_resources",
		Description: "List all Vrooli resources (PostgreSQL, Redis, Ollama, Qdrant, etc.) with their current status, health, and configuration.",
		Category:    "app_discovery",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
			Required:   []string{},
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
			Tags:               []string{"discovery", "resources", "list"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all resources",
					map[string]interface{}{},
				),
			},
		},
	}
}
