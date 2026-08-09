package artifacts_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"vrooli-bridge/internal/artifacts"
	"vrooli-bridge/internal/artifacts/mocks"
	testmocks "vrooli-bridge/internal/testutil/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func newService(nodes *mocks.FakeNodeReader, delivery *mocks.FakeDelivery) (artifacts.Service, *mocks.FakeRepository) {
	repo := mocks.NewFakeRepository()
	clk := testmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	return artifacts.NewService(repo, nodes, delivery, clk), repo
}

func okNodes() *mocks.FakeNodeReader {
	return &mocks.FakeNodeReader{Nodes: map[string]artifacts.TargetNode{"n1": {ID: "n1"}}}
}

func input() artifacts.DistributeInput {
	return artifacts.DistributeInput{
		Actor: "owner", NodeID: "n1", Name: "app-1.4.0-setup.exe",
		SourceRef: "blob://builds/app-1.4.0-setup.exe", DestinationPath: "/opt/app/setup.exe",
	}
}

// [REQ:BRG-P1-003] Distribution delegates the byte move to the device-sync-hub
// directed-delivery seam; bridge implements no byte transport of its own. The
// durable record references the device-sync-hub delivery.
func TestDistribute_DelegatesToDeviceSyncHub(t *testing.T) {
	nodes := okNodes()
	delivery := &mocks.FakeDelivery{Delivered: false} // in flight (PENDING)
	svc, _ := newService(nodes, delivery)

	dec, err := svc.Distribute(context.Background(), input())
	require.NoError(t, err)
	require.NotEmpty(t, dec.DistributionID)
	require.Equal(t, artifacts.StatusPending, dec.Status)
	require.NotEmpty(t, dec.DeliveryRef, "the device-sync-hub delivery ref is recorded")

	reqs := delivery.DeliveredRequests()
	require.Len(t, reqs, 1, "exactly one handoff to device-sync-hub")
	require.Equal(t, "n1", reqs[0].NodeID)
	require.Equal(t, "/opt/app/setup.exe", reqs[0].DestinationPath)
	require.Equal(t, "blob://builds/app-1.4.0-setup.exe", reqs[0].SourceRef)
}

// [REQ:BRG-P1-003] A dry-run validates and short-circuits before recording or
// delivering anything.
func TestDistribute_DryRunShortCircuits(t *testing.T) {
	delivery := &mocks.FakeDelivery{}
	svc, repo := newService(okNodes(), delivery)

	in := input()
	in.DryRun = true
	dec, err := svc.Distribute(context.Background(), in)
	require.NoError(t, err)
	require.True(t, dec.DryRun)
	require.Empty(t, dec.DistributionID)

	require.Empty(t, delivery.DeliveredRequests(), "nothing delivered on a dry-run")
	list, _ := repo.List(context.Background(), artifacts.ListFilter{})
	require.Empty(t, list, "nothing recorded on a dry-run")
}

// [REQ:BRG-P1-003] Missing required fields are rejected before any node lookup.
func TestDistribute_ValidatesRequiredFields(t *testing.T) {
	svc, _ := newService(okNodes(), &mocks.FakeDelivery{})
	for _, mut := range []func(*artifacts.DistributeInput){
		func(in *artifacts.DistributeInput) { in.NodeID = "" },
		func(in *artifacts.DistributeInput) { in.SourceRef = "" },
		func(in *artifacts.DistributeInput) { in.DestinationPath = "" },
	} {
		in := input()
		mut(&in)
		_, err := svc.Distribute(context.Background(), in)
		var invalid artifacts.ErrInvalidDistribution
		require.ErrorAs(t, err, &invalid)
	}
}

// [REQ:BRG-P1-003] A revoked node is rejected before any delivery.
func TestDistribute_RevokedNodeRejected(t *testing.T) {
	nodes := &mocks.FakeNodeReader{Nodes: map[string]artifacts.TargetNode{"n1": {ID: "n1", Revoked: true}}}
	delivery := &mocks.FakeDelivery{}
	svc, _ := newService(nodes, delivery)

	_, err := svc.Distribute(context.Background(), input())
	var revoked artifacts.ErrNodeRevoked
	require.ErrorAs(t, err, &revoked)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(artifacts.ToConnectError(err)))
	require.Empty(t, delivery.DeliveredRequests())
}

// [REQ:BRG-P1-003] When device-sync-hub rejects the handoff the distribution is
// recorded FAILED and the call fails (Unavailable).
func TestDistribute_DeliveryFailureRecordedFailed(t *testing.T) {
	delivery := &mocks.FakeDelivery{Err: errors.New("device-sync-hub unreachable")}
	svc, repo := newService(okNodes(), delivery)

	dec, err := svc.Distribute(context.Background(), input())
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(artifacts.ToConnectError(err)))

	got, gerr := repo.Get(context.Background(), dec.DistributionID)
	require.NoError(t, gerr)
	require.Equal(t, artifacts.StatusFailed, got.Status, "the failed delivery leaves a FAILED trail")
}

func TestProducedArtifactUploadRequiresRunOwnerAndRoundTrips(t *testing.T) {
	produced := mocks.NewFakeProducedRepository()
	runs := &mocks.FakeRunReader{Targets: map[string]artifacts.RunTarget{"run-1": {ID: "run-1", NodeID: "n1"}}}
	clk := testmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	svc := artifacts.NewService(mocks.NewFakeRepository(), okNodes(), &mocks.FakeDelivery{}, clk,
		artifacts.WithProducedRepository(produced), artifacts.WithRunReader(runs))

	got, err := svc.UploadRunArtifact(context.Background(), "n1", artifacts.ProducedArtifact{
		RunID: "run-1", Name: "screenshot.png", MediaType: "image/png", Data: []byte("png"),
	})
	require.NoError(t, err)
	require.Equal(t, "bridge://run/run-1/screenshot.png", got.ArtifactRef)
	require.Equal(t, int64(3), got.SizeBytes)

	read, err := svc.GetRunArtifact(context.Background(), "run-1", "screenshot.png")
	require.NoError(t, err)
	require.Equal(t, []byte("png"), read.Data)

	_, err = svc.UploadRunArtifact(context.Background(), "other-node", artifacts.ProducedArtifact{
		RunID: "run-1", Name: "forged.png", Data: []byte("png"),
	})
	var mismatch artifacts.ErrArtifactNodeMismatch
	require.ErrorAs(t, err, &mismatch)
}

func TestProducedArtifactUploadRejectsOversize(t *testing.T) {
	clk := testmocks.NewFakeClock(time.Unix(0, 0).UTC())
	svc := artifacts.NewService(mocks.NewFakeRepository(), okNodes(), &mocks.FakeDelivery{}, clk,
		artifacts.WithProducedRepository(mocks.NewFakeProducedRepository()),
		artifacts.WithRunReader(&mocks.FakeRunReader{Targets: map[string]artifacts.RunTarget{"run-1": {NodeID: "n1"}}}))
	_, err := svc.UploadRunArtifact(context.Background(), "n1", artifacts.ProducedArtifact{
		RunID: "run-1", Name: "too-large", Data: make([]byte, artifacts.MaxProducedArtifactBytes+1),
	})
	var invalid artifacts.ErrInvalidProducedArtifact
	require.ErrorAs(t, err, &invalid)
}
