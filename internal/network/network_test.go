package network

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLockFileParsesScenarioPIDAndStaleness(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".port_21234.lock")
	if err := os.WriteFile(lockPath, []byte("alpha:999999:1\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	lock, err := ReadLockFile(lockPath)
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}
	if lock.Port != 21234 || lock.Scenario != "alpha" || lock.PID != 999999 {
		t.Fatalf("lock = %#v", lock)
	}
	if !lock.Stale {
		t.Fatalf("expected stale lock, got %#v", lock)
	}
}

func TestListLocksSortsByFilename(t *testing.T) {
	home := t.TempDir()
	stateDir := filepath.Join(home, ".vrooli", "state", "scenarios")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, ".port_30000.lock"), []byte("alpha:999999:1\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, ".port_20000.lock"), []byte("beta:999999:1\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	locks, err := ListLocks(home)
	if err != nil {
		t.Fatalf("ListLocks: %v", err)
	}
	if len(locks) != 2 {
		t.Fatalf("locks = %#v", locks)
	}
	if locks[0].Port != 20000 || locks[1].Port != 30000 {
		t.Fatalf("locks not sorted = %#v", locks)
	}
}
