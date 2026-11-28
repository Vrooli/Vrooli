package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestRH3FixtureSecurityRequiresPinnedTempLeaseAndContainedGitMetadata(t *testing.T) {
	root, err := os.MkdirTemp("", "gct-rh3-live-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	dbPath := filepath.Join(root, "fixture.sqlite")
	primaryPath := filepath.Join(t.TempDir(), "primary.sqlite")
	db, err := coredb.Open(context.Background(), coredb.Config{
		Driver: coredb.DriverSQLite, TestDriver: coredb.DriverSQLite,
		DSN: primaryPath, MaxOpenConns: 1, MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open routed DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	const leaseID = "rh3-fixture-security-test"
	if err := db.InstallTestPool(context.Background(), dbPath, leaseID, time.Minute); err != nil {
		t.Fatalf("install test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.ClearTestPool(leaseID) })

	rootDirs := storage.Paths{
		ConfigDir: filepath.Join(t.TempDir(), "config"),
		DataDir:   filepath.Join(t.TempDir(), "data"),
		CacheDir:  filepath.Join(t.TempDir(), "cache"),
		LogsDir:   filepath.Join(t.TempDir(), "logs"),
		StateDir:  filepath.Join(t.TempDir(), "state"),
	}
	for _, dir := range []string{rootDirs.ConfigDir, rootDirs.DataDir, rootDirs.CacheDir, rootDirs.LogsDir, rootDirs.StateDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	roots := filerouting.New(rootDirs)
	if err := roots.InstallTestRoots(rootDirs, leaseID, time.Minute); err != nil {
		t.Fatalf("install file roots: %v", err)
	}
	ctx, err := db.PinTestPool(coredb.WithTestMode(context.Background()), leaseID)
	if err != nil {
		t.Fatalf("pin test DB: %v", err)
	}
	ctx, err = roots.PinTestRoots(ctx, leaseID)
	if err != nil {
		t.Fatalf("pin test roots: %v", err)
	}

	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, repo, "init", "--initial-branch=main")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("first candidate\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, repo, "add", "tracked.txt")
	if err := os.WriteFile(filepath.Join(root, rh3BenignFixtureMarker), []byte(leaseID+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	provider, fixtureOnly, unavailable := selectCommitCandidateSecurity(ctx, nil, db, roots, &ExecGitRunner{}, repo)
	if _, ok := provider.(rh3BenignCandidateSecurity); !ok {
		t.Fatalf("contained fixture repo did not receive fixture provider: %#v", provider)
	}
	if !fixtureOnly || unavailable != "" {
		t.Fatalf("commit candidate-security selection = fixtureOnly %t, reason %q", fixtureOnly, unavailable)
	}
	admission := scanStagedCandidate(ctx, &ExecGitRunner{}, repo, provider)
	if admission.Status != "passed" || !strings.Contains(admission.Summary, "Fixture-only benign provider passed") {
		t.Fatalf("commit admission gate did not consume fixture provider: %#v", admission)
	}
	result, err := provider.Scan(ctx, map[string][]byte{"tracked.txt": []byte("first candidate\n")})
	if err != nil || !result.Complete || !result.FixtureOnly {
		t.Fatalf("fixture provider result = %#v, %v", result, err)
	}
	unexpected, err := provider.Scan(ctx, map[string][]byte{"tracked.txt": []byte("secret=not-allowed\n")})
	if err != nil || unexpected.Complete {
		t.Fatalf("fixture provider accepted arbitrary content: %#v, %v", unexpected, err)
	}

	if got := rh3FixtureCandidateSecurity(context.Background(), db, roots, &ExecGitRunner{}, repo); got != nil {
		t.Fatalf("provider active without test-mode pin: %#v", got)
	}
	if err := os.Remove(filepath.Join(root, rh3BenignFixtureMarker)); err != nil {
		t.Fatal(err)
	}
	if got := rh3FixtureCandidateSecurity(ctx, db, roots, &ExecGitRunner{}, repo); got != nil {
		t.Fatalf("provider active without exact fixture marker: %#v", got)
	}
	var unavailableProvider CandidateSecurityScanner
	unavailableProvider, fixtureOnly, unavailable = selectCommitCandidateSecurity(ctx, nil, db, roots, &ExecGitRunner{}, repo)
	if unavailableProvider == nil || fixtureOnly || unavailable == "" {
		t.Fatalf("missing test fixture did not select fail-closed scanner: provider=%#v fixture=%t reason=%q", unavailableProvider, fixtureOnly, unavailable)
	}
	unavailableResult := scanStagedCandidate(ctx, &ExecGitRunner{}, repo, unavailableProvider)
	if unavailableResult.Status != "incomplete" || strings.Contains(unavailableResult.Summary, "first candidate") {
		t.Fatalf("missing fixture security coverage was not incomplete and redacted: %#v", unavailableResult)
	}
	if err := os.WriteFile(filepath.Join(root, rh3BenignFixtureMarker), []byte(leaseID+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	external := filepath.Join(t.TempDir(), "external-repo")
	if err := os.MkdirAll(external, 0o700); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, external, "init", "--initial-branch=main")
	gitCommand(t, external, "config", "user.name", "Fixture")
	gitCommand(t, external, "config", "user.email", "fixture@example.invalid")
	if err := os.WriteFile(filepath.Join(external, "base.txt"), []byte("external\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, external, "add", "base.txt")
	gitCommand(t, external, "commit", "-m", "test: external repository")
	if got := rh3FixtureCandidateSecurity(ctx, db, roots, &ExecGitRunner{}, external); got != nil {
		t.Fatalf("provider accepted repo outside fixture root: %#v", got)
	}
	linked := filepath.Join(root, "linked-worktree")
	gitCommand(t, external, "worktree", "add", "--detach", linked, "HEAD")
	if got := rh3FixtureCandidateSecurity(ctx, db, roots, &ExecGitRunner{}, linked); got != nil {
		t.Fatalf("provider accepted external common Git directory: %#v", got)
	}
}

func gitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
