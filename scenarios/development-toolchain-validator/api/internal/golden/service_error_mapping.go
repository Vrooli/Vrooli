package golden

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error
// model. Unknown errors intentionally map to internal so callers never
// depend on raw storage or implementation details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidGolden
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrGoldenNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var exists ErrGoldenAlreadyExists
	if errors.As(err, &exists) {
		return connect.NewError(connect.CodeAlreadyExists, exists)
	}
	var regen ErrRegenerateFailed
	if errors.As(err, &regen) {
		return connect.NewError(connect.CodeInternal, regen)
	}
	return connect.NewError(connect.CodeInternal, err)
}
