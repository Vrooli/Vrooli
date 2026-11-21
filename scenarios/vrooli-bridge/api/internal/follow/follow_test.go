package follow

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// The branch head is read with the operator's global git configuration
// neutralized: an `insteadOf` rewrite to SSH broke this read on a server with
// no agent (2026-09-15, reading refs/heads/agi from the public repository).
func TestLsRemoteIgnoresTheOperatorsURLRewrite(t *testing.T) {
	origin := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", origin}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	run("init", "-q", "-b", "agi")
	require.NoError(t, os.WriteFile(filepath.Join(origin, "a.txt"), []byte("x"), 0o644))
	run("add", ".")
	run("commit", "-q", "-m", "base")
	want := run("rev-parse", "HEAD")

	url := "file://" + origin
	gitconfig := filepath.Join(t.TempDir(), ".gitconfig")
	require.NoError(t, os.WriteFile(gitconfig, []byte("[url \"git@example.invalid:\"]\n\tinsteadOf = "+url+"\n"), 0o644))
	t.Setenv("GIT_CONFIG_GLOBAL", gitconfig)

	isolated, err := GitHeads{}.lsRemote(context.Background(), url, "refs/heads/agi", true)
	require.NoError(t, err)
	require.Contains(t, isolated, want)

	_, err = GitHeads{}.lsRemote(context.Background(), url, "refs/heads/agi", false)
	require.Error(t, err, "the rewritten URL must fail, which is what the isolated read avoids")
}
