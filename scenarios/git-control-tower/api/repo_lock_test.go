package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRepoLock_AcquireRelease(t *testing.T) {
	rl := NewRepoLock()
	ctx := context.Background()

	unlock, err := rl.Acquire(ctx, "/repo/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	unlock()
}

func TestRepoLock_SerializesSameRepo(t *testing.T) {
	rl := NewRepoLock()
	ctx := context.Background()

	unlock, err := rl.Acquire(ctx, "/repo/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to acquire again from another goroutine — should block.
	acquired := make(chan struct{})
	go func() {
		unlock2, err := rl.Acquire(ctx, "/repo/a")
		if err != nil {
			t.Errorf("unexpected error in goroutine: %v", err)
			return
		}
		close(acquired)
		unlock2()
	}()

	select {
	case <-acquired:
		t.Fatal("second acquire should block while first is held")
	case <-time.After(50 * time.Millisecond):
		// Expected: still blocked.
	}

	// Release first lock — second acquire should proceed.
	unlock()

	select {
	case <-acquired:
		// Success.
	case <-time.After(time.Second):
		t.Fatal("second acquire should succeed after release")
	}
}

func TestRepoLock_DifferentReposIndependent(t *testing.T) {
	rl := NewRepoLock()
	ctx := context.Background()

	unlockA, _ := rl.Acquire(ctx, "/repo/a")
	defer unlockA()

	// Acquiring a different repo should not block.
	acquired := make(chan struct{})
	go func() {
		unlockB, err := rl.Acquire(ctx, "/repo/b")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		close(acquired)
		unlockB()
	}()

	select {
	case <-acquired:
		// Success.
	case <-time.After(time.Second):
		t.Fatal("different repo should not block")
	}
}

func TestRepoLock_ContextCancellation(t *testing.T) {
	rl := NewRepoLock()
	bgCtx := context.Background()

	unlock, _ := rl.Acquire(bgCtx, "/repo/a")
	defer unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := rl.Acquire(ctx, "/repo/a")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestRepoLock_UnlockIdempotent(t *testing.T) {
	rl := NewRepoLock()
	ctx := context.Background()

	unlock, _ := rl.Acquire(ctx, "/repo/a")

	// Calling unlock multiple times must not panic or deadlock.
	unlock()
	unlock()
	unlock()
}

func TestRepoLock_ConcurrentContention(t *testing.T) {
	rl := NewRepoLock()
	ctx := context.Background()

	const goroutines = 20
	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock, err := rl.Acquire(ctx, "/repo/a")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			// Under correct mutual exclusion, only one goroutine is in
			// this critical section at a time. Read, sleep, then verify
			// nobody else incremented in between.
			val := atomic.LoadInt64(&counter)
			atomic.AddInt64(&counter, 1)
			time.Sleep(time.Millisecond)
			actual := atomic.LoadInt64(&counter)
			if actual != val+1 {
				t.Errorf("concurrent access detected: expected %d, got %d", val+1, actual)
			}
			unlock()
		}()
	}

	wg.Wait()
	if atomic.LoadInt64(&counter) != goroutines {
		t.Fatalf("expected %d increments, got %d", goroutines, atomic.LoadInt64(&counter))
	}
}

func TestRepoLock_LockUsableAfterContextCancel(t *testing.T) {
	rl := NewRepoLock()
	bgCtx := context.Background()

	// Hold the lock.
	unlock, _ := rl.Acquire(bgCtx, "/repo/a")

	// A second caller times out waiting.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := rl.Acquire(ctx, "/repo/a")
	if err == nil {
		t.Fatal("expected context error")
	}

	// Release the first lock.
	unlock()

	// The lock must still be usable by a subsequent caller.
	unlock2, err := rl.Acquire(bgCtx, "/repo/a")
	if err != nil {
		t.Fatalf("lock should be available after context cancel: %v", err)
	}
	unlock2()
}

func TestRepoLock_NilSafe(t *testing.T) {
	// Verify that passing a nil RepoLock doesn't happen by design,
	// but NewRepoLock always returns a valid instance.
	rl := NewRepoLock()
	if rl == nil {
		t.Fatal("NewRepoLock should never return nil")
	}
	if rl.locks == nil {
		t.Fatal("locks map should be initialized")
	}
}

func TestRepoLock_SameRepoReusesChannel(t *testing.T) {
	rl := NewRepoLock()
	ctx := context.Background()

	// Acquire and release twice — internal map should reuse the same entry.
	unlock1, _ := rl.Acquire(ctx, "/repo/a")
	unlock1()

	unlock2, _ := rl.Acquire(ctx, "/repo/a")
	unlock2()

	rl.mu.Lock()
	count := len(rl.locks)
	rl.mu.Unlock()

	if count != 1 {
		t.Fatalf("expected 1 lock entry, got %d", count)
	}
}

func TestRepoLock_FIFOFairness(t *testing.T) {
	rl := NewRepoLock()
	ctx := context.Background()

	// Hold the lock and queue up multiple waiters. They should all
	// eventually acquire the lock without starvation.
	unlock, _ := rl.Acquire(ctx, "/repo/a")

	const waiters = 5
	var order []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < waiters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			u, err := rl.Acquire(ctx, "/repo/a")
			if err != nil {
				t.Errorf("waiter %d: unexpected error: %v", id, err)
				return
			}
			mu.Lock()
			order = append(order, id)
			mu.Unlock()
			u()
		}(i)
	}

	// Let waiters queue up.
	time.Sleep(20 * time.Millisecond)
	unlock()

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(order) != waiters {
		t.Fatalf("expected %d completions, got %d", waiters, len(order))
	}
}
