package maintenance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	platform "github.com/vrooli/platform-go"
)

const ScenarioLockV1 = "scenario-lock-v1"

// ScenarioInterlock implements the control plane's scenario-lock-v1 contract.
// Resume takes this nonblocking lock BEFORE the gate mutex and retains it through
// persistence. Status never takes it: the lifecycle owner reads status under its
// own lock. The lock file is never removed, preserving the shared inode.
type ScenarioInterlock struct {
	path string
	mu   sync.Mutex
}

func NewScenarioInterlock(ownerHome string) (*ScenarioInterlock, error) {
	if !filepath.IsAbs(ownerHome) {
		return nil, fmt.Errorf("maintenance interlock requires the lifecycle owner's absolute home")
	}
	return &ScenarioInterlock{path: filepath.Join(ownerHome, ".vrooli", "state", "locks", "scenario-agent-manager.lock")}, nil
}

func (l *ScenarioInterlock) acquire(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !l.mu.TryLock() {
		return nil, fmt.Errorf("%w: lifecycle operation holds the scenario lock", ErrConflict)
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		l.mu.Unlock()
		return nil, err
	}
	f, err := os.OpenFile(l.path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		l.mu.Unlock()
		return nil, err
	}
	unlock, err := platform.LockFile(f, true)
	if err != nil {
		_ = f.Close()
		l.mu.Unlock()
		if errors.Is(err, platform.ErrLockUnavailable) {
			return nil, fmt.Errorf("%w: lifecycle operation holds the scenario lock", ErrConflict)
		}
		return nil, err
	}
	var once sync.Once
	release := func() { once.Do(func() { unlock(); _ = f.Close(); l.mu.Unlock() }) }
	if err := ctx.Err(); err != nil {
		release()
		return nil, err
	}
	return release, nil
}
