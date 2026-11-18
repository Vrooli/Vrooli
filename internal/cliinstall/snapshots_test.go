package cliinstall

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

func TestSnapshotLockExcludesConcurrentPublishers(t *testing.T) {
	root := t.TempDir()
	held := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- WithSnapshotLock(root, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	entered := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- WithSnapshotLock(root, func() error { close(entered); return nil })
	}()
	select {
	case <-entered:
		t.Fatal("second publisher entered while root snapshot lock was held")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotLockChildProcessHelper(t *testing.T) {
	if os.Getenv("VROOLI_SNAPSHOT_LOCK_CHILD") != "1" {
		return
	}
	root := os.Getenv("VROOLI_SNAPSHOT_LOCK_ROOT")
	started := os.Getenv("VROOLI_SNAPSHOT_LOCK_STARTED")
	acquired := os.Getenv("VROOLI_SNAPSHOT_LOCK_ACQUIRED")
	if err := os.WriteFile(started, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := WithSnapshotLock(root, func() error { return os.WriteFile(acquired, nil, 0600) }); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotLockExcludesSeparateProcess(t *testing.T) {
	root := t.TempDir()
	started := filepath.Join(root, "child.started")
	acquired := filepath.Join(root, "child.acquired")
	var child *exec.Cmd
	err := WithSnapshotLock(root, func() error {
		child = exec.Command(os.Args[0], "-test.run=^TestSnapshotLockChildProcessHelper$")
		child.Env = append(os.Environ(), "VROOLI_SNAPSHOT_LOCK_CHILD=1", "VROOLI_SNAPSHOT_LOCK_ROOT="+root, "VROOLI_SNAPSHOT_LOCK_STARTED="+started, "VROOLI_SNAPSHOT_LOCK_ACQUIRED="+acquired)
		if err := child.Start(); err != nil {
			return err
		}
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(started); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if _, err := os.Stat(started); err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond)
		if _, err := os.Stat(acquired); err == nil {
			return fmt.Errorf("child acquired the root lock while parent held it")
		}
		return nil
	})
	if child != nil {
		if waitErr := child.Wait(); err == nil && waitErr != nil {
			err = waitErr
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(acquired); err != nil {
		t.Fatalf("child did not acquire after parent released: %v", err)
	}
}

func TestSnapshotLeaseHandoffReleasesRootAndKeepsLease(t *testing.T) {
	root := t.TempDir()
	snapshot := makeSnapshot(t, root, "rebaseline-hub", "pending01", ".ready")
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = WithSnapshotLease(root, snapshot, func() error {
			// A different key exercises the native lock directly while the lease
			// callback runs; the root lock must already have been handed off.
			release, lockErr := platform.AcquireFileLock(filepath.Join(root, "snapshots.lock"))
			if lockErr != nil {
				return lockErr
			}
			release()
			acquired, leaseErr := TryWithSnapshotLease(root, snapshot, func() error { return nil })
			if leaseErr != nil {
				return leaseErr
			}
			if acquired {
				return os.ErrInvalid
			}
			return nil
		})
	}()
	wg.Wait()
	if err != nil {
		t.Fatalf("lease handoff: %v", err)
	}
}

func TestTrySnapshotLeaseWaitsForRootLockBeforeOpeningLease(t *testing.T) {
	root := t.TempDir()
	snapshot := makeSnapshot(t, root, "install-bundle", "race0001", "")
	leasePath := filepath.Join(snapshot, ".lease")
	if err := os.Remove(leasePath); err != nil {
		t.Fatal(err)
	}

	held := make(chan struct{})
	release := make(chan struct{})
	rootDone := make(chan error, 1)
	go func() {
		rootDone <- WithSnapshotLock(root, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held

	started := make(chan struct{})
	type leaseResult struct {
		acquired bool
		err      error
	}
	tryDone := make(chan leaseResult, 1)
	go func() {
		close(started)
		acquired, err := TryWithSnapshotLease(root, snapshot, func() error { return nil })
		tryDone <- leaseResult{acquired: acquired, err: err}
	}()
	<-started
	select {
	case result := <-tryDone:
		t.Fatalf("lease admission bypassed snapshots.lock: acquired=%v err=%v", result.acquired, result.err)
	case <-time.After(30 * time.Millisecond):
	}
	if _, err := os.Stat(leasePath); !os.IsNotExist(err) {
		t.Fatalf("lease marker was opened or recreated before root lock release: stat err=%v", err)
	}

	close(release)
	if err := <-rootDone; err != nil {
		t.Fatal(err)
	}
	result := <-tryDone
	if result.err != nil {
		t.Fatal(result.err)
	}
	if !result.acquired {
		t.Fatal("lease was not acquired after root lock release")
	}
}

func TestPruneInstallSnapshotsRequiresOldAndUnleasedMarker(t *testing.T) {
	root := t.TempDir()
	stale := makeSnapshot(t, root, "install-bundle", "stale001", "")
	active := makeSnapshot(t, root, "install-bundle", "active01", "")
	noLease := makeSnapshot(t, root, "install-bundle", "nolease1", "")
	if err := os.Remove(filepath.Join(noLease, ".lease")); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	for _, path := range []string{stale, active, noLease} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	activeLease, err := platform.AcquireFileLock(filepath.Join(active, ".lease"))
	if err != nil {
		t.Fatal(err)
	}
	defer activeLease()
	if err := PruneInstallSnapshots(root, time.Now()); err != nil {
		t.Fatal(err)
	}
	assertExists(t, stale, false)
	assertExists(t, active, true)
	assertExists(t, noLease, false)
}

func TestPruneHubPreservesCurrentRecoverableAndBusySnapshots(t *testing.T) {
	root := t.TempDir()
	completed := makeSnapshot(t, root, "rebaseline-hub", "complete", ".completed")
	orphan := makeSnapshot(t, root, "rebaseline-hub", "orphan01", "")
	recoverable := makeSnapshot(t, root, "rebaseline-hub", "pending0", ".ready")
	current := makeSnapshot(t, root, "rebaseline-hub", "current1", ".completed")
	retirable := makeSnapshot(t, root, "rebaseline-hub", "retire01", ".ready")
	busy := makeSnapshot(t, root, "rebaseline-hub", "busy0001", ".completed")
	old := time.Now().Add(-48 * time.Hour)
	for _, path := range []string{completed, orphan, recoverable, current, retirable, busy} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	busyLease, err := platform.AcquireFileLock(filepath.Join(busy, ".lease"))
	if err != nil {
		t.Fatal(err)
	}
	defer busyLease()
	var retired []string
	if err := PruneHubSnapshots(root, current, time.Now(), func(path string, _ *os.File) error {
		retired = append(retired, path)
		if path == retirable {
			return nil
		}
		return os.ErrPermission
	}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{completed, orphan, retirable} {
		assertExists(t, path, false)
	}
	for _, path := range []string{recoverable, current, busy} {
		assertExists(t, path, true)
	}
	if len(retired) != 2 {
		t.Fatalf("retirement callbacks = %v, want only pending and retirable snapshots", retired)
	}
}

func TestPruneHubReadsCurrentPointerAfterWaitingForRootLock(t *testing.T) {
	root := t.TempDir()
	previous := makeSnapshot(t, root, "rebaseline-hub", "previous", ".completed")
	current := makeSnapshot(t, root, "rebaseline-hub", "current01", ".completed")
	old := time.Now().Add(-48 * time.Hour)
	for _, path := range []string{previous, current} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	currentPointer := filepath.Join(root, "current")
	if err := os.WriteFile(currentPointer, []byte(previous), 0600); err != nil {
		t.Fatal(err)
	}
	rootHeld := make(chan struct{})
	releaseRoot := make(chan struct{})
	rootDone := make(chan error, 1)
	go func() {
		rootDone <- WithSnapshotLock(root, func() error {
			close(rootHeld)
			<-releaseRoot
			return nil
		})
	}()
	<-rootHeld

	pruneStarted := make(chan struct{})
	pruneDone := make(chan error, 1)
	go func() {
		close(pruneStarted)
		pruneDone <- PruneHubSnapshotsFromPointers(root, currentPointer, filepath.Join(root, "legacy-current"), time.Now(), nil)
	}()
	<-pruneStarted
	// Let pruning reach the root-lock wait before switching the active locator.
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(currentPointer, []byte(current), 0600); err != nil {
		t.Fatal(err)
	}
	close(releaseRoot)
	if err := <-rootDone; err != nil {
		t.Fatal(err)
	}
	if err := <-pruneDone; err != nil {
		t.Fatal(err)
	}
	assertExists(t, previous, false)
	assertExists(t, current, true)
}

func makeSnapshot(t *testing.T, root, prefix, suffix, marker string) string {
	t.Helper()
	path := filepath.Join(root, prefix+"."+suffix)
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, ".lease"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if marker != "" {
		if err := os.WriteFile(filepath.Join(path, marker), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func assertExists(t *testing.T, path string, exists bool) {
	t.Helper()
	_, err := os.Stat(path)
	if exists && err != nil {
		t.Errorf("expected %q to exist: %v", path, err)
	}
	if !exists && !os.IsNotExist(err) {
		t.Errorf("expected %q to be pruned; stat error=%v", path, err)
	}
}

// A refused transition leaves its working copy behind. The prune must remove it
// once it is old, and must keep a fresh one that another run may still be using.
func TestPruneInstallSnapshotsAlsoRemovesAbandonedTransitionBundles(t *testing.T) {
	root := t.TempDir()
	stale := makeSnapshot(t, root, "rebaseline-bundle", "stale001", "")
	fresh := makeSnapshot(t, root, "rebaseline-bundle", "fresh001", "")
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	if err := PruneInstallSnapshots(root, time.Now()); err != nil {
		t.Fatal(err)
	}
	assertExists(t, stale, false)
	assertExists(t, fresh, true)
}
