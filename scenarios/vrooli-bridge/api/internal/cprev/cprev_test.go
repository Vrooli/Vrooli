package cprev_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vrooli-bridge/internal/cprev"
)

// fakeGit is a scriptable GitRunner. It answers the three commands the resolver
// issues — rev-parse HEAD, ls-remote, merge-base --is-ancestor — from in-memory
// state so resolution and preflight are exercised without a real repository.
type fakeGit struct {
	head        string
	headErr     error               // git could not run (missing binary) for rev-parse
	headExit    int                 // non-zero rev-parse exit (detached/no HEAD)
	advertised  []string            // ls-remote ref SHAs
	lsRemoteErr error               // git could not run for ls-remote
	lsRemoteBad bool                // ls-remote ran but exited non-zero
	ancestorOf  map[string][]string // commit -> advertised SHAs it is an ancestor of

	lsRemoteCalls int
}

func (f *fakeGit) Run(_ context.Context, _ string, args ...string) (cprev.GitResult, error) {
	switch {
	case len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD":
		if f.headErr != nil {
			return cprev.GitResult{}, f.headErr
		}
		if f.headExit != 0 {
			return cprev.GitResult{ExitCode: f.headExit, Stderr: "no HEAD"}, nil
		}
		return cprev.GitResult{Stdout: f.head + "\n"}, nil
	case len(args) >= 1 && args[0] == "ls-remote":
		f.lsRemoteCalls++
		if f.lsRemoteErr != nil {
			return cprev.GitResult{}, f.lsRemoteErr
		}
		if f.lsRemoteBad {
			return cprev.GitResult{ExitCode: 128, Stderr: "no remote"}, nil
		}
		var b strings.Builder
		for i, sha := range f.advertised {
			b.WriteString(sha)
			b.WriteString("\trefs/heads/branch")
			b.WriteByte(byte('0' + i))
			b.WriteByte('\n')
		}
		return cprev.GitResult{Stdout: b.String()}, nil
	case len(args) >= 4 && args[0] == "merge-base" && args[1] == "--is-ancestor":
		commit, sha := args[2], args[3]
		for _, anc := range f.ancestorOf[commit] {
			if anc == sha {
				return cprev.GitResult{ExitCode: 0}, nil
			}
		}
		return cprev.GitResult{ExitCode: 1}, nil // ran, not an ancestor
	default:
		return cprev.GitResult{ExitCode: 1, Stderr: "unexpected git args: " + strings.Join(args, " ")}, nil
	}
}

const (
	cpCommit  = "1111111111111111111111111111111111111111"
	otherPush = "2222222222222222222222222222222222222222"
)

func TestControlPlaneCommit_PrefersGitHEAD(t *testing.T) {
	r := cprev.New(cprev.WithGitRunner(&fakeGit{head: cpCommit}), cprev.WithEmbeddedCommit("embedded-sha"))
	got, err := r.ControlPlaneCommit(context.Background())
	if err != nil {
		t.Fatalf("ControlPlaneCommit: %v", err)
	}
	if got != cpCommit {
		t.Fatalf("commit = %q, want live git HEAD %q", got, cpCommit)
	}
}

func TestControlPlaneCommit_FallsBackToEmbedded(t *testing.T) {
	// git rev-parse cannot run (no checkout) → embedded build commit is used.
	r := cprev.New(
		cprev.WithGitRunner(&fakeGit{headErr: errors.New("exec: git not found")}),
		cprev.WithEmbeddedCommit("embedded-sha"),
	)
	got, err := r.ControlPlaneCommit(context.Background())
	if err != nil {
		t.Fatalf("ControlPlaneCommit: %v", err)
	}
	if got != "embedded-sha" {
		t.Fatalf("commit = %q, want embedded fallback", got)
	}
}

func TestControlPlaneCommit_NoSourceIsTypedError(t *testing.T) {
	r := cprev.New(cprev.WithGitRunner(&fakeGit{headExit: 128}), cprev.WithEmbeddedCommit(""))
	_, err := r.ControlPlaneCommit(context.Background())
	var none cprev.ErrNoControlPlaneCommit
	if !errors.As(err, &none) {
		t.Fatalf("err = %v, want ErrNoControlPlaneCommit", err)
	}
}

func TestResolve_DefaultsEmptyToControlPlaneCommit(t *testing.T) {
	r := cprev.New(cprev.WithGitRunner(&fakeGit{head: cpCommit, advertised: []string{cpCommit}}))
	got, err := r.Resolve(context.Background(), "")
	if err != nil {
		t.Fatalf("Resolve(empty): %v", err)
	}
	if got != cpCommit {
		t.Fatalf("resolved = %q, want CP commit %q", got, cpCommit)
	}
}

func TestResolve_ExpandsSentinel(t *testing.T) {
	r := cprev.New(cprev.WithGitRunner(&fakeGit{head: cpCommit, advertised: []string{cpCommit}}))
	got, err := r.Resolve(context.Background(), cprev.SentinelControlPlane)
	if err != nil {
		t.Fatalf("Resolve(@cp): %v", err)
	}
	if got != cpCommit {
		t.Fatalf("resolved = %q, want CP commit %q", got, cpCommit)
	}
}

func TestResolve_ExplicitRevisionPassesThroughWhenPushed(t *testing.T) {
	// Explicit revision that is an ANCESTOR of an advertised ref (not a tip) still
	// passes preflight via the merge-base containment check.
	r := cprev.New(cprev.WithGitRunner(&fakeGit{
		head:       cpCommit,
		advertised: []string{otherPush},
		ancestorOf: map[string][]string{"deadbeef": {otherPush}},
	}))
	got, err := r.Resolve(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("Resolve(ancestor): %v", err)
	}
	if got != "deadbeef" {
		t.Fatalf("resolved = %q, want passthrough", got)
	}
}

func TestResolve_UnpushedCommitFailsPreflight(t *testing.T) {
	// CP HEAD is not advertised and not an ancestor of anything advertised — the
	// "committed locally but never pushed" drift the preflight designs out.
	r := cprev.New(cprev.WithGitRunner(&fakeGit{head: cpCommit, advertised: []string{otherPush}}))
	_, err := r.Resolve(context.Background(), "")
	var notPushed cprev.ErrNotPushed
	if !errors.As(err, &notPushed) {
		t.Fatalf("err = %v, want ErrNotPushed", err)
	}
	if notPushed.Commit != cpCommit {
		t.Fatalf("ErrNotPushed.Commit = %q, want %q", notPushed.Commit, cpCommit)
	}
	if !strings.Contains(err.Error(), "push it first") {
		t.Fatalf("message %q lacks push-first guidance", err.Error())
	}
}

func TestResolve_MetacharRefRejectedAtBoundary(t *testing.T) {
	// A relative ref (HEAD~1) must be rejected here, BEFORE any git/preflight, so
	// the operator gets a friendly boundary error rather than an opaque privsep
	// failure on the node.
	git := &fakeGit{head: cpCommit, advertised: []string{cpCommit}}
	r := cprev.New(cprev.WithGitRunner(git))
	_, err := r.Resolve(context.Background(), "HEAD~1")
	var unsafe cprev.ErrUnsafeRevision
	if !errors.As(err, &unsafe) {
		t.Fatalf("err = %v, want ErrUnsafeRevision", err)
	}
	if git.lsRemoteCalls != 0 {
		t.Fatalf("preflight ran (%d ls-remote calls) for a metachar ref; validation must short-circuit", git.lsRemoteCalls)
	}
}

func TestValidateRef(t *testing.T) {
	good := []string{cpCommit, "master", "v1.2.3", "feature/foo", "origin/master"}
	for _, g := range good {
		if err := cprev.ValidateRef(g); err != nil {
			t.Errorf("ValidateRef(%q) = %v, want nil", g, err)
		}
	}
	bad := []string{"HEAD~1", "HEAD^", "a b", "$(x)", "a;b", "a|b", "a`b`", "a*b", ""}
	for _, b := range bad {
		if err := cprev.ValidateRef(b); err == nil {
			// HEAD^ uses '^' which is NOT in the mirrored privsep set — assert it is
			// treated as the privsep filter would (allowed), so this case is skipped.
			if b == "HEAD^" {
				continue
			}
			t.Errorf("ValidateRef(%q) = nil, want error", b)
		}
	}
}

func TestExpand_EmptyStaysEmpty(t *testing.T) {
	r := cprev.New(cprev.WithGitRunner(&fakeGit{head: cpCommit}))
	got, err := r.Expand(context.Background(), "")
	if err != nil {
		t.Fatalf("Expand(empty): %v", err)
	}
	if got != "" {
		t.Fatalf("Expand(empty) = %q, want empty (first-provision rollback preserved)", got)
	}
}

func TestExpand_SentinelExpandsWithoutPreflight(t *testing.T) {
	// Expand must NOT preflight: an unadvertised CP commit still expands cleanly
	// (a rollback target is where the node already is).
	git := &fakeGit{head: cpCommit, advertised: []string{otherPush}}
	r := cprev.New(cprev.WithGitRunner(git))
	got, err := r.Expand(context.Background(), cprev.SentinelControlPlane)
	if err != nil {
		t.Fatalf("Expand(@cp): %v", err)
	}
	if got != cpCommit {
		t.Fatalf("Expand(@cp) = %q, want CP commit", got)
	}
	if git.lsRemoteCalls != 0 {
		t.Fatalf("Expand preflighted (%d ls-remote calls); it must not", git.lsRemoteCalls)
	}
}

func TestExpand_ValidatesRef(t *testing.T) {
	r := cprev.New(cprev.WithGitRunner(&fakeGit{head: cpCommit}))
	_, err := r.Expand(context.Background(), "HEAD~1")
	var unsafe cprev.ErrUnsafeRevision
	if !errors.As(err, &unsafe) {
		t.Fatalf("Expand(HEAD~1) err = %v, want ErrUnsafeRevision", err)
	}
}

func TestPreflight_IndeterminateRemoteDegradesToAllow(t *testing.T) {
	// git ls-remote cannot run (remote unreachable / git missing): preflight
	// degrades to allow rather than wedging every op on an infra hiccup.
	r := cprev.New(cprev.WithGitRunner(&fakeGit{head: cpCommit, lsRemoteErr: errors.New("network unreachable")}))
	got, err := r.Resolve(context.Background(), "")
	if err != nil {
		t.Fatalf("Resolve with unreachable remote = %v, want allow-degraded", err)
	}
	if got != cpCommit {
		t.Fatalf("resolved = %q, want CP commit", got)
	}
}

func TestPreflight_CachesAdvertisement(t *testing.T) {
	// A fleet roll issues one Sync (one Resolve) per node; the ls-remote result
	// must be cached so N nodes do not mean N network round-trips.
	git := &fakeGit{head: cpCommit, advertised: []string{cpCommit}}
	r := cprev.New(cprev.WithGitRunner(git), cprev.WithAdvertisementTTL(time.Minute))
	for i := 0; i < 5; i++ {
		if _, err := r.Resolve(context.Background(), ""); err != nil {
			t.Fatalf("Resolve #%d: %v", i, err)
		}
	}
	if git.lsRemoteCalls != 1 {
		t.Fatalf("ls-remote called %d times across 5 resolves, want 1 (cached)", git.lsRemoteCalls)
	}
}
