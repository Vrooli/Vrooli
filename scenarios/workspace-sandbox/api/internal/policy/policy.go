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
//   - AttributionPolicy: Controls commit authorship and message format
//   - ValidationPolicy: Runs pre-commit validation hooks
//   - TeardownPolicy: Runs pre-teardown hooks before sandbox unmount/delete
//
// # Usage
//
// Policies are injected into the Service at construction time. The service
// consults policies at decision points rather than containing the decision logic.
package policy

import (
	"context"
	"time"

	"workspace-sandbox/internal/types"
)

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

// TeardownPolicy runs pre-teardown hooks before sandbox unmount/delete.
//
// When a sandbox is stopped or deleted, its overlayfs mount disappears. External
// systems (e.g., scenario lifecycle managers) may have processes running from the
// sandbox's merged directory. Without pre-teardown coordination, those processes
// become orphaned — still alive but unable to access their filesystem.
//
// TeardownPolicy provides a hook point for external systems to gracefully evacuate
// processes before the filesystem disappears. Unlike ValidationPolicy, teardown
// hooks are always best-effort: failures are logged but never block teardown.
// A sandbox must always be cleanable regardless of hook outcomes.
type TeardownPolicy interface {
	// RunPreTeardownHooks executes hooks before sandbox teardown.
	//
	// The reason parameter indicates why teardown is happening:
	//   - "stop": explicit Stop() call — overlay will be unmounted
	//   - "delete": explicit Delete() call — overlay will be unmounted and removed
	//
	// Returns results for each hook. Failures are informational only.
	RunPreTeardownHooks(ctx context.Context, sandbox *types.Sandbox, reason string) []TeardownHookResult
}

// TeardownHook represents a pre-teardown shell command.
// Unlike ValidationHook, there is no Required field — all teardown hooks
// are best-effort because teardown must never be blocked.
type TeardownHook struct {
	Name        string
	Description string
	Command     string   // Shell command to run
	Args        []string // Arguments for the command
	Timeout     time.Duration
}

// TeardownHookResult captures the outcome of a teardown hook execution.
type TeardownHookResult struct {
	HookName string
	Success  bool
	Output   string
	Error    error
}
