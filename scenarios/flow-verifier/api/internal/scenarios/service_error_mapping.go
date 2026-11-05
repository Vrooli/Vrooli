package scenarios

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates scenarios domain sentinels into Connect codes.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrScenarioNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
