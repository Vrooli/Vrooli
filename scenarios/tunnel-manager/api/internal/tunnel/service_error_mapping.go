package tunnel

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error model.
// Unknown errors intentionally map to internal so callers never depend on raw
// storage or upstream-scrape details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var noMetrics ErrNoMetrics
	if errors.As(err, &noMetrics) {
		return connect.NewError(connect.CodeNotFound, noMetrics)
	}
	return connect.NewError(connect.CodeInternal, err)
}
