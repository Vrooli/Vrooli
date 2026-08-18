// Package artifactlock serializes convergence of one declared artifact across
// concurrent Vrooli processes. It is shared by host tools and resources so a
// second install always reinspects state under the same cross-process lock.
package artifactlock

import (
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

var mutexes sync.Map

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
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("artifact lock key is required")
	}
	mutexValue, _ := mutexes.LoadOrStore(key, &sync.Mutex{})
	mutex := mutexValue.(*sync.Mutex)
	mutex.Lock()
	path, err := pathFn(key)
	if err != nil {
		mutex.Unlock()
		return nil, err
	}
	if _, err := config.EnsureOwnedDir(filepath.Dir(path)); err != nil {
		mutex.Unlock()
		return nil, fmt.Errorf("create artifact lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		mutex.Unlock()
		return nil, fmt.Errorf("open artifact lock %q: %w", path, err)
	}
	_ = os.Chmod(path, 0o666)
	_ = config.ChownToInvokingUser(path)
	releaseFile, err := platform.LockFile(file, false)
	if err != nil {
		_ = file.Close()
		mutex.Unlock()
		return nil, fmt.Errorf("lock artifact %q: %w", key, err)
	}
	return func() {
		releaseFile()
		_ = file.Close()
		mutex.Unlock()
	}, nil
}

func safeName(key string) string {
	name := unsafeName.ReplaceAllString(strings.TrimSpace(key), "-")
	if name == "" {
		return "unknown"
	}
	return name
}
