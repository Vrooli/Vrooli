package themes

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError maps themes-domain sentinels into Connect codes.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var notFound ErrThemeNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var invalid ErrInvalidDesignMD
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var missing ErrScenarioDesignMDMissing
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, missing)
	}
	return connect.NewError(connect.CodeInternal, err)
}
