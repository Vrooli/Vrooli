package devices

import (
	"errors"

	"device-sync-hub/internal/auth"

	"connectrpc.com/connect"
)

// ToConnectError translates domain (and auth) sentinels into Connect's typed
// error model. Unknown errors map to internal so callers never depend on raw
// storage or implementation details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidDevice
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var badCode ErrInvalidPairingCode
	if errors.As(err, &badCode) {
		return connect.NewError(connect.CodeInvalidArgument, badCode)
	}
	var notFound ErrDeviceNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var conflict ErrDeviceConflict
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeFailedPrecondition, conflict)
	}
	var notOwner ErrNotOwner
	if errors.As(err, &notOwner) {
		return connect.NewError(connect.CodePermissionDenied, notOwner)
	}
	if errors.Is(err, auth.ErrUnauthenticated) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}
	if errors.Is(err, auth.ErrAuthUnavailable) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
