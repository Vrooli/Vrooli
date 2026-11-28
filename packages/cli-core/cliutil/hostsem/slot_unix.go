//go:build !windows

package hostsem

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func tryAcquireSlot(dir string, slot int) (func(), bool, error) {
	path := filepath.Join(dir, fmt.Sprintf("slot-%d.lock", slot))
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, false, fmt.Errorf("hostsem: open slot %d: %w", slot, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("hostsem: flock slot %d: %w", slot, err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, true, nil
}
