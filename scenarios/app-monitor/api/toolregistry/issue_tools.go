// Package toolregistry provides tool definitions for app-monitor.
//
// This file defines issue tracking integration tools.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// IssueToolProvider provides issue tracking tools.
type IssueToolProvider struct{}

// NewIssueToolProvider creates a new IssueToolProvider.
func NewIssueToolProvider() *IssueToolProvider {
	return &IssueToolProvider{}
}

// Name returns the provider identifier.
func (p *IssueToolProvider) Name() string {
	return "app-monitor-issues"
}

// Categories returns the tool categories for issue tools.
func (p *IssueToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "issues",
			Name:         "Issue Management",
			Description:  "Tools for listing and reporting issues via app-issue-tracker",
			Icon:         "bug",
			DisplayOrder: 5,
		},
	}
}

// Tools returns the issue tracking tool definitions.
func (p *IssueToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.listAppIssuesTool(),
		p.reportAppIssueTool(),
	}
}

// listAppIssuesTool returns the issue listing tool.
func (p *IssueToolProvider) listAppIssuesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_app_issues",
		Description: "List issues for an application from app-issue-tracker. Returns open, active, and total issue counts with issue details and links to the tracker.",
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
			Tags:               []string{"issues", "tracking"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List issues for agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// reportAppIssueTool returns the issue reporting tool.
func (p *IssueToolProvider) reportAppIssueTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "report_app_issue",
		Description: "Report a new issue for an application to app-issue-tracker. Can include screenshots, console logs, network requests, and health check results for rich diagnostic context.",
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
					Description: "Description of the issue",
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
			RequiresApproval:   false, // Creating issues is generally safe
			TimeoutSeconds:     60,
			RateLimitPerMinute: 20,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         false, // Creates a new issue each time
			ModifiesState:      true,
			Tags:               []string{"issues", "reporting"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Report a simple issue",
					map[string]interface{}{
						"app_id":  "agent-inbox",
						"message": "Chat messages are not loading after page refresh",
					},
				),
				NewToolExample(
					"Report an issue with diagnostic context",
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
