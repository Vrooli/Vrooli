package fetch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func repoSource(revision string) Source {
	return Source{Repo: RepoSpec{RepoID: "org/model", Revision: revision}}
}

func TestStageDirIsDeterministicPerDestinationAndSource(t *testing.T) {
	root := "/data"
	dest := filepath.Join(root, "models", "flux")
	a, b := StageDir(root, dest, repoSource("rev-1")), StageDir(root, dest, repoSource("rev-1"))
	if a != b {
		t.Fatalf("same identity gave %s and %s", a, b)
	}
	if StageDir(root, dest, repoSource("rev-2")) == a {
		t.Fatal("a different revision must not share a staging directory")
	}
	if StageDir(root, filepath.Join(root, "models", "other"), repoSource("rev-1")) == a {
		t.Fatal("a different destination must not share a staging directory")
	}
	if want := filepath.Join(root, "staging", "models"); filepath.Dir(a) != want {
		t.Fatalf("stage %s is not under %s: staging must sit outside the declared-durable models directory", a, want)
	}
	if adapter := StageDir(root, filepath.Join(root, "adapters", "flux"), repoSource("rev-1")); filepath.Dir(adapter) != filepath.Join(root, "staging", "adapters") {
		t.Fatalf("adapter stage %s", adapter)
	}
	got, ok := StageDestination(a)
	if !ok || got != dest {
		t.Fatalf("StageDestination(%s) = %q, %v", a, got, ok)
	}
	if got, ok := StageDestination(a + PreviousSuffix); !ok || got != dest {
		t.Fatalf("previous form maps to %q, %v", got, ok)
	}
	if _, ok := StageDestination("/data/models/.acquiring-858204562"); ok {
		t.Fatal("legacy random names have no known destination")
	}
	if _, ok := StageDestination("/data/models/.acquiring-flux-0123456789ab"); ok {
		t.Fatal("a name beside the weights is not in the staging directory")
	}
}

func TestInterruptedRepoSnapshotIsResumedNotRestarted(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "models", "flux")
	source := repoSource("rev-1")
	// A previous process died mid-download: its partial file is still staged.
	stage := StageDir(root, dest, source)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "partial.bin"), []byte("half"), 0o600); err != nil {
		t.Fatal(err)
	}
	var sawPartial bool
	var messages []string
	acquirer := Acquirer{Root: root, RepoFetch: repoFetcherFunc(func(_ context.Context, _ RepoSpec, dir string, _ func(int64, int64)) error {
		if dir != stage {
			t.Fatalf("fetch dir %s, want the deterministic staging dir %s", dir, stage)
		}
		_, err := os.Stat(filepath.Join(dir, "partial.bin"))
		sawPartial = err == nil
		return os.WriteFile(filepath.Join(dir, "model.bin"), []byte("whole"), 0o600)
	})}
	if _, err := acquirer.Acquire(context.Background(), source, dest, "", 1, func(_ int, m string) { messages = append(messages, m) }); err != nil {
		t.Fatal(err)
	}
	if !sawPartial {
		t.Fatal("the fetcher did not see the earlier partial snapshot")
	}
	if len(messages) == 0 || messages[0] != "resuming partial download left by an interrupted install" {
		t.Fatalf("progress did not announce the resume: %v", messages)
	}
	if _, err := os.Lstat(stage); !os.IsNotExist(err) {
		t.Fatalf("staging not consumed by publication: %v", err)
	}
	if StageHeld(stage) {
		t.Fatal("staging ownership leaked after Acquire")
	}
}

func TestAcquisitionOwnsItsStagingWhileRunning(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "models", "flux")
	source := repoSource("rev-1")
	stage := StageDir(root, dest, source)
	acquirer := Acquirer{Root: root, RepoFetch: repoFetcherFunc(func(_ context.Context, _ RepoSpec, dir string, _ func(int64, int64)) error {
		if !StageHeld(stage) {
			t.Error("running acquisition does not own its staging directory")
		}
		// A concurrent acquisition of the same capability must wait, not share it.
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()
		if _, _, err := claimStage(ctx, root, dest, source); !errors.Is(err, ErrStageBusy) {
			t.Errorf("concurrent claim err = %v, want ErrStageBusy", err)
		}
		return os.WriteFile(filepath.Join(dir, "model.bin"), []byte("whole"), 0o600)
	})}
	if _, err := acquirer.Acquire(context.Background(), source, dest, "", 1, nil); err != nil {
		t.Fatal(err)
	}
}

func TestClaimWaitsForSweepToReleaseStaging(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "models", "flux")
	source := repoSource("rev-1")
	stage := StageDir(root, dest, source)
	if !HoldStage(stage) {
		t.Fatal("sweep could not hold the stage")
	}
	go func() {
		time.Sleep(150 * time.Millisecond)
		ReleaseStage(stage)
	}()
	got, _, err := claimStage(context.Background(), root, dest, source)
	if err != nil || got != stage {
		t.Fatalf("claim after release = %q, %v", got, err)
	}
	ReleaseStage(got)
}

func TestFailedAcquisitionStillCleansItsStaging(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "models", "flux")
	source := repoSource("rev-1")
	acquirer := Acquirer{Root: root, RepoFetch: repoFetcherFunc(func(_ context.Context, _ RepoSpec, dir string, _ func(int64, int64)) error {
		_ = os.WriteFile(filepath.Join(dir, "partial.bin"), []byte("half"), 0o600)
		return errors.New("network down")
	})}
	if _, err := acquirer.Acquire(context.Background(), source, dest, "", 1, nil); err == nil {
		t.Fatal("expected failure")
	}
	if _, err := os.Lstat(StageDir(root, dest, source)); !os.IsNotExist(err) {
		t.Fatalf("in-process failure left staging behind: %v", err)
	}
}

func TestCrashLeftoverPreviousIsRecoveredOrDiscarded(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "models", "flux")
	source := repoSource("rev-1")
	previous := StageDir(root, dest, source) + PreviousSuffix
	write := func(dir, name string) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	// Crash between the two publication renames: the destination is gone.
	write(previous, "old.bin")
	if err := recoverPrevious(dest, previous); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "old.bin")); err != nil {
		t.Fatalf("replaced installation not restored: %v", err)
	}
	// A destination already exists: the set-aside is stale.
	write(previous, "stale.bin")
	if err := recoverPrevious(dest, previous); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(previous); !os.IsNotExist(err) {
		t.Fatalf("stale set-aside kept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "old.bin")); err != nil {
		t.Fatalf("destination touched: %v", err)
	}
}
