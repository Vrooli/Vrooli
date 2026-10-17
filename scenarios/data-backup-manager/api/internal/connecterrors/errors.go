// Package connecterrors centralizes domain-to-Connect error translation.
package connecterrors

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError maps one domain's invalid and not-found sentinels while
// keeping unknown errors opaque behind an internal Connect status.
func ToConnectError[Invalid, NotFound error](err error) error {
	if err == nil {
		return nil
	}
	var invalid Invalid
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound NotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	return connect.NewError(connect.CodeInternal, err)
}
