package focus

import (
	"errors"

	"connectrpc.com/connect"
)

func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidFocus
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var missing ErrSessionNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, missing)
	}
	var transition ErrInvalidTransition
	if errors.As(err, &transition) {
		return connect.NewError(connect.CodeFailedPrecondition, transition)
	}
	var conflict ErrSessionConflict
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeAlreadyExists, conflict)
	}
	var revision ErrRevisionConflict
	if errors.As(err, &revision) {
		return connect.NewError(connect.CodeAborted, revision)
	}
	return connect.NewError(connect.CodeInternal, err)
}
