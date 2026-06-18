package artifacts_test

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/artifacts"
	"vrooli-bridge/internal/artifacts/mocks"
	"vrooli-bridge/internal/testutil/db"
	testmocks "vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "vrooli-bridge/internal/database"
)

// newSqliteService wires the artifacts service against a REAL sqlite repository
// and a device-sync-hub directed-delivery fake — the integration shape (domain +
// persistence + delegation). device-sync-hub itself carries an environmental
// authenticator blocker, so the fake stands in for the directed-delivery seam
// (it mirrors the seam contract); the point under test is that bridge ORCHESTRATES
// and RECORDS the delivery and moves no bytes itself.
func newSqliteService(t *testing.T, nodes *mocks.FakeNodeReader, delivery *mocks.FakeDelivery) artifacts.Service {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(artifacts.Schema),
	))
	clk := testmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	return artifacts.NewService(artifacts.NewSQLiteRepository(d, clk), nodes, delivery, clk)
}

// [REQ:BRG-P1-003] An installer artifact reaches the target node via
// device-sync-hub and is available to the job: the durable distribution flips to
// DELIVERED with the device-sync-hub reference, and that record round-trips from
// persistence.
func TestDistribute_ArtifactReachesNodeViaDeviceSyncHub(t *testing.T) {
	nodes := &mocks.FakeNodeReader{Nodes: map[string]artifacts.TargetNode{"mac-1": {ID: "mac-1"}}}
	delivery := &mocks.FakeDelivery{Delivered: true} // device-sync-hub confirms receipt
	svc := newSqliteService(t, nodes, delivery)

	dec, err := svc.Distribute(context.Background(), artifacts.DistributeInput{
		Actor: "owner", NodeID: "mac-1", Name: "MyApp-1.4.0.dmg",
		SourceRef: "blob://builds/MyApp-1.4.0.dmg", DestinationPath: "/Users/runner/MyApp.dmg",
	})
	require.NoError(t, err)
	require.Equal(t, artifacts.StatusDelivered, dec.Status, "device-sync-hub reports the artifact reached the node")
	require.Equal(t, "dsh://mac-1/MyApp-1.4.0.dmg", dec.DeliveryRef)

	// The distribution round-trips from real persistence with its reference.
	got, err := svc.GetDistribution(context.Background(), dec.DistributionID)
	require.NoError(t, err)
	require.Equal(t, artifacts.StatusDelivered, got.Status)
	require.Equal(t, "/Users/runner/MyApp.dmg", got.DestinationPath)
	require.Equal(t, "dsh://mac-1/MyApp-1.4.0.dmg", got.DeliveryRef)

	// bridge handed off exactly once and moved no bytes itself.
	require.Len(t, delivery.DeliveredRequests(), 1)
}

// [REQ:BRG-P1-003] Distributions are listed newest-first and filterable by node.
func TestListDistributions_NewestFirstAndFilter(t *testing.T) {
	nodes := &mocks.FakeNodeReader{Nodes: map[string]artifacts.TargetNode{"a": {ID: "a"}, "b": {ID: "b"}}}
	svc := newSqliteService(t, nodes, &mocks.FakeDelivery{Delivered: true})
	ctx := context.Background()

	_, err := svc.Distribute(ctx, artifacts.DistributeInput{NodeID: "a", Name: "one", SourceRef: "s1", DestinationPath: "/d1"})
	require.NoError(t, err)
	_, err = svc.Distribute(ctx, artifacts.DistributeInput{NodeID: "b", Name: "two", SourceRef: "s2", DestinationPath: "/d2"})
	require.NoError(t, err)

	all, err := svc.ListDistributions(ctx, artifacts.ListFilter{})
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, "two", all[0].Name, "newest first")

	onlyA, err := svc.ListDistributions(ctx, artifacts.ListFilter{NodeID: "a"})
	require.NoError(t, err)
	require.Len(t, onlyA, 1)
	require.Equal(t, "a", onlyA[0].NodeID)
}
