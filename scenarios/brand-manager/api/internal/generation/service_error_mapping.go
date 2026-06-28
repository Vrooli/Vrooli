package generation

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error model.
// Unknown errors map to internal so callers never depend on raw provider or
// storage details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidGeneration
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrBrandNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var unavailable ErrProvidersUnavailable
	if errors.As(err, &unavailable) {
		return connect.NewError(connect.CodeUnavailable, unavailable)
	}
	return connect.NewError(connect.CodeInternal, err)
}
