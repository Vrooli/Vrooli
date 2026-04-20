package ports

import (
	"testing"
)

// newManagerForLockTests constructs a Manager with the minimum surrounding
// state the production constructor insists on (port registry). Using the
// shared test helpers would pull in a lot of postgres fixture plumbing that
// these tiny tests do not need.
func newManagerForLockTests(t *testing.T) *Manager {
	t.Helper()
	root := t.TempDir()
	writePortRegistry(t, root, nil)
	m, err := NewManager(root, t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return m
}

func TestConfirmLock_UpdatesPID(t *testing.T) {
	m := newManagerForLockTests(t)
	if err := m.WriteLock(21234, "swarm-manager", 1000); err != nil {
		t.Fatalf("WriteLock: %v", err)
	}

	if err := m.ConfirmLock(21234, "swarm-manager", 42); err != nil {
		t.Fatalf("ConfirmLock: %v", err)
	}
	lock, exists, err := m.ReadLock(21234)
	if err != nil || !exists {
		t.Fatalf("lock missing: exists=%v err=%v", exists, err)
	}
	if lock.PID != 42 {
		t.Errorf("PID = %d, want 42", lock.PID)
	}
	if lock.Scenario != "swarm-manager" {
		t.Errorf("Scenario = %q, want swarm-manager", lock.Scenario)
	}
}

func TestConfirmLock_NoOpIfScenarioMismatch(t *testing.T) {
	m := newManagerForLockTests(t)
	if err := m.WriteLock(21234, "other", 1000); err != nil {
		t.Fatalf("WriteLock: %v", err)
	}
	if err := m.ConfirmLock(21234, "swarm-manager", 42); err != nil {
		t.Fatalf("ConfirmLock: %v", err)
	}
	lock, _, _ := m.ReadLock(21234)
	if lock.Scenario != "other" || lock.PID != 1000 {
		t.Errorf("lock should be untouched: %+v", lock)
	}
}

func TestConfirmLock_NoOpIfLockMissing(t *testing.T) {
	m := newManagerForLockTests(t)
	if err := m.ConfirmLock(21234, "swarm-manager", 42); err != nil {
		t.Fatalf("ConfirmLock: %v", err)
	}
	_, exists, _ := m.ReadLock(21234)
	if exists {
		t.Errorf("lock should not have been created")
	}
}

func TestAbandonLock_RemovesOwnedLock(t *testing.T) {
	m := newManagerForLockTests(t)
	if err := m.WriteLock(21234, "swarm-manager", 1000); err != nil {
		t.Fatalf("WriteLock: %v", err)
	}
	if err := m.AbandonLock(21234, "swarm-manager"); err != nil {
		t.Fatalf("AbandonLock: %v", err)
	}
	_, exists, _ := m.ReadLock(21234)
	if exists {
		t.Errorf("lock should have been removed")
	}
}

func TestAbandonLock_LeavesOtherScenarioLock(t *testing.T) {
	m := newManagerForLockTests(t)
	if err := m.WriteLock(21234, "other", 1000); err != nil {
		t.Fatalf("WriteLock: %v", err)
	}
	if err := m.AbandonLock(21234, "swarm-manager"); err != nil {
		t.Fatalf("AbandonLock: %v", err)
	}
	lock, exists, _ := m.ReadLock(21234)
	if !exists || lock.Scenario != "other" {
		t.Errorf("other scenario's lock should be untouched: %+v (exists=%v)", lock, exists)
	}
}

func TestAbandonLock_MissingIsNoOp(t *testing.T) {
	m := newManagerForLockTests(t)
	if err := m.AbandonLock(21234, "swarm-manager"); err != nil {
		t.Errorf("AbandonLock on missing lock should be no-op: %v", err)
	}
}
