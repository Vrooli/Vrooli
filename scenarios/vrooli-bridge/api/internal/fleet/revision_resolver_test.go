package fleet_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"vrooli-bridge/internal/fleet"
	"vrooli-bridge/internal/fleet/mocks"
	testmocks "vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

type fakeRevResolver struct {
	resolved string
	err      error
	calls    int
}

func (f *fakeRevResolver) Resolve(_ context.Context, requested string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	if f.resolved != "" {
		return f.resolved, nil
	}
	return requested, nil
}

func newResolverService(nodes *mocks.FakeNodeLister, presence *mocks.FakePresence, prov *mocks.FakeProvisioner, res fleet.RevisionResolver) (fleet.Service, *mocks.FakeRepository) {
	repo := mocks.NewFakeRepository()
	clk := testmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	return fleet.NewService(repo, nodes, presence, prov, clk, fleet.WithRevisionResolver(res)), repo
}

const cpCommit = "1111111111111111111111111111111111111111"

// TestRoll_SentinelResolvedOnceAndPinsWholeFleet is the fleet half of the
// phase-6 acceptance: "@cp" resolves ONCE (one preflight) and every dispatched
// node — plus the rollout record — carries the same resolved commit, so a roll is
// atomic across the fleet.
func TestRoll_SentinelResolvedOnceAndPinsWholeFleet(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{{ID: "n1"}, {ID: "n2"}, {ID: "n3"}}}
	presence := &mocks.FakePresence{Online: map[string]bool{"n1": true, "n2": true, "n3": true}}
	prov := &mocks.FakeProvisioner{}
	res := &fakeRevResolver{resolved: cpCommit}
	svc, repo := newResolverService(nodes, presence, prov, res)

	dec, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "owner", TargetRevision: "@cp"})
	require.NoError(t, err)
	require.Equal(t, 1, res.calls, "the roll resolves the target exactly once, not per node")

	// Every dispatched node was pinned to the SAME resolved commit.
	require.Len(t, prov.Requested, 3)
	for _, r := range prov.Requested {
		require.Equal(t, cpCommit, r.TargetRevision, "node %s not pinned to the resolved commit", r.NodeID)
	}

	// The rollout record shows the resolved commit, not the raw "@cp".
	rollout, _, err := svc.GetRollout(context.Background(), dec.RolloutID)
	require.NoError(t, err)
	require.Equal(t, cpCommit, rollout.TargetRevision)
	_ = repo
}

// TestRoll_UnpushedTargetFailsWholeRoll asserts a preflight failure aborts the
// roll before any node is dispatched or any rollout is persisted.
func TestRoll_UnpushedTargetFailsWholeRoll(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []fleet.NodeRef{{ID: "n1"}, {ID: "n2"}}}
	presence := &mocks.FakePresence{Online: map[string]bool{"n1": true, "n2": true}}
	prov := &mocks.FakeProvisioner{}
	notPushed := errors.New("commit abc is not on remote \"origin\"; push it first")
	svc, repo := newResolverService(nodes, presence, prov, &fakeRevResolver{err: notPushed})

	_, err := svc.Roll(context.Background(), fleet.RollInput{Actor: "owner", TargetRevision: "abc"})
	require.ErrorIs(t, err, notPushed)
	require.Empty(t, prov.Requested, "no node dispatched when the roll target fails preflight")
	list, _ := repo.List(context.Background(), fleet.ListFilter{})
	require.Empty(t, list, "no rollout persisted when preflight fails")
}
