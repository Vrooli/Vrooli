package flow_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"treasury/internal/approval/flow"
	"treasury/internal/approval/flow/generated"
)

func TestApprovalFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(status generated.ApprovalStatus, event generated.ApprovalEvent) (generated.ApprovalStatus, error) {
		next, err := flow.TransitionApproval(flow.ApprovalState{Status: status}, event)
		return next.Status, err
	})
}

func TestGeneratedMatrixCoversEveryPair(t *testing.T) {
	for _, status := range generated.AllApprovalStatuses() {
		for _, event := range generated.AllApprovalEvents() {
			next, err := flow.TransitionApproval(flow.ApprovalState{Status: status}, event)
			require.Equal(t, generated.ApprovalNextStatus(status, event), next.Status)
			require.Equal(t, !generated.ApprovalIsValidEvent(status, event), err != nil)
		}
	}
}
