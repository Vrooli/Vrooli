package collectors

// DOC: docs/internal/SEAMS.md#host-snapshot-provider

import (
	"context"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// SnapshotProvider returns a host-inventory snapshot. The cpu/memory/gpu
// collectors share one provider so a collection cycle probes the host
// (/proc/meminfo, /proc/loadavg, nvidia-smi) ONCE instead of three independent
// uncached hostinventory.Collect calls.
type SnapshotProvider interface {
	Snapshot(ctx context.Context) (hostinventory.Snapshot, error)
}

// CachedSnapshotProvider memoizes the most recent snapshot for a short TTL so
// collectors that fire within the same cycle (or back-to-back on-demand reads)
// reuse one probe. It is safe for concurrent use.
type CachedSnapshotProvider struct {
	ttl     time.Duration
	collect func(ctx context.Context) (hostinventory.Snapshot, error)
	now     func() time.Time

	mu       sync.Mutex
	cached   hostinventory.Snapshot
	cachedAt time.Time
	hasValue bool
}

// NewCachedSnapshotProvider builds a provider backed by the real host inventory.
// A non-positive ttl falls back to 5s — long enough to cover one collection
// cycle's collectors, short enough that on-demand reads stay fresh.
func NewCachedSnapshotProvider(ttl time.Duration) *CachedSnapshotProvider {
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	return &CachedSnapshotProvider{
		ttl:     ttl,
		collect: hostinventory.Collect,
		now:     time.Now,
	}
}

// Snapshot returns a cached snapshot when fresh, otherwise collects a new one.
func (p *CachedSnapshotProvider) Snapshot(ctx context.Context) (hostinventory.Snapshot, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.hasValue && p.now().Sub(p.cachedAt) < p.ttl {
		return p.cached, nil
	}
	snap, err := p.collect(ctx)
	if err != nil {
		// Serve a stale snapshot rather than nothing when a probe transiently
		// fails; collectors already tolerate zero-valued fields.
		if p.hasValue {
			return p.cached, nil
		}
		return snap, err
	}
	p.cached = snap
	p.cachedAt = p.now()
	p.hasValue = true
	return snap, nil
}

// defaultSnapshotProvider is the per-collector fallback used when no shared
// provider was injected (e.g. a collector constructed directly in a test). It
// caches at the same short TTL so even un-wired collectors don't re-probe on
// every call.
func defaultSnapshotProvider() SnapshotProvider {
	return NewCachedSnapshotProvider(0)
}
