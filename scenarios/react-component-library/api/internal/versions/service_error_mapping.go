package versions

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError maps the versions domain's typed sentinels to Connect
// codes. Unknown errors return nil so the caller can fall back to
// CodeInternal with the raw message.
func ToConnectError(err error) *connect.Error {
	if err == nil {
		return nil
	}
	var (
		notFound ErrVersionNotFound
		invalid  ErrInvalidVersion
		exists   ErrVersionExists
	)
	switch {
	case errors.As(err, &notFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.As(err, &exists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	return nil
}
