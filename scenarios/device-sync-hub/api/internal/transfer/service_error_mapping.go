package transfer

import (
	"errors"

	"device-sync-hub/internal/devices"

	"connectrpc.com/connect"
)

// ToConnectError translates transfer (and device-trust) sentinels into Connect's
// typed error model. Unknown errors map to internal so callers never depend on
// raw storage or implementation details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidItem
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var badTarget ErrInvalidTarget
	if errors.As(err, &badTarget) {
		return connect.NewError(connect.CodeInvalidArgument, badTarget)
	}
	var quota ErrQuotaExceeded
	if errors.As(err, &quota) {
		return connect.NewError(connect.CodeResourceExhausted, quota)
	}
	var notFound ErrItemNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	if errors.Is(err, devices.ErrUntrustedDevice) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
