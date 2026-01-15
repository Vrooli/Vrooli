// Package toolregistry provides tool definitions for scenario-to-desktop.
//
// This file defines pipeline orchestration tools for desktop application packaging.
// These tools provide unified control over the full desktop build pipeline
// (bundle -> preflight -> generate -> build -> distribution -> smoketest).
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// PipelineToolProvider provides pipeline orchestration tools.
type PipelineToolProvider struct{}

// NewPipelineToolProvider creates a new PipelineToolProvider.
func NewPipelineToolProvider() *PipelineToolProvider {
	return &PipelineToolProvider{}
}

// Name returns the provider identifier.
func (p *PipelineToolProvider) Name() string {
	return "scenario-to-desktop-pipeline"
}

// Categories returns the tool categories for pipeline tools.
func (p *PipelineToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "pipeline",
			Name:         "Pipeline Orchestration",
			Description:  "Tools for running and managing the desktop build pipeline",
			Icon:         "git-branch",
			DisplayOrder: 1,
		},
	}
}

// Tools returns the pipeline tool definitions.
func (p *PipelineToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.runPipelineTool(),
		p.checkPipelineStatusTool(),
		p.cancelPipelineTool(),
		p.resumePipelineTool(),
		p.listPipelinesTool(),
	}
}

// runPipelineTool returns the main pipeline execution tool (async).
func (p *PipelineToolProvider) runPipelineTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "run_pipeline",
		Description: "Run the desktop application build pipeline for a scenario. The pipeline executes stages in order: bundle -> preflight -> generate -> build -> distribution -> smoketest. You can stop at any stage using stop_after_stage, then resume later with resume_pipeline. This is a long-running async operation.",
		Category:    "pipeline",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario_name": {
					Type:        "string",
					Description: "Name of the scenario to build a desktop app for (required)",
				},
				"platforms": {
					Type:        "array",
					Description: "Target platforms to build for: win, mac, linux. Defaults to current platform.",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"deployment_mode": {
					Type:        "string",
					Default:     StringValue("bundled"),
					Description: "Deployment mode: 'bundled' (includes scenario runtime) or 'proxy' (connects to external server)",
				},
				"stop_after_stage": {
					Type:        "string",
					Description: "Stop pipeline after this stage completes: bundle, preflight, generate, build, distribution, smoketest. Empty runs all stages.",
				},
				"skip_preflight": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Skip preflight validation stage",
				},
				"skip_smoke_test": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Skip smoke test stage",
				},
				"proxy_url": {
					Type:        "string",
					Description: "Override the proxy URL for the scenario's UI (required when deployment_mode is 'proxy')",
				},
				"template_type": {
					Type:        "string",
					Default:     StringValue("basic"),
					Description: "Electron template type: basic (single window), advanced (multi-window), kiosk",
				},
				"distribute": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Enable distribution stage to upload artifacts to configured targets",
				},
				"distribution_targets": {
					Type:        "array",
					Description: "Distribution target names to upload to. Empty means all enabled targets.",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"sign": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Enable code signing during build stage",
				},
				"clean": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Force a clean build (removes existing desktop output)",
				},
				"version": {
					Type:        "string",
					Description: "Release version for distribution (e.g., 1.0.0)",
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
			Tags:               []string{"pipeline", "build", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Run full pipeline for all platforms",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"platforms":     []string{"win", "mac", "linux"},
					},
				),
				NewToolExample(
					"Run pipeline and stop after generate stage",
					map[string]interface{}{
						"scenario_name":    "agent-inbox",
						"stop_after_stage": "generate",
					},
				),
				NewToolExample(
					"Run pipeline with distribution enabled",
					map[string]interface{}{
						"scenario_name": "agent-inbox",
						"platforms":     []string{"linux"},
						"distribute":    true,
						"version":       "1.0.0",
					},
				),
				NewToolExample(
					"Run pipeline in proxy mode",
					map[string]interface{}{
						"scenario_name":   "agent-inbox",
						"deployment_mode": "proxy",
						"proxy_url":       "https://app.example.com/apps/agent-inbox/proxy/",
						"platforms":       []string{"win"},
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "check_pipeline_status",
					OperationIdField:       "pipeline_id",
					StatusToolIdParam:      "pipeline_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 3600, // 60 minutes max
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"completed"},
					FailureValues: []string{"failed", "cancelled"},
					PendingValues: []string{"pending", "running"},
				},
				ProgressTracking: &toolspb.ProgressTracking{
					ProgressField: "progress",
					MessageField:  "current_stage",
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:        "cancel_pipeline",
					CancelToolIdParam: "pipeline_id",
				},
			},
		},
	}
}

// checkPipelineStatusTool returns the pipeline status checking tool (polling target).
func (p *PipelineToolProvider) checkPipelineStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "check_pipeline_status",
		Description: "Check the current status and progress of a pipeline run. Shows which stages have completed, the current stage, any errors, and final artifacts when complete. This is the polling endpoint for run_pipeline.",
		Category:    "pipeline",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"pipeline_id": {
					Type:        "string",
					Description: "The pipeline ID to check status for (returned from run_pipeline or resume_pipeline)",
				},
			},
			Required: []string{"pipeline_id"},
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
			Tags:               []string{"pipeline", "status", "polling"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check pipeline progress",
					map[string]interface{}{
						"pipeline_id": "pipeline-abc123",
					},
				),
			},
		},
	}
}

// cancelPipelineTool returns the pipeline cancellation tool.
func (p *PipelineToolProvider) cancelPipelineTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "cancel_pipeline",
		Description: "Cancel a running pipeline. This stops the current stage and marks the pipeline as cancelled. Idempotent - calling on an already cancelled or completed pipeline has no effect.",
		Category:    "pipeline",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"pipeline_id": {
					Type:        "string",
					Description: "The pipeline ID to cancel (returned from run_pipeline or resume_pipeline)",
				},
			},
			Required: []string{"pipeline_id"},
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
			Tags:               []string{"pipeline", "cancel"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Cancel a running pipeline",
					map[string]interface{}{
						"pipeline_id": "pipeline-abc123",
					},
				),
			},
		},
	}
}

// resumePipelineTool returns the pipeline resume tool (async).
func (p *PipelineToolProvider) resumePipelineTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "resume_pipeline",
		Description: "Resume a pipeline that was stopped at a specific stage (using stop_after_stage). The resumed pipeline continues from the next stage, using results from the parent pipeline. Returns a new pipeline ID while linking to the parent.",
		Category:    "pipeline",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"pipeline_id": {
					Type:        "string",
					Description: "The parent pipeline ID to resume from (must have been stopped with stop_after_stage)",
				},
				"stop_after_stage": {
					Type:        "string",
					Description: "Optionally stop again after this stage. Empty runs remaining stages.",
				},
			},
			Required: []string{"pipeline_id"},
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
			Tags:               []string{"pipeline", "resume", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Resume a stopped pipeline",
					map[string]interface{}{
						"pipeline_id": "pipeline-abc123",
					},
				),
				NewToolExample(
					"Resume and stop after build stage",
					map[string]interface{}{
						"pipeline_id":      "pipeline-abc123",
						"stop_after_stage": "build",
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "check_pipeline_status",
					OperationIdField:       "pipeline_id",
					StatusToolIdParam:      "pipeline_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 3600,
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"completed"},
					FailureValues: []string{"failed", "cancelled"},
					PendingValues: []string{"pending", "running"},
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:        "cancel_pipeline",
					CancelToolIdParam: "pipeline_id",
				},
			},
		},
	}
}

// listPipelinesTool returns the pipeline listing tool.
func (p *PipelineToolProvider) listPipelinesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_pipelines",
		Description: "List all tracked pipeline runs with optional filtering by status or scenario.",
		Category:    "pipeline",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"status": {
					Type:        "string",
					Description: "Filter by pipeline status: pending, running, completed, failed, cancelled",
				},
				"scenario_name": {
					Type:        "string",
					Description: "Filter by scenario name",
				},
				"limit": {
					Type:        "integer",
					Default:     IntValue(50),
					Description: "Maximum number of pipelines to return",
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
			Tags:               []string{"pipeline", "list", "query"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all pipelines",
					map[string]interface{}{},
				),
				NewToolExample(
					"List running pipelines for a scenario",
					map[string]interface{}{
						"status":        "running",
						"scenario_name": "agent-inbox",
					},
				),
				NewToolExample(
					"List pipelines that can be resumed",
					map[string]interface{}{
						"status": "completed",
					},
				),
			},
		},
	}
}
