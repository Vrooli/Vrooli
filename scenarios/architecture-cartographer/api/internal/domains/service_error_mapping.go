package domains

import (
	"errors"

	"connectrpc.com/connect"
)

// ErrorToConnectCode translates domains typed sentinels into Connect codes.
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return 0
	}
	var (
		notFound    ErrScenarioNotFound
		noAuthority ErrNoAuthority
	)
	switch {
	case errors.As(err, &notFound):
		return connect.CodeNotFound
	case errors.As(err, &noAuthority):
		return connect.CodeFailedPrecondition
	default:
		return connect.CodeInternal
	}
}
