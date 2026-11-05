package settings

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error
// model. Unknown errors map to internal so callers never depend on raw
// storage or implementation details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var ve ValidationError
	if errors.As(err, &ve) {
		return connect.NewError(connect.CodeInvalidArgument, ve)
	}
	return connect.NewError(connect.CodeInternal, err)
}
