// Package domain defines the core domain entities for agent-manager.
//
// This file contains EXPLICIT DECISION HELPERS that make key business
// decisions visible, testable, and easy to locate. Each function name
// clearly expresses what decision is being made.

package domain

// =============================================================================
// STATE TRANSITION DECISIONS
// =============================================================================
// These functions answer: "Can entity X transition to state Y?"
// All state machine logic is centralized here for clarity and testability.

// CanTaskTransitionTo returns whether a task can move to the target status.
// Returns (allowed bool, reason string) where reason explains denial.
func (t TaskStatus) CanTransitionTo(target TaskStatus) (bool, string) {
	transitions := taskTransitions[t]
	for _, allowed := range transitions {
		if allowed == target {
			return true, ""
		}
	}
	return false, taskTransitionDenialReason(t, target)
}

// taskTransitions defines the valid state machine for tasks.
// Key: current state, Value: allowed next states
var taskTransitions = map[TaskStatus][]TaskStatus{
	TaskStatusQueued: {
		TaskStatusRunning,
		TaskStatusCancelled,
	},
	TaskStatusRunning: {
		TaskStatusNeedsReview,
		TaskStatusFailed,
		TaskStatusCancelled,
	},
	TaskStatusNeedsReview: {
		TaskStatusApproved,
		TaskStatusRejected,
	},
	// Terminal states - no transitions allowed
	TaskStatusApproved:  {},
	TaskStatusRejected:  {},
	TaskStatusFailed:    {},
	TaskStatusCancelled: {},
}

func taskTransitionDenialReason(from, to TaskStatus) string {
	switch from {
	case TaskStatusApproved, TaskStatusRejected, TaskStatusFailed, TaskStatusCancelled:
		return "task is in terminal state"
	default:
		return "transition not allowed from " + string(from) + " to " + string(to)
	}
}

// CanRunTransitionTo returns whether a run can move to the target status.
func (r RunStatus) CanTransitionTo(target RunStatus) (bool, string) {
	transitions := runTransitions[r]
	for _, allowed := range transitions {
		if allowed == target {
			return true, ""
		}
	}
	return false, runTransitionDenialReason(r, target)
}

// runTransitions defines the valid state machine for runs.
var runTransitions = map[RunStatus][]RunStatus{
	RunStatusPending: {
		RunStatusStarting,
		RunStatusCancelled,
		RunStatusFailed,
	},
	RunStatusStarting: {
		RunStatusRunning,
		RunStatusFailed,
		RunStatusCancelled,
	},
	RunStatusRunning: {
		RunStatusNeedsReview,
		RunStatusComplete,
		RunStatusFailed,
		RunStatusCancelled,
	},
	RunStatusNeedsReview: {
		RunStatusComplete, // After approval
		RunStatusFailed,   // Rejection or error
	},
	// Terminal states
	RunStatusComplete:  {},
	RunStatusFailed:    {},
	RunStatusCancelled: {},
}

func runTransitionDenialReason(from, to RunStatus) string {
	switch from {
	case RunStatusComplete, RunStatusFailed, RunStatusCancelled:
		return "run is in terminal state"
	default:
		return "transition not allowed from " + string(from) + " to " + string(to)
	}
}

// =============================================================================
// CANCELLATION DECISIONS
// =============================================================================

// IsCancellable returns whether a task can be cancelled in its current state.
func (t *Task) IsCancellable() bool {
	return t.Status == TaskStatusQueued || t.Status == TaskStatusRunning
}

// IsStoppable returns whether a run can be stopped in its current state.
func (r *Run) IsStoppable() bool {
	return r.Status == RunStatusStarting || r.Status == RunStatusRunning
}

// =============================================================================
// APPROVAL DECISIONS
// =============================================================================

// IsApprovable returns whether a run can be approved in its current state.
// Returns (allowed bool, reason string).
func (r *Run) IsApprovable() (bool, string) {
	if r.Status != RunStatusNeedsReview {
		return false, "run must be in needs_review status to approve"
	}
	if r.SandboxID == nil {
		return false, "run has no sandbox to apply changes from"
	}
	if r.ApprovalState == ApprovalStateApproved {
		return false, "run is already approved"
	}
	if r.ApprovalState == ApprovalStateRejected {
		return false, "run has been rejected and cannot be approved"
	}
	return true, ""
}

// IsRejectable returns whether a run can be rejected in its current state.
func (r *Run) IsRejectable() (bool, string) {
	if r.Status != RunStatusNeedsReview {
		return false, "run must be in needs_review status to reject"
	}
	if r.ApprovalState == ApprovalStateRejected {
		return false, "run is already rejected"
	}
	return true, ""
}

// =============================================================================
// RUN MODE DECISIONS
// =============================================================================

// RunModeDecision captures the decision about which run mode to use.
type RunModeDecision struct {
	Mode           RunMode
	Reason         string
	PolicyOverride bool
	ExplicitChoice bool
}

// DecideRunMode determines which run mode should be used based on inputs.
// This is a pure function that makes the decision logic explicit and testable.
//
// Decision priority (highest to lowest):
// 1. Explicit mode request from caller (if provided)
// 2. Policy override (in-place allowed by policy)
// 3. Default to sandboxed
func DecideRunMode(
	requestedMode *RunMode,
	forceInPlace bool,
	policyAllowsInPlace bool,
	profileRequiresSandbox bool,
) RunModeDecision {
	// Priority 1: Explicit request
	if requestedMode != nil {
		return RunModeDecision{
			Mode:           *requestedMode,
			Reason:         "explicitly requested by caller",
			ExplicitChoice: true,
		}
	}

	// Priority 2: Force in-place with policy permission
	if forceInPlace && policyAllowsInPlace {
		return RunModeDecision{
			Mode:           RunModeInPlace,
			Reason:         "force in-place requested and allowed by policy",
			PolicyOverride: true,
		}
	}

	// Priority 3: Profile requires sandbox
	if profileRequiresSandbox {
		return RunModeDecision{
			Mode:   RunModeSandboxed,
			Reason: "agent profile requires sandbox",
		}
	}

	// Priority 4: Policy allows in-place but not forced
	if policyAllowsInPlace {
		// Even if allowed, default to sandboxed for safety
		return RunModeDecision{
			Mode:   RunModeSandboxed,
			Reason: "defaulting to sandbox (in-place allowed but not requested)",
		}
	}

	// Default: Sandboxed
	return RunModeDecision{
		Mode:   RunModeSandboxed,
		Reason: "sandbox-first default policy",
	}
}

// =============================================================================
// RESULT CLASSIFICATION DECISIONS
// =============================================================================

// RunOutcome classifies how a run completed.
type RunOutcome string

const (
	RunOutcomeSuccess     RunOutcome = "success"      // Completed, needs review
	RunOutcomeExitError   RunOutcome = "exit_error"   // Non-zero exit code
	RunOutcomeException   RunOutcome = "exception"    // Execution threw error
	RunOutcomeCancelled   RunOutcome = "cancelled"    // User cancelled
	RunOutcomeTimeout     RunOutcome = "timeout"      // Exceeded time limit
	RunOutcomeSandboxFail RunOutcome = "sandbox_fail" // Sandbox creation failed
	RunOutcomeRunnerFail  RunOutcome = "runner_fail"  // Runner not available
)

// ClassifyRunOutcome determines the outcome category from execution results.
// This makes result classification logic explicit and testable.
func ClassifyRunOutcome(
	executionErr error,
	exitCode *int,
	wasCancelled bool,
	timedOut bool,
) RunOutcome {
	if wasCancelled {
		return RunOutcomeCancelled
	}
	if timedOut {
		return RunOutcomeTimeout
	}
	if executionErr != nil {
		return RunOutcomeException
	}
	if exitCode != nil && *exitCode != 0 {
		return RunOutcomeExitError
	}
	return RunOutcomeSuccess
}

// RequiresReview returns whether this outcome should trigger review workflow.
func (o RunOutcome) RequiresReview() bool {
	return o == RunOutcomeSuccess
}

// IsTerminalFailure returns whether this outcome is a final failure state.
func (o RunOutcome) IsTerminalFailure() bool {
	switch o {
	case RunOutcomeExitError, RunOutcomeException, RunOutcomeTimeout,
		RunOutcomeSandboxFail, RunOutcomeRunnerFail:
		return true
	default:
		return false
	}
}

// =============================================================================
// SCOPE CONFLICT DECISIONS
// =============================================================================

// ScopesOverlap determines if two scope paths have a parent-child relationship.
// This is the core logic for detecting conflicting scope locks.
//
// Examples:
//   - ScopesOverlap("src/", "src/foo") → true (parent-child)
//   - ScopesOverlap("src/foo", "src/") → true (child-parent)
//   - ScopesOverlap("src/", "tests/") → false (siblings)
//   - ScopesOverlap("src/foo", "src/foo") → true (identical)
func ScopesOverlap(scopeA, scopeB string) bool {
	// Identical scopes overlap
	if scopeA == scopeB {
		return true
	}

	// Normalize paths (ensure consistent trailing slash handling)
	normA := normalizeScopePath(scopeA)
	normB := normalizeScopePath(scopeB)

	// Check if A is ancestor of B or vice versa
	return isAncestorOf(normA, normB) || isAncestorOf(normB, normA)
}

func normalizeScopePath(path string) string {
	if path == "" {
		return "/"
	}
	// Ensure consistent leading slash, no trailing slash (except root)
	if path[0] != '/' {
		path = "/" + path
	}
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return path
}

func isAncestorOf(ancestor, descendant string) bool {
	if ancestor == "/" {
		return true // Root is ancestor of everything
	}
	// Ancestor must be a prefix followed by /
	if len(descendant) <= len(ancestor) {
		return false
	}
	return descendant[:len(ancestor)] == ancestor && descendant[len(ancestor)] == '/'
}
