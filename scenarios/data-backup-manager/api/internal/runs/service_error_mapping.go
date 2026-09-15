package runs

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error model.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidRun
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrRunNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var active ErrRunAlreadyActive
	if errors.As(err, &active) {
		return connect.NewError(connect.CodeAlreadyExists, active)
	}
	return connect.NewError(connect.CodeInternal, err)
}
