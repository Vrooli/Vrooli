package connecterrors

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError preserves existing transport errors and maps domain errors
// through the owning domain's code policy.
func ToConnectError(err error, codeFor func(error) connect.Code) error {
	if err == nil {
		return nil
	}
	if connectErr := new(connect.Error); errors.As(err, &connectErr) {
		return err
	}
	return connect.NewError(codeFor(err), err)
}
