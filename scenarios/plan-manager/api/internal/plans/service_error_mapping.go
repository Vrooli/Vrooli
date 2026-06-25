package plans

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates plans domain sentinels into Connect's typed error
// model. Unknown errors map to internal so callers never depend on raw storage
// or implementation details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidPlan
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrPlanNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var phaseNotFound ErrPhaseNotFound
	if errors.As(err, &phaseNotFound) {
		return connect.NewError(connect.CodeNotFound, phaseNotFound)
	}
	var tmplNotFound ErrTemplateNotFound
	if errors.As(err, &tmplNotFound) {
		return connect.NewError(connect.CodeNotFound, tmplNotFound)
	}
	return connect.NewError(connect.CodeInternal, err)
}
