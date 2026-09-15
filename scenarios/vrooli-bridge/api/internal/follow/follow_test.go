package follow

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type fakeHeads struct {
	head string
	err  error
}

func (f *fakeHeads) Head(context.Context, string, string) (string, error) { return f.head, f.err }

type fakeProvisioner struct {
	calls []string
	err   error
}

func (f *fakeProvisioner) Provision(_ context.Context, nodeID, revision, actor string) (string, error) {
	f.calls = append(f.calls, nodeID+"@"+revision+" by "+actor)
	if f.err != nil {
		return "", f.err
	}
	return "op-" + revision[:4], nil
}

func newTestService(t *testing.T, heads *fakeHeads, prov *fakeProvisioner) *Service {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	return NewService(db, heads, prov, nil, func() time.Time { return time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC) })
}

const (
	headA = "aaaa000000000000000000000000000000000000"
	headB = "bbbb000000000000000000000000000000000000"
)

// Following starts at the branch's current head: a node that already carries
// work (a working-tree ship) is not reset until the branch moves.
func TestFollowStartsWithTheNextCommitAndUpdatesWhenTheBranchMoves(t *testing.T) {
	heads := &fakeHeads{head: headA}
	prov := &fakeProvisioner{}
	svc := newTestService(t, heads, prov)

	policy, err := svc.Follow(context.Background(), "node-1", "agi", "", false)
	require.NoError(t, err)
	require.Equal(t, DefaultRepoURL, policy.RepoURL)
	require.Equal(t, headA, policy.LastSeenHead)

	svc.CheckAll(context.Background())
	require.Empty(t, prov.calls, "an unchanged branch dispatches nothing")

	heads.head = headB
	svc.CheckAll(context.Background())
	require.Equal(t, []string{"node-1@" + headB + " by bridge:follow:agi"}, prov.calls)
	got, err := svc.Get(context.Background(), "node-1")
	require.NoError(t, err)
	require.Equal(t, headB, got.LastSeenHead)
	require.Equal(t, "op-bbbb", got.LastOpID)

	svc.CheckAll(context.Background())
	require.Len(t, prov.calls, 1, "a dispatched head is never sent again")
}

// A refused dispatch (node offline, helper missing) is retried on the next
// check instead of being marked seen.
func TestRefusedDispatchIsRetriedOnTheNextCheck(t *testing.T) {
	heads := &fakeHeads{head: headA}
	prov := &fakeProvisioner{}
	svc := newTestService(t, heads, prov)
	_, err := svc.Follow(context.Background(), "node-1", "agi", "", false)
	require.NoError(t, err)

	heads.head = headB
	prov.err = errors.New("node offline")
	policy, err := svc.Check(context.Background(), "node-1")
	require.NoError(t, err)
	require.Equal(t, headA, policy.LastSeenHead)
	require.Contains(t, policy.LastResult, "not dispatched")

	prov.err = nil
	policy, err = svc.Check(context.Background(), "node-1")
	require.NoError(t, err)
	require.Equal(t, headB, policy.LastSeenHead)
	require.Len(t, prov.calls, 2)
}

func TestFollowNowDispatchesTheCurrentHead(t *testing.T) {
	prov := &fakeProvisioner{}
	svc := newTestService(t, &fakeHeads{head: headA}, prov)
	policy, err := svc.Follow(context.Background(), "node-1", "agi", "", true)
	require.NoError(t, err)
	require.Equal(t, headA, policy.LastSeenHead)
	require.Len(t, prov.calls, 1)
}

func TestFollowRefusesUnsafeInput(t *testing.T) {
	svc := newTestService(t, &fakeHeads{head: headA}, &fakeProvisioner{})
	for _, branch := range []string{"-x", "a..b", "a b", "refs/heads/x;rm", ""} {
		_, err := svc.Follow(context.Background(), "node-1", branch, "", false)
		var invalid ErrInvalid
		require.ErrorAs(t, err, &invalid, "branch %q", branch)
	}
	for _, repo := range []string{"ext::sh -c id", "/tmp/repo", "--upload-pack=x", "file:///tmp/repo"} {
		_, err := svc.Follow(context.Background(), "node-1", "agi", repo, false)
		var invalid ErrInvalid
		require.ErrorAs(t, err, &invalid, "repo %q", repo)
	}
	require.ErrorIs(t, svc.Unfollow(context.Background(), "node-1"), ErrNotFollowing)
}
