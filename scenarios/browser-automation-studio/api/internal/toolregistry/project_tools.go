// Package toolregistry provides tool definitions for project and workflow management.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// ProjectToolProvider provides project and workflow management tools.
type ProjectToolProvider struct{}

// NewProjectToolProvider creates a new ProjectToolProvider.
func NewProjectToolProvider() *ProjectToolProvider {
	return &ProjectToolProvider{}
}

// Name returns the provider name.
func (p *ProjectToolProvider) Name() string {
	return "bas-project"
}

// Categories returns the categories for project tools.
func (p *ProjectToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		NewCategory("project_management", "Project Management", "Tools for creating and managing automation projects", "folder", 3),
		NewCategory("workflow_management", "Workflow Management", "Tools for creating and modifying workflows", "edit", 4),
	}
}

// Tools returns the project tool definitions.
func (p *ProjectToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.createWorkflowTool(),
		p.updateWorkflowTool(),
		p.validateWorkflowTool(),
		p.createProjectTool(),
		p.listProjectsTool(),
	}
}

// createWorkflowTool defines the create_workflow tool.
func (p *ProjectToolProvider) createWorkflowTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"name":        StringParam("Name for the new workflow"),
			"project_id":  StringParamWithFormat("Project ID to create the workflow in (optional)", "uuid"),
			"description": StringParam("Description of what the workflow does (optional)"),
			"definition":  ObjectParam("Workflow definition in JSON format (steps, actions, etc.)"),
		},
		[]string{"name"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 20,
		CostEstimate:       "low",
		Idempotent:         false,
		ModifiesState:      true,
		Tags:               []string{"create", "workflow"},
	})

	return NewToolDefinition(
		"create_workflow",
		"Create a new browser automation workflow. Provide a name and optionally a workflow definition in JSON format.",
		"workflow_management",
		params,
		meta,
	)
}

// updateWorkflowTool defines the update_workflow tool.
func (p *ProjectToolProvider) updateWorkflowTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"workflow_id": StringParamWithFormat("The ID of the workflow to update", "uuid"),
			"name":        StringParam("New name for the workflow (optional)"),
			"description": StringParam("New description (optional)"),
			"definition":  ObjectParam("Updated workflow definition in JSON format (optional)"),
		},
		[]string{"workflow_id"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 20,
		CostEstimate:       "low",
		Idempotent:         true,
		ModifiesState:      true,
		Tags:               []string{"update", "workflow"},
	})

	return NewToolDefinition(
		"update_workflow",
		"Update an existing workflow. Can modify name, description, or the workflow definition.",
		"workflow_management",
		params,
		meta,
	)
}

// validateWorkflowTool defines the validate_workflow tool.
func (p *ProjectToolProvider) validateWorkflowTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"definition": ObjectParam("Workflow definition in JSON format to validate"),
		},
		[]string{"definition"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     15,
		RateLimitPerMinute: 60,
		CostEstimate:       "low",
		Idempotent:         true,
		Tags:               []string{"validate", "workflow"},
	})

	return NewToolDefinition(
		"validate_workflow",
		"Validate a workflow definition without saving it. Returns validation errors if the definition is invalid.",
		"workflow_management",
		params,
		meta,
	)
}

// createProjectTool defines the create_project tool.
func (p *ProjectToolProvider) createProjectTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"name":        StringParam("Name for the new project"),
			"description": StringParam("Description of the project (optional)"),
		},
		[]string{"name"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 10,
		CostEstimate:       "low",
		Idempotent:         false,
		ModifiesState:      true,
		Tags:               []string{"create", "project"},
	})

	return NewToolDefinition(
		"create_project",
		"Create a new automation project. Projects organize related workflows together.",
		"project_management",
		params,
		meta,
	)
}

// listProjectsTool defines the list_projects tool.
func (p *ProjectToolProvider) listProjectsTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"limit":  IntParamWithDefault("Maximum number of projects to return", 50),
			"offset": IntParamWithDefault("Number of projects to skip", 0),
		},
		[]string{}, // No required params
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 30,
		CostEstimate:       "low",
		Idempotent:         true,
		Tags:               []string{"listing", "project"},
	})

	return NewToolDefinition(
		"list_projects",
		"List all automation projects. Returns project names, IDs, and workflow counts.",
		"project_management",
		params,
		meta,
	)
}
