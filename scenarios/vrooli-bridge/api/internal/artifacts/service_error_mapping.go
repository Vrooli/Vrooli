package artifacts

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates the artifacts domain's typed sentinels into Connect
// error codes at the transport edge:
//
//   - structural request errors → InvalidArgument
//   - unknown node / unknown distribution → NotFound
//   - revoked node → FailedPrecondition
//   - delivery failure → Unavailable (transient)
//   - anything else → Internal
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var (
		invalid         ErrInvalidDistribution
		nodeNF          ErrNodeNotFound
		distNF          ErrDistributionNotFound
		revoked         ErrNodeRevoked
		deliv           ErrDeliveryFailed
		invalidProduced ErrInvalidProducedArtifact
		mismatch        ErrArtifactNodeMismatch
		producedNF      ErrProducedArtifactNotFound
	)
	switch {
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	case errors.As(err, &nodeNF):
		return connect.NewError(connect.CodeNotFound, nodeNF)
	case errors.As(err, &distNF):
		return connect.NewError(connect.CodeNotFound, distNF)
	case errors.As(err, &revoked):
		return connect.NewError(connect.CodeFailedPrecondition, revoked)
	case errors.As(err, &deliv):
		return connect.NewError(connect.CodeUnavailable, deliv)
	case errors.As(err, &invalidProduced):
		return connect.NewError(connect.CodeInvalidArgument, invalidProduced)
	case errors.As(err, &mismatch):
		return connect.NewError(connect.CodePermissionDenied, mismatch)
	case errors.As(err, &producedNF):
		return connect.NewError(connect.CodeNotFound, producedNF)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
