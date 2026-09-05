package main

import (
	"os"
	"path/filepath"
	"testing"

	platform "github.com/vrooli/platform-go"
)

func TestWebConsoleCleanupRecoveryLockHonorsStorageRecoveryHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "storage-manager", "recovery.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	heldFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	heldRelease, err := platform.LockFile(heldFile, true)
	if err != nil {
		_ = heldFile.Close()
		t.Fatal(err)
	}
	defer func() {
		heldRelease()
		_ = heldFile.Close()
	}()

	h := &webConsoleCleanup{recoveryLockPath: path}
	if _, err := h.acquireRecoveryLock(); err == nil {
		t.Fatal("owner cleanup acquired a lock held by storage recovery")
	}
}
