package dispatch

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates the dispatch domain's typed sentinels into Connect
// error codes at the transport edge:
//
//   - allowlist/scope/unsafe-token denials → PermissionDenied (the caller is
//     not authorized to run this verb on this node)
//   - structural job errors → InvalidArgument
//   - unknown node → NotFound
//   - revoked/offline node → FailedPrecondition (the node cannot accept it now)
//   - delivery failure → Unavailable (transient)
//   - anything else → Internal
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var (
		invalid     ErrInvalidJob
		notInMan    ErrVerbNotInManifest
		outOfScope  ErrVerbOutOfScope
		unsafe      ErrUnsafeToken
		notFound    ErrNodeNotFound
		revoked     ErrNodeRevoked
		offline     ErrNodeOffline
		needsUpdate ErrNodeNeedsUpdate
		kind        ErrUnsupportedNodeKind
		delivery    ErrDeliveryFailed
		leaseReq    ErrDeviceLeaseRequired
		leaseHeld   ErrDeviceLeaseNotHeld
	)
	switch {
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	case errors.As(err, &notInMan):
		return connect.NewError(connect.CodePermissionDenied, notInMan)
	case errors.As(err, &outOfScope):
		return connect.NewError(connect.CodePermissionDenied, outOfScope)
	case errors.As(err, &unsafe):
		return connect.NewError(connect.CodePermissionDenied, unsafe)
	case errors.As(err, &notFound):
		return connect.NewError(connect.CodeNotFound, notFound)
	case errors.As(err, &revoked):
		return connect.NewError(connect.CodeFailedPrecondition, revoked)
	case errors.As(err, &offline):
		return connect.NewError(connect.CodeFailedPrecondition, offline)
	case errors.As(err, &needsUpdate):
		return connect.NewError(connect.CodeFailedPrecondition, needsUpdate)
	case errors.As(err, &kind):
		return connect.NewError(connect.CodeFailedPrecondition, kind)
	case errors.As(err, &leaseReq):
		return connect.NewError(connect.CodeFailedPrecondition, leaseReq)
	case errors.As(err, &leaseHeld):
		return connect.NewError(connect.CodeFailedPrecondition, leaseHeld)
	case errors.As(err, &delivery):
		return connect.NewError(connect.CodeUnavailable, delivery)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// IsRejection reports whether err is an allowlist/scope/precondition rejection
// (as opposed to an internal failure). The handler uses it to decide log noise.
func IsRejection(err error) bool {
	var (
		invalid     ErrInvalidJob
		notInMan    ErrVerbNotInManifest
		outOfScope  ErrVerbOutOfScope
		unsafe      ErrUnsafeToken
		notFound    ErrNodeNotFound
		revoked     ErrNodeRevoked
		offline     ErrNodeOffline
		needsUpdate ErrNodeNeedsUpdate
		kind        ErrUnsupportedNodeKind
		leaseReq    ErrDeviceLeaseRequired
		leaseHeld   ErrDeviceLeaseNotHeld
	)
	return errors.As(err, &invalid) || errors.As(err, &notInMan) || errors.As(err, &outOfScope) ||
		errors.As(err, &unsafe) || errors.As(err, &notFound) || errors.As(err, &revoked) ||
		errors.As(err, &offline) || errors.As(err, &needsUpdate) || errors.As(err, &kind) ||
		errors.As(err, &leaseReq) || errors.As(err, &leaseHeld)
}
