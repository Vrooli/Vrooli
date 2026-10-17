package discovery

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error model.
// Unknown errors map to internal so callers never depend on raw filesystem or
// cross-domain details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidDiscovery
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var scenarioNotFound ErrScenarioNotFound
	if errors.As(err, &scenarioNotFound) {
		return connect.NewError(connect.CodeNotFound, scenarioNotFound)
	}
	var noBranding ErrNoBrandingFound
	if errors.As(err, &noBranding) {
		return connect.NewError(connect.CodeFailedPrecondition, noBranding)
	}
	return connect.NewError(connect.CodeInternal, err)
}
