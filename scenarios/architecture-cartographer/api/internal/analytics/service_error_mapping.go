package analytics

import (
	"errors"

	"connectrpc.com/connect"
)

// ErrorToConnectCode translates analytics typed sentinels into Connect
// error codes. Handlers wrap the returned error via connect.NewError so
// the wire shape matches api-steer §11.
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return 0
	}
	var (
		invEvent    ErrInvalidEvent
		invOverride ErrInvalidOverride
	)
	switch {
	case errors.As(err, &invEvent), errors.As(err, &invOverride):
		return connect.CodeInvalidArgument
	default:
		return connect.CodeInternal
	}
}
