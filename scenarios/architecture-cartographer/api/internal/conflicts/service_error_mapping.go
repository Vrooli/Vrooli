package conflicts

import (
	"errors"

	"connectrpc.com/connect"
)

// ErrorToConnectCode translates conflicts typed sentinels into Connect codes.
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return 0
	}
	var (
		notFound ErrConflictNotFound
		invInput ErrInvalidInput
		deferred ErrResolverDeferred
	)
	switch {
	case errors.As(err, &notFound):
		return connect.CodeNotFound
	case errors.As(err, &invInput):
		return connect.CodeInvalidArgument
	case errors.As(err, &deferred):
		// Deferred is not surfaced as a Connect error; handlers render
		// the deferral as part of the response body. The zero Code is
		// a sentinel meaning "do not wrap as a Connect error."
		return 0
	default:
		return connect.CodeInternal
	}
}
