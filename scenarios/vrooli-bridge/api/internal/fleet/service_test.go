package fleet_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"vrooli-bridge/internal/fleet"
	"vrooli-bridge/internal/fleet/mocks"
	testmocks "vrooli-bridge/internal/testutil/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func newFakeService(nodes *mocks.FakeNodeLister, presence *mocks.FakePresence, prov *mocks.FakeProvisioner) (fleet.Service, *mocks.FakeRepository) {
	repo := mocks.NewFakeRepository()
	clk := testmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	return fleet.NewService(repo, nodes, presence, prov, clk), repo
}

// [REQ:BRG-P1-001] A dry-run classifies every node and reports what it WOULD
// dispatch, but creates no rollout and dispatches no provisioning op.
func TestRoll_DryRunShortCircuits(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{{ID: "n1"}, {ID: "n2"}}}
	presence := &mocks.FakePresence{Online: map[string]bool{"n1": true, "n2": true}}
	prov := &mocks.FakeProvisioner{}
	svc, repo := newFakeService(nodes, presence, prov)

	dec, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "owner", TargetRevision: "rev-B", DryRun: true})
	require.NoError(t, err)
	require.True(t, dec.DryRun)
	require.Empty(t, dec.RolloutID)
	require.Len(t, dec.Results, 2)

	require.Empty(t, prov.RequestedNodes(), "nothing provisioned on a dry-run")
	list, _ := repo.List(context.Background(), fleet.ListFilter{})
	require.Empty(t, list, "no rollout persisted on a dry-run")
}

// A working-tree node (dirty provenance) is pinned to no fetchable commit, so a
// revision roll must EXCLUDE it with a needs-reprovision disposition rather than
// dispatch a sync that would fail on the node.
func TestRoll_WorkingTreeNodeExcludedNeedsReprovision(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{
		{ID: "n1"},
		{ID: "n2", WorkingTree: true},
	}}
	presence := &mocks.FakePresence{Online: map[string]bool{"n1": true, "n2": true}}
	prov := &mocks.FakeProvisioner{}
	svc, _ := newFakeService(nodes, presence, prov)

	dec, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "owner", TargetRevision: "rev-B"})
	require.NoError(t, err)

	byID := map[string]fleet.NodeResult{}
	for _, r := range dec.Results {
		byID[r.NodeID] = r
	}
	require.Equal(t, fleet.DispositionDispatched, byID["n1"].Disposition, "pinned node still dispatches")
	require.Equal(t, fleet.DispositionSkippedWorkingTree, byID["n2"].Disposition, "working-tree node excluded")
	require.Contains(t, byID["n2"].Detail, "reprovision")
	// The excluded node was never handed to the provisioner.
	require.NotContains(t, prov.RequestedNodes(), "n2")
	require.Contains(t, prov.RequestedNodes(), "n1")
}

// [REQ:BRG-P1-001] An empty target revision is rejected with InvalidArgument
// before any enumeration.
func TestRoll_EmptyTargetRejected(t *testing.T) {
	svc, _ := newFakeService(&mocks.FakeNodeLister{}, &mocks.FakePresence{}, &mocks.FakeProvisioner{})
	_, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "owner"})
	require.Error(t, err)
	var invalid fleet.ErrInvalidRoll
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(fleet.ToConnectError(err)))
}

// [REQ:BRG-P1-001] A node whose provisioning dispatch fails is recorded FAILED
// with the error detail; the rollout is PARTIAL when others dispatched.
func TestRoll_ProvisionFailureRecordedPerNode(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{{ID: "n1"}, {ID: "n2"}}}
	presence := &mocks.FakePresence{Online: map[string]bool{"n1": true, "n2": true}}
	prov := &mocks.FakeProvisioner{FailNodes: map[string]error{"n2": errors.New("delivery failed")}}
	svc, _ := newFakeService(nodes, presence, prov)

	dec, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "owner", TargetRevision: "rev-B"})
	require.NoError(t, err)
	require.Equal(t, fleet.StatusPartial, dec.Status)

	byNode := map[string]fleet.NodeResult{}
	for _, r := range dec.Results {
		byNode[r.NodeID] = r
	}
	require.Equal(t, fleet.DispositionDispatched, byNode["n1"].Disposition)
	require.Equal(t, fleet.DispositionFailed, byNode["n2"].Disposition)
	require.Contains(t, byNode["n2"].Detail, "delivery failed")
}

// [REQ:BRG-P1-001] GetRollout on an unknown id is NotFound.
func TestGetRollout_UnknownIsNotFound(t *testing.T) {
	svc, _ := newFakeService(&mocks.FakeNodeLister{}, &mocks.FakePresence{}, &mocks.FakeProvisioner{})
	_, _, err := svc.GetRollout(context.Background(), "nope")
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(fleet.ToConnectError(err)))
}

// [REQ:BRG-P1-001] ListRollouts returns rollouts newest-first.
func TestListRollouts_NewestFirst(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{{ID: "n1"}}}
	presence := &mocks.FakePresence{Online: map[string]bool{"n1": true}}
	svc, _ := newFakeService(nodes, presence, &mocks.FakeProvisioner{})

	_, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "o", TargetRevision: "rev-1"})
	require.NoError(t, err)
	_, err = svc.Roll(context.Background(), fleet.RollInput{Actor: "o", TargetRevision: "rev-2"})
	require.NoError(t, err)

	list, err := svc.ListRollouts(context.Background(), fleet.ListFilter{})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "rev-2", list[0].TargetRevision, "newest first")
}
