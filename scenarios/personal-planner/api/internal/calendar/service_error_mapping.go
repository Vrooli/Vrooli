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
	return connect.NewError(connect.CodeInternal, fmt.Errorf("calendar: %w", err))
}
