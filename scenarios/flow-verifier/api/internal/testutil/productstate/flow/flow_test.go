package flow

import (
	"testing"

	"flow-verifier/internal/testutil/productstate/flow/generated"
)

func TestFlowFormalReplay(t *testing.T) {
	generated.RunReplay(t, TransitionProduct)
}

func TestProductObservationEffectReplay(t *testing.T) {
	generated.AssertProductTraceObservations(t, "production_finalize", []generated.ProductObservation{{
		State: generated.ProductState{
			Settlement: generated.ProductSettlementSettled,
			Wallet:     generated.ProductWalletNonnegative,
		},
		Effects: map[string]int64{"allocations": 1, "usage": 5},
	}})
}
