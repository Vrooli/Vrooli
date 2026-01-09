// Package policy - validation.go implements pre-commit validation hooks.
// [OT-P1-005] Pre-commit Validation Hooks
package policy

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

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

// HookValidationPolicy runs configured validation hooks before applying changes.
// [OT-P1-005] Pre-commit Validation Hooks
type HookValidationPolicy struct {
	hooks          []ValidationHook
	globalTimeout  time.Duration
	logger         ValidationLogger
	continueOnFail bool // If true, continue running hooks even after a failure
}

// ValidationLogger is an optional interface for logging hook execution.
type ValidationLogger interface {
	LogHookStart(hookName string, sandbox *types.Sandbox)
	LogHookResult(result *ValidationResult)
}

// HookPolicyOption configures the HookValidationPolicy.
type HookPolicyOption func(*HookValidationPolicy)

// WithGlobalTimeout sets the global timeout for all hooks.
func WithGlobalTimeout(timeout time.Duration) HookPolicyOption {
	return func(p *HookValidationPolicy) {
		p.globalTimeout = timeout
	}
}

// WithValidationLogger sets a logger for hook execution.
func WithValidationLogger(logger ValidationLogger) HookPolicyOption {
	return func(p *HookValidationPolicy) {
		p.logger = logger
	}
}

// WithContinueOnFail configures whether to continue running hooks after a failure.
func WithContinueOnFail(continueOnFail bool) HookPolicyOption {
	return func(p *HookValidationPolicy) {
		p.continueOnFail = continueOnFail
	}
}

// NewHookValidationPolicy creates a policy with the given hooks.
func NewHookValidationPolicy(hooks []ValidationHook, opts ...HookPolicyOption) *HookValidationPolicy {
	p := &HookValidationPolicy{
		hooks:         hooks,
		globalTimeout: 5 * time.Minute, // Default timeout
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ValidateBeforeApply runs all hooks and returns an error if any required hook fails.
// [OT-P1-005] Pre-commit Validation Hooks
//
// # Hook Execution Environment
//
// Each hook is executed with these environment variables:
//   - SANDBOX_ID: The sandbox UUID
//   - SANDBOX_SCOPE_PATH: The scope path being sandboxed
//   - SANDBOX_PROJECT_ROOT: The project root directory
//   - SANDBOX_UPPER_DIR: The overlay upper directory (where changes are)
//   - SANDBOX_MERGED_DIR: The merged view directory
//   - SANDBOX_CHANGED_FILES: Comma-separated list of changed file paths
//   - SANDBOX_CHANGE_COUNT: Number of changed files
//
// # Return Behavior
//
// - Returns nil if all hooks pass
// - Returns ValidationHookError if any required hook fails
// - Non-required hook failures are logged but don't block approval
func (p *HookValidationPolicy) ValidateBeforeApply(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange) error {
	if len(p.hooks) == 0 {
		return nil
	}

	// Apply global timeout if not already set in context
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && p.globalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.globalTimeout)
		defer cancel()
	}

	// Build environment variables
	env := buildHookEnv(sandbox, changes)

	var failures []ValidationResult
	for _, hook := range p.hooks {
		if p.logger != nil {
			p.logger.LogHookStart(hook.Name, sandbox)
		}

		result := p.executeHook(ctx, hook, sandbox, env)

		if p.logger != nil {
			p.logger.LogHookResult(&result)
		}

		if !result.Success {
			failures = append(failures, result)
			// Stop on first required failure unless configured to continue
			if hook.Required && !p.continueOnFail {
				return &ValidationHookError{
					HookName: hook.Name,
					Output:   result.Output,
					Err:      result.Error,
				}
			}
		}
	}

	// Check if any required hooks failed
	for _, f := range failures {
		for _, hook := range p.hooks {
			if hook.Name == f.HookName && hook.Required {
				return &ValidationHookError{
					HookName: f.HookName,
					Output:   f.Output,
					Err:      f.Error,
				}
			}
		}
	}

	return nil
}

// executeHook runs a single validation hook and returns the result.
func (p *HookValidationPolicy) executeHook(ctx context.Context, hook ValidationHook, sandbox *types.Sandbox, env []string) ValidationResult {
	result := ValidationResult{
		HookName: hook.Name,
	}

	// Create command
	cmd := exec.CommandContext(ctx, hook.Command, hook.Args...)
	cmd.Env = env

	// Set working directory to the sandbox merged directory if available
	if sandbox.MergedDir != "" {
		cmd.Dir = sandbox.MergedDir
	} else if sandbox.ProjectRoot != "" {
		cmd.Dir = sandbox.ProjectRoot
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Combine output
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

// buildHookEnv creates the environment variables for hook execution.
func buildHookEnv(sandbox *types.Sandbox, changes []*types.FileChange) []string {
	// Start with current process environment
	env := []string{}

	// Add sandbox-specific variables
	env = append(env,
		fmt.Sprintf("SANDBOX_ID=%s", sandbox.ID.String()),
		fmt.Sprintf("SANDBOX_SCOPE_PATH=%s", sandbox.ScopePath),
		fmt.Sprintf("SANDBOX_PROJECT_ROOT=%s", sandbox.ProjectRoot),
		fmt.Sprintf("SANDBOX_UPPER_DIR=%s", sandbox.UpperDir),
		fmt.Sprintf("SANDBOX_MERGED_DIR=%s", sandbox.MergedDir),
		fmt.Sprintf("SANDBOX_CHANGE_COUNT=%d", len(changes)),
	)

	// Build comma-separated list of changed files
	if len(changes) > 0 {
		paths := make([]string, len(changes))
		for i, c := range changes {
			paths[i] = c.FilePath
		}
		env = append(env, fmt.Sprintf("SANDBOX_CHANGED_FILES=%s", strings.Join(paths, ",")))
	}

	return env
}

// GetValidationHooks returns the configured hooks.
func (p *HookValidationPolicy) GetValidationHooks() []ValidationHook {
	return p.hooks
}

// ValidationHookError is returned when a required validation hook fails.
type ValidationHookError struct {
	HookName string
	Output   string
	Err      error
}

func (e *ValidationHookError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("validation hook '%s' failed: %v", e.HookName, e.Err)
	}
	return fmt.Sprintf("validation hook '%s' failed", e.HookName)
}

// Unwrap returns the underlying error.
func (e *ValidationHookError) Unwrap() error {
	return e.Err
}

// HTTPStatus returns the HTTP status code for this error type.
func (e *ValidationHookError) HTTPStatus() int {
	return 422 // Unprocessable Entity - validation failed
}

// IsRetryable indicates this error type should not be automatically retried.
func (e *ValidationHookError) IsRetryable() bool {
	return false
}

// Hint provides actionable guidance for resolving validation failures.
func (e *ValidationHookError) Hint() string {
	return fmt.Sprintf("Review the hook output and fix the issues before retrying. Hook '%s' must pass for approval to proceed.", e.HookName)
}

// Verify interfaces are implemented.
var (
	_ ValidationPolicy  = (*NoOpValidationPolicy)(nil)
	_ ValidationPolicy  = (*HookValidationPolicy)(nil)
	_ types.DomainError = (*ValidationHookError)(nil)
)
