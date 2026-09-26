package runs

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates runs domain sentinels into Connect codes.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var nf ErrNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, nf)
	}
	return connect.NewError(connect.CodeInternal, err)
}
