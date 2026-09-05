package cprev_test

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"vrooli-bridge/internal/cprev"
)

// TestResolve_RealGit_UnpushedCommitFailsPreflight exercises the whole pipeline
// against REAL git: a scratch clone with a local commit that was never pushed
// must fail preflight with push-first guidance, while a commit that IS on the
// remote resolves cleanly. No real remote is ever contacted — the "remote" is a
// bare repo in a temp dir.
func TestResolve_RealGit_UnpushedCommitFailsPreflight(t *testing.T) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}

	root := t.TempDir()
	remoteDir := filepath.Join(root, "remote.git")
	workDir := filepath.Join(root, "work")

	run := func(dir string, args ...string) string {
		t.Helper()
		full := append([]string{
			"-c", "user.email=test@vrooli.dev",
			"-c", "user.name=Test",
			"-c", "commit.gpgsign=false",
			"-c", "init.defaultBranch=main",
		}, args...)
		cmd := exec.Command(gitBin, full...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s (in %s): %v\n%s", strings.Join(args, " "), dir, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	// Bare "remote" + a work clone with one pushed commit (A).
	run(root, "init", "--bare", remoteDir)
	run(root, "clone", remoteDir, workDir)
	run(workDir, "commit", "--allow-empty", "-m", "A")
	pushedA := run(workDir, "rev-parse", "HEAD")
	run(workDir, "push", "origin", "HEAD:refs/heads/main")

	// A second commit (B) made LOCALLY and never pushed — the drift being caught.
	run(workDir, "commit", "--allow-empty", "-m", "B")
	unpushedB := run(workDir, "rev-parse", "HEAD")

	r := cprev.New(cprev.WithRepoDir(workDir), cprev.WithRemote("origin"))
	ctx := context.Background()

	// The control plane's HEAD is now the unpushed B → default resolution fails
	// preflight with push-first guidance naming the commit and remote.
	_, err = r.Resolve(ctx, "")
	var notPushed cprev.ErrNotPushed
	if !errors.As(err, &notPushed) {
		t.Fatalf("Resolve(default HEAD=B) = %v, want ErrNotPushed", err)
	}
	if notPushed.Commit != unpushedB {
		t.Fatalf("ErrNotPushed.Commit = %q, want unpushed B %q", notPushed.Commit, unpushedB)
	}
	if !strings.Contains(err.Error(), "origin") || !strings.Contains(err.Error(), "push it first") {
		t.Fatalf("message %q lacks commit/remote/push-first guidance", err.Error())
	}

	// The pushed commit A IS on the remote → resolves cleanly (exact ls-remote
	// match on the branch tip).
	got, err := r.Resolve(ctx, pushedA)
	if err != nil {
		t.Fatalf("Resolve(pushed A) = %v, want success", err)
	}
	if got != pushedA {
		t.Fatalf("resolved = %q, want pushed A %q", got, pushedA)
	}

	// ControlPlaneCommit reports the live HEAD (B) regardless of push state.
	head, err := r.ControlPlaneCommit(ctx)
	if err != nil {
		t.Fatalf("ControlPlaneCommit: %v", err)
	}
	if head != unpushedB {
		t.Fatalf("ControlPlaneCommit = %q, want live HEAD B %q", head, unpushedB)
	}
}
