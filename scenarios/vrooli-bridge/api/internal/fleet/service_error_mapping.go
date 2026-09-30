package fleet

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates the fleet domain's typed sentinels into Connect
// error codes at the transport edge:
//
//   - structural request errors → InvalidArgument
//   - unknown rollout → NotFound
//   - anything else → Internal
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var (
		invalid   ErrInvalidRoll
		rolloutNF ErrRolloutNotFound
	)
	switch {
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	case errors.As(err, &rolloutNF):
		return connect.NewError(connect.CodeNotFound, rolloutNF)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
