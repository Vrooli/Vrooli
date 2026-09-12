package provision

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates the provision domain's typed sentinels into Connect
// error codes at the transport edge:
//
//   - structural request errors → InvalidArgument
//   - unknown node / unknown op / no recorded version → NotFound
//   - revoked/offline node → FailedPrecondition (cannot accept it now)
//   - delivery failure → Unavailable (transient)
//   - anything else → Internal
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var (
		invalid   ErrInvalidOp
		nodeNF    ErrNodeNotFound
		opNF      ErrOpNotFound
		noVersion ErrNoNodeVersion
		revoked   ErrNodeRevoked
		offline   ErrNodeOffline
		kind      ErrUnsupportedNodeKind
		delivery  ErrDeliveryFailed
	)
	switch {
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	case errors.As(err, &nodeNF):
		return connect.NewError(connect.CodeNotFound, nodeNF)
	case errors.As(err, &opNF):
		return connect.NewError(connect.CodeNotFound, opNF)
	case errors.As(err, &noVersion):
		return connect.NewError(connect.CodeNotFound, noVersion)
	case errors.As(err, &revoked):
		return connect.NewError(connect.CodeFailedPrecondition, revoked)
	case errors.As(err, &offline):
		return connect.NewError(connect.CodeFailedPrecondition, offline)
	case errors.As(err, &kind):
		return connect.NewError(connect.CodeFailedPrecondition, kind)
	case errors.As(err, &delivery):
		return connect.NewError(connect.CodeUnavailable, delivery)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
