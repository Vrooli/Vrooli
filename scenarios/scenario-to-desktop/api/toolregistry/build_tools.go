// Package toolregistry provides tool definitions for scenario-to-desktop.
//
// This file defines build and generation tools for desktop application packaging.
// These tools generate Electron wrappers and build desktop applications.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// BuildToolProvider provides desktop build and generation tools.
type BuildToolProvider struct{}

// NewBuildToolProvider creates a new BuildToolProvider.
func NewBuildToolProvider() *BuildToolProvider {
	return &BuildToolProvider{}
}

// Name returns the provider identifier.
func (p *BuildToolProvider) Name() string {
	return "scenario-to-desktop-build"
}

// Categories returns the tool categories for build tools.
func (p *BuildToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "desktop_build",
			Name:         "Desktop Build",
			Description:  "Tools for generating and building desktop applications from scenarios",
			Icon:         "package",
			DisplayOrder: 1,
		},
	}
}

// Tools returns the build tool definitions.
func (p *BuildToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.generateDesktopWrapperTool(),
		p.buildForPlatformTool(),
		p.cancelBuildTool(),
		p.listBuildsTool(),
	}
}

// generateDesktopWrapperTool returns the desktop wrapper generation tool (async).
func (p *BuildToolProvider) generateDesktopWrapperTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "generate_desktop_wrapper",
		Description: "Generate an Electron wrapper for a scenario's UI and queue the build. Returns a build_id for status polling. This is a long-running operation that generates the Electron project structure and builds the desktop application.",
		Category:    "desktop_build",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario to generate a desktop app for",
				},
				"template_type": {
					Type:        "string",
					Default:     StringValue("universal"),
					Description: "Template type to use: universal (auto-detect), basic (single window), advanced (multi-window), multi_window, or kiosk",
				},
				"platforms": {
					Type:        "array",
					Description: "Target platforms to build for: win, mac, linux. Defaults to current platform if not specified.",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"proxy_url": {
					Type:        "string",
					Description: "Override the proxy URL for the scenario's UI (useful for development)",
				},
				"auto_manage_vrooli": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Whether the desktop app should auto-manage Vrooli lifecycle (start/stop)",
				},
			},
			Required: []string{"scenario_name"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"build", "electron", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Generate desktop app for agent-inbox",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"platforms":     []string{"win", "mac", "linux"},
					},
				),
				NewToolExample(
					"Generate kiosk-mode app",
					map[string]interface{}{
						"scenario_name": "browser-automation-studio",
						"template_type": "kiosk",
						"platforms":     []string{"linux"},
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "check_build_status",
					OperationIdField:       "build_id",
					StatusToolIdParam:      "build_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 1800, // 30 minutes max
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"ready", "partial"},
					FailureValues: []string{"failed"},
					PendingValues: []string{"building", "queued"},
				},
				ProgressTracking: &toolspb.ProgressTracking{
					ProgressField: "progress_percent",
					MessageField:  "current_step",
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:        "cancel_build",
					CancelToolIdParam: "build_id",
				},
			},
		},
	}
}

// buildForPlatformTool returns the platform-specific build tool (async).
func (p *BuildToolProvider) buildForPlatformTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "build_for_platform",
		Description: "Build a desktop application for specific platforms. Requires an existing desktop wrapper (run generate_desktop_wrapper first). Returns a build_id for status polling.",
		Category:    "desktop_build",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario with existing desktop wrapper",
				},
				"platforms": {
					Type:        "array",
					Description: "Target platforms to build for: win, mac, linux",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"clean": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Clean build artifacts before building (slower but ensures fresh build)",
				},
			},
			Required: []string{"scenario_name", "platforms"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"build", "platform", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Build for Windows and macOS",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"platforms":     []string{"win", "mac"},
					},
				),
				NewToolExample(
					"Clean build for Linux",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"platforms":     []string{"linux"},
						"clean":         true,
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "check_build_status",
					OperationIdField:       "build_id",
					StatusToolIdParam:      "build_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 1800,
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"ready", "partial"},
					FailureValues: []string{"failed"},
					PendingValues: []string{"building", "queued"},
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:        "cancel_build",
					CancelToolIdParam: "build_id",
				},
			},
		},
	}
}

// cancelBuildTool returns the build cancellation tool.
func (p *BuildToolProvider) cancelBuildTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "cancel_build",
		Description: "Cancel an in-progress build operation. This is an idempotent operation - calling it on an already cancelled or completed build has no effect.",
		Category:    "desktop_build",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"build_id": {
					Type:        "string",
					Description: "The build ID to cancel (returned from generate_desktop_wrapper or build_for_platform)",
				},
			},
			Required: []string{"build_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"build", "cancel"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Cancel a build",
					map[string]interface{}{
						"build_id": "build-abc123",
					},
				),
			},
		},
	}
}

// listBuildsTool returns the build listing tool.
func (p *BuildToolProvider) listBuildsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_builds",
		Description: "List all tracked build operations with optional filtering by status or scenario.",
		Category:    "desktop_build",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"status": {
					Type:        "string",
					Description: "Filter by build status: building, ready, partial, failed",
				},
				"scenario_name": {
					Type:        "string",
					Description: "Filter by scenario name",
				},
				"limit": {
					Type:        "integer",
					Default:     IntValue(50),
					Description: "Maximum number of builds to return",
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
			Tags:               []string{"build", "list", "query"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all builds",
					map[string]interface{}{},
				),
				NewToolExample(
					"List failed builds for a scenario",
					map[string]interface{}{
						"status":        "failed",
						"scenario_name": "agent-inbox",
					},
				),
			},
		},
	}
}
