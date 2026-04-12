package network

import (
	"os"
	"path/filepath"
	"strconv"
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

func TestPruneStaleLocksRemovesOnlyDeadOwners(t *testing.T) {
	home := t.TempDir()
	stateDir := filepath.Join(home, ".vrooli", "state", "scenarios")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	stalePath := filepath.Join(stateDir, ".port_21234.lock")
	livePath := filepath.Join(stateDir, ".port_21235.lock")
	if err := os.WriteFile(stalePath, []byte("alpha:999999:1\n"), 0o644); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}
	if err := os.WriteFile(livePath, []byte("beta:"+strconv.Itoa(os.Getpid())+":1\n"), 0o644); err != nil {
		t.Fatalf("write live lock: %v", err)
	}

	cleaned, err := PruneStaleLocks(home)
	if err != nil {
		t.Fatalf("PruneStaleLocks: %v", err)
	}
	if len(cleaned) != 1 || cleaned[0].Port != 21234 {
		t.Fatalf("cleaned = %#v", cleaned)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected stale lock removal, stat err=%v", err)
	}
	if _, err := os.Stat(livePath); err != nil {
		t.Fatalf("expected live lock to remain: %v", err)
	}
}
