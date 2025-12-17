package policy

import (
	"context"

	"workspace-sandbox/internal/types"
)

// NoOpValidationPolicy is a validation policy that performs no validation.
// This is the default policy for fast path when no hooks are configured.
type NoOpValidationPolicy struct{}

// NewNoOpValidationPolicy creates a validation policy that does nothing.
func NewNoOpValidationPolicy() *NoOpValidationPolicy {
	return &NoOpValidationPolicy{}
}

// ValidateBeforeApply always succeeds.
func (p *NoOpValidationPolicy) ValidateBeforeApply(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange) error {
	return nil
}

// GetValidationHooks returns an empty list.
func (p *NoOpValidationPolicy) GetValidationHooks() []ValidationHook {
	return nil
}

// HookValidationPolicy runs configured validation hooks.
type HookValidationPolicy struct {
	hooks []ValidationHook
}

// NewHookValidationPolicy creates a policy with the given hooks.
func NewHookValidationPolicy(hooks []ValidationHook) *HookValidationPolicy {
	return &HookValidationPolicy{hooks: hooks}
}

// ValidateBeforeApply runs all hooks and returns the first required failure.
func (p *HookValidationPolicy) ValidateBeforeApply(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange) error {
	// Note: Actual hook execution would happen here.
	// For now, this is a placeholder for the extension point.
	// Implementation would:
	// 1. Set up environment with sandbox paths
	// 2. Run each hook command
	// 3. Collect results
	// 4. Return error if any required hook fails

	return nil
}

// GetValidationHooks returns the configured hooks.
func (p *HookValidationPolicy) GetValidationHooks() []ValidationHook {
	return p.hooks
}

// Verify interfaces are implemented.
var (
	_ ValidationPolicy = (*NoOpValidationPolicy)(nil)
	_ ValidationPolicy = (*HookValidationPolicy)(nil)
)
