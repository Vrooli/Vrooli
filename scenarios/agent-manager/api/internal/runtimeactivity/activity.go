// Package runtimeactivity projects current execution observations for ordinary
// product telemetry. It does not expose admission fences or lifecycle authority.
package runtimeactivity

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/maintenance"
)

type Observer interface {
	ObserveActivity(context.Context) (maintenance.Inventory, error)
}

// Snapshot counts physically observed execution scopes, not logical run states.
// A missing ExecutingRuns is unknown, even if ObservedExecutingRuns is zero.
// LogicalQueuedRuns is independent pending/starting metadata, never work pulse.
type Snapshot struct {
	ObservedAt            time.Time
	Complete              bool
	ExecutingRuns         *uint32
	ObservedExecutingRuns uint32
	LogicalQueuedRuns     uint32
	Unknown               []string
	// PhysicalRunIDs is the complete set of positively observed run IDs when
	// Complete is true. It is telemetry provenance, not an admission input.
	PhysicalRunIDs []string
}

const (
	defaultCacheTTL = time.Minute
	refreshTimeout  = 30 * time.Second
)

// Reader caches only ordinary activity telemetry. Lifecycle and maintenance
// callers continue to use InventoryReader directly for fresh safety evidence.
type Reader struct {
	observer Observer
	ttl      time.Duration
	now      func() time.Time

	mu            sync.Mutex
	cache         Snapshot
	cacheAt       time.Time
	refreshDone   chan struct{}
	refreshResult Snapshot
}

func New(observer Observer) *Reader { return NewWithCacheTTL(observer, defaultCacheTTL) }

// NewWithCacheTTL sets the maximum age of a telemetry snapshot. Non-positive
// values use the production default, keeping refresh bounded for every reader.
func NewWithCacheTTL(observer Observer, ttl time.Duration) *Reader {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &Reader{observer: observer, ttl: ttl, now: time.Now}
}

func (r *Reader) Read(ctx context.Context) Snapshot {
	if ctx.Err() != nil {
		return unavailable(ctx.Err().Error())
	}
	if r == nil || r.observer == nil {
		return unavailable("physical execution observer unavailable")
	}
	for {
		r.mu.Lock()
		now := r.now()
		if !r.cacheAt.IsZero() && !now.Before(r.cacheAt) && now.Sub(r.cacheAt) < r.ttl {
			result := cloneSnapshot(r.cache)
			r.mu.Unlock()
			return result
		}
		if done := r.refreshDone; done != nil {
			r.mu.Unlock()
			select {
			case <-done:
				if ctx.Err() != nil {
					return unavailable(ctx.Err().Error())
				}
				r.mu.Lock()
				result := cloneSnapshot(r.refreshResult)
				r.mu.Unlock()
				return result
			case <-ctx.Done():
				return unavailable(ctx.Err().Error())
			}
		}
		done := make(chan struct{})
		r.refreshDone = done
		observer := r.observer
		r.mu.Unlock()

		// The refresh is shared telemetry work. A canceled HTTP caller stops
		// waiting without canceling the common read for other callers.
		go r.refresh(observer, done)
		select {
		case <-done:
			if ctx.Err() != nil {
				return unavailable(ctx.Err().Error())
			}
			r.mu.Lock()
			result := cloneSnapshot(r.refreshResult)
			r.mu.Unlock()
			return result
		case <-ctx.Done():
			return unavailable(ctx.Err().Error())
		}
	}
}

func (r *Reader) refresh(observer Observer, done chan struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
	defer cancel()
	result := observe(ctx, observer)
	r.mu.Lock()
	r.cache = cloneSnapshot(result)
	r.cacheAt = result.ObservedAt
	if r.cacheAt.IsZero() || r.cacheAt.After(r.now()) {
		r.cacheAt = r.now()
	}
	r.refreshResult = cloneSnapshot(result)
	r.refreshDone = nil
	close(done)
	r.mu.Unlock()
}

func observe(ctx context.Context, observer Observer) Snapshot {
	result := Snapshot{Unknown: []string{}}
	if observer == nil {
		return unavailable("physical execution observer unavailable")
	}
	inventory, err := observer.ObserveActivity(ctx)
	result.ObservedAt = inventory.ObservedAt
	for _, reason := range inventory.Unknown {
		result.addUnknown(reason)
	}
	if err != nil {
		result.addUnknown(err.Error())
	}
	if ctx.Err() != nil {
		result.addUnknown(ctx.Err().Error())
	}
	if inventory.ObservedExecutors < 0 || inventory.LogicalQueuedRuns < 0 || uint64(inventory.ObservedExecutors) > uint64(^uint32(0)) || uint64(inventory.LogicalQueuedRuns) > uint64(^uint32(0)) {
		result.addUnknown("invalid runtime observation counts")
		return result
	}
	result.ObservedExecutingRuns = uint32(inventory.ObservedExecutors)
	result.PhysicalRunIDs = append([]string(nil), inventory.PhysicalRunIDs...)
	result.LogicalQueuedRuns = uint32(inventory.LogicalQueuedRuns)
	if result.ObservedAt.IsZero() {
		result.addUnknown("physical observation timestamp unavailable")
	}
	if err == nil && !inventory.PhysicalComplete && len(result.Unknown) < 64 {
		result.addUnknown("physical execution scope coverage is incomplete")
	}
	if err == nil && inventory.PhysicalComplete && len(result.Unknown) == 0 {
		result.Complete = true
		count := result.ObservedExecutingRuns
		result.ExecutingRuns = &count
	}
	return result
}

func unavailable(reason string) Snapshot {
	result := Snapshot{Unknown: []string{}}
	result.addUnknown(reason)
	return result
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	snapshot.Unknown = append([]string{}, snapshot.Unknown...)
	snapshot.PhysicalRunIDs = append([]string{}, snapshot.PhysicalRunIDs...)
	if snapshot.ExecutingRuns != nil {
		count := *snapshot.ExecutingRuns
		snapshot.ExecutingRuns = &count
	}
	return snapshot
}

func (s *Snapshot) addUnknown(reason string) {
	if len(s.Unknown) >= 64 {
		return
	}
	reason = strings.ToValidUTF8(reason, "?")
	text := []rune(reason)
	if len(text) > 512 {
		reason = string(text[:512]) + " (truncated)"
	}
	s.Unknown = append(s.Unknown, reason)
}

func (s Snapshot) Reason() string {
	if s.Complete {
		return "Complete canonical physical execution-scope observation; queued records are not executing work."
	}
	if s.ObservedExecutingRuns > 0 {
		return fmt.Sprintf("Physical execution is present (at least %d scopes); total coverage is incomplete.", s.ObservedExecutingRuns)
	}
	return "Physical execution coverage is incomplete; zero observed is not zero activity."
}
