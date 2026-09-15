package recoverylock

import (
	"errors"
	"os"
	"testing"
)

func TestAcquireRejectsConcurrentOwner(t *testing.T) {
	path := t.TempDir() + "/recovery.lock"
	release, err := AcquireFor(path, "run-123")
	if err != nil {
		t.Fatalf("Acquire first: %v", err)
	}
	defer release()
	if got := string(mustRead(t, path)); got != "run-123" {
		t.Fatalf("holder file = %q, want run-123", got)
	}
	if _, err := Acquire(path); err == nil {
		t.Fatal("second Acquire succeeded while first owner held the lock")
	} else if !errors.Is(err, ErrLockHeld) {
		t.Fatalf("second Acquire error = %v, want ErrLockHeld", err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
