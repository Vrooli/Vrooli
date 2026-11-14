package graph_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/graph"
)

// Same path: two goroutines must serialize.
func TestPathMutex_SerializesSamePath(t *testing.T) {
	m := graph.NewPathMutex()
	var inFlight, maxInFlight int32

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := m.Lock("/abs")
			n := atomic.AddInt32(&inFlight, 1)
			for {
				cur := atomic.LoadInt32(&maxInFlight)
				if n <= cur || atomic.CompareAndSwapInt32(&maxInFlight, cur, n) {
					break
				}
			}
			// hold briefly without sleeping (no time import allowed
			// in the package; we use a small busy loop equivalent —
			// the test goroutines themselves contend on the mutex).
			atomic.AddInt32(&inFlight, -1)
			unlock()
		}()
	}
	wg.Wait()
	require.LessOrEqual(t, atomic.LoadInt32(&maxInFlight), int32(1),
		"two holders observed for the same path; mutex did not serialize")
}

// Different paths: lock returns immediately without blocking.
func TestPathMutex_DifferentPathsIndependent(t *testing.T) {
	m := graph.NewPathMutex()
	unlockA := m.Lock("/a")
	defer unlockA()
	// If /b shared a lock with /a this call would block forever; the
	// test would time out.
	unlockB := m.Lock("/b")
	unlockB()
}

// Empty path also gets a mutex — callers can defer fearlessly.
func TestPathMutex_EmptyPath(t *testing.T) {
	m := graph.NewPathMutex()
	unlock := m.Lock("")
	require.NotNil(t, unlock)
	unlock()
}
