// Package toolregistry provides tool definitions for scenario-to-cloud.
//
// This file defines state-modifying deployment lifecycle tools.
// These tools can create, execute, stop, and start deployments.
// State-modifying tools require human approval before execution.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// DeploymentToolProvider provides deployment lifecycle tools.
type DeploymentToolProvider struct{}

// NewDeploymentToolProvider creates a new DeploymentToolProvider.
func NewDeploymentToolProvider() *DeploymentToolProvider {
	return &DeploymentToolProvider{}
}

// Name returns the provider identifier.
func (p *DeploymentToolProvider) Name() string {
	return "scenario-to-cloud-deployment"
}

// Categories returns the tool categories for deployment tools.
func (p *DeploymentToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "deployment_lifecycle",
			Name:         "Deployment Lifecycle",
			Description:  "Tools for creating, executing, and stopping deployments",
			Icon:         "rocket",
			DisplayOrder: 1,
		},
	}
}

// Tools returns the deployment lifecycle tool definitions.
func (p *DeploymentToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.createDeploymentTool(),
		p.executeDeploymentTool(),
		p.stopDeploymentTool(),
		p.startDeploymentTool(),
	}
}

// createDeploymentTool returns the deployment creation tool.
func (p *DeploymentToolProvider) createDeploymentTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "create_deployment",
		Description: "Create a new deployment record from a manifest. This prepares a deployment but does not execute it. Use execute_deployment to run the deployment after creation.",
		Category:    "deployment_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"name": {
					Type:        "string",
					Description: "Human-readable name for this deployment (auto-generated if not provided)",
				},
				"manifest": {
					Type:        "object",
					Description: "Deployment manifest defining target VPS, scenario, dependencies, ports, and edge configuration",
				},
			},
			Required: []string{"manifest"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false, // Just creates a record, doesn't execute
			TimeoutSeconds:     30,
			RateLimitPerMinute: 20,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"deployment", "create"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Create a named deployment",
					map[string]interface{}{
						"name": "agent-inbox-production",
						"manifest": map[string]interface{}{
							"version":     "1.0.0",
							"environment": "production",
							"scenario": map[string]interface{}{
								"id": "agent-inbox",
							},
						},
					},
				),
			},
		},
	}
}

// executeDeploymentTool returns the deployment execution tool (async, long-running).
func (p *DeploymentToolProvider) executeDeploymentTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "execute_deployment",
		Description: "Execute a deployment to deploy a scenario to a VPS. This runs the full deployment pipeline: bundle building, VPS setup, resource provisioning, and scenario startup. Use check_deployment_status to monitor progress. This is a long-running operation that can take 10-30 minutes.",
		Category:    "deployment_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"deployment": {
					Type:        "string",
					Description: "Deployment identifier - either the deployment name or UUID",
				},
				"run_preflight": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Run preflight checks before deployment (recommended for first deploy)",
				},
				"force_bundle_build": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Force rebuilding the bundle even if one exists with matching hash",
				},
				"provided_secrets": {
					Type:        "object",
					Description: "User-provided secrets for deployment (e.g., API keys, database passwords)",
				},
			},
			Required: []string{"deployment"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // State-modifying: deploys to VPS
			TimeoutSeconds:     60,   // Initial response timeout (returns run_id)
			RateLimitPerMinute: 5,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"deployment", "vps", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Deploy with preflight checks",
					map[string]interface{}{
						"deployment":    "my-production-deploy",
						"run_preflight": true,
					},
				),
				NewToolExample(
					"Force rebuild and deploy",
					map[string]interface{}{
						"deployment":         "my-production-deploy",
						"force_bundle_build": true,
					},
				),
				NewToolExample(
					"Deploy with secrets",
					map[string]interface{}{
						"deployment": "my-production-deploy",
						"provided_secrets": map[string]interface{}{
							"OPENROUTER_API_KEY": "sk-or-...",
						},
					},
				),
			},
			// Define async behavior for status polling
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "check_deployment_status",
					OperationIdField:       "run_id",
					StatusToolIdParam:      "deployment",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 1800, // 30 minutes max
					Backoff: &toolspb.PollingBackoff{
						InitialIntervalSeconds: 5,
						MaxIntervalSeconds:     30,
						Multiplier:             1.5,
					},
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"deployed"},
					FailureValues: []string{"failed"},
					PendingValues: []string{"pending", "setup_running", "setup_complete", "deploying"},
					ErrorField:    "error_message",
					ResultField:   "deploy_result",
				},
				ProgressTracking: &toolspb.ProgressTracking{
					ProgressField: "progress_percent",
					MessageField:  "progress_step",
					PhaseField:    "status",
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:           "stop_deployment",
					CancelToolIdParam:    "deployment",
					Graceful:             true,
					CancelTimeoutSeconds: 60,
				},
			},
		},
	}
}

// stopDeploymentTool returns the deployment stop tool.
func (p *DeploymentToolProvider) stopDeploymentTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "stop_deployment",
		Description: "Stop a running deployment on the VPS. This gracefully stops the scenario processes, but leaves the VPS and resources intact. Use start_deployment to restart.",
		Category:    "deployment_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"deployment": {
					Type:        "string",
					Description: "Deployment identifier - either the deployment name or UUID",
				},
				"force": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Force stop with SIGKILL if graceful shutdown fails",
				},
			},
			Required: []string{"deployment"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // State-modifying: stops running services
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"deployment", "stop", "lifecycle"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Gracefully stop a deployment",
					map[string]interface{}{
						"deployment": "my-production-deploy",
					},
				),
				NewToolExample(
					"Force stop a stuck deployment",
					map[string]interface{}{
						"deployment": "my-production-deploy",
						"force":      true,
					},
				),
			},
		},
	}
}

// startDeploymentTool returns the deployment start tool (async).
func (p *DeploymentToolProvider) startDeploymentTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "start_deployment",
		Description: "Start or resume a stopped deployment. This restarts the scenario processes on the VPS without re-running the full setup. Faster than execute_deployment for previously deployed scenarios.",
		Category:    "deployment_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"deployment": {
					Type:        "string",
					Description: "Deployment identifier - either the deployment name or UUID",
				},
			},
			Required: []string{"deployment"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // State-modifying: starts services
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        true,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"deployment", "start", "lifecycle", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Start a stopped deployment",
					map[string]interface{}{
						"deployment": "my-production-deploy",
					},
				),
			},
			// Async behavior for start operation
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "check_deployment_status",
					OperationIdField:       "run_id",
					StatusToolIdParam:      "deployment",
					PollIntervalSeconds:    3,
					MaxPollDurationSeconds: 300, // 5 minutes max for start
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"deployed"},
					FailureValues: []string{"failed"},
					PendingValues: []string{"deploying"},
					ErrorField:    "error_message",
				},
			},
		},
	}
}
