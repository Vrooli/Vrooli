package apply

import (
	"errors"

	"connectrpc.com/connect"
)

// ErrorToConnectCode translates apply typed sentinels into Connect codes.
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return 0
	}
	var (
		invReq      ErrInvalidPlanRequest
		unimplented ErrApplyUnimplemented
	)
	switch {
	case errors.As(err, &invReq):
		return connect.CodeInvalidArgument
	case errors.As(err, &unimplented):
		return connect.CodeUnimplemented
	default:
		return connect.CodeInternal
	}
}
