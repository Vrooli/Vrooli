// Package toolregistry provides tool definitions for scenario-to-desktop.
//
// This file defines inspection and status tools for desktop builds.
// These are read-only tools for checking build status, pipeline progress, and system prerequisites.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// InspectionToolProvider provides inspection and status tools.
type InspectionToolProvider struct{}

// NewInspectionToolProvider creates a new InspectionToolProvider.
func NewInspectionToolProvider() *InspectionToolProvider {
	return &InspectionToolProvider{}
}

// Name returns the provider identifier.
func (p *InspectionToolProvider) Name() string {
	return "scenario-to-desktop-inspection"
}

// Categories returns the tool categories for inspection tools.
func (p *InspectionToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "build_status",
			Name:         "Build Status",
			Description:  "Tools for checking build status and progress",
			Icon:         "activity",
			DisplayOrder: 4,
		},
		{
			Id:           "desktop_inspection",
			Name:         "Desktop Inspection",
			Description:  "Tools for inspecting generated desktop applications and configurations",
			Icon:         "search",
			DisplayOrder: 5,
		},
	}
}

// Tools returns the inspection tool definitions.
func (p *InspectionToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.checkBuildStatusTool(),
		p.getPipelineStatusTool(),
		p.listGeneratedWrappersTool(),
		p.validateConfigurationTool(),
		p.getSystemPrerequisitesTool(),
		p.checkDistributionStatusTool(),
	}
}

// checkBuildStatusTool returns the build status checking tool (polling target).
func (p *InspectionToolProvider) checkBuildStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "check_build_status",
		Description: "Check the current status and progress of a build operation. This is the polling endpoint for async build operations (generate_desktop_wrapper, build_for_platform).",
		Category:    "build_status",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"build_id": {
					Type:        "string",
					Description: "The build ID to check status for (returned from generate_desktop_wrapper or build_for_platform)",
				},
			},
			Required: []string{"build_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 120, // High rate for polling
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"status", "polling", "build"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check build progress",
					map[string]interface{}{
						"build_id": "build-abc123",
					},
				),
			},
		},
	}
}

// getPipelineStatusTool returns the pipeline status tool.
func (p *InspectionToolProvider) getPipelineStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_pipeline_status",
		Description: "Get the status of a full pipeline run (bundle -> generate -> build -> distribute). Shows progress through each stage and any errors.",
		Category:    "build_status",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"pipeline_id": {
					Type:        "string",
					Description: "The pipeline ID to check status for",
				},
			},
			Required: []string{"pipeline_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"status", "pipeline", "polling"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check pipeline progress",
					map[string]interface{}{
						"pipeline_id": "pipeline-xyz789",
					},
				),
			},
		},
	}
}

// listGeneratedWrappersTool returns the wrapper listing tool.
func (p *InspectionToolProvider) listGeneratedWrappersTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_generated_wrappers",
		Description: "List all scenarios that have generated desktop wrappers. Shows which scenarios have Electron projects and their current build status.",
		Category:    "desktop_inspection",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"limit": {
					Type:        "integer",
					Default:     IntValue(50),
					Description: "Maximum number of wrappers to return",
				},
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
			Tags:               []string{"inspection", "list", "wrappers"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all generated wrappers",
					map[string]interface{}{},
				),
			},
		},
	}
}

// validateConfigurationTool returns the configuration validation tool.
func (p *InspectionToolProvider) validateConfigurationTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "validate_configuration",
		Description: "Validate a scenario's desktop configuration without actually building. Checks template compatibility, required files, UI availability, and port configuration.",
		Category:    "desktop_inspection",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario to validate",
				},
			},
			Required: []string{"scenario_name"},
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
			Tags:               []string{"validation", "config", "preflight"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Validate scenario configuration",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
					},
				),
			},
		},
	}
}

// getSystemPrerequisitesTool returns the system prerequisites checking tool.
func (p *InspectionToolProvider) getSystemPrerequisitesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_system_prerequisites",
		Description: "Check system prerequisites for desktop application building. Verifies availability of Wine (for Windows builds on non-Windows), Xcode (for macOS signing), Node.js, npm, and other required tools.",
		Category:    "desktop_inspection",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				// No required parameters
			},
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
			Tags:               []string{"system", "prerequisites", "validation"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check system prerequisites",
					map[string]interface{}{},
				),
			},
		},
	}
}

// checkDistributionStatusTool returns the distribution status checking tool (polling target).
func (p *InspectionToolProvider) checkDistributionStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "check_distribution_status",
		Description: "Check the status of a distribution operation (upload or publish). This is the polling endpoint for async distribution operations.",
		Category:    "build_status",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"distribution_id": {
					Type:        "string",
					Description: "The distribution operation ID to check status for (returned from upload_artifact or publish_release)",
				},
			},
			Required: []string{"distribution_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 120, // High rate for polling
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"status", "polling", "distribution"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check upload progress",
					map[string]interface{}{
						"distribution_id": "dist-abc123",
					},
				),
			},
		},
	}
}
