// Package toolregistry provides tool definitions for scenario-to-desktop.
//
// This file defines distribution tools for desktop applications.
// These tools handle uploading artifacts to S3/R2 and publishing releases.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// DistributionToolProvider provides distribution tools.
type DistributionToolProvider struct{}

// NewDistributionToolProvider creates a new DistributionToolProvider.
func NewDistributionToolProvider() *DistributionToolProvider {
	return &DistributionToolProvider{}
}

// Name returns the provider identifier.
func (p *DistributionToolProvider) Name() string {
	return "scenario-to-desktop-distribution"
}

// Categories returns the tool categories for distribution tools.
func (p *DistributionToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "distribution",
			Name:         "Distribution",
			Description:  "Tools for distributing desktop applications to users",
			Icon:         "upload-cloud",
			DisplayOrder: 3,
		},
	}
}

// Tools returns the distribution tool definitions.
func (p *DistributionToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.uploadArtifactTool(),
		p.publishReleaseTool(),
		p.listArtifactsTool(),
		p.listDistributionTargetsTool(),
		p.validateDistributionTargetTool(),
	}
}

// uploadArtifactTool returns the artifact upload tool (async).
func (p *DistributionToolProvider) uploadArtifactTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "upload_artifact",
		Description: "Upload a built artifact to configured distribution targets (S3, R2, etc.). This is an async operation that uploads the file in the background. Requires approval as it uploads to external services.",
		Category:    "distribution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario whose artifact to upload",
				},
				"artifact_path": {
					Type:        "string",
					Description: "Path to the artifact file to upload",
				},
				"targets": {
					Type:        "array",
					Description: "Distribution target names to upload to. Defaults to all enabled targets.",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"version": {
					Type:        "string",
					Description: "Version tag for this upload (e.g., 1.0.0, latest)",
				},
			},
			Required: []string{"scenario_name", "artifact_path"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // Uploads to external
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        true,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"distribution", "upload", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Upload to all targets",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"artifact_path": "/path/to/AgentInbox-1.0.0.dmg",
						"version":       "1.0.0",
					},
				),
				NewToolExample(
					"Upload to specific target",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"artifact_path": "/path/to/AgentInbox-1.0.0.dmg",
						"targets":       []string{"r2-production"},
						"version":       "1.0.0",
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "check_distribution_status",
					OperationIdField:       "distribution_id",
					StatusToolIdParam:      "distribution_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 600, // 10 minutes max
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"uploaded", "completed"},
					FailureValues: []string{"failed"},
					PendingValues: []string{"uploading", "pending"},
				},
			},
		},
	}
}

// publishReleaseTool returns the release publishing tool (async).
func (p *DistributionToolProvider) publishReleaseTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "publish_release",
		Description: "Publish a full release with all platform artifacts to distribution targets. This uploads all specified artifacts and creates a release manifest. Requires approval.",
		Category:    "distribution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario to publish a release for",
				},
				"version": {
					Type:        "string",
					Description: "Version string for this release (e.g., 1.0.0)",
				},
				"release_notes": {
					Type:        "string",
					Description: "Release notes or changelog for this version",
				},
				"artifacts": {
					Type:        "object",
					Description: "Map of platform to artifact paths: {\"win\": \"/path/to/setup.exe\", \"mac\": \"/path/to/app.dmg\", \"linux\": \"/path/to/app.AppImage\"}",
				},
			},
			Required: []string{"scenario_name", "version", "artifacts"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // Publishes to external
			TimeoutSeconds:     120,
			RateLimitPerMinute: 5,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"distribution", "release", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Publish a multi-platform release",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"version":       "1.0.0",
						"release_notes": "Initial release with core features",
						"artifacts": map[string]interface{}{
							"win":   "/dist/AgentInbox-Setup-1.0.0.exe",
							"mac":   "/dist/AgentInbox-1.0.0.dmg",
							"linux": "/dist/AgentInbox-1.0.0.AppImage",
						},
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "check_distribution_status",
					OperationIdField:       "distribution_id",
					StatusToolIdParam:      "distribution_id",
					PollIntervalSeconds:    10,
					MaxPollDurationSeconds: 1800, // 30 minutes
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"published", "completed"},
					FailureValues: []string{"failed"},
					PendingValues: []string{"publishing", "uploading", "pending"},
				},
			},
		},
	}
}

// listArtifactsTool returns the artifact listing tool.
func (p *DistributionToolProvider) listArtifactsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_artifacts",
		Description: "List distributed artifacts for a scenario. Shows all uploaded artifacts with their versions, platforms, and distribution targets.",
		Category:    "distribution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario to list artifacts for",
				},
				"target": {
					Type:        "string",
					Description: "Filter by distribution target name",
				},
			},
			Required: []string{"scenario_name"},
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
			Tags:               []string{"distribution", "list", "query"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all artifacts",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
					},
				),
				NewToolExample(
					"List artifacts from specific target",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"target":        "r2-production",
					},
				),
			},
		},
	}
}

// listDistributionTargetsTool returns the distribution target listing tool.
func (p *DistributionToolProvider) listDistributionTargetsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_distribution_targets",
		Description: "List all configured distribution targets. Shows target names, types (S3, R2, local), and their enabled status.",
		Category:    "distribution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				// No required parameters
			},
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
			Tags:               []string{"distribution", "targets", "config"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all distribution targets",
					map[string]interface{}{},
				),
			},
		},
	}
}

// validateDistributionTargetTool returns the target validation tool.
func (p *DistributionToolProvider) validateDistributionTargetTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "validate_distribution_target",
		Description: "Test connectivity and permissions for a distribution target. Verifies that the target is accessible and properly configured for uploads.",
		Category:    "distribution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"target_name": {
					Type:        "string",
					Description: "Name of the distribution target to validate",
				},
			},
			Required: []string{"target_name"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 20,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"distribution", "validate", "connectivity"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Validate R2 target",
					map[string]interface{}{
						"target_name": "r2-production",
					},
				),
			},
		},
	}
}
