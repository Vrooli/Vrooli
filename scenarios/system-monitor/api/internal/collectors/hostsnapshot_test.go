package collectors

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestCachedSnapshotProvider_OneProbePerCycle(t *testing.T) {
	var calls int32
	now := time.Unix(0, 0)
	p := &CachedSnapshotProvider{
		ttl: 5 * time.Second,
		now: func() time.Time { return now },
		collect: func(context.Context) (hostinventory.Snapshot, error) {
			atomic.AddInt32(&calls, 1)
			return hostinventory.Snapshot{OS: "linux"}, nil
		},
	}

	// Three collectors (cpu/memory/gpu) within one cycle share the provider:
	// only the first probe should hit collect.
	for i := 0; i < 3; i++ {
		if _, err := p.Snapshot(context.Background()); err != nil {
			t.Fatalf("snapshot: %v", err)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("collect called %d times within TTL, want 1", got)
	}

	// After the TTL elapses, the next probe refreshes.
	now = now.Add(6 * time.Second)
	if _, err := p.Snapshot(context.Background()); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("collect called %d times after TTL, want 2", got)
	}
}

func TestCachedSnapshotProvider_ServesStaleOnError(t *testing.T) {
	var fail bool
	now := time.Unix(0, 0)
	p := &CachedSnapshotProvider{
		ttl: time.Second,
		now: func() time.Time { return now },
		collect: func(context.Context) (hostinventory.Snapshot, error) {
			if fail {
				return hostinventory.Snapshot{}, context.DeadlineExceeded
			}
			return hostinventory.Snapshot{OS: "linux", Arch: "amd64"}, nil
		},
	}
	if _, err := p.Snapshot(context.Background()); err != nil {
		t.Fatalf("warm-up: %v", err)
	}
	now = now.Add(2 * time.Second) // expire cache
	fail = true
	snap, err := p.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("should serve stale rather than error: %v", err)
	}
	if snap.Arch != "amd64" {
		t.Fatalf("expected stale snapshot served, got %+v", snap)
	}
}
