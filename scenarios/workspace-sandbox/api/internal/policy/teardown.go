// Package policy - teardown.go implements pre-teardown hooks for sandbox lifecycle.
//
// # Problem
//
// When a sandbox is stopped or deleted, its overlayfs mount disappears. But
// external systems may have processes actively running from the sandbox's merged/
// directory. For example, the Vrooli scenario lifecycle system can start a
// scenario from a sandbox path — when that sandbox is torn down, the scenario
// process becomes orphaned (still alive but unable to read files, spawn children,
// or reload).
//
// # Solution
//
// This file provides a TeardownPolicy implementation that runs configurable shell
// commands before sandbox unmount/delete operations. The hooks are always
// best-effort: failures are logged but never block teardown. This ensures sandboxes
// are always cleanable regardless of hook outcomes.
//
// # Integration with Vrooli
//
// In the Vrooli ecosystem, the teardown hook is configured to call:
//
//	vrooli scenario heal-from-sandbox
//
// This CLI command reads SANDBOX_MERGED_DIR from the environment, scans process
// metadata to find scenarios running from that path, stops them, and restarts
// them from the canonical repo location via the native `vrooli scenario
// heal-from-sandbox` implementation.
//
// # Hook Environment
//
// Each hook receives these environment variables:
//   - SANDBOX_ID: sandbox UUID
//   - SANDBOX_SCOPE_PATH: the scope path being sandboxed
//   - SANDBOX_PROJECT_ROOT: the project root directory
//   - SANDBOX_UPPER_DIR: the overlay upper directory
//   - SANDBOX_MERGED_DIR: the merged view directory (where processes run from)
//   - SANDBOX_TEARDOWN_REASON: why teardown is happening ("stop" or "delete")
package policy

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"workspace-sandbox/internal/types"
)

// NoOpTeardownPolicy is a teardown policy that does nothing.
// This is the default when no teardown hooks are configured.
type NoOpTeardownPolicy struct{}

// NewNoOpTeardownPolicy creates a teardown policy that does nothing.
func NewNoOpTeardownPolicy() *NoOpTeardownPolicy {
	return &NoOpTeardownPolicy{}
}

// RunPreTeardownHooks returns empty results (no hooks to run).
func (p *NoOpTeardownPolicy) RunPreTeardownHooks(ctx context.Context, sandbox *types.Sandbox, reason string) []TeardownHookResult {
	return nil
}

// HookTeardownPolicy runs configured shell commands before sandbox teardown.
type HookTeardownPolicy struct {
	hooks         []TeardownHook
	globalTimeout time.Duration
}

// TeardownPolicyOption configures the HookTeardownPolicy.
type TeardownPolicyOption func(*HookTeardownPolicy)

// WithTeardownGlobalTimeout sets the global timeout for all teardown hooks.
func WithTeardownGlobalTimeout(timeout time.Duration) TeardownPolicyOption {
	return func(p *HookTeardownPolicy) {
		p.globalTimeout = timeout
	}
}

// NewHookTeardownPolicy creates a policy with the given hooks.
func NewHookTeardownPolicy(hooks []TeardownHook, opts ...TeardownPolicyOption) *HookTeardownPolicy {
	p := &HookTeardownPolicy{
		hooks:         hooks,
		globalTimeout: 30 * time.Second, // Default: shorter than validation (teardown should be fast)
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// RunPreTeardownHooks executes all configured hooks before sandbox teardown.
//
// All hooks run regardless of individual failures (best-effort). Each hook's
// outcome is captured in the returned results for logging by the caller.
func (p *HookTeardownPolicy) RunPreTeardownHooks(ctx context.Context, sandbox *types.Sandbox, reason string) []TeardownHookResult {
	if len(p.hooks) == 0 {
		return nil
	}

	// Apply global timeout if not already set in context
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && p.globalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.globalTimeout)
		defer cancel()
	}

	env := buildTeardownHookEnv(sandbox, reason)

	results := make([]TeardownHookResult, 0, len(p.hooks))
	for _, hook := range p.hooks {
		result := p.executeHook(ctx, hook, sandbox, env)
		results = append(results, result)
		// Always continue to next hook — teardown hooks are best-effort
	}

	return results
}

// executeHook runs a single teardown hook and returns the result.
func (p *HookTeardownPolicy) executeHook(ctx context.Context, hook TeardownHook, sandbox *types.Sandbox, env []string) TeardownHookResult {
	result := TeardownHookResult{
		HookName: hook.Name,
	}

	// Apply per-hook timeout if configured
	if hook.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, hook.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, hook.Command, hook.Args...)
	cmd.Env = env

	// Set working directory to sandbox merged directory if available
	if sandbox.MergedDir != "" {
		cmd.Dir = sandbox.MergedDir
	} else if sandbox.ProjectRoot != "" {
		cmd.Dir = sandbox.ProjectRoot
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result.Output = stdout.String()
	if stderr.Len() > 0 {
		if result.Output != "" {
			result.Output += "\n"
		}
		result.Output += stderr.String()
	}

	if err != nil {
		result.Success = false
		result.Error = err
	} else {
		result.Success = true
	}

	return result
}

// buildTeardownHookEnv creates the environment variables for teardown hook execution.
//
// This provides the same sandbox metadata as validation hooks (SANDBOX_ID,
// SANDBOX_SCOPE_PATH, etc.) plus SANDBOX_TEARDOWN_REASON so the hook knows
// why teardown is happening.
func buildTeardownHookEnv(sandbox *types.Sandbox, reason string) []string {
	return []string{
		fmt.Sprintf("SANDBOX_ID=%s", sandbox.ID.String()),
		fmt.Sprintf("SANDBOX_SCOPE_PATH=%s", sandbox.ScopePath),
		fmt.Sprintf("SANDBOX_PROJECT_ROOT=%s", sandbox.ProjectRoot),
		fmt.Sprintf("SANDBOX_UPPER_DIR=%s", sandbox.UpperDir),
		fmt.Sprintf("SANDBOX_MERGED_DIR=%s", sandbox.MergedDir),
		fmt.Sprintf("SANDBOX_TEARDOWN_REASON=%s", reason),
	}
}

// Verify interfaces are implemented.
var (
	_ TeardownPolicy = (*NoOpTeardownPolicy)(nil)
	_ TeardownPolicy = (*HookTeardownPolicy)(nil)
)
