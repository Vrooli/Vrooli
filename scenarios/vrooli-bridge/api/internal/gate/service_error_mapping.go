package gate

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates the gate domain's typed sentinels into Connect error
// codes at the transport edge:
//
//   - structural request errors → InvalidArgument
//   - unknown gate → NotFound
//   - anything else → Internal
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var (
		invalid ErrInvalidGate
		gateNF  ErrGateNotFound
	)
	switch {
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	case errors.As(err, &gateNF):
		return connect.NewError(connect.CodeNotFound, gateNF)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
