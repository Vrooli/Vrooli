// Package toolregistry provides the tool discovery service for workspace-sandbox.
//
// This file defines the sandbox lifecycle tools (Tier 1) that are exposed
// via the Tool Discovery Protocol.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// SandboxToolProvider provides the sandbox lifecycle tools.
// These tools enable creating, managing, and destroying sandboxes.
type SandboxToolProvider struct{}

// NewSandboxToolProvider creates a new SandboxToolProvider.
func NewSandboxToolProvider() *SandboxToolProvider {
	return &SandboxToolProvider{}
}

// Name returns the provider identifier.
func (p *SandboxToolProvider) Name() string {
	return "workspace-sandbox-lifecycle"
}

// Categories returns the tool categories for sandbox lifecycle.
func (p *SandboxToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "sandbox_lifecycle",
			Name:        "Sandbox Lifecycle",
			Description: "Tools for creating, managing, and destroying isolated workspaces",
			Icon:        "box",
		},
	}
}

// Tools returns the tool definitions for sandbox lifecycle.
func (p *SandboxToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.createSandboxTool(),
		p.getSandboxTool(),
		p.listSandboxesTool(),
		p.deleteSandboxTool(),
		p.startSandboxTool(),
		p.stopSandboxTool(),
	}
}

func (p *SandboxToolProvider) createSandboxTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "create_sandbox",
		Description: "Create a new isolated sandbox workspace with copy-on-write filesystem. The sandbox provides an isolated environment for making changes without affecting the original files.",
		Category:    "sandbox_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scope_path": {
					Type:        "string",
					Description: "Path within the project to scope the sandbox to (e.g., 'src/components'). The sandbox will only track changes within this path.",
				},
				"project_root": {
					Type:        "string",
					Description: "Root path of the project. Defaults to VROOLI_ROOT if not specified.",
				},
				"owner": {
					Type:        "string",
					Description: "Identifier for the sandbox owner (e.g., agent run ID or user ID).",
				},
				"owner_type": {
					Type:        "string",
					Enum:        []string{"agent", "user", "system"},
					Default:     StringValue("agent"),
					Description: "Type of the owner for attribution and audit purposes.",
				},
				"metadata": {
					Type:        "object",
					Description: "Additional metadata to store with the sandbox (e.g., task description, context).",
				},
				"no_lock": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "If true, create sandbox without reserving the scope path (allows multiple sandboxes on same path).",
				},
			},
			Required: []string{"scope_path"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         false,
			Tags:               []string{"sandbox", "create", "workspace"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Create a sandbox for the components directory",
					map[string]interface{}{
						"scope_path": "src/components",
						"owner":      "agent-run-123",
						"owner_type": "agent",
					},
				),
				NewToolExample(
					"Create an unlocked sandbox with metadata",
					map[string]interface{}{
						"scope_path": "src/api",
						"no_lock":    true,
						"metadata": map[string]interface{}{
							"task": "Implement new API endpoint",
						},
					},
				),
			},
		},
	}
}

func (p *SandboxToolProvider) getSandboxTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_sandbox",
		Description: "Get details about a specific sandbox including its status, paths, and metadata.",
		Category:    "sandbox_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The unique identifier of the sandbox to retrieve.",
				},
			},
			Required: []string{"sandbox_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 120,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"sandbox", "status", "info"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get sandbox details",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				),
			},
		},
	}
}

func (p *SandboxToolProvider) listSandboxesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_sandboxes",
		Description: "List sandboxes with optional filtering by status, owner, or project root.",
		Category:    "sandbox_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"status": {
					Type:        "string",
					Enum:        []string{"creating", "active", "stopped", "approved", "rejected", "deleted"},
					Description: "Filter by sandbox status.",
				},
				"owner": {
					Type:        "string",
					Description: "Filter by owner identifier.",
				},
				"project_root": {
					Type:        "string",
					Description: "Filter by project root path.",
				},
				"limit": {
					Type:        "integer",
					Default:     IntValue(50),
					Description: "Maximum number of sandboxes to return.",
				},
				"offset": {
					Type:        "integer",
					Default:     IntValue(0),
					Description: "Number of sandboxes to skip for pagination.",
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
			Tags:               []string{"sandbox", "list", "query"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all active sandboxes",
					map[string]interface{}{
						"status": "active",
					},
				),
				NewToolExample(
					"List sandboxes for a specific owner",
					map[string]interface{}{
						"owner": "agent-run-123",
						"limit": 10,
					},
				),
			},
		},
	}
}

func (p *SandboxToolProvider) deleteSandboxTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "delete_sandbox",
		Description: "Delete a sandbox and clean up all associated resources. This permanently removes the sandbox and any uncommitted changes.",
		Category:    "sandbox_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The unique identifier of the sandbox to delete.",
				},
				"force": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "If true, force deletion even if the sandbox has pending changes or running processes.",
				},
			},
			Required: []string{"sandbox_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"sandbox", "delete", "cleanup"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Delete a sandbox",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				),
				NewToolExample(
					"Force delete a sandbox with pending changes",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"force":      true,
					},
				),
			},
		},
	}
}

func (p *SandboxToolProvider) startSandboxTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "start_sandbox",
		Description: "Start (remount) a previously stopped sandbox. This restores access to the sandbox's isolated filesystem.",
		Category:    "sandbox_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The unique identifier of the sandbox to start.",
				},
			},
			Required: []string{"sandbox_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"sandbox", "start", "mount"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Start a stopped sandbox",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				),
			},
		},
	}
}

func (p *SandboxToolProvider) stopSandboxTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "stop_sandbox",
		Description: "Stop (unmount) a sandbox while preserving its data. The sandbox can be restarted later to continue work.",
		Category:    "sandbox_lifecycle",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The unique identifier of the sandbox to stop.",
				},
			},
			Required: []string{"sandbox_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"sandbox", "stop", "unmount"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Stop an active sandbox",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				),
			},
		},
	}
}
