package provision_test

import (
	"context"
	"errors"
	"testing"

	"vrooli-bridge/internal/provision"
	"vrooli-bridge/internal/provision/mocks"

	"github.com/vrooli/api-core/schedule"

	"github.com/stretchr/testify/require"
)

// fakeRevResolver scripts provision.RevisionResolver. resolveErr fails the target
// pipeline (Resolve); resolved/expanded map an input to an output, defaulting to
// pass-through when unset.
type fakeRevResolver struct {
	resolved   string
	resolveErr error
	expanded   string
	expandErr  error

	lastResolve string
	lastExpand  string
}

func (f *fakeRevResolver) Resolve(_ context.Context, requested string) (string, error) {
	f.lastResolve = requested
	if f.resolveErr != nil {
		return "", f.resolveErr
	}
	if f.resolved != "" {
		return f.resolved, nil
	}
	return requested, nil
}

func (f *fakeRevResolver) Expand(_ context.Context, requested string) (string, error) {
	f.lastExpand = requested
	if f.expandErr != nil {
		return "", f.expandErr
	}
	if requested == "" {
		return "", nil
	}
	if f.expanded != "" {
		return f.expanded, nil
	}
	return requested, nil
}

func newResolverService(t *testing.T, res provision.RevisionResolver) (provision.Service, *mocks.FakeRepository, *mocks.FakeCommandPusher, *mocks.FakeAuditSink) {
	t.Helper()
	repo := mocks.NewFakeRepository()
	nodes := &mocks.FakeNodeReader{Nodes: map[string]provision.TargetNode{"n1": {ID: "n1"}}}
	pres := &mocks.FakePresence{Online: map[string]bool{"n1": true}}
	audit := &mocks.FakeAuditSink{}
	pusher := &mocks.FakeCommandPusher{Delivered: 1}
	svc := provision.NewService(repo, nodes, pres, audit, pusher, schedule.System(), provision.WithRevisionResolver(res))
	return svc, repo, pusher, audit
}

const cpCommit = "1111111111111111111111111111111111111111"

// TestSync_OmittedTargetDefaultsViaResolver asserts Sync no longer requires an
// explicit target: an omitted revision resolves to the control plane's commit.
func TestSync_OmittedTargetDefaultsViaResolver(t *testing.T) {
	res := &fakeRevResolver{resolved: cpCommit}
	svc, _, pusher, _ := newResolverService(t, res)

	dec, err := svc.Sync(context.Background(), provision.SyncInput{Actor: "owner", NodeID: "n1"})
	require.NoError(t, err)
	require.Equal(t, "", res.lastResolve, "resolver received the omitted (empty) target")
	require.Equal(t, cpCommit, dec.TargetRevision)
	require.Equal(t, cpCommit, pusher.PushedCommands()[0].TargetRevision)
}

func TestSync_SentinelTargetExpandsViaResolver(t *testing.T) {
	res := &fakeRevResolver{resolved: cpCommit}
	svc, _, pusher, _ := newResolverService(t, res)

	dec, err := svc.Sync(context.Background(), provision.SyncInput{Actor: "owner", NodeID: "n1", TargetRevision: "@cp"})
	require.NoError(t, err)
	require.Equal(t, "@cp", res.lastResolve)
	require.Equal(t, cpCommit, dec.TargetRevision)
	require.Equal(t, cpCommit, pusher.PushedCommands()[0].TargetRevision)
}

// TestSync_UnpushedTargetFailsClosed asserts a preflight failure aborts Sync
// before any op, audit, or push.
func TestSync_UnpushedTargetFailsClosed(t *testing.T) {
	notPushed := errors.New("commit abc is not on remote \"origin\"; push it first")
	res := &fakeRevResolver{resolveErr: notPushed}
	svc, _, pusher, audit := newResolverService(t, res)

	_, err := svc.Sync(context.Background(), provision.SyncInput{Actor: "owner", NodeID: "n1", TargetRevision: "abc"})
	require.ErrorIs(t, err, notPushed)
	require.Empty(t, pusher.PushedCommands(), "nothing pushed when preflight fails")
	require.Empty(t, audit.Recorded(), "no audit written when the target is rejected pre-node")
}

// TestSync_ExplicitRollbackExpandedNotPreflighted asserts an explicit rollback
// goes through Expand (so @cp works) rather than Resolve (no preflight).
func TestSync_ExplicitRollbackExpandedViaResolver(t *testing.T) {
	res := &fakeRevResolver{resolved: cpCommit, expanded: "9999999999999999999999999999999999999999"}
	svc, _, pusher, _ := newResolverService(t, res)

	dec, err := svc.Sync(context.Background(), provision.SyncInput{
		Actor: "owner", NodeID: "n1", TargetRevision: "@cp", RollbackRevision: "@cp",
	})
	require.NoError(t, err)
	require.Equal(t, "@cp", res.lastExpand, "explicit rollback flows through Expand")
	require.Equal(t, "9999999999999999999999999999999999999999", dec.RollbackRevision)
	require.Equal(t, "9999999999999999999999999999999999999999", pusher.PushedCommands()[0].RollbackRevision)
}
