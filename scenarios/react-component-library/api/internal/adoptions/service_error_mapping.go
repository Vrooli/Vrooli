package adoptions

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates adoptions-domain sentinels into Connect's
// typed error model. Unknown errors map to internal so callers never
// depend on raw storage details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var notFound ErrAdoptionNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var invalid ErrInvalidAdoption
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	return connect.NewError(connect.CodeInternal, err)
}
