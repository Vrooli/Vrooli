package planlog

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates log-domain sentinels into Connect's typed error
// model. Unknown errors map to internal so callers never depend on raw storage
// or implementation details. Downstream-unavailability is NOT mapped here — it
// is captured as a degraded sync state on the durable entry, never surfaced as a
// request error.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidEntry
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notPromotable ErrNotPromotable
	if errors.As(err, &notPromotable) {
		return connect.NewError(connect.CodeFailedPrecondition, notPromotable)
	}
	var notFound ErrEntryNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	return connect.NewError(connect.CodeInternal, err)
}
