package flow

import (
	"testing"

	"treasury/internal/settlement/flow/generated"
)

func TestSettlementFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(status generated.SettlementStatus, event generated.SettlementEvent) (generated.SettlementStatus, error) {
		next, err := TransitionSettlement(SettlementState{Status: status}, event)
		return next.Status, err
	})
}
