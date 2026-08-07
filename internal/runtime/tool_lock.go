package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

var toolInstallMutexes sync.Map

var toolInstallLockPathFn = func(tool string) (string, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home for host-tool lock: %w", err)
	}
	stateRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", fmt.Errorf("resolve runtime state for host-tool lock: %w", err)
	}
	return filepath.Join(stateRoot, "locks", "host-tool-"+safeToolLockName(tool)+".lock"), nil
}

var unsafeToolLockName = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

func safeToolLockName(tool string) string {
	name := unsafeToolLockName.ReplaceAllString(strings.TrimSpace(tool), "-")
	if name == "" {
		return "unknown"
	}
	return name
}

// acquireToolInstallLock serializes a managed host-tool convergence across
// concurrent vrooli processes. The process-local mutex avoids needless flock
// contention; the advisory file lock covers separate processes.
func acquireToolInstallLock(tool string) (func(), error) {
	key := strings.TrimSpace(tool)
	mutexValue, _ := toolInstallMutexes.LoadOrStore(key, &sync.Mutex{})
	mutex := mutexValue.(*sync.Mutex)
	mutex.Lock()

	path, err := toolInstallLockPathFn(key)
	if err != nil {
		mutex.Unlock()
		return nil, err
	}
	if _, err := config.EnsureOwnedDir(filepath.Dir(path)); err != nil {
		mutex.Unlock()
		return nil, fmt.Errorf("create host-tool lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		mutex.Unlock()
		return nil, fmt.Errorf("open host-tool lock %q: %w", path, err)
	}
	_ = os.Chmod(path, 0o666)
	_ = config.ChownToInvokingUser(path)
	if err := lockToolInstallFile(file); err != nil {
		_ = file.Close()
		mutex.Unlock()
		return nil, fmt.Errorf("lock host-tool %q: %w", key, err)
	}

	return func() {
		_ = unlockToolInstallFile(file)
		_ = file.Close()
		mutex.Unlock()
	}, nil
}
