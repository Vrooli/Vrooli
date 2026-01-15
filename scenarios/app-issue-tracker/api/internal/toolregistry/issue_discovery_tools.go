// Package toolregistry provides tool definitions for app-issue-tracker.
//
// This file defines read-only discovery tools for finding and listing issues.
// These tools do not modify state and never require approval.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// IssueDiscoveryProvider provides read-only issue discovery tools.
type IssueDiscoveryProvider struct{}

// NewIssueDiscoveryProvider creates a new IssueDiscoveryProvider.
func NewIssueDiscoveryProvider() *IssueDiscoveryProvider {
	return &IssueDiscoveryProvider{}
}

// Name returns the provider identifier.
func (p *IssueDiscoveryProvider) Name() string {
	return "issue-tracker-discovery"
}

// Categories returns the tool categories for discovery tools.
func (p *IssueDiscoveryProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "issue_discovery",
			Name:         "Issue Discovery",
			Description:  "Tools for finding and listing issues",
			Icon:         "search",
			DisplayOrder: 1,
		},
	}
}

// Tools returns the read-only discovery tool definitions.
func (p *IssueDiscoveryProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.listIssuesTool(),
		p.getIssueTool(),
		p.searchIssuesTool(),
		p.getIssueStatisticsTool(),
		p.listApplicationsTool(),
	}
}

// listIssuesTool returns the issue listing tool.
func (p *IssueDiscoveryProvider) listIssuesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_issues",
		Description: "List all issues with optional filtering by status, priority, type, or target. Returns issue ID, title, status, priority, and timestamps.",
		Category:    "issue_discovery",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"status": {
					Type:        "string",
					Enum:        []string{"open", "active", "completed", "failed", "archived"},
					Description: "Filter by issue status. If not provided, returns all issues.",
				},
				"priority": {
					Type:        "string",
					Enum:        []string{"critical", "high", "medium", "low"},
					Description: "Filter by priority level.",
				},
				"type": {
					Type:        "string",
					Enum:        []string{"bug", "feature", "task", "improvement"},
					Description: "Filter by issue type.",
				},
				"target_id": {
					Type:        "string",
					Description: "Filter by target scenario or resource ID.",
				},
				"target_type": {
					Type:        "string",
					Enum:        []string{"scenario", "resource"},
					Description: "Filter by target type (scenario or resource).",
				},
				"limit": {
					Type:        "integer",
					Default:     IntValue(50),
					Description: "Maximum number of issues to return (default: 50, max: 200)",
				},
			},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"list", "query", "issues"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all open issues",
					map[string]interface{}{
						"status": "open",
					},
				),
				NewToolExample(
					"List high-priority bugs",
					map[string]interface{}{
						"priority": "high",
						"type":     "bug",
					},
				),
				NewToolExample(
					"List issues for a specific scenario",
					map[string]interface{}{
						"target_id":   "agent-inbox",
						"target_type": "scenario",
						"limit":       20,
					},
				),
			},
		},
	}
}

// getIssueTool returns the single issue retrieval tool.
func (p *IssueDiscoveryProvider) getIssueTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_issue",
		Description: "Get complete details for a single issue including metadata, error context, investigation results, and fix information.",
		Category:    "issue_discovery",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"issue_id": {
					Type:        "string",
					Description: "The unique identifier of the issue to retrieve",
				},
			},
			Required: []string{"issue_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"get", "details", "issues"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get issue by ID",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
					},
				),
			},
		},
	}
}

// searchIssuesTool returns the issue search tool.
func (p *IssueDiscoveryProvider) searchIssuesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "search_issues",
		Description: "Search for issues by keyword across title, description, and error messages. Returns matching issues sorted by relevance.",
		Category:    "issue_discovery",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"query": {
					Type:        "string",
					Description: "Search query - keywords to match against issue content",
				},
				"limit": {
					Type:        "integer",
					Default:     IntValue(20),
					Description: "Maximum number of results to return (default: 20, max: 100)",
				},
			},
			Required: []string{"query"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"search", "query", "issues"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Search for connection timeout issues",
					map[string]interface{}{
						"query": "connection timeout",
					},
				),
				NewToolExample(
					"Search for authentication errors",
					map[string]interface{}{
						"query": "authentication failed 401",
						"limit": 10,
					},
				),
			},
		},
	}
}

// getIssueStatisticsTool returns the statistics tool.
func (p *IssueDiscoveryProvider) getIssueStatisticsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_issue_statistics",
		Description: "Get dashboard statistics including total issues, open count, completion rates, average resolution time, top applications, and failure metrics.",
		Category:    "issue_discovery",
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
			Tags:               []string{"statistics", "dashboard", "metrics"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get current issue statistics",
					map[string]interface{}{},
				),
			},
		},
	}
}

// listApplicationsTool returns the applications listing tool.
func (p *IssueDiscoveryProvider) listApplicationsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_applications",
		Description: "List all applications (scenarios and resources) that have filed issues, with issue counts per application.",
		Category:    "issue_discovery",
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
			Tags:               []string{"applications", "list", "aggregation"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all applications with issues",
					map[string]interface{}{},
				),
			},
		},
	}
}
