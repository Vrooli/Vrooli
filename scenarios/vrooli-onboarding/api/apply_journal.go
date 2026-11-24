package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	platform "github.com/vrooli/platform-go"
)

// mutatePersistedApplyRun serializes small control-plane edits with runner
// progress writes. It always edits the latest durable snapshot, so a heartbeat
// or cancellation request cannot restore stale item outcomes over newer work.
func mutatePersistedApplyRun(id string, mutate func(*applyRun) error) (applyRun, error) {
	var run applyRun
	err := withApplyJournalLock(id, func() error {
		var err error
		run, err = loadPersistedApplyRun(id)
		if err != nil {
			return fmt.Errorf("read latest apply journal: %w", err)
		}
		if err := mutate(&run); err != nil {
			return err
		}
		for index := range run.Items {
			run.Items[index].Disposition = applyDisposition(run.Items[index].Outcome)
		}
		return persistApplyRunUpdate(run)
	})
	if err != nil {
		return run, err
	}
	applyRuns.Lock()
	applyRuns.items[id] = run
	applyRuns.Unlock()
	return run, nil
}

func withApplyJournalLock(id string, action func() error) error {
	path, err := applyRunPath(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	lock, err := platform.OpenLockFile(path+".journal.lock", 0o600)
	if err != nil {
		return err
	}
	release, err := platform.LockFile(lock, true)
	if err != nil {
		_ = lock.Close()
		return err
	}
	defer func() { release(); _ = lock.Close() }()
	if err := platform.VerifyLockFile(lock, path+".journal.lock"); err != nil {
		return err
	}
	return action()
}

func persistApplyRunProgress(run applyRun) error {
	return withApplyJournalLock(run.ID, func() error {
		latest, err := loadPersistedApplyRun(run.ID)
		if err == nil {
			// These fields are monotonic/control-plane state. A runner's local
			// progress snapshot must not clear a cancellation or roll back the
			// liveness stamp written by its heartbeat goroutine.
			run.CancelRequested = run.CancelRequested || latest.CancelRequested
			if latest.RunnerPID != 0 {
				run.RunnerPID = latest.RunnerPID
			}
			if latest.Heartbeat > run.Heartbeat {
				run.Heartbeat = latest.Heartbeat
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("read latest apply journal before progress write: %w", err)
		}
		return persistApplyRunUpdate(run)
	})
}

func restampApplyRunHeartbeat(id string) error {
	_, err := mutatePersistedApplyRun(id, func(run *applyRun) error {
		if run.Status.Terminal() {
			return nil
		}
		run.RunnerPID = os.Getpid()
		run.Heartbeat = operatorStateNow().UTC().Format(time.RFC3339)
		return nil
	})
	return err
}
