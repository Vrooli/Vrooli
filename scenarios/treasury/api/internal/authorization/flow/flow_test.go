package flow

import (
	"testing"

	"treasury/internal/authorization/flow/generated"
)

func TestAuthorizationFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(status generated.AuthorizationStatus, event generated.AuthorizationEvent) (generated.AuthorizationStatus, error) {
		next, err := TransitionAuthorization(AuthorizationState{Status: status}, event)
		return next.Status, err
	})
}
