package flow

import (
	"testing"

	"ai-gateway/internal/routing/flow/generated"
)

func TestMediaReceiptFlowReplay(t *testing.T) {
	generated.RunReplay(t, func(state generated.MediaStatus, event generated.MediaEvent) (generated.MediaStatus, error) {
		receipt, err := Representation(state)
		if err != nil {
			return state, err
		}
		next, err := TransitionReceipt(receipt.Status, receipt.ErrorCode, event)
		if err != nil {
			return state, err
		}
		return State(next.Status, next.ErrorCode)
	})
}
