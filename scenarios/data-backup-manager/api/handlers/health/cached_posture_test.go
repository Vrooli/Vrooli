package health

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type slowPosture struct {
	calls   atomic.Int32
	release chan struct{}
	detail  atomic.Value
	err     error
}

func (s *slowPosture) OverdueOrFailed(ctx context.Context) (bool, string, error) {
	s.calls.Add(1)
	if s.release != nil {
		select {
		case <-s.release:
		case <-ctx.Done():
			return false, "", ctx.Err()
		}
	}
	d, _ := s.detail.Load().(string)
	return d != "", d, s.err
}

func TestPostureProbeNeverWaitsOnTheWorkBehindIt(t *testing.T) {
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	inner := &slowPosture{}
	inner.detail.Store("39 issues")
	c := newCachedPosture(inner, 15*time.Second, 10*time.Minute, clock)

	// First answer is computed in the open.
	if d, detail, err := c.OverdueOrFailed(context.Background()); err != nil || !d || detail != "39 issues" || inner.calls.Load() != 1 {
		t.Fatalf("first = %v %q %v calls=%d", d, detail, err, inner.calls.Load())
	}
	// Within the fresh window: no work at all.
	now = now.Add(10 * time.Second)
	c.OverdueOrFailed(context.Background())
	if inner.calls.Load() != 1 {
		t.Fatalf("a fresh answer must not recompute: %d calls", inner.calls.Load())
	}

	// Past fresh: the stale answer comes back at once even though the refresh is
	// stuck behind a long backup, and only one refresh is started.
	inner.release = make(chan struct{})
	inner.detail.Store("")
	now = now.Add(time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, detail, _ := c.OverdueOrFailed(context.Background())
			if !d || detail != "39 issues" {
				t.Errorf("probe got %v %q, want the cached answer", d, detail)
			}
		}()
	}
	wg.Wait() // returned without the refresh finishing
	waitFor(t, func() bool { return inner.calls.Load() == 2 })
	close(inner.release)
	waitFor(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return !c.refreshing && !c.degraded
	})
	if inner.calls.Load() != 2 {
		t.Fatalf("five probes started %d refreshes, want one", inner.calls.Load()-1)
	}
	if d, _, _ := c.OverdueOrFailed(context.Background()); d {
		t.Fatal("the refreshed answer was not used")
	}
}

func TestPostureTooOldIsRecomputedInTheOpenAndErrorsShow(t *testing.T) {
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	inner := &slowPosture{}
	c := newCachedPosture(inner, time.Second, time.Minute, func() time.Time { return now })
	c.OverdueOrFailed(context.Background())
	inner.err = errors.New("database is locked")
	now = now.Add(2 * time.Minute)
	if _, _, err := c.OverdueOrFailed(context.Background()); err == nil {
		t.Fatal("an answer older than maxStale must be recomputed, and its error shown")
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	for i := 0; i < 400; i++ {
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not reached")
}
