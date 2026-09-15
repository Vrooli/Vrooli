package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
)

var artifactStageLocks sync.Map

func stageArtifact(target string, directory bool) (stageRoot, staged string, cleanup func(), err error) {
	parent := filepath.Dir(target)
	lock := artifactStageLock(parent)
	lock.Lock()
	released := false
	fileLockPath := filepath.Join(parent, ".vrooli-artifact-stage.lock")
	fileLock, err := os.OpenFile(fileLockPath, os.O_RDWR|os.O_CREATE, tuning.PermFile)
	if err != nil {
		lock.Unlock()
		return "", "", func() {}, err
	}
	var releaseFile func()
	if err := AwaitContext(context.Background(), AwaitClock{Now: time.Now, Sleep: time.Sleep}, AwaitPolicy{Timeout: tuning.DailyRetentionWindow(), Interval: tuning.FastHealthPollInterval()}, func() (bool, error) {
		releaseFile, err = lockFileFn(fileLock, true)
		if errors.Is(err, platform.ErrLockUnavailable) {
			return false, nil
		}
		return err == nil, err
	}); err != nil {
		_ = fileLock.Close()
		lock.Unlock()
		return "", "", func() {}, err
	}
	release := func() {
		if released {
			return
		}
		released = true
		if releaseFile != nil {
			releaseFile()
		}
		_ = fileLock.Close()
		lock.Unlock()
	}
	if err := reapArtifactStages(parent); err != nil {
		release()
		return "", "", func() {}, err
	}
	stageRoot, err = os.MkdirTemp(parent, ".vrooli-artifact-stage-")
	if err != nil {
		release()
		return "", "", func() {}, err
	}
	cleanup = func() { _ = os.RemoveAll(stageRoot); release() }
	if directory {
		return stageRoot, stageRoot, cleanup, nil
	}
	return stageRoot, filepath.Join(stageRoot, filepath.Base(target)), cleanup, nil
}

func artifactStageLock(parent string) *sync.Mutex {
	if existing, ok := artifactStageLocks.Load(parent); ok {
		return existing.(*sync.Mutex)
	}
	created := &sync.Mutex{}
	actual, _ := artifactStageLocks.LoadOrStore(parent, created)
	return actual.(*sync.Mutex)
}

// reapArtifactStages removes abandoned staging directories left by a process
// that was interrupted before its deferred cleanup ran. Only the exact
// lifecycle-owned prefix is eligible; ordinary sibling directories are never
// touched.
func reapArtifactStages(parent string) error {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".vrooli-artifact-stage-") {
			continue
		}
		if err := os.RemoveAll(filepath.Join(parent, entry.Name())); err != nil {
			return fmt.Errorf("reap abandoned artifact stage %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func swapArtifact(staged, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), tuning.PermDir); err != nil {
		return err
	}
	if err := platform.AtomicReplace(staged, target); err != nil {
		return fmt.Errorf("atomic artifact replace %s: %w", target, err)
	}
	return nil
}
