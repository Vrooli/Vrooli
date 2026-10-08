package plans

import (
	"errors"
	"fmt"
)

var (
	ErrPlanManagerUnavailable = errors.New("plan-manager unavailable")
	ErrPlanManagerTimeout     = errors.New("plan-manager timeout")
	ErrPlanManagerHTTPStatus  = errors.New("plan-manager http status")
	ErrPlanManagerNotFound    = errors.New("plan-manager not found")
	ErrPlanManagerInvalid     = errors.New("plan-manager invalid request")
	ErrPlanManagerConflict    = errors.New("plan-manager conflict")
	ErrPlanManagerServer      = errors.New("plan-manager server error")
)

func shouldUseMirrorFallback(err error) bool {
	return errors.Is(err, ErrPlanManagerUnavailable) || errors.Is(err, ErrPlanManagerTimeout)
}

func fallbackWarning(err error) string {
	if err == nil {
		return "Plan Manager unavailable; reading markdown mirror fallback"
	}
	return fmt.Sprintf("Plan Manager unavailable; reading markdown mirror fallback: %v", err)
}

func firstNonNil(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
