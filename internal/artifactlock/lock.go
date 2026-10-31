// Package artifactlock serializes convergence of one declared artifact across
// concurrent Vrooli processes. It is shared by host tools and resources so a
// second install always reinspects state under the same cross-process lock.
package artifactlock

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

var gates sync.Map

const artifactLockMode = 0o600

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// LockPath returns the per-user lock path for a logical artifact key.
func LockPath(key string) (string, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve invoking user home: %w", err)
	}
	stateRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", fmt.Errorf("resolve runtime state: %w", err)
	}
	return filepath.Join(stateRoot, "locks", "artifact-"+safeName(key)+".lock"), nil
}

// Acquire obtains the shared lock for key and returns its release function.
func Acquire(key string) (func(), error) {
	return AcquireWithPath(key, LockPath)
}

// AcquireWithPath is the injectable form used by callers' tests. The path
// callback is still responsible for selecting a user-owned location; this
// package owns directory creation, file locking, and release ordering.
func AcquireWithPath(key string, pathFn func(string) (string, error)) (func(), error) {
	return AcquireWithPathContext(context.Background(), key, pathFn)
}

// AcquireContext preserves artifact identity and ownership while allowing a
// queued caller to leave without stranding a mutex or native-lock goroutine.
func AcquireContext(ctx context.Context, key string) (func(), error) {
	return AcquireWithPathContext(ctx, key, LockPath)
}

// AcquireWithPathContext is the injectable context-aware owner boundary.
func AcquireWithPathContext(ctx context.Context, key string, pathFn func(string) (string, error)) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("artifact lock key is required")
	}
	gateValue, _ := gates.LoadOrStore(key, make(chan struct{}, 1))
	gate := gateValue.(chan struct{})
	select {
	case gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	unlock := func() { <-gate }
	path, err := pathFn(key)
	if err != nil {
		unlock()
		return nil, err
	}
	if _, err := config.EnsureOwnedDir(filepath.Dir(path)); err != nil {
		unlock()
		return nil, fmt.Errorf("create artifact lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, artifactLockMode)
	if err != nil {
		unlock()
		return nil, fmt.Errorf("open artifact lock %q: %w", path, err)
	}
	_ = os.Chmod(path, artifactLockMode)
	_ = config.ChownToInvokingUser(path)
	_ = file.Close()
	releaseFile, err := platform.AcquireFileLockContext(ctx, path)
	if err != nil {
		unlock()
		return nil, fmt.Errorf("lock artifact %q: %w", key, err)
	}
	var once sync.Once
	return func() { once.Do(func() { releaseFile(); unlock() }) }, nil
}

func safeName(key string) string {
	name := unsafeName.ReplaceAllString(strings.TrimSpace(key), "-")
	if name == "" {
		return "unknown"
	}
	return name
}
