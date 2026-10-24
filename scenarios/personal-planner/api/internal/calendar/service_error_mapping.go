package calendar

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
)

func ToConnectError(err error) error {
	var invalid ErrInvalidAllocation
	var missing ErrWorkItemNotFound
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var allocationMissing ErrAllocationNotFound
	if errors.As(err, &allocationMissing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var routineMissing ErrRoutineNotFound
	if errors.As(err, &routineMissing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var routineConflict ErrRoutineRevisionConflict
	if errors.As(err, &routineConflict) {
		return connect.NewError(connect.CodeAborted, err)
	}
	var conflict ErrAllocationConflict
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	var demand ErrDemandExceeded
	if errors.As(err, &demand) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	var revision ErrScheduleRevisionConflict
	if errors.As(err, &revision) {
		return connect.NewError(connect.CodeAborted, err)
	}
	var proposalMissing ErrProposalNotFound
	if errors.As(err, &proposalMissing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var proposalNotFeasible ErrProposalNotFeasible
	if errors.As(err, &proposalNotFeasible) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	var proposalApplied ErrProposalAlreadyApplied
	if errors.As(err, &proposalApplied) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	var eventMissing ErrEventNotFound
	if errors.As(err, &eventMissing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var eventRevision ErrEventRevisionConflict
	if errors.As(err, &eventRevision) {
		return connect.NewError(connect.CodeAborted, err)
	}
	var eventIdempotency ErrEventIdempotencyConflict
	if errors.As(err, &eventIdempotency) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	var eventIdentity ErrEventIdentityConflict
	if errors.As(err, &eventIdentity) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("calendar: %w", err))
}
