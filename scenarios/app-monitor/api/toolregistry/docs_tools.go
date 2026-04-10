// Package toolregistry provides tool definitions for app-monitor.
//
// This file defines documentation retrieval tools.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// DocsToolProvider provides documentation tools.
type DocsToolProvider struct{}

// NewDocsToolProvider creates a new DocsToolProvider.
func NewDocsToolProvider() *DocsToolProvider {
	return &DocsToolProvider{}
}

// Name returns the provider identifier.
func (p *DocsToolProvider) Name() string {
	return "app-monitor-docs"
}

// Categories returns the tool categories for documentation tools.
func (p *DocsToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "docs",
			Name:         "Documentation",
			Description:  "Tools for listing, retrieving, and searching scenario documentation",
			Icon:         "book",
			DisplayOrder: 6,
		},
	}
}

// Tools returns the documentation tool definitions.
func (p *DocsToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.listAppDocsTool(),
		p.getAppDocTool(),
		p.searchAppDocsTool(),
	}
}

// listAppDocsTool returns the documentation listing tool.
func (p *DocsToolProvider) listAppDocsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_app_docs",
		Description: "List all documentation files for an application. Returns paths and names of available documentation.",
		Category:    "docs",
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
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"docs", "list"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List docs for agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// getAppDocTool returns the single document retrieval tool.
func (p *DocsToolProvider) getAppDocTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_app_doc",
		Description: "Get a specific documentation file for an application. Returns the content of the requested document.",
		Category:    "docs",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
				"path": {
					Type:        "string",
					Description: "Path to the documentation file (e.g., 'README.md', 'docs/api.md')",
				},
			},
			Required: []string{"app_id", "path"},
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
			Tags:               []string{"docs", "read"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get README for agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
						"path":   "README.md",
					},
				),
			},
		},
	}
}

// searchAppDocsTool returns the documentation search tool.
func (p *DocsToolProvider) searchAppDocsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "search_app_docs",
		Description: "Search documentation content for an application. Returns matching snippets and file locations.",
		Category:    "docs",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
				"query": {
					Type:        "string",
					Description: "Search query string",
				},
			},
			Required: []string{"app_id", "query"},
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
			Tags:               []string{"docs", "search"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Search for API documentation",
					map[string]interface{}{
						"app_id": "agent-inbox",
						"query":  "authentication",
					},
				),
			},
		},
	}
}
