package cprev

import (
	"errors"

	"connectrpc.com/connect"
)

// ConnectError maps a revision-resolution error to a friendly Connect error, or
// returns nil when err is not a cprev error so the caller can fall back to its
// own domain mapping. It is the single translation point for the resolver's
// typed errors, shared by the onboard and provision handlers:
//
//   - ErrUnsafeRevision   → InvalidArgument (the operator passed a bad ref)
//   - ErrNotPushed        → FailedPrecondition (push first, then retry)
//   - ErrNoControlPlaneCommit → FailedPrecondition (pass an explicit revision)
//
// The error VALUE is preserved so its message (which names the commit/remote or
// the disallowed character) reaches the client verbatim.
func ConnectError(err error) error {
	if err == nil {
		return nil
	}
	var (
		unsafe   ErrUnsafeRevision
		notPush  ErrNotPushed
		noCommit ErrNoControlPlaneCommit
	)
	switch {
	case errors.As(err, &unsafe):
		return connect.NewError(connect.CodeInvalidArgument, unsafe)
	case errors.As(err, &notPush):
		return connect.NewError(connect.CodeFailedPrecondition, notPush)
	case errors.As(err, &noCommit):
		return connect.NewError(connect.CodeFailedPrecondition, noCommit)
	default:
		return nil
	}
}
