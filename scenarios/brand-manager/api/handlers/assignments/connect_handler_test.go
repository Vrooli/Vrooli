package assignments_test

import (
	"context"
	"testing"

	"brand-manager/handlers/assignments"
	internalassignments "brand-manager/internal/assignments"
	mocks "brand-manager/internal/assignments/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	assignmentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments"
	assignmentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments/assignments_v1connect"
)

// newClient wires the real internal service over in-memory fakes behind the
// generated Connect handler, exercising handler + adapter + service together.
// brands holds the version-by-id map the resolver returns.
func newClient(t *testing.T, brands map[string]int) assignmentsconnect.AssignmentsServiceClient {
	t.Helper()
	repo := &mocks.FakeRepository{}
	resolver := mocks.FakeBrandResolver{Versions: brands}
	logger, _ := connectxtest.NewLogger(t)
	svc := internalassignments.NewService(repo, resolver, logger)
	path, handler := assignmentsconnect.NewAssignmentsServiceHandler(assignments.NewConnectHandler(assignments.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return assignmentsconnect.NewAssignmentsServiceClient(server.Client(), server.URL)
}

func TestConnect_AssignThenStatus(t *testing.T) {
	client := newClient(t, map[string]int{"b1": 2})
	ctx := context.Background()

	assigned, err := client.AssignBrand(ctx, connect.NewRequest(&assignmentsv1.AssignBrandRequest{
		BrandId:      "b1",
		ScenarioName: "web-console",
		Elements:     []string{"logo", "colors"},
	}))
	require.NoError(t, err)
	require.Equal(t, int32(2), assigned.Msg.Assignment.BrandVersion)
	require.NotEmpty(t, assigned.Msg.Assignment.Id)

	status, err := client.GetScenarioStatus(ctx, connect.NewRequest(&assignmentsv1.GetScenarioStatusRequest{ScenarioName: "web-console"}))
	require.NoError(t, err)
	require.True(t, status.Msg.Status.HasBrand)
	require.Equal(t, "b1", status.Msg.Status.BrandId)
	require.Equal(t, int32(2), status.Msg.Status.BrandVersion)
	require.Equal(t, []string{"logo", "colors"}, status.Msg.Status.Elements)
	require.NotNil(t, status.Msg.Status.AppliedAt)
}

func TestConnect_StatusUnassignedHasNoBrand(t *testing.T) {
	client := newClient(t, nil)
	status, err := client.GetScenarioStatus(context.Background(), connect.NewRequest(&assignmentsv1.GetScenarioStatusRequest{ScenarioName: "lonely"}))
	require.NoError(t, err)
	require.False(t, status.Msg.Status.HasBrand)
	require.Equal(t, "lonely", status.Msg.Status.Scenario)
	require.Nil(t, status.Msg.Status.AppliedAt, "no applied-at for an unbranded scenario")
}

func TestConnect_AssignRejectsMissingFields(t *testing.T) {
	client := newClient(t, map[string]int{"b1": 1})

	_, err := client.AssignBrand(context.Background(), connect.NewRequest(&assignmentsv1.AssignBrandRequest{ScenarioName: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnect_AssignRejectsUnknownBrand(t *testing.T) {
	client := newClient(t, map[string]int{"b1": 1})

	_, err := client.AssignBrand(context.Background(), connect.NewRequest(&assignmentsv1.AssignBrandRequest{
		BrandId:      "ghost",
		ScenarioName: "x",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnect_UnassignIsIdempotent(t *testing.T) {
	client := newClient(t, map[string]int{"b1": 1})
	ctx := context.Background()

	// Unassigning a scenario that was never branded still succeeds.
	_, err := client.UnassignScenario(ctx, connect.NewRequest(&assignmentsv1.UnassignScenarioRequest{ScenarioName: "ghost"}))
	require.NoError(t, err)

	_, err = client.AssignBrand(ctx, connect.NewRequest(&assignmentsv1.AssignBrandRequest{BrandId: "b1", ScenarioName: "web-console"}))
	require.NoError(t, err)
	_, err = client.UnassignScenario(ctx, connect.NewRequest(&assignmentsv1.UnassignScenarioRequest{ScenarioName: "web-console"}))
	require.NoError(t, err)

	status, err := client.GetScenarioStatus(ctx, connect.NewRequest(&assignmentsv1.GetScenarioStatusRequest{ScenarioName: "web-console"}))
	require.NoError(t, err)
	require.False(t, status.Msg.Status.HasBrand, "scenario is unbranded after unassign")
}

func TestConnect_ListFiltersByBrand(t *testing.T) {
	client := newClient(t, map[string]int{"b1": 1, "b2": 1})
	ctx := context.Background()
	_, err := client.AssignBrand(ctx, connect.NewRequest(&assignmentsv1.AssignBrandRequest{BrandId: "b1", ScenarioName: "a"}))
	require.NoError(t, err)
	_, err = client.AssignBrand(ctx, connect.NewRequest(&assignmentsv1.AssignBrandRequest{BrandId: "b2", ScenarioName: "b"}))
	require.NoError(t, err)

	resp, err := client.ListAssignments(ctx, connect.NewRequest(&assignmentsv1.ListAssignmentsRequest{BrandId: "b1"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Assignments, 1)
	require.Equal(t, "a", resp.Msg.Assignments[0].ScenarioName)
}
