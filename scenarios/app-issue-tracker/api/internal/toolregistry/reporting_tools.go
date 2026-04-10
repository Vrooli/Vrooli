// Package toolregistry provides tool definitions for app-issue-tracker.
//
// This file defines tools for reporting and exporting issue data.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// ReportingToolProvider provides reporting and export tools.
type ReportingToolProvider struct{}

// NewReportingToolProvider creates a new ReportingToolProvider.
func NewReportingToolProvider() *ReportingToolProvider {
	return &ReportingToolProvider{}
}

// Name returns the provider identifier.
func (p *ReportingToolProvider) Name() string {
	return "issue-tracker-reporting"
}

// Categories returns the tool categories for reporting tools.
func (p *ReportingToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "reporting",
			Name:         "Reporting & Export",
			Description:  "Tools for generating reports and exporting issue data",
			Icon:         "file-text",
			DisplayOrder: 5,
		},
	}
}

// Tools returns the reporting tool definitions.
func (p *ReportingToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.exportIssuesTool(),
		p.analyzeFailuresTool(),
	}
}

// exportIssuesTool returns the issue export tool.
func (p *ReportingToolProvider) exportIssuesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "export_issues",
		Description: "Export issues in JSON, CSV, or Markdown format with optional filtering. Useful for generating reports or backing up issue data.",
		Category:    "reporting",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"format": {
					Type:        "string",
					Enum:        []string{"json", "csv", "markdown"},
					Default:     StringValue("json"),
					Description: "Output format for the export",
				},
				"status": {
					Type:        "string",
					Enum:        []string{"open", "active", "completed", "failed", "archived"},
					Description: "Filter by issue status",
				},
				"priority": {
					Type:        "string",
					Enum:        []string{"critical", "high", "medium", "low"},
					Description: "Filter by priority level",
				},
				"type": {
					Type:        "string",
					Enum:        []string{"bug", "feature", "task", "improvement"},
					Description: "Filter by issue type",
				},
				"target_id": {
					Type:        "string",
					Description: "Filter by target scenario or resource",
				},
			},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"export", "report", "data"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Export all issues as JSON",
					map[string]interface{}{
						"format": "json",
					},
				),
				NewToolExample(
					"Export open bugs as CSV",
					map[string]interface{}{
						"format": "csv",
						"status": "open",
						"type":   "bug",
					},
				),
				NewToolExample(
					"Generate markdown report of failed issues",
					map[string]interface{}{
						"format": "markdown",
						"status": "failed",
					},
				),
			},
		},
	}
}

// analyzeFailuresTool returns the failure analysis tool.
func (p *ReportingToolProvider) analyzeFailuresTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "analyze_failures",
		Description: "Analyze failed issues to identify patterns and common failure reasons. Returns breakdown by failure type, affected targets, and trends.",
		Category:    "reporting",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"since": {
					Type:        "string",
					Format:      "date-time",
					Description: "Only analyze failures since this date (ISO 8601 format)",
				},
				"target_id": {
					Type:        "string",
					Description: "Filter to failures affecting a specific target",
				},
			},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 20,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"analysis", "failures", "patterns"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Analyze all failures",
					map[string]interface{}{},
				),
				NewToolExample(
					"Analyze recent failures for a specific scenario",
					map[string]interface{}{
						"since":     "2024-01-01T00:00:00Z",
						"target_id": "agent-inbox",
					},
				),
			},
		},
	}
}
