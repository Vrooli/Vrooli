package validation

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	planmodel "plan-manager/internal/planmodel"
)

// ErrPhaseNotFound is returned when a phase id is not on the named plan.
type ErrPhaseNotFound struct {
	PlanID  string
	PhaseID string
}

// ErrOperationNotFound is returned when a durable validation id is unknown.
type ErrOperationNotFound struct{ ID string }

// ErrProducerTicketRequired makes the former worker-owned routes honest: a
// validation ticket only projects producer commands, and Git Control Tower or
// Test Genie owns the long-running work and native wait.
type ErrProducerTicketRequired struct{ PlanID string }

func (e ErrProducerTicketRequired) Error() string {
	return fmt.Sprintf("plan %q requires a producer-owned validation ticket: use validate start, the rendered producer wait command, then validate sync", e.PlanID)
}

func (e ErrOperationNotFound) Error() string {
	return fmt.Sprintf("validation operation %q not found", e.ID)
}

// ErrAttachmentEnded means only this wait attachment ended. The operation id is
// durable and server-owned work continues; callers must inspect or resume it.
type ErrAttachmentEnded struct {
	OperationID string
	Cause       error
}

func (e ErrAttachmentEnded) Error() string {
	return fmt.Sprintf("validation wait attachment ended for %q; inspect or resume the durable operation: %v", e.OperationID, e.Cause)
}

func (e ErrAttachmentEnded) Unwrap() error { return e.Cause }

func (e ErrPhaseNotFound) Error() string {
	return fmt.Sprintf("phase %q not found on plan %q", e.PhaseID, e.PlanID)
}

// ToConnectError translates validation/plans sentinels into Connect's typed
// error model. Unknown errors map to internal.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var phaseNotFound ErrPhaseNotFound
	if errors.As(err, &phaseNotFound) {
		return connect.NewError(connect.CodeNotFound, phaseNotFound)
	}
	var operationNotFound ErrOperationNotFound
	if errors.As(err, &operationNotFound) {
		return connect.NewError(connect.CodeNotFound, operationNotFound)
	}
	var producerTicketRequired ErrProducerTicketRequired
	if errors.As(err, &producerTicketRequired) {
		return connect.NewError(connect.CodeFailedPrecondition, producerTicketRequired)
	}
	var attachmentEnded ErrAttachmentEnded
	if errors.As(err, &attachmentEnded) {
		if errors.Is(attachmentEnded.Cause, context.DeadlineExceeded) {
			return connect.NewError(connect.CodeDeadlineExceeded, attachmentEnded)
		}
		return connect.NewError(connect.CodeCanceled, attachmentEnded)
	}
	var planNotFound planmodel.ErrPlanNotFound
	if errors.As(err, &planNotFound) {
		return connect.NewError(connect.CodeNotFound, planNotFound)
	}
	var invalid planmodel.ErrInvalidPlan
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	return connect.NewError(connect.CodeInternal, err)
}
