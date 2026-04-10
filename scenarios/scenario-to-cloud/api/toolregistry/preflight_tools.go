// Package toolregistry provides tool definitions for scenario-to-cloud.
//
// This file defines validation and preflight tools.
// These tools help verify manifests and VPS readiness before deployment.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// ValidationToolProvider provides validation and preflight tools.
type ValidationToolProvider struct{}

// NewValidationToolProvider creates a new ValidationToolProvider.
func NewValidationToolProvider() *ValidationToolProvider {
	return &ValidationToolProvider{}
}

// Name returns the provider identifier.
func (p *ValidationToolProvider) Name() string {
	return "scenario-to-cloud-validation"
}

// Categories returns the tool categories for validation tools.
func (p *ValidationToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "validation",
			Name:         "Validation & Preflight",
			Description:  "Tools for validating manifests and running preflight checks",
			Icon:         "check-circle",
			DisplayOrder: 4,
		},
	}
}

// Tools returns the validation tool definitions.
func (p *ValidationToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.validateManifestTool(),
		p.runPreflightTool(),
	}
}

// validateManifestTool returns the manifest validation tool.
func (p *ValidationToolProvider) validateManifestTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "validate_manifest",
		Description: "Validate a deployment manifest for errors and warnings. Checks required fields, port configurations, dependency compatibility, and TLS settings. Does not connect to any VPS - purely validates the manifest structure.",
		Category:    "validation",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"manifest": {
					Type:        "object",
					Description: "The deployment manifest to validate. Must include target, scenario, dependencies, ports, and edge configuration.",
				},
			},
			Required: []string{"manifest"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"validation", "manifest", "preflight"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Validate a basic deployment manifest",
					map[string]interface{}{
						"manifest": map[string]interface{}{
							"version":     "1.0.0",
							"environment": "production",
							"target": map[string]interface{}{
								"type": "vps",
								"vps": map[string]interface{}{
									"host":     "192.168.1.100",
									"port":     22,
									"user":     "root",
									"key_path": "/home/user/.ssh/id_rsa",
									"workdir":  "/root/Vrooli",
								},
							},
							"scenario": map[string]interface{}{
								"id":  "agent-inbox",
								"ref": "main",
							},
						},
					},
				),
			},
		},
	}
}

// runPreflightTool returns the preflight check tool.
func (p *ValidationToolProvider) runPreflightTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "run_preflight",
		Description: "Run comprehensive preflight checks against a VPS target. Validates SSH connectivity, OS compatibility, disk space, port availability, DNS resolution, and firewall rules. Use this before execute_deployment to catch issues early.",
		Category:    "validation",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"manifest": {
					Type:        "object",
					Description: "The deployment manifest containing target VPS configuration",
				},
				"check_dns": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Check DNS resolution for the domain",
				},
				"check_ports": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Check if required ports are available",
				},
				"check_disk": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Check disk space availability",
				},
				"check_firewall": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Check firewall configuration for required ports",
				},
			},
			Required: []string{"manifest"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"preflight", "validation", "vps"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Run all preflight checks",
					map[string]interface{}{
						"manifest": map[string]interface{}{
							"target": map[string]interface{}{
								"type": "vps",
								"vps": map[string]interface{}{
									"host":     "192.168.1.100",
									"port":     22,
									"user":     "root",
									"key_path": "/home/user/.ssh/id_rsa",
								},
							},
						},
					},
				),
				NewToolExample(
					"Run only SSH and port checks",
					map[string]interface{}{
						"manifest": map[string]interface{}{
							"target": map[string]interface{}{
								"type": "vps",
								"vps": map[string]interface{}{
									"host":     "192.168.1.100",
									"port":     22,
									"user":     "root",
									"key_path": "/home/user/.ssh/id_rsa",
								},
							},
						},
						"check_dns":      false,
						"check_disk":     false,
						"check_firewall": false,
					},
				),
			},
		},
	}
}
