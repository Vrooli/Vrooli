// Package toolregistry provides tool definitions for app-issue-tracker.
//
// This file defines tools for creating and modifying issues.
// These tools modify state but generally don't require approval.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// IssueManagementProvider provides issue CRUD tools.
type IssueManagementProvider struct{}

// NewIssueManagementProvider creates a new IssueManagementProvider.
func NewIssueManagementProvider() *IssueManagementProvider {
	return &IssueManagementProvider{}
}

// Name returns the provider identifier.
func (p *IssueManagementProvider) Name() string {
	return "issue-tracker-management"
}

// Categories returns the tool categories for management tools.
func (p *IssueManagementProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "issue_management",
			Name:         "Issue Management",
			Description:  "Tools for creating and modifying issues",
			Icon:         "edit",
			DisplayOrder: 2,
		},
	}
}

// Tools returns the issue management tool definitions.
func (p *IssueManagementProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.createIssueTool(),
		p.updateIssueTool(),
		p.transitionIssueTool(),
		p.addIssueArtifactsTool(),
	}
}

// createIssueTool returns the issue creation tool.
func (p *IssueManagementProvider) createIssueTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "create_issue",
		Description: "Create a new issue with title, description, priority, type, targets, and optional error context. Returns the created issue with its assigned ID.",
		Category:    "issue_management",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"title": {
					Type:        "string",
					Description: "Brief, descriptive title for the issue",
				},
				"description": {
					Type:        "string",
					Description: "Detailed description of the issue",
				},
				"type": {
					Type:        "string",
					Enum:        []string{"bug", "feature", "task", "improvement"},
					Default:     StringValue("bug"),
					Description: "Type of issue",
				},
				"priority": {
					Type:        "string",
					Enum:        []string{"critical", "high", "medium", "low"},
					Default:     StringValue("medium"),
					Description: "Priority level",
				},
				"targets": {
					Type:        "array",
					Description: "Target scenarios or resources this issue affects",
					Items: &toolspb.ParameterSchema{
						Type: "object",
						Properties: map[string]*toolspb.ParameterSchema{
							"id": {
								Type:        "string",
								Description: "Target identifier (scenario or resource name)",
							},
							"type": {
								Type:        "string",
								Enum:        []string{"scenario", "resource"},
								Description: "Target type",
							},
						},
					},
				},
				"tags": {
					Type:        "array",
					Description: "Tags for categorizing the issue",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"error_context": {
					Type:        "object",
					Description: "Error context including message, logs, and stack trace",
					Properties: map[string]*toolspb.ParameterSchema{
						"error_message": {
							Type:        "string",
							Description: "The error message",
						},
						"logs": {
							Type:        "string",
							Description: "Relevant log output",
						},
						"stack_trace": {
							Type:        "string",
							Description: "Stack trace if available",
						},
						"environment": {
							Type:        "string",
							Description: "Environment details (OS, version, etc.)",
						},
					},
				},
				"reporter": {
					Type:        "object",
					Description: "Reporter information",
					Properties: map[string]*toolspb.ParameterSchema{
						"name": {
							Type:        "string",
							Description: "Reporter name or identifier",
						},
						"email": {
							Type:        "string",
							Description: "Reporter email",
						},
					},
				},
			},
			Required: []string{"title"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"create", "issues"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Create a simple bug report",
					map[string]interface{}{
						"title":       "API returns 500 on login",
						"description": "The login endpoint returns a 500 error when using valid credentials",
						"type":        "bug",
						"priority":    "high",
					},
				),
				NewToolExample(
					"Create issue with full context",
					map[string]interface{}{
						"title":       "Database connection timeout",
						"description": "Postgres connection times out after 30 seconds",
						"type":        "bug",
						"priority":    "critical",
						"targets": []interface{}{
							map[string]interface{}{
								"id":   "agent-inbox",
								"type": "scenario",
							},
						},
						"error_context": map[string]interface{}{
							"error_message": "connection timed out after 30000ms",
							"logs":          "2024-01-15 10:30:00 ERROR pg: connection failed",
						},
					},
				),
			},
		},
	}
}

// updateIssueTool returns the issue update tool.
func (p *IssueManagementProvider) updateIssueTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "update_issue",
		Description: "Update an existing issue's metadata including title, description, priority, type, and tags. Does not change status (use transition_issue for that).",
		Category:    "issue_management",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"issue_id": {
					Type:        "string",
					Description: "The unique identifier of the issue to update",
				},
				"title": {
					Type:        "string",
					Description: "New title for the issue",
				},
				"description": {
					Type:        "string",
					Description: "New description for the issue",
				},
				"priority": {
					Type:        "string",
					Enum:        []string{"critical", "high", "medium", "low"},
					Description: "New priority level",
				},
				"type": {
					Type:        "string",
					Enum:        []string{"bug", "feature", "task", "improvement"},
					Description: "New issue type",
				},
				"tags": {
					Type:        "array",
					Description: "New tags (replaces existing tags)",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
			},
			Required: []string{"issue_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"update", "issues"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Update issue priority",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
						"priority": "critical",
					},
				),
				NewToolExample(
					"Update issue title and description",
					map[string]interface{}{
						"issue_id":    "issue-2024-01-15-001",
						"title":       "Updated: API returns 500 on login with special characters",
						"description": "The login endpoint fails when username contains special characters",
					},
				),
			},
		},
	}
}

// transitionIssueTool returns the status transition tool.
func (p *IssueManagementProvider) transitionIssueTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "transition_issue",
		Description: "Change an issue's status. Valid transitions: open→active (start investigation), active→completed (resolved), active→failed (investigation failed), any→archived (close without resolution).",
		Category:    "issue_management",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"issue_id": {
					Type:        "string",
					Description: "The unique identifier of the issue to transition",
				},
				"status": {
					Type:        "string",
					Enum:        []string{"open", "active", "completed", "failed", "archived"},
					Description: "The new status for the issue",
				},
				"reason": {
					Type:        "string",
					Description: "Optional reason for the transition (especially useful for failed/archived)",
				},
			},
			Required: []string{"issue_id", "status"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"transition", "status", "issues"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Mark issue as completed",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
						"status":   "completed",
						"reason":   "Fix applied and verified",
					},
				),
				NewToolExample(
					"Mark investigation as failed",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
						"status":   "failed",
						"reason":   "Unable to reproduce the issue",
					},
				),
			},
		},
	}
}

// addIssueArtifactsTool returns the artifact attachment tool.
func (p *IssueManagementProvider) addIssueArtifactsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "add_issue_artifacts",
		Description: "Attach artifacts (logs, screenshots, debug info) to an existing issue. Artifacts are stored alongside the issue metadata.",
		Category:    "issue_management",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"issue_id": {
					Type:        "string",
					Description: "The unique identifier of the issue to attach artifacts to",
				},
				"artifacts": {
					Type:        "array",
					Description: "Array of artifacts to attach",
					Items: &toolspb.ParameterSchema{
						Type: "object",
						Properties: map[string]*toolspb.ParameterSchema{
							"name": {
								Type:        "string",
								Description: "Filename for the artifact",
							},
							"type": {
								Type:        "string",
								Enum:        []string{"log", "screenshot", "debug", "network", "custom"},
								Description: "Type of artifact",
							},
							"content": {
								Type:        "string",
								Description: "Content of the artifact (text or base64 encoded)",
							},
							"encoding": {
								Type:        "string",
								Enum:        []string{"text", "base64"},
								Default:     StringValue("text"),
								Description: "Content encoding",
							},
							"description": {
								Type:        "string",
								Description: "Optional description of the artifact",
							},
						},
					},
				},
			},
			Required: []string{"issue_id", "artifacts"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 20,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"artifacts", "attachments", "issues"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Attach log file to issue",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
						"artifacts": []interface{}{
							map[string]interface{}{
								"name":        "error.log",
								"type":        "log",
								"content":     "2024-01-15 10:30:00 ERROR: Connection refused\n2024-01-15 10:30:01 ERROR: Retry failed",
								"description": "Application error logs around the time of failure",
							},
						},
					},
				),
			},
		},
	}
}
