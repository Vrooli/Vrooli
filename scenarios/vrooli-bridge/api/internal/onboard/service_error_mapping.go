package onboard

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates the onboard domain's typed sentinels into Connect
// error codes at the transport edge:
//
//   - structural request errors → InvalidArgument
//   - unknown op → NotFound
//   - anything else → Internal
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var (
		invalid  ErrInvalid
		conflict ErrConflict
		opNF     ErrOpNotFound
	)
	switch {
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	case errors.As(err, &conflict):
		return connect.NewError(connect.CodeFailedPrecondition, conflict)
	case errors.As(err, &opNF):
		return connect.NewError(connect.CodeNotFound, opNF)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
