package flow

import (
	"testing"

	"plan-manager/internal/authoring/flow/generated"
)

func TestAuthoringFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(s generated.AuthoringStatus, e generated.AuthoringEvent) (generated.AuthoringStatus, error) {
		next, err := TransitionAuthoring(AuthoringState{Status: s}, e)
		return next.Status, err
	})
}
