package workspace

import (
	"errors"

	"connectrpc.com/connect"
)

func ToConnectError(err error) error {
	var invalid ErrInvalidProfile
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.As(err, new(ErrRevisionConflict)) {
		return connect.NewError(connect.CodeAborted, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
