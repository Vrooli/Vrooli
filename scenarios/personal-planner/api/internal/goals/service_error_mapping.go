package goals

import (
	"errors"

	"connectrpc.com/connect"
)

func ToConnectError(err error) error {
	var inv ErrInvalidGoal
	if errors.As(err, &inv) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var nf ErrGoalNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var mnf ErrMilestoneNotFound
	if errors.As(err, &mnf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var rc ErrRevisionConflict
	if errors.As(err, &rc) {
		return connect.NewError(connect.CodeAborted, err)
	}
	var pi ErrPrerequisitesIncomplete
	if errors.As(err, &pi) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
