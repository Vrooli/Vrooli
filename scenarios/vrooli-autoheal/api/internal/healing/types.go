// Package healing provides recovery action execution and healing strategies.
// This package separates healing concerns from detection concerns (checks package).
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package healing

import (
	"context"
	"time"

	"vrooli-autoheal/internal/checks"
)

// ActionExecutor executes a single recovery action.
// This interface enables dependency injection for testing action logic
// without actually executing system commands.
// [REQ:TEST-SEAM-001]
type ActionExecutor interface {
	// Execute runs the action and returns the result.
	// The context controls cancellation/timeout.
	Execute(ctx context.Context) checks.ActionResult
}

// ActionExecutorFunc is a function adapter for ActionExecutor.
type ActionExecutorFunc func(ctx context.Context) checks.ActionResult

// Execute implements ActionExecutor.
func (f ActionExecutorFunc) Execute(ctx context.Context) checks.ActionResult {
	return f(ctx)
}

// ActionFactory creates ActionExecutor instances.
// It receives the action ID, check ID, and last result to create
// an appropriately configured executor.
// [REQ:TEST-SEAM-001]
type ActionFactory interface {
	// CreateExecutor creates an executor for the given action.
	CreateExecutor(actionID, checkID string, lastResult *checks.Result) ActionExecutor
}

// ActionFactoryFunc is a function adapter for ActionFactory.
type ActionFactoryFunc func(actionID, checkID string, lastResult *checks.Result) ActionExecutor

// CreateExecutor implements ActionFactory.
func (f ActionFactoryFunc) CreateExecutor(actionID, checkID string, lastResult *checks.Result) ActionExecutor {
	return f(actionID, checkID, lastResult)
}

// Verifier checks whether an action was successful by re-running the check.
// This interface enables testing verification logic independently.
// [REQ:TEST-SEAM-001]
type Verifier interface {
	// Verify checks that the system is healthy after an action.
	// Returns true if recovery was successful.
	Verify(ctx context.Context) bool
}

// VerifierFunc is a function adapter for Verifier.
type VerifierFunc func(ctx context.Context) bool

// Verify implements Verifier.
func (f VerifierFunc) Verify(ctx context.Context) bool {
	return f(ctx)
}

// ActionConfig defines configuration for a recovery action.
// This enables declarative action definitions that can be tested independently.
type ActionConfig struct {
	ID          string
	Name        string
	Description string
	Dangerous   bool

	// AvailabilityFn determines if the action is available based on state.
	// If nil, the action is always available.
	AvailabilityFn func(lastResult *checks.Result) bool

	// ExecutorFactory creates the executor for this action.
	// This enables dependency injection for testing.
	ExecutorFactory ActionFactory

	// Verifier optionally verifies recovery after execution.
	// If nil, no verification is performed.
	Verifier Verifier

	// VerifyDelay is the time to wait before verification.
	VerifyDelay time.Duration
}

// ToRecoveryAction converts an ActionConfig to a checks.RecoveryAction.
// This bridges the new healing architecture with the existing check interfaces.
func (c *ActionConfig) ToRecoveryAction(lastResult *checks.Result) checks.RecoveryAction {
	available := true
	if c.AvailabilityFn != nil {
		available = c.AvailabilityFn(lastResult)
	}

	return checks.RecoveryAction{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Dangerous:   c.Dangerous,
		Available:   available,
	}
}

// Execute runs the action and returns the result.
// It handles verification if configured.
func (c *ActionConfig) Execute(ctx context.Context, checkID string, lastResult *checks.Result) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  c.ID,
		CheckID:   checkID,
		Timestamp: start,
	}

	if c.ExecutorFactory == nil {
		result.Success = false
		result.Error = "no executor configured for action: " + c.ID
		result.Duration = time.Since(start)
		return result
	}

	// Create and execute the action
	executor := c.ExecutorFactory.CreateExecutor(c.ID, checkID, lastResult)
	result = executor.Execute(ctx)

	// If verification is configured and execution succeeded, verify
	if result.Success && c.Verifier != nil && c.VerifyDelay > 0 {
		time.Sleep(c.VerifyDelay)

		if c.Verifier.Verify(ctx) {
			result.Success = true
			result.Output += "\n\n=== Verification ===\nRecovery verified successfully"
		} else {
			result.Success = false
			result.Error = "verification failed after " + c.ID
			result.Output += "\n\n=== Verification Failed ===\nSystem not healthy after action"
		}
	}

	result.Duration = time.Since(start)
	return result
}

// Healer manages recovery actions for a specific check.
// This interface enables checks to delegate healing logic to a separate component.
// [REQ:HEAL-ACTION-001]
type Healer interface {
	// CheckID returns the ID of the check this healer supports.
	CheckID() string

	// Actions returns the available recovery actions for this check.
	Actions(lastResult *checks.Result) []checks.RecoveryAction

	// Execute runs the specified recovery action.
	Execute(ctx context.Context, actionID string, lastResult *checks.Result) checks.ActionResult
}

// BaseHealer provides a common implementation of Healer using ActionConfigs.
// Checks can embed this to get standardized action handling.
type BaseHealer struct {
	checkID       string
	actionConfigs []ActionConfig
}

// NewBaseHealer creates a BaseHealer with the given action configurations.
func NewBaseHealer(checkID string, configs []ActionConfig) *BaseHealer {
	return &BaseHealer{
		checkID:       checkID,
		actionConfigs: configs,
	}
}

// CheckID returns the check ID.
func (h *BaseHealer) CheckID() string {
	return h.checkID
}

// Actions returns all configured recovery actions.
func (h *BaseHealer) Actions(lastResult *checks.Result) []checks.RecoveryAction {
	actions := make([]checks.RecoveryAction, 0, len(h.actionConfigs))
	for _, config := range h.actionConfigs {
		actions = append(actions, config.ToRecoveryAction(lastResult))
	}
	return actions
}

// Execute runs the specified action.
func (h *BaseHealer) Execute(ctx context.Context, actionID string, lastResult *checks.Result) checks.ActionResult {
	for _, config := range h.actionConfigs {
		if config.ID == actionID {
			return config.Execute(ctx, h.checkID, lastResult)
		}
	}

	return checks.ActionResult{
		ActionID:  actionID,
		CheckID:   h.checkID,
		Success:   false,
		Error:     "unknown action: " + actionID,
		Timestamp: time.Now(),
	}
}

// Ensure BaseHealer implements Healer.
var _ Healer = (*BaseHealer)(nil)
