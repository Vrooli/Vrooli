// Package domain defines the core domain entities for agent-manager.
//
// This file contains EXPLICIT DECISION HELPERS that make key business
// decisions visible, testable, and easy to locate. Each function name
// clearly expresses what decision is being made.

package domain

import (
	"time"

	"github.com/google/uuid"
)

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

// RunModeDecision captures the decision about which run mode to use,
// along with a human-readable reason for audit/event logging.
type RunModeDecision struct {
	Mode           RunMode
	Reason         string
	ExplicitChoice bool
	PolicyDenied   bool
}

// DeriveRunMode returns the RunMode for a given resolved SandboxConfig.
//
// SandboxConfig.Mode is the single source of truth for "is this run
// sandboxed?" — every Mode except [SandboxModeOff] yields
// [RunModeSandboxed]. Spawn surfaces resolve the SandboxConfig (via
// orchestration.resolveSandboxConfig) before calling this function, so
// nil here implies "no sandbox config" and is treated identically to
// [SandboxModeOff].
//
// Mapping:
//   - SandboxModeOff   → RunModeInPlace  (explicit no-sandbox)
//   - any other Mode   → RunModeSandboxed (incl Tracking, Protected)
//   - nil cfg          → RunModeInPlace  (treated as Off; in practice
//     the orchestrator always populates a non-nil cfg)
//
// This function does NOT consult any other input. Callers that need to
// override the derived mode (e.g. an explicit req.RunMode on CreateRun)
// should compose:
//
//	mode := DeriveRunMode(cfg.SandboxConfig)
//	if req.RunMode != nil {
//	    mode = *req.RunMode
//	}
//
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md (RunMode decision boundary).
// DOC: scenarios/agent-manager/docs/internal/INVARIANTS.md (run mode invariant).
func DeriveRunMode(cfg *SandboxConfig) RunMode {
	if cfg == nil {
		return RunModeInPlace
	}
	if cfg.Mode.Effective() == SandboxModeOff {
		return RunModeInPlace
	}
	return RunModeSandboxed
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

// ContractRunOutcome is the 4-value outcome enum from the auditability
// contract that gets recorded on per-run provenance (ProvenanceRunGroup.runOutcome).
// See scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md Finding 2.
type ContractRunOutcome string

const (
	ContractRunOutcomeSuccess   ContractRunOutcome = "success"
	ContractRunOutcomeFailure   ContractRunOutcome = "failure"
	ContractRunOutcomeCancelled ContractRunOutcome = "cancelled"
	ContractRunOutcomeTimeout   ContractRunOutcome = "timeout"
)

// ToContract maps the agent-manager 7-value RunOutcome to the 4-value
// auditability-contract enum. The mapping is intentionally lossy: failure
// modes (exit_error, exception, sandbox_fail, runner_fail) all collapse to
// "failure" for GCT rendering purposes. The original RunOutcome remains on
// the Run record for triage; only the contract value is sent on the apply
// call (see Decision D5 in
// scenarios/swarm-manager/execute/agent-manager-sandbox-auto-apply-defaults/plan.md).
func (o RunOutcome) ToContract() ContractRunOutcome {
	switch o {
	case RunOutcomeSuccess:
		return ContractRunOutcomeSuccess
	case RunOutcomeCancelled:
		return ContractRunOutcomeCancelled
	case RunOutcomeTimeout:
		return ContractRunOutcomeTimeout
	case RunOutcomeExitError, RunOutcomeException, RunOutcomeSandboxFail, RunOutcomeRunnerFail:
		return ContractRunOutcomeFailure
	default:
		// Unknown outcome → conservatively classify as failure rather than
		// silently dropping the provenance write.
		return ContractRunOutcomeFailure
	}
}

// =============================================================================
// CONVERSATION ID RESOLUTION
// =============================================================================

// ParentLookup is the seam for resolving a parent run's ConversationID without
// pulling the orchestrator's run repository into the domain package. Callers
// pass a closure that fetches the parent run by ID; the resolver only reads
// ConversationID from the result.
type ParentLookup func(parentID uuid.UUID) (string, bool)

// ResolveConversationID picks the ConversationID for a newly created run using
// the precedence locked by Decision D7:
//
//  1. spawner-supplied value wins (run.ConversationID is non-empty)
//  2. else inherit from ParentRunID's run via parentLookup
//  3. else generate a fresh UUID
//
// parentLookup may be nil; it is only consulted when (1) is empty and the run
// has a ParentRunID. This keeps the domain layer free of repository imports.
func ResolveConversationID(run *Run, parentLookup ParentLookup) string {
	if run == nil {
		return uuid.NewString()
	}
	// (1) Spawner-supplied wins.
	if run.ConversationID != "" {
		return run.ConversationID
	}
	// (2) Inherit from parent.
	if run.ParentRunID != nil && parentLookup != nil {
		if parentConv, ok := parentLookup(*run.ParentRunID); ok && parentConv != "" {
			return parentConv
		}
	}
	// (3) Fresh UUID.
	return uuid.NewString()
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
	// Normalize paths first (ensure consistent slash handling)
	normA := normalizeScopePath(scopeA)
	normB := normalizeScopePath(scopeB)

	// Identical scopes overlap
	if normA == normB {
		return true
	}

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

// =============================================================================
// RESUMPTION DECISIONS
// =============================================================================
// These functions determine when and how runs can be resumed after interruption.

// ResumptionDecision captures the decision about whether a run can be resumed.
type ResumptionDecision struct {
	CanResume     bool
	Reason        string
	ResumePhase   RunPhase
	SkippedPhases []RunPhase
}

// DecideResumption determines whether a run can be resumed and from which phase.
// This is a pure function that makes resumption logic explicit and testable.
//
// Decision criteria:
// 1. Run must be in non-terminal status (not complete, failed, cancelled)
// 2. Run's phase must support resumption
// 3. Checkpoint must be available if run was past initialization
func DecideResumption(
	run *Run,
	checkpoint *RunCheckpoint,
	staleDuration time.Duration,
) ResumptionDecision {
	// Check terminal status
	if run.Status == RunStatusComplete {
		return ResumptionDecision{
			CanResume: false,
			Reason:    "run is already complete",
		}
	}
	if run.Status == RunStatusFailed {
		return ResumptionDecision{
			CanResume: false,
			Reason:    "run has failed - create a new run instead",
		}
	}
	if run.Status == RunStatusCancelled {
		return ResumptionDecision{
			CanResume: false,
			Reason:    "run was cancelled - create a new run instead",
		}
	}

	// Check if phase supports resumption
	phase := run.Phase
	if checkpoint != nil {
		phase = checkpoint.Phase
	}

	if !phase.CanResumeFromPhase() {
		return ResumptionDecision{
			CanResume: false,
			Reason:    "phase " + string(phase) + " does not support resumption",
		}
	}

	// Calculate which phases can be skipped
	var skipped []RunPhase
	switch phase {
	case RunPhaseExecuting:
		skipped = []RunPhase{RunPhaseQueued, RunPhaseInitializing, RunPhaseSandboxCreating, RunPhaseRunnerAcquiring}
	case RunPhaseRunnerAcquiring:
		skipped = []RunPhase{RunPhaseQueued, RunPhaseInitializing, RunPhaseSandboxCreating}
	case RunPhaseSandboxCreating:
		skipped = []RunPhase{RunPhaseQueued, RunPhaseInitializing}
	case RunPhaseInitializing:
		skipped = []RunPhase{RunPhaseQueued}
	}

	return ResumptionDecision{
		CanResume:     true,
		Reason:        "run can be resumed from " + string(phase),
		ResumePhase:   phase,
		SkippedPhases: skipped,
	}
}

// =============================================================================
// STALE RUN DECISIONS
// =============================================================================

// StaleRunDecision captures the decision about what to do with a stale run.
type StaleRunDecision struct {
	IsStale        bool
	TimeSinceHeart time.Duration
	Action         StaleRunAction
	Reason         string
}

// StaleRunAction indicates what action should be taken for a stale run.
type StaleRunAction string

const (
	StaleRunActionNone   StaleRunAction = "none"   // Not stale, no action needed
	StaleRunActionResume StaleRunAction = "resume" // Try to resume the run
	StaleRunActionFail   StaleRunAction = "fail"   // Mark as failed
	StaleRunActionAlert  StaleRunAction = "alert"  // Alert operator but don't change state
)

// DecideStaleRunAction determines what action to take for a potentially stale run.
//
// Decision priority:
// 1. If not stale, no action
// 2. If resumable, try to resume
// 3. If retries exhausted, mark as failed
// 4. Otherwise, alert operator
func DecideStaleRunAction(
	run *Run,
	checkpoint *RunCheckpoint,
	staleDuration time.Duration,
	maxRetries int,
) StaleRunDecision {
	// Check if actually stale
	if !run.IsStale(staleDuration) {
		return StaleRunDecision{
			IsStale: false,
			Action:  StaleRunActionNone,
			Reason:  "run is not stale",
		}
	}

	timeSinceHeart := time.Duration(0)
	if run.LastHeartbeat != nil {
		timeSinceHeart = time.Since(*run.LastHeartbeat)
	}

	// Check retry count
	retryCount := 0
	if checkpoint != nil {
		retryCount = checkpoint.RetryCount
	}

	if retryCount >= maxRetries {
		return StaleRunDecision{
			IsStale:        true,
			TimeSinceHeart: timeSinceHeart,
			Action:         StaleRunActionFail,
			Reason:         "max retries exceeded",
		}
	}

	// Check if resumable
	if run.IsResumable() {
		return StaleRunDecision{
			IsStale:        true,
			TimeSinceHeart: timeSinceHeart,
			Action:         StaleRunActionResume,
			Reason:         "run is stale but can be resumed",
		}
	}

	// Otherwise alert
	return StaleRunDecision{
		IsStale:        true,
		TimeSinceHeart: timeSinceHeart,
		Action:         StaleRunActionAlert,
		Reason:         "run is stale and cannot be resumed automatically",
	}
}
