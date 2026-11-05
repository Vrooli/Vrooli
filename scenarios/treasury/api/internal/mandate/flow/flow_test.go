package flow

import (
	"testing"

	"treasury/internal/mandate/flow/generated"
)

func TestMandateFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(status generated.MandateStatus, event generated.MandateEvent) (generated.MandateStatus, error) {
		next, err := TransitionMandate(MandateState{Status: status}, event)
		return next.Status, err
	})
}
