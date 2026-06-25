package validation

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"

	internalplans "plan-manager/internal/plans"
)

// ErrPhaseNotFound is returned when a phase id is not on the named plan.
type ErrPhaseNotFound struct {
	PlanID  string
	PhaseID string
}

func (e ErrPhaseNotFound) Error() string {
	return fmt.Sprintf("phase %q not found on plan %q", e.PhaseID, e.PlanID)
}

// ToConnectError translates validation/plans sentinels into Connect's typed
// error model. Unknown errors map to internal.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var phaseNotFound ErrPhaseNotFound
	if errors.As(err, &phaseNotFound) {
		return connect.NewError(connect.CodeNotFound, phaseNotFound)
	}
	var planNotFound internalplans.ErrPlanNotFound
	if errors.As(err, &planNotFound) {
		return connect.NewError(connect.CodeNotFound, planNotFound)
	}
	var invalid internalplans.ErrInvalidPlan
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	return connect.NewError(connect.CodeInternal, err)
}
