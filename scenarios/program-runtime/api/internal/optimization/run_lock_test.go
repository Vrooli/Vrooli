package optimization

import (
	"context"
	"testing"
	"time"
)

func TestRunLockSetSerializesSameRunAndReleasesKeys(t *testing.T) {
	var locks runLockSet
	unlockFirst, err := locks.lock(context.Background(), "run-a")
	if err != nil {
		t.Fatal(err)
	}

	acquired := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		unlock, err := locks.lock(context.Background(), "run-a")
		if err != nil {
			return
		}
		close(acquired)
		unlock()
		close(finished)
	}()

	select {
	case <-acquired:
		t.Fatal("second caller acquired the same run lock concurrently")
	case <-time.After(20 * time.Millisecond):
	}

	unlockFirst()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("second caller did not acquire the lock after release")
	}

	locks.mu.Lock()
	defer locks.mu.Unlock()
	if len(locks.locks) != 0 {
		t.Fatalf("run lock keys leaked after release: %v", locks.locks)
	}
}

func TestRunLockSetCanceledWaitReleasesItsReference(t *testing.T) {
	var locks runLockSet
	unlock, err := locks.lock(context.Background(), "run-a")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := locks.lock(ctx, "run-a"); err == nil {
		t.Fatal("canceled waiter unexpectedly acquired the run lock")
	}
	unlock()
	locks.mu.Lock()
	defer locks.mu.Unlock()
	if len(locks.locks) != 0 {
		t.Fatalf("canceled waiter leaked a run lock key: %v", locks.locks)
	}
}
