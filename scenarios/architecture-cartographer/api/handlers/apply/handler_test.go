package apply_test

import (
	"context"
	"errors"
	"testing"

	applyh "architecture-cartographer/handlers/apply"
	"architecture-cartographer/internal/apply"
	"architecture-cartographer/internal/apply/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply/apply_v1connect"
)

func TestHandler_PlanApply_HappyPath(t *testing.T) {
	svc := &mocks.FakeService{
		Plan: apply.Plan{
			ID: "plan-1", Scenario: "demo", Domain: "graph",
			Operations: []apply.Operation{{ID: "op-1", Kind: apply.OperationKindMoveFile, FromPath: "a.go", ToPath: "b.go"}},
		},
	}
	h := applyh.NewHandler(svc)
	resp, err := h.PlanApply(context.Background(), connect.NewRequest(&applyv1.PlanApplyRequest{
		Scenario: "demo", Domain: "graph",
	}))
	require.NoError(t, err)
	require.Equal(t, "plan-1", resp.Msg.GetPlan().GetId())
	require.Len(t, resp.Msg.GetPlan().GetOperations(), 1)
	require.Equal(t, applyv1.OperationKind_OPERATION_KIND_MOVE_FILE, resp.Msg.GetPlan().GetOperations()[0].GetKind())
}

func TestHandler_PlanApply_RejectsMissingFields(t *testing.T) {
	h := applyh.NewHandler(&mocks.FakeService{})
	for _, tc := range []struct {
		name string
		req  *applyv1.PlanApplyRequest
	}{
		{"missing scenario", &applyv1.PlanApplyRequest{Domain: "graph"}},
		{"missing domain", &applyv1.PlanApplyRequest{Scenario: "demo"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.PlanApply(context.Background(), connect.NewRequest(tc.req))
			require.Error(t, err)
			var ce *connect.Error
			require.ErrorAs(t, err, &ce)
			require.Equal(t, connect.CodeInvalidArgument, ce.Code())
		})
	}
}

func TestHandler_RunApply_ReturnsUnimplementedFromService(t *testing.T) {
	// FakeService.RunApply returns ErrApplyUnimplemented by default.
	svc := &mocks.FakeService{}
	h := applyh.NewHandler(svc)
	_, err := h.RunApply(context.Background(), connect.NewRequest(&applyv1.RunApplyRequest{PlanId: "plan-1"}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeUnimplemented, ce.Code(), "RunApply must surface CodeUnimplemented in v0.1")
	var typed apply.ErrApplyUnimplemented
	require.True(t, errors.As(err, &typed), "typed sentinel must be preserved through wrap")
}

func TestHandler_ListApplyHistory_Empty(t *testing.T) {
	h := applyh.NewHandler(&mocks.FakeService{})
	resp, err := h.ListApplyHistory(context.Background(), connect.NewRequest(&applyv1.ListApplyHistoryRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.GetRuns())
}

func TestHandler_GetBuildBaseline_Empty(t *testing.T) {
	h := applyh.NewHandler(&mocks.FakeService{})
	resp, err := h.GetBuildBaseline(context.Background(), connect.NewRequest(&applyv1.GetBuildBaselineRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetBaseline())
}

func TestHandler_InterfaceSatisfied(t *testing.T) {
	var _ apply_v1connect.ApplyServiceHandler = (*applyh.Handler)(nil)
}
