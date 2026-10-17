package config

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error
// model. Unknown errors intentionally map to internal so callers never
// depend on raw storage details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidConfig
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var unavailable ErrRemoteUnavailable
	if errors.As(err, &unavailable) {
		// Creds-absent is a precondition the caller must fix (configure CF
		// credentials), not a transient outage — map to FailedPrecondition.
		return connect.NewError(connect.CodeFailedPrecondition, unavailable)
	}
	return connect.NewError(connect.CodeInternal, err)
}
