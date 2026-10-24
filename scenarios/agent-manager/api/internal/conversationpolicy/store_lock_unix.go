//go:build !windows

package conversationpolicy

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// lockPolicyFile takes an exclusive advisory lock on a sidecar lock file so the
// caller's read-modify-write of the policy file is serialized against other
// processes. The returned release closure unlocks and closes the lock fd.
func lockPolicyFile(path string) (func(), error) {
	lockPath := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("prepare conversation policy lock dir: %w", err)
	}
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open conversation policy lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock conversation policy: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
