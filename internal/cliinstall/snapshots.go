package cliinstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
)

const snapshotRetention = 24 * time.Hour

// WithSnapshotLock serializes snapshot publication and maintenance across
// processes using the repository's cross-platform native file-lock owner.
func WithSnapshotLock(root string, fn func() error) error {
	file, release, err := acquireSnapshotRootLock(root)
	if err != nil {
		return err
	}
	defer file.Close()
	defer release()
	return fn()
}

// WithSnapshotLockFile exposes the held native lock handle so subprocess
// owners can pass it to children on platforms with descriptor-based locks.
// On Windows, LockFileEx ownership stays with this process.
func WithSnapshotLockFile(root string, fn func(*os.File) error) error {
	file, release, err := acquireSnapshotRootLock(root)
	if err != nil {
		return err
	}
	defer file.Close()
	defer release()
	// Do not call LOCK_UN here: Unix child descriptors share this lock's open
	// file description. Closing the owner's copy lets escaped descendants keep
	// exclusion until their final inherited descriptor closes.
	return fn(file)
}

// WithSnapshotLease takes the root lock while claiming the snapshot lease,
// then releases the root lock before running fn. This preserves the Makefile's
// root-to-lease handoff while allowing unrelated snapshot work to proceed.
func WithSnapshotLease(root, snapshot string, fn func() error) error {
	return WithSnapshotLeaseFile(root, snapshot, func(*os.File) error { return fn() })
}

// WithSnapshotLeaseFile retains the root-to-lease handoff and exposes the
// held lease handle for subprocess lock inheritance where supported. Windows
// LockFileEx ownership cannot outlive its process, so every later admission
// and prune also checks the named Job Object until its assigned commands drain.
func WithSnapshotLeaseFile(root, snapshot string, fn func(*os.File) error) error {
	return withSnapshotLeaseFileUsing(root, snapshot, fn, inspectSnapshotLeaseJob, time.Sleep)
}

func withSnapshotLeaseFileUsing(root, snapshot string, fn func(*os.File) error, probe func(string, string) (snapshotJobState, error), sleep func(time.Duration)) error {
	for attempt := 0; ; attempt++ {
		leaseFile, acquired, err := claimSnapshotLeaseWithRootLockUsing(root, snapshot, probe)
		if err != nil {
			return err
		}
		if acquired {
			return func() error {
				defer leaseFile.Close()
				return fn(leaseFile)
			}()
		}
		// Another helper may have died while its Job Object members were still
		// draining. Retry only after dropping both the lease and root locks.
		sleep(snapshotLeaseRetryDelay(attempt))
	}
}

// TryWithSnapshotLease skips fn when another process owns the lease. It creates
// the marker when absent, matching shell `exec >.lease` cleanup behavior.
func TryWithSnapshotLease(root, snapshot string, fn func() error) (bool, error) {
	return TryWithSnapshotLeaseFile(root, snapshot, func(*os.File) error { return fn() })
}

// TryWithSnapshotLeaseFile retains TryWithSnapshotLease's no-op-on-contention
// behavior and exposes the lock handle to its subprocess owner.
func TryWithSnapshotLeaseFile(root, snapshot string, fn func(*os.File) error) (bool, error) {
	leaseFile, acquired, err := claimSnapshotLeaseWithRootLock(root, snapshot)
	if err != nil || !acquired {
		return false, err
	}
	defer leaseFile.Close()
	return true, fn(leaseFile)
}

func claimSnapshotLeaseWithRootLock(root, snapshot string) (*os.File, bool, error) {
	return claimSnapshotLeaseWithRootLockUsing(root, snapshot, inspectSnapshotLeaseJob)
}

func claimSnapshotLeaseWithRootLockUsing(root, snapshot string, probe func(string, string) (snapshotJobState, error)) (*os.File, bool, error) {
	rootFile, releaseRoot, err := acquireSnapshotRootLock(root)
	if err != nil {
		return nil, false, err
	}
	defer rootFile.Close()
	defer releaseRoot()
	return claimSnapshotLeaseFileUnderRootUsing(root, snapshot, probe)
}

func acquireSnapshotRootLock(root string) (*os.File, func(), error) {
	return acquireSnapshotRootLockUsing(root, inspectSnapshotRootJob, time.Sleep)
}

func acquireSnapshotRootLockUsing(root string, probe func(string) (snapshotJobState, error), sleep func(time.Duration)) (*os.File, func(), error) {
	for attempt := 0; ; attempt++ {
		file, release, err := acquireFileLock(filepath.Join(root, "snapshots.lock"), false)
		if err != nil {
			return nil, nil, err
		}
		state, probeErr := probe(root)
		allowed, policyErr := snapshotRootAdmission(state, probeErr)
		if policyErr != nil {
			release()
			_ = file.Close()
			return nil, nil, policyErr
		}
		if !allowed {
			release()
			_ = file.Close()
			sleep(snapshotLeaseRetryDelay(attempt))
			continue
		}
		return file, release, nil
	}
}

// tryWithSnapshotLeaseFileUnderRoot must only be called while snapshots.lock
// is held. It lets pruning inspect or remove a bundle without recursive lock
// acquisition, while keeping every lease admission serialized against removal.
func tryWithSnapshotLeaseFileUnderRoot(root, snapshot string, fn func(*os.File) error) (bool, error) {
	return tryWithSnapshotLeaseFileUnderRootUsing(root, snapshot, fn, inspectSnapshotLeaseJob)
}

func tryWithSnapshotLeaseFileUnderRootUsing(root, snapshot string, fn func(*os.File) error, probe func(string, string) (snapshotJobState, error)) (bool, error) {
	file, acquired, err := claimSnapshotLeaseFileUnderRootUsing(root, snapshot, probe)
	if err != nil || !acquired {
		return false, err
	}
	defer file.Close()
	return true, fn(file)
}

func claimSnapshotLeaseFileUnderRootUsing(root, snapshot string, probe func(string, string) (snapshotJobState, error)) (*os.File, bool, error) {
	leasePath, err := snapshotLeasePath(root, snapshot)
	if err != nil {
		return nil, false, err
	}
	file, err := acquireFileHandleLock(leasePath, true)
	if errors.Is(err, platform.ErrLockUnavailable) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	state, probeErr := probe(root, snapshot)
	allowed, policyErr := snapshotLeaseAdmission(true, state, probeErr)
	if policyErr != nil || !allowed {
		_ = file.Close()
		return nil, false, policyErr
	}
	return file, true, nil
}

// PruneInstallSnapshots removes only old ordinary bundles. Leased bundles
// survive; markerless stale directories can be removed because publication
// holds the root lock and writes its lease before writing any images.
func PruneInstallSnapshots(root string, now time.Time) error {
	return WithSnapshotLock(root, func() error {
		entries, err := os.ReadDir(root)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() || !snapshotName(entry.Name(), "install-bundle.") {
				continue
			}
			candidate := filepath.Join(root, entry.Name())
			info, err := entry.Info()
			if err != nil || now.Sub(info.ModTime()) <= snapshotRetention {
				continue
			}
			// Make holds the root lock from directory creation until .lease is
			// durable. A stale markerless bundle therefore predates this protocol
			// or was abandoned before the marker write and is safe to remove.
			if _, err := os.Stat(filepath.Join(candidate, ".lease")); errors.Is(err, os.ErrNotExist) {
				state, probeErr := inspectSnapshotLeaseJob(root, candidate)
				allowed, policyErr := snapshotLeasePruneAllowed(state, probeErr)
				if policyErr != nil {
					return fmt.Errorf("inspect install snapshot job %q: %w", candidate, policyErr)
				}
				if !allowed {
					continue
				}
				if err := os.RemoveAll(candidate); err != nil {
					return fmt.Errorf("prune markerless install snapshot %q: %w", candidate, err)
				}
				continue
			} else if err != nil {
				return err
			}
			_, err = tryWithSnapshotLeaseFileUnderRoot(root, candidate, func(*os.File) error { return os.RemoveAll(candidate) })
			if err != nil {
				return fmt.Errorf("prune install snapshot %q: %w", candidate, err)
			}
		}
		return nil
	})
}

type snapshotJobState uint8

const (
	snapshotJobAbsent snapshotJobState = iota
	snapshotJobIdle
	snapshotJobActive
)

func snapshotLeaseAdmission(leaseLocked bool, state snapshotJobState, probeErr error) (bool, error) {
	if !leaseLocked {
		return false, nil
	}
	return snapshotJobAllows(state, probeErr)
}

func snapshotRootAdmission(state snapshotJobState, probeErr error) (bool, error) {
	return snapshotJobAllows(state, probeErr)
}

func snapshotLeasePruneAllowed(state snapshotJobState, probeErr error) (bool, error) {
	return snapshotJobAllows(state, probeErr)
}

func snapshotJobAllows(state snapshotJobState, probeErr error) (bool, error) {
	if probeErr != nil {
		return false, fmt.Errorf("inspect snapshot Job Object: %w", probeErr)
	}
	switch state {
	case snapshotJobAbsent, snapshotJobIdle:
		return true, nil
	case snapshotJobActive:
		return false, nil
	default:
		return false, fmt.Errorf("inspect snapshot Job Object: invalid state %d", state)
	}
}

func inspectSnapshotJobNamesUsing(names []string, probe func(string) (snapshotJobState, error)) (snapshotJobState, error) {
	if len(names) == 0 || probe == nil {
		return snapshotJobAbsent, fmt.Errorf("inspect snapshot Job Object: no witness names or probe")
	}
	state := snapshotJobAbsent
	for _, name := range names {
		candidate, err := probe(name)
		if err != nil {
			return snapshotJobAbsent, fmt.Errorf("inspect snapshot Job Object %q: %w", name, err)
		}
		switch candidate {
		case snapshotJobAbsent:
		case snapshotJobIdle:
			if state == snapshotJobAbsent {
				state = snapshotJobIdle
			}
		case snapshotJobActive:
			state = snapshotJobActive
		default:
			return snapshotJobAbsent, fmt.Errorf("inspect snapshot Job Object %q: invalid state %d", name, candidate)
		}
	}
	return state, nil
}

func snapshotLeaseRetryDelay(attempt int) time.Duration {
	delay := 50 * time.Millisecond
	for i := 0; i < attempt && delay < 500*time.Millisecond; i++ {
		delay *= 2
	}
	if delay > 500*time.Millisecond {
		return 500 * time.Millisecond
	}
	return delay
}

// SnapshotLeaseJobName returns the stable global Windows Job Object name for
// the validated .lease path. Non-Windows callers use the same name for parity
// and portable policy tests, even though their process groups need no witness.
func SnapshotLeaseJobName(root, snapshot string) (string, error) {
	path, err := snapshotLeasePath(root, snapshot)
	if err != nil {
		return "", err
	}
	return snapshotJobObjectName(`Global\VrooliSnapshotLease-`, path)
}

// SnapshotRootJobName returns a stable global Windows Job Object name for the
// root snapshots.lock. Root-scoped helper commands use it as a recovery witness
// while creating and preparing a snapshot before its per-snapshot lease exists.
func SnapshotRootJobName(root string) (string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve absolute snapshot root: %w", err)
	}
	return snapshotJobObjectName(`Global\VrooliSnapshotRoot-`, filepath.Join(absoluteRoot, "snapshots.lock"))
}

// PruneHubSnapshots preserves the active locator and recoverable pending
// snapshots. Incomplete and completed snapshots are disposable; pending ready
// snapshots require the caller's retirement decision before removal.
func PruneHubSnapshots(root, current string, now time.Time, retire func(string, *os.File) error) error {
	return WithSnapshotLock(root, func() error {
		return pruneHubSnapshotsUnderRoot(root, current, now, retire)
	})
}

// PruneHubSnapshotsFromPointers reads the current and legacy locator files only
// after acquiring snapshots.lock, so pruning cannot use a stale pointer while
// waiting for root admission.
func PruneHubSnapshotsFromPointers(root, currentPointer, legacyPointer string, now time.Time, retire func(string, *os.File) error) error {
	return WithSnapshotLock(root, func() error {
		current, err := readCurrentSnapshotPointer(currentPointer, legacyPointer)
		if err != nil {
			return err
		}
		return pruneHubSnapshotsUnderRoot(root, current, now, retire)
	})
}

func readCurrentSnapshotPointer(currentPointer, legacyPointer string) (string, error) {
	current, err := os.ReadFile(currentPointer)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	currentPath := strings.TrimSpace(string(current))
	if currentPath != "" {
		return currentPath, nil
	}
	legacy, err := os.ReadFile(legacyPointer)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return strings.TrimSpace(string(legacy)), nil
}

func pruneHubSnapshotsUnderRoot(root, current string, now time.Time, retire func(string, *os.File) error) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !snapshotName(entry.Name(), "rebaseline-hub.") {
			continue
		}
		candidate := filepath.Join(root, entry.Name())
		if samePath(candidate, current) {
			continue
		}
		info, err := entry.Info()
		if err != nil || now.Sub(info.ModTime()) <= snapshotRetention {
			continue
		}
		_, err = tryWithSnapshotLeaseFileUnderRoot(root, candidate, func(lease *os.File) error {
			if _, err := os.Stat(filepath.Join(candidate, ".completed")); err == nil {
				return os.RemoveAll(candidate)
			}
			if _, err := os.Stat(filepath.Join(candidate, ".ready")); errors.Is(err, os.ErrNotExist) {
				return os.RemoveAll(candidate)
			} else if err != nil {
				return err
			}
			if retire != nil && retire(candidate, lease) == nil {
				return os.RemoveAll(candidate)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("prune Hub snapshot %q: %w", candidate, err)
		}
	}
	return nil
}

func acquireFileLock(path string, nonBlocking bool) (*os.File, func(), error) {
	file, err := platform.OpenLockFile(path, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open snapshot lock %q: %w", path, err)
	}
	release, err := platform.LockFile(file, nonBlocking)
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if err := platform.VerifyLockFile(file, path); err != nil {
		release()
		_ = file.Close()
		return nil, nil, err
	}
	return file, release, nil
}

func acquireFileHandleLock(path string, nonBlocking bool) (*os.File, error) {
	file, _, err := acquireFileLock(path, nonBlocking)
	return file, err
}

func snapshotLeasePath(root, snapshot string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	snapshotAbs, err := filepath.Abs(snapshot)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, snapshotAbs)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.ContainsRune(rel, filepath.Separator) {
		return "", fmt.Errorf("snapshot path is outside the snapshot root")
	}
	return filepath.Join(snapshotAbs, ".lease"), nil
}

func snapshotName(name, prefix string) bool {
	return strings.HasPrefix(name, prefix) && len(name) == len(prefix)+8 && filepath.Base(name) == name
}

func samePath(a, b string) bool {
	if b == "" {
		return false
	}
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	return errA == nil && errB == nil && aa == bb
}
