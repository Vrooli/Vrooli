// Package toolregistry provides the tool discovery service for workspace-sandbox.
//
// This file defines the diff and approval tools (Tier 4) that are exposed
// via the Tool Discovery Protocol.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// DiffToolProvider provides the diff and approval tools.
// These tools enable viewing diffs and approving/rejecting changes.
type DiffToolProvider struct{}

// NewDiffToolProvider creates a new DiffToolProvider.
func NewDiffToolProvider() *DiffToolProvider {
	return &DiffToolProvider{}
}

// Name returns the provider identifier.
func (p *DiffToolProvider) Name() string {
	return "workspace-sandbox-diff"
}

// Categories returns the tool categories for diff operations.
func (p *DiffToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "diff_approval",
			Name:        "Diff & Approval",
			Description: "Tools for viewing diffs and approving or rejecting changes",
			Icon:        "git-compare",
		},
	}
}

// Tools returns the tool definitions for diff operations.
func (p *DiffToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.getDiffTool(),
		p.approveChangesTool(),
		p.rejectChangesTool(),
		p.discardFilesTool(),
	}
}

func (p *DiffToolProvider) getDiffTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_diff",
		Description: "Generate a unified diff showing all changes made in a sandbox compared to the original files. Shows added, modified, and deleted files.",
		Category:    "diff_approval",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox to generate the diff for.",
				},
				"context_lines": {
					Type:        "integer",
					Default:     IntValue(3),
					Description: "Number of context lines to include around each change.",
				},
				"path_filter": {
					Type:        "string",
					Description: "Only include changes matching this path pattern (glob syntax).",
				},
			},
			Required: []string{"sandbox_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 30,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"diff", "changes", "review"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get diff with default context",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				),
				NewToolExample(
					"Get diff with more context lines",
					map[string]interface{}{
						"sandbox_id":    "550e8400-e29b-41d4-a716-446655440000",
						"context_lines": 10,
					},
				),
			},
		},
	}
}

func (p *DiffToolProvider) approveChangesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "approve_changes",
		Description: "Approve and apply changes from a sandbox to the canonical repository. This copies the modified files from the sandbox overlay to the original project, making the changes permanent.",
		Category:    "diff_approval",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox containing the changes to approve.",
				},
				"files": {
					Type:        "array",
					Description: "Optional list of specific file paths to approve. If not provided, all changes are approved.",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"commit_message": {
					Type:        "string",
					Description: "Optional commit message for the changes. Used if auto-commit is enabled.",
				},
				"actor": {
					Type:        "string",
					Description: "Identifier of who is approving the changes (for audit purposes).",
				},
			},
			Required: []string{"sandbox_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // Modifies canonical repo - requires human approval
			TimeoutSeconds:     120,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"approval", "merge", "commit", "permanent"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Approve all changes in a sandbox",
					map[string]interface{}{
						"sandbox_id":     "550e8400-e29b-41d4-a716-446655440000",
						"commit_message": "feat: implement user authentication",
					},
				),
				NewToolExample(
					"Approve only specific files",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"files":      []string{"src/auth/login.ts", "src/auth/session.ts"},
					},
				),
			},
		},
	}
}

func (p *DiffToolProvider) rejectChangesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "reject_changes",
		Description: "Reject and discard all changes in a sandbox. This marks the sandbox as rejected and cleans up resources. No changes are applied to the original files.",
		Category:    "diff_approval",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox to reject.",
				},
				"reason": {
					Type:        "string",
					Description: "Optional reason for rejection (for audit purposes).",
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
			Tags:               []string{"rejection", "discard", "cleanup"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Reject changes with a reason",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"reason":     "Changes don't meet code review standards",
					},
				),
			},
		},
	}
}

func (p *DiffToolProvider) discardFilesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "discard_files",
		Description: "Discard changes to specific files in a sandbox, reverting them to the original state. Useful for undoing unwanted changes while keeping other modifications.",
		Category:    "diff_approval",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox containing the files.",
				},
				"files": {
					Type:        "array",
					Description: "List of file paths to discard changes for.",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
			},
			Required: []string{"sandbox_id", "files"},
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
			Tags:               []string{"discard", "revert", "undo"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Discard changes to specific files",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"files":      []string{"src/config.ts", "package.json"},
					},
				),
			},
		},
	}
}
