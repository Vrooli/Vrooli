package rewrite

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"typescript-code-graph/internal/sidecar"
)

// ErrorToConnectCode translates a Service error to its canonical
// Connect error code. Unknown errors map to CodeInternal.
//
// Mapping (per plan §8.2):
//
//	RewriteErrorInvalidInput        → InvalidArgument
//	RewriteErrorInvalidOperation    → InvalidArgument
//	RewriteErrorPlanNotFound        → FailedPrecondition
//	RewriteErrorSidecarUnavailable  → Unavailable
//	RewriteErrorSidecarTimeout      → DeadlineExceeded
//	RewriteErrorInternal            → Internal
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return connect.Code(0)
	}
	var re RewriteError
	if errors.As(err, &re) {
		switch re.Kind {
		case RewriteErrorInvalidInput, RewriteErrorInvalidOperation:
			return connect.CodeInvalidArgument
		case RewriteErrorPlanNotFound:
			return connect.CodeFailedPrecondition
		case RewriteErrorSidecarUnavailable:
			return connect.CodeUnavailable
		case RewriteErrorSidecarTimeout:
			return connect.CodeDeadlineExceeded
		case RewriteErrorInternal:
			return connect.CodeInternal
		default:
			return connect.CodeInternal
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return connect.CodeDeadlineExceeded
	}
	if errors.Is(err, context.Canceled) {
		return connect.CodeCanceled
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

// fromSidecarError translates a sidecar.* sentinel or typed error
// returned from RewriteApply into the domain's RewriteError. Used by
// Service.Apply so the handler only ever sees domain errors.
func fromSidecarError(path string, planID PlanID, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sidecar.ErrSidecarUnavailable) ||
		errors.Is(err, sidecar.ErrSidecarPermanentlyUnhealthy) {
		return RewriteError{Kind: RewriteErrorSidecarUnavailable, Path: path, PlanID: planID, Cause: err}
	}
	if errors.Is(err, sidecar.ErrSidecarTimeout) {
		return RewriteError{Kind: RewriteErrorSidecarTimeout, Path: path, PlanID: planID, Cause: err}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return RewriteError{Kind: RewriteErrorSidecarTimeout, Path: path, PlanID: planID, Cause: err}
	}
	if errors.Is(err, context.Canceled) {
		return RewriteError{Kind: RewriteErrorInternal, Path: path, PlanID: planID, Cause: err}
	}
	var ree *sidecar.RewriteError
	if errors.As(err, &ree) {
		return RewriteError{Kind: RewriteErrorInternal, Path: path, PlanID: planID, Message: ree.Message, Cause: ree}
	}
	return RewriteError{Kind: RewriteErrorInternal, Path: path, PlanID: planID, Cause: err}
}
