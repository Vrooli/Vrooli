package artifacts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/coreset"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection/selectionv1connect"
	internalartifacts "vrooli-bridge/internal/artifacts"
)

type fakeSelectionService struct {
	receipt       *selectionv1.GetHandoffResponse
	authorization string
}

func (f fakeSelectionService) ListScenarios(context.Context, *connect.Request[selectionv1.ListScenariosRequest]) (*connect.Response[selectionv1.ListScenariosResponse], error) {
	return connect.NewResponse(&selectionv1.ListScenariosResponse{}), nil
}
func (f fakeSelectionService) GetCoreSet(context.Context, *connect.Request[selectionv1.GetCoreSetRequest]) (*connect.Response[selectionv1.GetCoreSetResponse], error) {
	return connect.NewResponse(&selectionv1.GetCoreSetResponse{}), nil
}
func (f fakeSelectionService) GetRecommendation(context.Context, *connect.Request[selectionv1.GetRecommendationRequest]) (*connect.Response[selectionv1.GetRecommendationResponse], error) {
	return connect.NewResponse(&selectionv1.GetRecommendationResponse{}), nil
}
func (f fakeSelectionService) AcceptRecommendation(context.Context, *connect.Request[selectionv1.AcceptRecommendationRequest]) (*connect.Response[selectionv1.AcceptRecommendationResponse], error) {
	return connect.NewResponse(&selectionv1.AcceptRecommendationResponse{}), nil
}
func (f fakeSelectionService) GetClosure(context.Context, *connect.Request[selectionv1.GetClosureRequest]) (*connect.Response[selectionv1.GetClosureResponse], error) {
	return connect.NewResponse(&selectionv1.GetClosureResponse{}), nil
}
func (f fakeSelectionService) GetUnion(context.Context, *connect.Request[selectionv1.GetUnionRequest]) (*connect.Response[selectionv1.GetUnionResponse], error) {
	return connect.NewResponse(&selectionv1.GetUnionResponse{}), nil
}
func (f fakeSelectionService) CreateHandoff(context.Context, *connect.Request[selectionv1.CreateHandoffRequest]) (*connect.Response[selectionv1.CreateHandoffResponse], error) {
	return connect.NewResponse(&selectionv1.CreateHandoffResponse{}), nil
}
func (f fakeSelectionService) GetHandoff(_ context.Context, req *connect.Request[selectionv1.GetHandoffRequest]) (*connect.Response[selectionv1.GetHandoffResponse], error) {
	if req.Header().Get("Authorization") != f.authorization {
		return nil, connect.NewError(connect.CodePermissionDenied, context.Canceled)
	}
	if req.Msg.GetSelectionDigest() != f.receipt.GetHandoff().GetSelectionDigest() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, context.Canceled)
	}
	return connect.NewResponse(f.receipt), nil
}

func TestOnboardingSelectionVerifierBindsOwnerReceiptAndCanonicalClosure(t *testing.T) {
	closure, err := coreset.Resolve(coreset.DependencyGraph{Components: []coreset.ComponentID{{Kind: "scenario", Name: "alpha"}}}, []coreset.ComponentID{{Kind: "scenario", Name: "alpha"}})
	require.NoError(t, err)
	receipt := &selectionv1.GetHandoffResponse{Handoff: &selectionv1.Handoff{Reference: "vrooli-onboarding://handoffs/h-1", DeploymentId: "deployment-1", Target: "node-1", MachineId: "machine-1", NodeId: "node-1", EnrollmentGeneration: 7, DesiredRevision: 12, SelectionDigest: "sha256:selection", ClosureDigest: closure.Digest, State: "pending"}, Selection: &setupv1.Selection{Scenarios: []string{"alpha"}}}
	const ownerCredential = "Bearer same-owner-token"
	path, handler := selectionconnect.NewSelectionServiceHandler(fakeSelectionService{receipt: receipt, authorization: ownerCredential})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	verifier := onboardingSelectionVerifier{resolve: func(context.Context) (string, error) { return server.URL, nil }}
	req := internalartifacts.ClosureShipmentRequest{TargetID: "node-1", Closure: closure, SelectionReceiptRef: receipt.GetHandoff().GetReference(), SelectionDigest: "sha256:selection", DeploymentID: "deployment-1", MachineID: "machine-1", NodeID: "node-1", EnrollmentGeneration: 7, DesiredRevision: 12, SelectionAuthorization: ownerCredential}
	require.NoError(t, verifier.Authorize(context.Background(), req))

	wrongOwner := req
	wrongOwner.SelectionAuthorization = "Bearer another-owner"
	require.Error(t, verifier.Authorize(context.Background(), wrongOwner))
	stale := req
	stale.DesiredRevision++
	require.Error(t, verifier.Authorize(context.Background(), stale))
	changed, err := coreset.Resolve(coreset.DependencyGraph{Components: []coreset.ComponentID{{Kind: "scenario", Name: "beta"}}}, []coreset.ComponentID{{Kind: "scenario", Name: "beta"}})
	require.NoError(t, err)
	changedMembership := req
	changedMembership.Closure = changed
	require.Error(t, verifier.Authorize(context.Background(), changedMembership))
	wrongNode := req
	wrongNode.NodeID = "node-2"
	require.Error(t, verifier.Authorize(context.Background(), wrongNode))
}
