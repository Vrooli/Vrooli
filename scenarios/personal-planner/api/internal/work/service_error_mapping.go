package work

import (
	"errors"

	"connectrpc.com/connect"
)

func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidWorkItem
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var missing ErrWorkItemNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, missing)
	}
	return connect.NewError(connect.CodeInternal, err)
}
