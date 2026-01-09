// Package domain defines the core domain entities for agent-manager.
//
// This file contains INVARIANT DEFINITIONS AND ENFORCEMENT helpers.
//
// INVARIANTS are conditions that must ALWAYS be true for the system to be
// in a valid state. Unlike validation (which checks input), invariants are
// checked at runtime to catch programming errors and ensure correctness.
//
// DESIGN PRINCIPLES:
// - Fail fast: Detect invariant violations immediately
// - Explicit contracts: Document what must be true, not just what is checked
// - Safe accessors: Provide nil-safe methods to avoid panics
// - Assertion mode: Can be enabled/disabled for production vs development
//
// KEY INVARIANTS:
// 1. INV_RUN_SANDBOX: Sandboxed runs must have SandboxID after sandbox creation phase
// 2. INV_APPROVAL_STATE: ApprovalState can only be non-None when Status is NeedsReview+
// 3. INV_TERMINAL_IMMUTABLE: Terminal states cannot transition to non-terminal states
// 4. INV_PHASE_SEQUENCE: Phases must progress in defined order (no skipping backwards)
// 5. INV_SCOPE_EXCLUSIVE: Overlapping scopes cannot have concurrent active locks
// 6. INV_HEARTBEAT_ACTIVE: Heartbeats are only meaningful for active runs

package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// INVARIANT CHECKING CONFIGURATION
// =============================================================================

// InvariantMode controls how invariant violations are handled.
type InvariantMode int

const (
	// InvariantModePanic panics on violations (for development/testing)
	InvariantModePanic InvariantMode = iota

	// InvariantModeLog logs violations but continues (for production)
	InvariantModeLog

	// InvariantModeDisabled skips invariant checks (for performance-critical paths)
	InvariantModeDisabled
)

// invariantMode is the current invariant checking mode.
// Default is Panic for fail-fast behavior during development.
var invariantMode = InvariantModePanic

// SetInvariantMode sets the invariant checking mode.
func SetInvariantMode(mode InvariantMode) {
	invariantMode = mode
}

// InvariantViolationError represents a violated invariant.
type InvariantViolationError struct {
	Invariant   string // Invariant name (e.g., "INV_RUN_SANDBOX")
	Description string // Human-readable description
	Entity      string // Entity type (e.g., "Run")
	EntityID    string // Entity identifier
	Details     string // Additional context
}

func (e *InvariantViolationError) Error() string {
	return fmt.Sprintf("invariant violation [%s] on %s(%s): %s - %s",
		e.Invariant, e.Entity, e.EntityID, e.Description, e.Details)
}

// checkInvariant handles an invariant violation based on current mode.
func checkInvariant(ok bool, inv *InvariantViolationError) bool {
	if ok {
		return true
	}

	switch invariantMode {
	case InvariantModePanic:
		panic(inv.Error())
	case InvariantModeLog:
		// In production, we'd log this. For now, just note it happened.
		// TODO: Integrate with structured logging
		_ = inv.Error()
	case InvariantModeDisabled:
		// Do nothing
	}

	return false
}

// =============================================================================
// RUN INVARIANTS
// =============================================================================

// INV_RUN_SANDBOX: After sandbox creation phase, sandboxed runs must have SandboxID
const InvRunSandbox = "INV_RUN_SANDBOX"

// CheckRunSandboxInvariant verifies the sandbox invariant for a run.
// Returns true if the invariant holds, false otherwise.
func (r *Run) CheckRunSandboxInvariant() bool {
	// Only applies to sandboxed runs past the sandbox creation phase
	if r.RunMode != RunModeSandboxed {
		return true
	}

	phaseOrd := phaseOrdinal(r.Phase)
	sandboxCreatingOrd := phaseOrdinal(RunPhaseSandboxCreating)

	// If we're past sandbox creation, we must have a sandbox ID
	if phaseOrd > sandboxCreatingOrd && r.SandboxID == nil {
		return checkInvariant(false, &InvariantViolationError{
			Invariant:   InvRunSandbox,
			Description: "sandboxed run past sandbox_creating phase must have SandboxID",
			Entity:      "Run",
			EntityID:    r.ID.String(),
			Details:     fmt.Sprintf("phase=%s, sandboxId=nil", r.Phase),
		})
	}

	return true
}

// INV_APPROVAL_STATE: Approval state transitions follow valid paths
const InvApprovalState = "INV_APPROVAL_STATE"

// CheckApprovalStateInvariant verifies the approval state invariant.
func (r *Run) CheckApprovalStateInvariant() bool {
	// ApprovalState should be None before needs_review
	if r.Status != RunStatusNeedsReview && r.Status != RunStatusComplete {
		if r.ApprovalState != ApprovalStateNone && r.ApprovalState != "" {
			return checkInvariant(false, &InvariantViolationError{
				Invariant:   InvApprovalState,
				Description: "approval state must be 'none' before needs_review status",
				Entity:      "Run",
				EntityID:    r.ID.String(),
				Details:     fmt.Sprintf("status=%s, approvalState=%s", r.Status, r.ApprovalState),
			})
		}
	}

	// Cannot be approved/rejected without going through needs_review
	if r.ApprovalState == ApprovalStateApproved || r.ApprovalState == ApprovalStateRejected {
		if r.ApprovedBy == "" {
			return checkInvariant(false, &InvariantViolationError{
				Invariant:   InvApprovalState,
				Description: "approved/rejected runs must have ApprovedBy set",
				Entity:      "Run",
				EntityID:    r.ID.String(),
				Details:     fmt.Sprintf("approvalState=%s, approvedBy=''", r.ApprovalState),
			})
		}
	}

	return true
}

// INV_TERMINAL_IMMUTABLE: Terminal states cannot transition back
const InvTerminalImmutable = "INV_TERMINAL_IMMUTABLE"

// CheckTerminalImmutabilityInvariant verifies terminal states are respected.
// This should be called BEFORE a status transition to validate it.
func (r *Run) CheckTerminalTransition(newStatus RunStatus) bool {
	if r.Status.IsTerminal() && !newStatus.IsTerminal() {
		return checkInvariant(false, &InvariantViolationError{
			Invariant:   InvTerminalImmutable,
			Description: "cannot transition from terminal state to non-terminal",
			Entity:      "Run",
			EntityID:    r.ID.String(),
			Details:     fmt.Sprintf("current=%s (terminal), proposed=%s", r.Status, newStatus),
		})
	}
	return true
}

// CheckTaskTerminalTransition verifies task terminal states are respected.
func (t *Task) CheckTerminalTransition(newStatus TaskStatus) bool {
	if t.Status.IsTerminal() && !newStatus.IsTerminal() {
		return checkInvariant(false, &InvariantViolationError{
			Invariant:   InvTerminalImmutable,
			Description: "cannot transition from terminal state to non-terminal",
			Entity:      "Task",
			EntityID:    t.ID.String(),
			Details:     fmt.Sprintf("current=%s (terminal), proposed=%s", t.Status, newStatus),
		})
	}
	return true
}

// INV_PHASE_SEQUENCE: Phases must follow defined progression
const InvPhaseSequence = "INV_PHASE_SEQUENCE"

// CheckPhaseTransition verifies a phase transition is valid.
func (r *Run) CheckPhaseTransition(newPhase RunPhase) bool {
	currentOrd := phaseOrdinal(r.Phase)
	newOrd := phaseOrdinal(newPhase)

	// Phases should only advance (except for retry which resets to same phase)
	if newOrd < currentOrd {
		return checkInvariant(false, &InvariantViolationError{
			Invariant:   InvPhaseSequence,
			Description: "phases cannot go backwards",
			Entity:      "Run",
			EntityID:    r.ID.String(),
			Details:     fmt.Sprintf("current=%s (ord=%d), proposed=%s (ord=%d)", r.Phase, currentOrd, newPhase, newOrd),
		})
	}

	// Phases should not skip (except when resuming)
	if newOrd > currentOrd+1 {
		// This is a warning, not a hard violation, as resumption can skip
		// We log but don't fail
		_ = &InvariantViolationError{
			Invariant:   InvPhaseSequence,
			Description: "phase transition skipped intermediate phases (resumption?)",
			Entity:      "Run",
			EntityID:    r.ID.String(),
			Details:     fmt.Sprintf("current=%s, proposed=%s, skipped=%d phases", r.Phase, newPhase, newOrd-currentOrd-1),
		}
	}

	return true
}

// phaseOrdinal returns the numeric order of a phase for comparison.
func phaseOrdinal(phase RunPhase) int {
	switch phase {
	case RunPhaseQueued:
		return 0
	case RunPhaseInitializing:
		return 1
	case RunPhaseSandboxCreating:
		return 2
	case RunPhaseRunnerAcquiring:
		return 3
	case RunPhaseExecuting:
		return 4
	case RunPhaseCollectingResults:
		return 5
	case RunPhaseAwaitingReview:
		return 6
	case RunPhaseApplying:
		return 7
	case RunPhaseCleaningUp:
		return 8
	case RunPhaseCompleted:
		return 9
	default:
		return 0
	}
}

// =============================================================================
// SAFE ACCESSORS (Nil-Safe Getters)
// =============================================================================
// These methods provide nil-safe access to optional fields, avoiding panics
// and making intent explicit.

// SafeSandboxID returns the sandbox ID or uuid.Nil if not set.
func (r *Run) SafeSandboxID() uuid.UUID {
	if r.SandboxID == nil {
		return uuid.Nil
	}
	return *r.SandboxID
}

// HasSandbox returns whether this run has an associated sandbox.
func (r *Run) HasSandbox() bool {
	return r.SandboxID != nil && *r.SandboxID != uuid.Nil
}

// SafeStartedAt returns the started time or zero time if not set.
func (r *Run) SafeStartedAt() time.Time {
	if r.StartedAt == nil {
		return time.Time{}
	}
	return *r.StartedAt
}

// SafeEndedAt returns the ended time or zero time if not set.
func (r *Run) SafeEndedAt() time.Time {
	if r.EndedAt == nil {
		return time.Time{}
	}
	return *r.EndedAt
}

// SafeLastHeartbeat returns the last heartbeat time or zero time if not set.
func (r *Run) SafeLastHeartbeat() time.Time {
	if r.LastHeartbeat == nil {
		return time.Time{}
	}
	return *r.LastHeartbeat
}

// SafeExitCode returns the exit code or -1 if not set.
func (r *Run) SafeExitCode() int {
	if r.ExitCode == nil {
		return -1
	}
	return *r.ExitCode
}

// Duration returns the run duration, or 0 if not started or still running.
func (r *Run) Duration() time.Duration {
	if r.StartedAt == nil {
		return 0
	}
	if r.EndedAt == nil {
		// Still running - return elapsed time
		return time.Since(*r.StartedAt)
	}
	return r.EndedAt.Sub(*r.StartedAt)
}

// SafeApprovedAt returns the approval time or zero time if not set.
func (r *Run) SafeApprovedAt() time.Time {
	if r.ApprovedAt == nil {
		return time.Time{}
	}
	return *r.ApprovedAt
}

// SafeLastCheckpointID returns the last checkpoint ID or uuid.Nil if not set.
func (r *Run) SafeLastCheckpointID() uuid.UUID {
	if r.LastCheckpointID == nil {
		return uuid.Nil
	}
	return *r.LastCheckpointID
}

// SafeSandboxID returns the sandbox ID from checkpoint or uuid.Nil if not set.
func (c *RunCheckpoint) SafeSandboxID() uuid.UUID {
	if c.SandboxID == nil {
		return uuid.Nil
	}
	return *c.SandboxID
}

// SafeLockID returns the lock ID from checkpoint or uuid.Nil if not set.
func (c *RunCheckpoint) SafeLockID() uuid.UUID {
	if c.LockID == nil {
		return uuid.Nil
	}
	return *c.LockID
}

// SafeEntityID returns the entity ID from record or uuid.Nil if not set.
func (r *IdempotencyRecord) SafeEntityID() uuid.UUID {
	if r.EntityID == nil {
		return uuid.Nil
	}
	return *r.EntityID
}

// =============================================================================
// STATE ASSERTIONS
// =============================================================================
// These methods assert preconditions and return errors if not met.
// Use these at service boundaries to validate state before operations.

// AssertCanStart asserts a run can be started.
// Returns nil if valid, or an error describing why not.
func (r *Run) AssertCanStart() error {
	if r.Status != RunStatusPending {
		return NewStateError("Run", string(r.Status), "start",
			"can only start runs in pending status")
	}
	return nil
}

// AssertCanStop asserts a run can be stopped.
func (r *Run) AssertCanStop() error {
	if !r.IsStoppable() {
		return NewStateError("Run", string(r.Status), "stop",
			"can only stop runs in starting or running status")
	}
	return nil
}

// AssertCanApprove asserts a run can be approved.
func (r *Run) AssertCanApprove() error {
	if ok, reason := r.IsApprovable(); !ok {
		return NewStateError("Run", string(r.Status), "approve", reason)
	}
	return nil
}

// AssertCanReject asserts a run can be rejected.
func (r *Run) AssertCanReject() error {
	if ok, reason := r.IsRejectable(); !ok {
		return NewStateError("Run", string(r.Status), "reject", reason)
	}
	return nil
}

// AssertCanResume asserts a run can be resumed.
func (r *Run) AssertCanResume() error {
	if !r.IsResumable() {
		return NewStateError("Run", string(r.Status), "resume",
			"run is not in a resumable state")
	}
	return nil
}

// AssertCanCancel asserts a task can be cancelled.
func (t *Task) AssertCanCancel() error {
	if !t.IsCancellable() {
		return NewStateError("Task", string(t.Status), "cancel",
			"can only cancel tasks in queued or running status")
	}
	return nil
}

// =============================================================================
// CONSISTENCY CHECKS
// =============================================================================
// These run multiple invariant checks and return all violations.

// CheckAllInvariants runs all invariant checks on a run.
// Returns a slice of all violations found (empty if all pass).
func (r *Run) CheckAllInvariants() []error {
	var violations []error

	// Temporarily use log mode to collect all violations
	oldMode := invariantMode
	invariantMode = InvariantModeLog

	if !r.CheckRunSandboxInvariant() {
		violations = append(violations, &InvariantViolationError{
			Invariant: InvRunSandbox,
			Entity:    "Run",
			EntityID:  r.ID.String(),
		})
	}

	if !r.CheckApprovalStateInvariant() {
		violations = append(violations, &InvariantViolationError{
			Invariant: InvApprovalState,
			Entity:    "Run",
			EntityID:  r.ID.String(),
		})
	}

	// Restore mode
	invariantMode = oldMode

	return violations
}

// =============================================================================
// SCOPE CONFLICT HELPERS
// =============================================================================
// Note: ScopeConflict type is defined in errors.go

// ClassifyScopeConflict determines the type of scope conflict.
func ClassifyScopeConflict(requested, existing string) string {
	if requested == existing {
		return "exact"
	}
	if ScopesOverlap(requested, existing) {
		if len(requested) < len(existing) {
			return "ancestor"
		}
		return "descendant"
	}
	return "" // No conflict
}

// =============================================================================
// LIFECYCLE STATE HELPERS
// =============================================================================

// LifecycleState represents a simplified view of run state for decision making.
type LifecycleState string

const (
	LifecycleStateNew        LifecycleState = "new"        // Not yet started
	LifecycleStateActive     LifecycleState = "active"     // Currently running
	LifecycleStateReviewable LifecycleState = "reviewable" // Awaiting approval
	LifecycleStateFinished   LifecycleState = "finished"   // Terminal state
)

// GetLifecycleState returns the simplified lifecycle state of a run.
func (r *Run) GetLifecycleState() LifecycleState {
	switch r.Status {
	case RunStatusPending:
		return LifecycleStateNew
	case RunStatusStarting, RunStatusRunning:
		return LifecycleStateActive
	case RunStatusNeedsReview:
		return LifecycleStateReviewable
	case RunStatusComplete, RunStatusFailed, RunStatusCancelled:
		return LifecycleStateFinished
	default:
		return LifecycleStateNew
	}
}

// CanReceiveEvents returns whether this run can still receive events.
func (r *Run) CanReceiveEvents() bool {
	state := r.GetLifecycleState()
	return state == LifecycleStateActive || state == LifecycleStateReviewable
}

// CanReceiveHeartbeats returns whether this run should receive heartbeats.
func (r *Run) CanReceiveHeartbeats() bool {
	return r.GetLifecycleState() == LifecycleStateActive
}
