package discovery

import (
	"context"
	"sync"
	"time"
)

// CachedTargetSourceScanner coalesces concurrent scans and keeps a short-lived
// snapshot of the expensive, read-only source inventory. Catalog and dismissal
// filtering still runs for every request, so registrations and dismissals are
// visible immediately; only newly discovered filesystem entries wait for the
// refresh window.
type CachedTargetSourceScanner struct {
	source TargetSourceScanner
	ttl    time.Duration
	now    func() time.Time

	mu        sync.Mutex
	cached    []TargetCandidate
	fetchedAt time.Time
	valid     bool
	loading   bool
	ready     chan struct{}
}

// NewCachedTargetSourceScanner wraps source with a bounded freshness window.
// A non-positive TTL disables reuse while retaining request coalescing.
func NewCachedTargetSourceScanner(source TargetSourceScanner, ttl time.Duration) *CachedTargetSourceScanner {
	return &CachedTargetSourceScanner{
		source: source,
		ttl:    ttl,
		now:    time.Now,
	}
}

var _ TargetSourceScanner = (*CachedTargetSourceScanner)(nil)

func (s *CachedTargetSourceScanner) Scan(ctx context.Context) ([]TargetCandidate, error) {
	for {
		s.mu.Lock()
		now := s.now()
		if s.valid && s.ttl > 0 && now.Sub(s.fetchedAt) < s.ttl {
			got := append([]TargetCandidate(nil), s.cached...)
			s.mu.Unlock()
			return got, nil
		}
		if s.loading {
			ready := s.ready
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ready:
				continue
			}
		}
		s.loading = true
		s.ready = make(chan struct{})
		ready := s.ready
		s.mu.Unlock()

		got, err := s.source.Scan(ctx)

		s.mu.Lock()
		if err == nil {
			s.cached = append([]TargetCandidate(nil), got...)
			s.fetchedAt = s.now()
			s.valid = true
		}
		s.loading = false
		close(ready)
		s.mu.Unlock()
		return got, err
	}
}
