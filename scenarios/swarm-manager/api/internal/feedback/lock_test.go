package feedback

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newLockInTempDir(t *testing.T) (*Lock, string) {
	t.Helper()
	root := t.TempDir()
	resolve := func(name string) string {
		return filepath.Join(root, "initiatives", name)
	}
	return &Lock{Dir: resolve, MaxAge: time.Hour}, root
}

func TestLock_AcquireThenRelease(t *testing.T) {
	lock, _ := newLockInTempDir(t)
	h := Holder{RunID: "run-1", Purpose: "feedback"}
	if err := lock.Acquire("i", h); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	got, err := lock.Inspect("i")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if got == nil || got.RunID != "run-1" {
		t.Fatalf("unexpected holder: %+v", got)
	}
	if err := lock.Release("i", "run-1"); err != nil {
		t.Fatalf("release: %v", err)
	}
	got, err = lock.Inspect("i")
	if err != nil {
		t.Fatalf("inspect after release: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil holder after release, got %+v", got)
	}
}

func TestLock_Acquire_RejectsLiveHolder(t *testing.T) {
	lock, _ := newLockInTempDir(t)
	if err := lock.Acquire("i", Holder{RunID: "a", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}
	err := lock.Acquire("i", Holder{RunID: "b", Purpose: "feedback"})
	if err == nil {
		t.Fatalf("expected conflict, got nil")
	}
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
	var conflict *LockConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("expected LockConflict, got %T", err)
	}
	if conflict.Holder.RunID != "a" {
		t.Fatalf("conflict should expose existing holder, got %+v", conflict.Holder)
	}
}

func TestLock_AcquireOverride_Preempts(t *testing.T) {
	lock, _ := newLockInTempDir(t)
	if err := lock.Acquire("i", Holder{RunID: "a", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}
	if err := lock.AcquireOverride("i", Holder{RunID: "b", Purpose: "feedback"}); err != nil {
		t.Fatalf("override: %v", err)
	}
	got, _ := lock.Inspect("i")
	if got == nil || got.RunID != "b" {
		t.Fatalf("expected override to replace holder, got %+v", got)
	}
}

func TestLock_Release_MismatchedRunID_Noop(t *testing.T) {
	lock, _ := newLockInTempDir(t)
	if err := lock.Acquire("i", Holder{RunID: "a", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release("i", "unrelated"); err != nil {
		t.Fatalf("release mismatched: %v", err)
	}
	got, _ := lock.Inspect("i")
	if got == nil || got.RunID != "a" {
		t.Fatalf("expected holder untouched by mismatched release, got %+v", got)
	}
}

func TestLock_StaleLockIsOverwritten(t *testing.T) {
	lock, _ := newLockInTempDir(t)

	now := time.Now().UTC()
	lock.Clock = func() time.Time { return now.Add(-2 * time.Hour) }
	if err := lock.Acquire("i", Holder{RunID: "old", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}

	lock.Clock = func() time.Time { return now }
	if err := lock.Acquire("i", Holder{RunID: "new", Purpose: "feedback"}); err != nil {
		t.Fatalf("expected stale lock to be overwritten, got %v", err)
	}
	got, _ := lock.Inspect("i")
	if got == nil || got.RunID != "new" {
		t.Fatalf("expected new holder, got %+v", got)
	}
}

func TestLock_SweepStale(t *testing.T) {
	lock, _ := newLockInTempDir(t)
	now := time.Now().UTC()
	lock.Clock = func() time.Time { return now.Add(-3 * time.Hour) }
	if err := lock.Acquire("i", Holder{RunID: "old", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}

	lock.Clock = func() time.Time { return now }
	swept, err := lock.SweepStale("i")
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if !swept {
		t.Fatalf("expected sweep to clear stale lock")
	}
	got, _ := lock.Inspect("i")
	if got != nil {
		t.Fatalf("expected no holder after sweep, got %+v", got)
	}
}
