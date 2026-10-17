package rewrite

import (
	"errors"

	"connectrpc.com/connect"
)

// ErrorToConnectCode maps a Service error to its canonical Connect
// code per plan §8.2. Unknown errors map to CodeInternal.
//
//	NoOperations | MalformedOperation | ApplyNotSet | PlanNotFound → InvalidArgument
//	PathMismatch                                                    → FailedPrecondition
//	Internal                                                        → Internal
//
// PlanNotFound maps to InvalidArgument (not NotFound) per the plan's
// rewrite-domain decision — an unknown plan_id is treated as a bad
// argument, not a missing resource, because plans are
// caller-derived and not addressable through the URL space.
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return connect.Code(0)
	}
	var rerr RewriteError
	if errors.As(err, &rerr) {
		switch rerr.Kind {
		case RewriteErrorNoOperations,
			RewriteErrorMalformedOperation,
			RewriteErrorApplyNotSet,
			RewriteErrorPlanNotFound:
			return connect.CodeInvalidArgument
		case RewriteErrorPathMismatch:
			return connect.CodeFailedPrecondition
		case RewriteErrorInternal:
			return connect.CodeInternal
		default:
			return connect.CodeInternal
		}
	}
	return connect.CodeInternal
}

// ToConnectError wraps err in a connect.Error tagged with the code
// returned by ErrorToConnectCode. nil in, nil out.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	return connect.NewError(ErrorToConnectCode(err), err)
}
