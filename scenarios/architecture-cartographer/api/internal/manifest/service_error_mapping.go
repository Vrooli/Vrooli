package manifest

import (
	"errors"

	"connectrpc.com/connect"
)

// ErrorToConnectCode translates manifest typed sentinels into Connect codes.
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return 0
	}
	var (
		notFound ErrManifestNotFound
		invalid  ErrInvalidManifest
	)
	switch {
	case errors.As(err, &notFound):
		return connect.CodeNotFound
	case errors.As(err, &invalid):
		return connect.CodeInvalidArgument
	default:
		return connect.CodeInternal
	}
}
