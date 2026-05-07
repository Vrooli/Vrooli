package hostsem

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewRejectsZeroSlots(t *testing.T) {
	if _, err := New(t.TempDir(), 0); err == nil {
		t.Fatal("expected error for slots=0")
	}
}

func TestAcquireRespectsSlotCount(t *testing.T) {
	const slots = 3
	const goroutines = 12

	s, err := New(t.TempDir(), slots)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var (
		current int32
		peak    int32
		wg      sync.WaitGroup
	)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := s.Acquire(context.Background())
			if err != nil {
				t.Errorf("Acquire: %v", err)
				return
			}
			n := atomic.AddInt32(&current, 1)
			for {
				p := atomic.LoadInt32(&peak)
				if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt32(&current, -1)
			release()
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&peak); got > slots {
		t.Fatalf("peak concurrent holders = %d, want <= %d", got, slots)
	}
	if got := atomic.LoadInt32(&peak); got < 1 {
		t.Fatalf("peak concurrent holders = %d, want >= 1 (no acquisition observed)", got)
	}
}

func TestAcquireHonorsContextCancel(t *testing.T) {
	s, err := New(t.TempDir(), 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	hold, err := s.Acquire(context.Background())
	if err != nil {
		t.Fatalf("priming Acquire: %v", err)
	}
	defer hold()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := s.Acquire(ctx); err == nil {
		t.Fatal("expected ctx deadline error from second Acquire")
	}
}

func TestReleaseAllowsReacquisition(t *testing.T) {
	s, err := New(t.TempDir(), 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r1, err := s.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	r1()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r2, err := s.Acquire(ctx)
	if err != nil {
		t.Fatalf("re-Acquire after release: %v", err)
	}
	r2()
}
