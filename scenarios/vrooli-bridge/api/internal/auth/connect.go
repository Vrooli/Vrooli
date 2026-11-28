package auth

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError maps the auth package's sentinels to Connect error codes at
// the transport edge: ErrUnauthenticated → Unauthenticated (401-equivalent),
// ErrAuthUnavailable → Unavailable (so "couldn't check right now" is a
// retryable 503-equivalent, never a 401), and a missing dispatch-only service
// scope → PermissionDenied. Any other error maps to Internal.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrUnauthenticated) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}
	if errors.Is(err, ErrAuthUnavailable) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if errors.Is(err, ErrDispatchServiceScope) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
