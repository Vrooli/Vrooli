// Package toolregistry provides tool definitions for app-monitor.
//
// This file defines Swarm Manager fix backlog integration tools.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// IssueToolProvider provides fix backlog tools.
type IssueToolProvider struct{}

// NewIssueToolProvider creates a new IssueToolProvider.
func NewIssueToolProvider() *IssueToolProvider {
	return &IssueToolProvider{}
}

// Name returns the provider identifier.
func (p *IssueToolProvider) Name() string {
	return "app-monitor-issues"
}

// Categories returns the tool categories for fix tools.
func (p *IssueToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "issues",
			Name:         "Fix Backlog",
			Description:  "Tools for listing and creating Swarm Manager fix backlog items",
			Icon:         "bug",
			DisplayOrder: 5,
		},
	}
}

// Tools returns the fix backlog tool definitions.
func (p *IssueToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.listAppIssuesTool(),
		p.reportAppIssueTool(),
	}
}

// listAppIssuesTool returns the fix listing tool.
func (p *IssueToolProvider) listAppIssuesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_app_issues",
		Description: "List Swarm Manager fix backlog items for an application. Returns active, archived, and total counts with links to backlog items.",
		Category:    "issues",
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
			Tags:               []string{"fixes", "swarm-manager", "tracking"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List fixes for agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// reportAppIssueTool returns the fix reporting tool.
func (p *IssueToolProvider) reportAppIssueTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "report_app_issue",
		Description: "Create a Swarm Manager fix backlog item for an application. Can include screenshots, console logs, network requests, and health check results as evidence files.",
		Category:    "issues",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
				"message": {
					Type:        "string",
					Description: "Description of the fix needed",
				},
				"screenshot_data": {
					Type:        "string",
					Description: "Base64-encoded screenshot data (optional)",
				},
				"console_logs": {
					Type:        "array",
					Description: "Array of console log entries from the browser (optional)",
				},
				"network_requests": {
					Type:        "array",
					Description: "Array of network request entries (optional)",
				},
				"health_checks": {
					Type:        "array",
					Description: "Array of health check results (optional)",
				},
			},
			Required: []string{"app_id", "message"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 20,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"fixes", "swarm-manager", "reporting"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Report a simple fix",
					map[string]interface{}{
						"app_id":  "agent-inbox",
						"message": "Chat messages are not loading after page refresh",
					},
				),
				NewToolExample(
					"Report a fix with diagnostic context",
					map[string]interface{}{
						"app_id":  "agent-inbox",
						"message": "API returning 500 errors",
						"console_logs": []interface{}{
							map[string]interface{}{
								"level":   "error",
								"message": "Failed to fetch messages: 500 Internal Server Error",
							},
						},
					},
				),
			},
		},
	}
}
