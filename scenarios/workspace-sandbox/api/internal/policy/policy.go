// Package policy defines extension points for workspace-sandbox behavior.
//
// # Purpose
//
// The policy package provides interfaces for the "volatile edges" of the system -
// areas expected to evolve with different modes and configurations. By extracting
// these decisions into interfaces, we enable:
//
//   - Easy testing with mock policies
//   - Configuration-driven behavior changes
//   - Future extension without modifying core service logic
//
// # Interfaces
//
//   - ApprovalPolicy: Decides whether changes can be auto-approved
//   - AttributionPolicy: Controls commit authorship and message format
//   - ValidationPolicy: Runs pre-commit validation hooks
//
// # Usage
//
// Policies are injected into the Service at construction time. The service
// consults policies at decision points rather than containing the decision logic.
package policy

import (
	"context"

	"workspace-sandbox/internal/types"
)

// ApprovalPolicy decides whether sandbox changes can be approved and how.
type ApprovalPolicy interface {
	// CanAutoApprove decides if the given changes can be automatically approved
	// without human intervention. Returns true if auto-approval is allowed,
	// false if human review is required.
	CanAutoApprove(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange) (bool, string)

	// ValidateApproval performs any additional validation before approval.
	// Returns an error if the approval should be blocked.
	ValidateApproval(ctx context.Context, sandbox *types.Sandbox, req *types.ApprovalRequest) error
}

// AttributionPolicy controls how commits are attributed.
type AttributionPolicy interface {
	// GetCommitAuthor returns the author string for the commit.
	// Format: "Name <email>"
	GetCommitAuthor(ctx context.Context, sandbox *types.Sandbox, actor string) string

	// GetCommitMessage returns the formatted commit message.
	GetCommitMessage(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange, userMessage string) string

	// GetCoAuthors returns any co-author lines to append to the commit message.
	GetCoAuthors(ctx context.Context, sandbox *types.Sandbox, actor string) []string
}

// ValidationPolicy runs pre-commit validation hooks.
type ValidationPolicy interface {
	// ValidateBeforeApply runs validation checks before applying changes.
	// Returns an error if validation fails and changes should not be applied.
	ValidateBeforeApply(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange) error

	// GetValidationHooks returns the list of validation hooks to run.
	GetValidationHooks() []ValidationHook
}

// ValidationHook represents a single validation check.
type ValidationHook struct {
	Name        string
	Description string
	Command     string   // Shell command to run
	Args        []string // Arguments for the command
	Required    bool     // If true, failure blocks the commit
}

// ValidationResult captures the outcome of a validation run.
type ValidationResult struct {
	HookName string
	Success  bool
	Output   string
	Error    error
}
