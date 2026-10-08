package adoptions

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates adoptions-domain sentinels into Connect's
// typed error model. Unknown errors map to internal so callers never
// depend on raw storage details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var notFound ErrAdoptionNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var invalid ErrInvalidAdoption
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var blocked ErrAdoptionValidationBlocked
	if errors.As(err, &blocked) {
		return connect.NewError(connect.CodeFailedPrecondition, blocked)
	}
	var tokens ErrAdoptionTokensUnsatisfied
	if errors.As(err, &tokens) {
		return connect.NewError(connect.CodeFailedPrecondition, tokens)
	}
	var readiness ErrAdoptionReadinessBlocked
	if errors.As(err, &readiness) {
		return connect.NewError(connect.CodeFailedPrecondition, readiness)
	}
	var dependencyConflict ErrBatchDependencyConflict
	if errors.As(err, &dependencyConflict) {
		return connect.NewError(connect.CodeFailedPrecondition, dependencyConflict)
	}
	return connect.NewError(connect.CodeInternal, err)
}
