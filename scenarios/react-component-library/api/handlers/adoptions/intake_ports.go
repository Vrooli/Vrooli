package adoptions

import (
	"context"

	"connectrpc.com/connect"
	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
)

// AdoptionIntakePort restricts the new service handler to the five scenario,
// recommendation, reconciliation, and provenance intake methods.
type AdoptionIntakePort interface {
	ListScenarios(context.Context, *connect.Request[adoptionsv1.ListScenariosRequest]) (*connect.Response[adoptionsv1.ListScenariosResponse], error)
	SuggestAdoptions(context.Context, *connect.Request[adoptionsv1.SuggestAdoptionsRequest]) (*connect.Response[adoptionsv1.SuggestAdoptionsResponse], error)
	ReconcileAdoptions(context.Context, *connect.Request[adoptionsv1.ReconcileAdoptionsRequest]) (*connect.Response[adoptionsv1.ReconcileAdoptionsResponse], error)
	DiscoverAdoptions(context.Context, *connect.Request[adoptionsv1.DiscoverAdoptionsRequest]) (*connect.Response[adoptionsv1.DiscoverAdoptionsResponse], error)
	ConfirmDiscovery(context.Context, *connect.Request[adoptionsv1.ConfirmDiscoveryRequest]) (*connect.Response[adoptionsv1.ConfirmDiscoveryResponse], error)
}

type adoptionIntakeHandler struct{ port AdoptionIntakePort }

func (h adoptionIntakeHandler) ListScenarios(ctx context.Context, req *connect.Request[adoptionsv1.ListScenariosRequest]) (*connect.Response[adoptionsv1.ListScenariosResponse], error) {
	return h.port.ListScenarios(ctx, req)
}

func (h adoptionIntakeHandler) SuggestAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.SuggestAdoptionsRequest]) (*connect.Response[adoptionsv1.SuggestAdoptionsResponse], error) {
	return h.port.SuggestAdoptions(ctx, req)
}

func (h adoptionIntakeHandler) ReconcileAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.ReconcileAdoptionsRequest]) (*connect.Response[adoptionsv1.ReconcileAdoptionsResponse], error) {
	return h.port.ReconcileAdoptions(ctx, req)
}

func (h adoptionIntakeHandler) DiscoverAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.DiscoverAdoptionsRequest]) (*connect.Response[adoptionsv1.DiscoverAdoptionsResponse], error) {
	return h.port.DiscoverAdoptions(ctx, req)
}

func (h adoptionIntakeHandler) ConfirmDiscovery(ctx context.Context, req *connect.Request[adoptionsv1.ConfirmDiscoveryRequest]) (*connect.Response[adoptionsv1.ConfirmDiscoveryResponse], error) {
	return h.port.ConfirmDiscovery(ctx, req)
}
