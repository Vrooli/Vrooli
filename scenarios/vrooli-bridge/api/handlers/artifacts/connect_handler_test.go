package artifacts

import (
	"context"
	"testing"
	"time"

	internalartifacts "vrooli-bridge/internal/artifacts"
	amocks "vrooli-bridge/internal/artifacts/mocks"
	"vrooli-bridge/internal/auth"
	testmocks "vrooli-bridge/internal/testutil/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
)

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

func newHarness(t *testing.T) *connectHandler {
	t.Helper()
	nodes := &amocks.FakeNodeReader{Nodes: map[string]internalartifacts.TargetNode{"n1": {ID: "n1"}}}
	delivery := &amocks.FakeDelivery{Delivered: true}
	clk := testmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	svc := internalartifacts.NewService(amocks.NewFakeRepository(), nodes, delivery, clk)
	return NewConnectHandler(Deps{Service: svc})
}

// [REQ:BRG-P1-003] The artifacts verbs are owner-gated: no identity →
// Unauthenticated.
func TestArtifactsHandler_RequiresOwner(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	_, err := h.DistributeArtifact(ctx, connect.NewRequest(&artifactsv1.DistributeArtifactRequest{NodeId: "n1", SourceRef: "s", DestinationPath: "/d"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.GetDistribution(ctx, connect.NewRequest(&artifactsv1.GetDistributionRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.ListDistributions(ctx, connect.NewRequest(&artifactsv1.ListDistributionsRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P1-003] DistributeArtifact records a distribution and it is
// retrievable by id; X-Dry-Run short-circuits.
func TestArtifactsHandler_DistributeAndGet(t *testing.T) {
	h := newHarness(t)

	resp, err := h.DistributeArtifact(ownerCtx(), connect.NewRequest(&artifactsv1.DistributeArtifactRequest{
		NodeId: "n1", Name: "setup.exe", SourceRef: "blob://s", DestinationPath: "/opt/s",
	}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.DistributionId)
	require.Equal(t, artifactsv1.DeliveryStatus_DELIVERY_STATUS_DELIVERED, resp.Msg.Status)

	got, err := h.GetDistribution(ownerCtx(), connect.NewRequest(&artifactsv1.GetDistributionRequest{Id: resp.Msg.DistributionId}))
	require.NoError(t, err)
	require.Equal(t, "/opt/s", got.Msg.Distribution.DestinationPath)

	dry := connect.NewRequest(&artifactsv1.DistributeArtifactRequest{NodeId: "n1", SourceRef: "blob://s", DestinationPath: "/opt/s"})
	dry.Header().Set(dryRunHeader, "true")
	dryResp, err := h.DistributeArtifact(ownerCtx(), dry)
	require.NoError(t, err)
	require.True(t, dryResp.Msg.DryRun)
	require.Empty(t, dryResp.Msg.DistributionId)
}
