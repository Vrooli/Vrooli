package graph_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	intgraph "go-code-graph/internal/graph"
)

// TestPathMutexSerializesSameKey asserts that two goroutines locking
// the same absolute path proceed in strict serial order — the second
// goroutine cannot enter the critical section until the first has
// released.
func TestPathMutexSerializesSameKey(t *testing.T) {
	t.Parallel()
	mu := intgraph.NewPathMutex()

	var inCS atomic.Int32
	var maxObserved atomic.Int32
	var wg sync.WaitGroup
	const N = 8
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := mu.Lock("/same/path")
			defer unlock()
			cur := inCS.Add(1)
			if cur > maxObserved.Load() {
				maxObserved.Store(cur)
			}
			// Brief work window so concurrency would be observable
			// if the mutex weren't doing its job.
			time.Sleep(2 * time.Millisecond)
			inCS.Add(-1)
		}()
	}
	wg.Wait()
	if got := maxObserved.Load(); got > 1 {
		t.Fatalf("same-key acquisitions ran concurrently (max observed=%d, want 1)", got)
	}
}

// TestPathMutexParallelDifferentKeys asserts that two goroutines
// locking different paths actually run concurrently — total wall-clock
// is bounded by the slower one, not the sum.
func TestPathMutexParallelDifferentKeys(t *testing.T) {
	t.Parallel()
	mu := intgraph.NewPathMutex()

	const hold = 30 * time.Millisecond
	start := time.Now()
	var wg sync.WaitGroup
	for _, key := range []string{"/a", "/b", "/c"} {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			unlock := mu.Lock(k)
			defer unlock()
			time.Sleep(hold)
		}(key)
	}
	wg.Wait()
	elapsed := time.Since(start)

	// Allow a generous margin for scheduler jitter. Three serial
	// holds would be 90ms; we expect the parallel total to be well
	// under that.
	if elapsed >= 3*hold-5*time.Millisecond {
		t.Fatalf("different-key acquisitions did not run in parallel: elapsed=%v", elapsed)
	}
}
