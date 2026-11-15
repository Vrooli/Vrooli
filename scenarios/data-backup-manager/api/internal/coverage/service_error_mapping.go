package coverage

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates coverage errors into Connect's typed error model.
// Most coverage failures are composition errors from a downstream seam. The
// explicit legacy sensitive bypass and an empty sensitive review are typed
// preconditions; per-item accept failures ride back in AcceptResult.Failed.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrSensitiveReviewRequired) || errors.Is(err, ErrNoSensitiveSuggestions) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
