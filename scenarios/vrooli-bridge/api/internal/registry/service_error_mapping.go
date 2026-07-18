package registry

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates the registry domain's typed sentinels into Connect
// error codes at the transport edge. Unknown errors map to Internal so a
// storage failure never leaks as a 400. Mirrors the device-sync-hub/notes
// convention.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidNode
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrNodeNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var active ErrNodeActive
	if errors.As(err, &active) {
		return connect.NewError(connect.CodeFailedPrecondition, active)
	}
	return connect.NewError(connect.CodeInternal, err)
}
