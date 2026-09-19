package graph

import (
	"errors"

	"connectrpc.com/connect"
)

// ErrorToConnectCode translates graph typed sentinels into Connect codes.
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return 0
	}
	var (
		notFound    ErrSnapshotNotFound
		invReq      ErrInvalidExtractRequest
		integration IntegrationError
	)
	switch {
	case errors.As(err, &notFound):
		return connect.CodeNotFound
	case errors.As(err, &invReq):
		return connect.CodeInvalidArgument
	case errors.As(err, &integration):
		return connect.CodeUnavailable
	default:
		return connect.CodeInternal
	}
}
