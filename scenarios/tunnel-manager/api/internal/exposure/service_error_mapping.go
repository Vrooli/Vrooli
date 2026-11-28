package exposure

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error
// model. Unknown errors map to internal.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidExposure
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrLeaseNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var invalidTransition ErrInvalidLeaseTransition
	if errors.As(err, &invalidTransition) {
		return connect.NewError(connect.CodeFailedPrecondition, invalidTransition)
	}
	var transitionConflict ErrLeaseWriteConflict
	if errors.As(err, &transitionConflict) {
		return connect.NewError(connect.CodeFailedPrecondition, transitionConflict)
	}
	var portErr ErrPortUnresolved
	if errors.As(err, &portErr) {
		return connect.NewError(connect.CodeFailedPrecondition, portErr)
	}
	return connect.NewError(connect.CodeInternal, err)
}
