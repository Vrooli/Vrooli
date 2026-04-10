// Package toolregistry provides tool definitions for scenario-to-cloud.
//
// This file defines read-only inspection tools for monitoring deployments.
// These tools do not modify state and never require approval.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// InspectionToolProvider provides read-only deployment inspection tools.
type InspectionToolProvider struct{}

// NewInspectionToolProvider creates a new InspectionToolProvider.
func NewInspectionToolProvider() *InspectionToolProvider {
	return &InspectionToolProvider{}
}

// Name returns the provider identifier.
func (p *InspectionToolProvider) Name() string {
	return "scenario-to-cloud-inspection"
}

// Categories returns the tool categories for inspection tools.
func (p *InspectionToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "deployment_status",
			Name:         "Deployment Status",
			Description:  "Tools for checking deployment status and progress",
			Icon:         "activity",
			DisplayOrder: 2,
		},
		{
			Id:           "deployment_inspection",
			Name:         "Deployment Inspection",
			Description:  "Tools for detailed inspection of deployed scenarios",
			Icon:         "search",
			DisplayOrder: 3,
		},
	}
}

// Tools returns the read-only inspection tool definitions.
func (p *InspectionToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.checkDeploymentStatusTool(),
		p.listDeploymentsTool(),
		p.inspectDeploymentTool(),
		p.getDeploymentLogsTool(),
		p.getLiveStateTool(),
	}
}

// checkDeploymentStatusTool returns the status checking tool (used for async polling).
func (p *InspectionToolProvider) checkDeploymentStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "check_deployment_status",
		Description: "Check the current status and progress of a deployment. Returns status, progress percentage, current step, and any error information. Use this to poll for completion after starting execute_deployment.",
		Category:    "deployment_status",
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
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"status", "monitoring", "polling"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check status of a deployment by name",
					map[string]interface{}{
						"deployment": "my-production-deploy",
					},
				),
				NewToolExample(
					"Check status of a deployment by UUID",
					map[string]interface{}{
						"deployment": "550e8400-e29b-41d4-a716-446655440000",
					},
				),
			},
		},
	}
}

// listDeploymentsTool returns the deployment listing tool.
func (p *InspectionToolProvider) listDeploymentsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_deployments",
		Description: "List all deployments with optional filtering by status or scenario. Returns deployment ID, name, status, scenario, and timestamps.",
		Category:    "deployment_status",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"status": {
					Type:        "string",
					Enum:        []string{"pending", "setup_running", "setup_complete", "deploying", "deployed", "failed", "stopped"},
					Description: "Filter by deployment status. If not provided, returns all deployments.",
				},
				"scenario": {
					Type:        "string",
					Description: "Filter by scenario ID. If not provided, returns deployments for all scenarios.",
				},
				"limit": {
					Type:        "integer",
					Default:     IntValue(50),
					Description: "Maximum number of deployments to return (default: 50, max: 200)",
				},
			},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"list", "query"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all deployments",
					map[string]interface{}{},
				),
				NewToolExample(
					"List failed deployments",
					map[string]interface{}{
						"status": "failed",
					},
				),
				NewToolExample(
					"List deployments for a specific scenario",
					map[string]interface{}{
						"scenario": "agent-inbox",
						"limit":    10,
					},
				),
			},
		},
	}
}

// inspectDeploymentTool returns the deployment inspection tool.
func (p *InspectionToolProvider) inspectDeploymentTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "inspect_deployment",
		Description: "Get detailed VPS state for a deployment including scenario status, running processes, resource health, and container states. Performs a live inspection of the deployed VPS.",
		Category:    "deployment_inspection",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"deployment": {
					Type:        "string",
					Description: "Deployment identifier - either the deployment name or UUID",
				},
				"include_resources": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Include resource status (postgres, redis, etc.) in the inspection",
				},
				"include_processes": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Include running process information",
				},
			},
			Required: []string{"deployment"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 20,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"inspection", "monitoring", "vps"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Full inspection of a deployment",
					map[string]interface{}{
						"deployment": "my-production-deploy",
					},
				),
				NewToolExample(
					"Quick inspection without resource details",
					map[string]interface{}{
						"deployment":        "my-production-deploy",
						"include_resources": false,
					},
				),
			},
		},
	}
}

// getDeploymentLogsTool returns the log retrieval tool.
func (p *InspectionToolProvider) getDeploymentLogsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_deployment_logs",
		Description: "Retrieve logs from a deployed scenario. Returns the most recent log output from the scenario API and UI processes.",
		Category:    "deployment_inspection",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"deployment": {
					Type:        "string",
					Description: "Deployment identifier - either the deployment name or UUID",
				},
				"tail_lines": {
					Type:        "integer",
					Default:     IntValue(100),
					Description: "Number of log lines to retrieve from the end (default: 100, max: 1000)",
				},
				"source": {
					Type:        "string",
					Enum:        []string{"all", "api", "ui", "caddy"},
					Default:     StringValue("all"),
					Description: "Log source to retrieve. 'all' combines logs from all sources.",
				},
			},
			Required: []string{"deployment"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     20,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"logs", "debugging"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get recent logs from all sources",
					map[string]interface{}{
						"deployment": "my-production-deploy",
					},
				),
				NewToolExample(
					"Get API logs only",
					map[string]interface{}{
						"deployment": "my-production-deploy",
						"source":     "api",
						"tail_lines": 200,
					},
				),
			},
		},
	}
}

// getLiveStateTool returns the real-time VPS state tool.
func (p *InspectionToolProvider) getLiveStateTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_live_state",
		Description: "Get real-time VPS state including running processes, port usage, Docker containers, and system resources. More detailed than inspect_deployment, useful for debugging.",
		Category:    "deployment_inspection",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"deployment": {
					Type:        "string",
					Description: "Deployment identifier - either the deployment name or UUID",
				},
				"include_docker": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Include Docker container status",
				},
				"include_ports": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Include listening ports and their processes",
				},
				"include_system": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Include system resources (CPU, memory, disk)",
				},
			},
			Required: []string{"deployment"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 20,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"live-state", "debugging", "vps"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get full live state",
					map[string]interface{}{
						"deployment": "my-production-deploy",
					},
				),
				NewToolExample(
					"Get only Docker and port information",
					map[string]interface{}{
						"deployment":     "my-production-deploy",
						"include_docker": true,
						"include_ports":  true,
						"include_system": false,
					},
				),
			},
		},
	}
}
