package maintenance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
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

// NewScenarioInterlock binds the lock file the control plane's lifecycle
// runner takes for agent-manager: <runtime-home state entry>/locks/
// scenario-agent-manager.lock under the lifecycle owner's home. The state
// entry comes from the repository contract's runtime_home, the authority the
// control plane's lock directory follows; it is deliberately not the
// scenario's storage-class state directory, which VROOLI_STORAGE_ROOT can move
// away from the control plane's lock.
func NewScenarioInterlock(ownerHome string) (*ScenarioInterlock, error) {
	if !filepath.IsAbs(ownerHome) {
		return nil, fmt.Errorf("maintenance interlock requires the lifecycle owner's absolute home")
	}
	stateRoot, err := repocontract.RuntimeHomeEntryPath(ownerHome, repocontract.HomeKeyState)
	if err != nil {
		return nil, fmt.Errorf("maintenance interlock: resolve runtime-home state entry: %w", err)
	}
	return &ScenarioInterlock{path: filepath.Join(stateRoot, "locks", "scenario-agent-manager.lock")}, nil
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
