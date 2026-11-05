package runs

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates the runs domain's typed sentinels into Connect
// error codes at the transport edge. Unknown errors map to Internal so a
// storage failure never leaks as a 400. Mirrors the registry convention.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidRun
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrRunNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	return connect.NewError(connect.CodeInternal, err)
}
