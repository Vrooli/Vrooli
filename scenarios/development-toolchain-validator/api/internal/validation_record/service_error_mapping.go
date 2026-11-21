package validation_record

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error
// model.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidRecord
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrRecordNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	return connect.NewError(connect.CodeInternal, err)
}
