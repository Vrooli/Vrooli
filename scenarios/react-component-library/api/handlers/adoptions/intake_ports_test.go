package adoptions

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
)

type intakeTraceContextKey struct{}

// intakePortStub embeds the interface so each test only needs to implement the
// RPC it exercises. Unexpected calls still fail immediately.
type intakePortStub struct {
	AdoptionIntakePort
	list      func(context.Context, *connect.Request[adoptionsv1.ListScenariosRequest]) (*connect.Response[adoptionsv1.ListScenariosResponse], error)
	reconcile func(context.Context, *connect.Request[adoptionsv1.ReconcileAdoptionsRequest]) (*connect.Response[adoptionsv1.ReconcileAdoptionsResponse], error)
	discover  func(context.Context, *connect.Request[adoptionsv1.DiscoverAdoptionsRequest]) (*connect.Response[adoptionsv1.DiscoverAdoptionsResponse], error)
	confirm   func(context.Context, *connect.Request[adoptionsv1.ConfirmDiscoveryRequest]) (*connect.Response[adoptionsv1.ConfirmDiscoveryResponse], error)
}

func (s intakePortStub) ListScenarios(ctx context.Context, req *connect.Request[adoptionsv1.ListScenariosRequest]) (*connect.Response[adoptionsv1.ListScenariosResponse], error) {
	return s.list(ctx, req)
}

func (s intakePortStub) ReconcileAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.ReconcileAdoptionsRequest]) (*connect.Response[adoptionsv1.ReconcileAdoptionsResponse], error) {
	return s.reconcile(ctx, req)
}

func (s intakePortStub) DiscoverAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.DiscoverAdoptionsRequest]) (*connect.Response[adoptionsv1.DiscoverAdoptionsResponse], error) {
	return s.discover(ctx, req)
}

func (s intakePortStub) ConfirmDiscovery(ctx context.Context, req *connect.Request[adoptionsv1.ConfirmDiscoveryRequest]) (*connect.Response[adoptionsv1.ConfirmDiscoveryResponse], error) {
	return s.confirm(ctx, req)
}

func TestAdoptionIntakeHandlerListScenariosForwardsRequestAndResult(t *testing.T) {
	ctx := context.WithValue(t.Context(), intakeTraceContextKey{}, "trace")
	req := connect.NewRequest(&adoptionsv1.ListScenariosRequest{})
	response := connect.NewResponse(&adoptionsv1.ListScenariosResponse{})
	h := adoptionIntakeHandler{port: intakePortStub{list: func(gotCtx context.Context, gotReq *connect.Request[adoptionsv1.ListScenariosRequest]) (*connect.Response[adoptionsv1.ListScenariosResponse], error) {
		require.Same(t, ctx, gotCtx)
		require.Same(t, req, gotReq)
		return response, nil
	}}}

	got, err := h.ListScenarios(ctx, req)
	require.NoError(t, err)
	require.Same(t, response, got)
}

func TestAdoptionIntakeHandlerReconcileAdoptionsForwardsRequestAndError(t *testing.T) {
	ctx := context.WithValue(t.Context(), intakeTraceContextKey{}, "trace")
	req := connect.NewRequest(&adoptionsv1.ReconcileAdoptionsRequest{Apply: true})
	wantErr := errors.New("reconciliation failed")
	h := adoptionIntakeHandler{port: intakePortStub{reconcile: func(gotCtx context.Context, gotReq *connect.Request[adoptionsv1.ReconcileAdoptionsRequest]) (*connect.Response[adoptionsv1.ReconcileAdoptionsResponse], error) {
		require.Same(t, ctx, gotCtx)
		require.Same(t, req, gotReq)
		return nil, wantErr
	}}}

	got, err := h.ReconcileAdoptions(ctx, req)
	require.Nil(t, got)
	require.Same(t, wantErr, err)
}

func TestAdoptionIntakeHandlerDiscoverAdoptionsForwardsRequestAndResult(t *testing.T) {
	ctx := context.WithValue(t.Context(), intakeTraceContextKey{}, "trace")
	req := connect.NewRequest(&adoptionsv1.DiscoverAdoptionsRequest{Scenario: "demo"})
	response := connect.NewResponse(&adoptionsv1.DiscoverAdoptionsResponse{Scanned: 2})
	h := adoptionIntakeHandler{port: intakePortStub{discover: func(gotCtx context.Context, gotReq *connect.Request[adoptionsv1.DiscoverAdoptionsRequest]) (*connect.Response[adoptionsv1.DiscoverAdoptionsResponse], error) {
		require.Same(t, ctx, gotCtx)
		require.Same(t, req, gotReq)
		return response, nil
	}}}

	got, err := h.DiscoverAdoptions(ctx, req)
	require.NoError(t, err)
	require.Same(t, response, got)
}

func TestAdoptionIntakeHandlerConfirmDiscoveryForwardsRequestAndError(t *testing.T) {
	ctx := context.WithValue(t.Context(), intakeTraceContextKey{}, "trace")
	req := connect.NewRequest(&adoptionsv1.ConfirmDiscoveryRequest{ComponentId: "button"})
	wantErr := errors.New("confirmation failed")
	h := adoptionIntakeHandler{port: intakePortStub{confirm: func(gotCtx context.Context, gotReq *connect.Request[adoptionsv1.ConfirmDiscoveryRequest]) (*connect.Response[adoptionsv1.ConfirmDiscoveryResponse], error) {
		require.Same(t, ctx, gotCtx)
		require.Same(t, req, gotReq)
		return nil, wantErr
	}}}

	got, err := h.ConfirmDiscovery(ctx, req)
	require.Nil(t, got)
	require.Same(t, wantErr, err)
}
