package fleet_test

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/fleet"
	"vrooli-bridge/internal/fleet/mocks"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "vrooli-bridge/internal/database"
)

// newSqliteService wires the fleet service against a real sqlite repository and
// the seam fakes — the integration shape (domain + persistence + delegation).
func newSqliteService(t *testing.T, nodes *mocks.FakeNodeLister, presence *mocks.FakePresence, prov *mocks.FakeProvisioner) (fleet.Service, fleet.Repository) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(fleet.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	repo := fleet.NewSQLiteRepository(d, clk)
	return fleet.NewService(repo, nodes, presence, prov, clk), repo
}

// [REQ:BRG-P1-001] A fleet-wide pin moves every HEALTHY node to the target
// revision and reports per-node results: each healthy node is dispatched a
// provisioning op; offline / needs-update / revoked nodes are skipped with the
// reason, and the rollout is persisted with an accurate ledger.
func TestRollFleet_MovesHealthyNodesAndReportsPerNodeResults(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{
		{ID: "healthy-1"},
		{ID: "healthy-2"},
		{ID: "offline-1"},
		{ID: "stale-1"}, // online but protocol-flagged
		{ID: "revoked-1", Revoked: true},
	}}
	presence := &mocks.FakePresence{
		Online:  map[string]bool{"healthy-1": true, "healthy-2": true, "stale-1": true},
		Flagged: map[string]bool{"stale-1": true},
	}
	prov := &mocks.FakeProvisioner{}
	svc, _ := newSqliteService(t, nodes, presence, prov)

	dec, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "owner", TargetRevision: "rev-B"})
	require.NoError(t, err)
	require.False(t, dec.DryRun)
	require.NotEmpty(t, dec.RolloutID)
	require.Equal(t, fleet.StatusPartial, dec.Status, "some dispatched, some skipped")

	// Only the two healthy nodes were actually provisioned.
	require.Equal(t, []string{"healthy-1", "healthy-2"}, prov.RequestedNodes())

	byNode := map[string]fleet.NodeResult{}
	for _, r := range dec.Results {
		byNode[r.NodeID] = r
	}
	require.Equal(t, fleet.DispositionDispatched, byNode["healthy-1"].Disposition)
	require.Equal(t, "op-healthy-1", byNode["healthy-1"].OpID)
	require.Equal(t, fleet.DispositionDispatched, byNode["healthy-2"].Disposition)
	require.Equal(t, fleet.DispositionSkippedOffline, byNode["offline-1"].Disposition)
	require.Equal(t, fleet.DispositionSkippedNeedsUpdate, byNode["stale-1"].Disposition)
	require.Equal(t, fleet.DispositionSkippedRevoked, byNode["revoked-1"].Disposition)

	// The persisted rollout matches the in-flight decision.
	rollout, results, err := svc.GetRollout(context.Background(), dec.RolloutID)
	require.NoError(t, err)
	require.Equal(t, "rev-B", rollout.TargetRevision)
	require.Equal(t, 5, rollout.TotalNodes)
	require.Equal(t, 2, rollout.Dispatched)
	require.Equal(t, 3, rollout.Skipped)
	require.Equal(t, 0, rollout.Failed)
	require.Len(t, results, 5)
}

// [REQ:BRG-P1-001] When every eligible node is dispatched, the rollout is
// DISPATCHED (no skips, no failures).
func TestRollFleet_AllHealthyIsDispatched(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{{ID: "n1"}, {ID: "n2"}}}
	presence := &mocks.FakePresence{Online: map[string]bool{"n1": true, "n2": true}}
	svc, _ := newSqliteService(t, nodes, presence, &mocks.FakeProvisioner{})

	dec, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "owner", TargetRevision: "rev-C"})
	require.NoError(t, err)
	require.Equal(t, fleet.StatusDispatched, dec.Status)
}

// [REQ:BRG-P1-001] A roll where every node is ineligible dispatches nothing and
// is FAILED (nothing rolled).
func TestRollFleet_NoEligibleNodesIsFailed(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{{ID: "n1"}, {ID: "n2"}}}
	presence := &mocks.FakePresence{Online: map[string]bool{}} // all offline
	prov := &mocks.FakeProvisioner{}
	svc, _ := newSqliteService(t, nodes, presence, prov)

	dec, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "owner", TargetRevision: "rev-D"})
	require.NoError(t, err)
	require.Equal(t, fleet.StatusFailed, dec.Status)
	require.Empty(t, prov.RequestedNodes())
}

// [REQ:BRG-P1-001] A roll narrowed to a subset only touches the named nodes.
func TestRollFleet_SubsetOnlyRollsNamedNodes(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{{ID: "n1"}, {ID: "n2"}, {ID: "n3"}}}
	presence := &mocks.FakePresence{Online: map[string]bool{"n1": true, "n2": true, "n3": true}}
	prov := &mocks.FakeProvisioner{}
	svc, _ := newSqliteService(t, nodes, presence, prov)

	dec, err := svc.Roll(context.Background(), fleet.RollInput{
		Actor: "owner", TargetRevision: "rev-E", NodeIDs: []string{"n2", "unknown"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"n2"}, prov.RequestedNodes(), "only the known subset node is rolled")
	require.Len(t, dec.Results, 1)
}
