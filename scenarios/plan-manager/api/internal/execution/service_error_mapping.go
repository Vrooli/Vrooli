package execution

import (
	"errors"

	"connectrpc.com/connect"

	planmodel "plan-manager/internal/planmodel"
)

// ToConnectError translates execution/plans domain sentinels into Connect's
// typed error model. Unknown errors map to internal so callers never depend on
// raw storage or implementation details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidExecution
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var validationRequired ErrValidationRequired
	if errors.As(err, &validationRequired) {
		return connect.NewError(connect.CodeFailedPrecondition, validationRequired)
	}
	var activeConflict ErrActiveExecutionConflict
	if errors.As(err, &activeConflict) {
		return connect.NewError(connect.CodeFailedPrecondition, activeConflict)
	}
	var execNotFound ErrExecutionNotFound
	if errors.As(err, &execNotFound) {
		return connect.NewError(connect.CodeNotFound, execNotFound)
	}
	// Plans-domain sentinels surface through the PlanStore seam (Start resolves a
	// plan; TransitionPhase mutates one) and must keep their codes.
	var planNotFound planmodel.ErrPlanNotFound
	if errors.As(err, &planNotFound) {
		return connect.NewError(connect.CodeNotFound, planNotFound)
	}
	var phaseNotFound planmodel.ErrPhaseNotFound
	if errors.As(err, &phaseNotFound) {
		return connect.NewError(connect.CodeNotFound, phaseNotFound)
	}
	var invalidPlan planmodel.ErrInvalidPlan
	if errors.As(err, &invalidPlan) {
		return connect.NewError(connect.CodeInvalidArgument, invalidPlan)
	}
	return connect.NewError(connect.CodeInternal, err)
}
