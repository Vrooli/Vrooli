package integrations

import (
	"connectrpc.com/connect"
)

func ToConnectError(err error) error {
	switch err.(type) {
	case ErrInvalid:
		return connect.NewError(connect.CodeInvalidArgument, err)
	case ErrConflict:
		return connect.NewError(connect.CodeAborted, err)
	case ErrNotFound:
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
