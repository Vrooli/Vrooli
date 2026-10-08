package apply

import (
	"errors"

	"architecture-cartographer/internal/domains"

	"connectrpc.com/connect"
)

// ErrorToConnectCode translates apply typed sentinels into Connect codes.
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return 0
	}
	var (
		invReq       ErrInvalidPlanRequest
		unimplented  ErrApplyUnimplemented
		unconfigured ErrSuppressionUnconfigured
		scenarioMiss domains.ErrScenarioNotFound
	)
	switch {
	case errors.As(err, &invReq):
		return connect.CodeInvalidArgument
	case errors.As(err, &unimplented):
		return connect.CodeUnimplemented
	case errors.As(err, &unconfigured):
		return connect.CodeFailedPrecondition
	case errors.As(err, &scenarioMiss):
		return connect.CodeNotFound
	default:
		return connect.CodeInternal
	}
}
