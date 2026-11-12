// Package operations owns durable cloud operations: the state machine, the
// fenced worker that executes an admitted plan, startup reconciliation and
// the wait/status/cancel contract. Persistence is the source of truth; the
// in-process worker pool only supplies compute, and operationcoord only
// closes waiter races.
//
// DOC: docs/reference/operation-lifecycle.md
package operations

import (
	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// State aliases the domain vocabulary so callers of this package never spell
// a state string twice.
type State = domain.OperationState

// The canonical operation vocabulary (design D of the certification plan).
const (
	Admitted        = domain.OperationAdmitted
	WaitingInput    = domain.OperationWaitingInput
	Running         = domain.OperationRunning
	Verifying       = domain.OperationVerifying
	Reconciling     = domain.OperationReconciling
	Recovering      = domain.OperationRecovering
	CancelRequested = domain.OperationCancelRequested
	Succeeded       = domain.OperationSucceeded
	Failed          = domain.OperationFailed
	FailedRecovery  = domain.OperationFailedRecovery
	Cancelled       = domain.OperationCancelled
)

// States lists every state in a stable order.
func States() []State {
	return domain.OperationStates()
}

// Legal reports whether from → to is an allowed transition.
func Legal(from, to State) bool {
	return from.CanTransitionTo(to)
}

// Transition validates from → to and returns a typed operation_conflict when
// the move is not in the table. A terminal state admits no transition.
func Transition(from, to State) error {
	op := &domain.CloudOperation{State: from}
	if err := op.TransitionTo(to); err != nil {
		return apierrors.New(apierrors.CodeOperationConflict, err.Error()).
			WithDetail("state", string(from)).WithDetail("requested", string(to))
	}
	return nil
}

// Recovery outcomes recorded on a failed operation. A failed change can still
// restore service; that is never reported as operation success.
const (
	RecoveryServiceRestored = "service_restored"
	RecoveryNotAttempted    = "not_attempted"
	RecoveryFailed          = "recovery_failed"
)

// Result, StepReceipt and UnknownEffect are the durable envelopes stored on
// the operation row; they live in domain so persistence can scan them.
type (
	Result        = domain.OperationResult
	StepReceipt   = domain.StepReceipt
	UnknownEffect = domain.UnknownEffect
	StepOutcome   = domain.StepOutcome
)

// Step outcomes. Unknown is recorded, never inferred away.
const (
	StepSucceeded = domain.StepSucceeded
	StepFailed    = domain.StepFailed
	StepSkipped   = domain.StepSkipped
	StepUnchanged = domain.StepUnchanged
	StepUnknown   = domain.StepUnknown
)
