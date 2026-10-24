package demand

import (
	"context"
	"strings"
	"sync"
)

// SharedHolder shares one renewable lease among concurrent callers that target
// the same scenario.
//
// A lease is acquired through the control-plane CLI, which means a process
// spawn and a write to the runtime registry. A caller that fans out N reads at
// one scenario therefore issued N concurrent writes to the same SQLite file,
// and the losers surfaced as binding failures indistinguishable from the target
// being down. The lease's natural scope is the caller's unit of work, not each
// individual request: holding one scenario in demand N times concurrently is
// the same statement made N times.
//
// Sharing is deliberately limited to overlapping callers. Sequential callers
// each take their own lease, so nothing is held after the last user releases
// it. That keeps the contention fix from turning into a lease that outlives the
// work it was protecting.
type SharedHolder struct {
	client LeaseClient
	reason string

	mu   sync.Mutex
	held map[string]*sharedEntry
}

type sharedEntry struct {
	ready chan struct{}
	hold  *Hold
	err   error
	refs  int
}

// NewSharedHolder returns a holder over client. A nil client yields a nil
// holder so lease-free callers keep working unchanged.
func NewSharedHolder(client LeaseClient, reason string) *SharedHolder {
	if client == nil {
		return nil
	}
	return &SharedHolder{client: client, reason: reason, held: map[string]*sharedEntry{}}
}

func sharedKey(req AcquireRequest) string {
	return strings.Join([]string{req.ConsumerID, req.Scenario, req.Variant}, "\x00")
}

// Acquire returns a hold on the lease for req's scenario, creating it if no
// live one exists and joining the existing one otherwise. Every successful
// Acquire must be matched by exactly one Close.
func (h *SharedHolder) Acquire(ctx context.Context, req AcquireRequest) (*SharedHold, error) {
	if h == nil {
		return nil, nil
	}
	key := sharedKey(req)
	// A lease can die between the map lookup and the join (renewal failure).
	// Retry rather than hand a caller a dead hold; the bound stops a pathological
	// target from spinning here.
	for attempt := 0; attempt < 3; attempt++ {
		entry, owner := h.reserve(key)
		if owner {
			// The hold must outlive the caller that happened to create it, so it
			// is not parented to that caller's context. Its lifetime is the
			// refcount, plus renewal success.
			hold, err := AcquireLifetime(context.WithoutCancel(ctx), h.client, req, h.reason)
			entry.hold, entry.err = hold, err
			close(entry.ready)
			if err != nil {
				h.release(key, entry)
				return nil, err
			}
			return h.attach(ctx, key, entry), nil
		}

		select {
		case <-entry.ready:
		case <-ctx.Done():
			h.release(key, entry)
			return nil, ctx.Err()
		}
		if entry.err != nil {
			h.release(key, entry)
			return nil, entry.err
		}
		if entry.hold.Context().Err() == nil {
			return h.attach(ctx, key, entry), nil
		}
		// The shared lease lost its renewal. Drop this reference and let the
		// next pass create a fresh one.
		h.release(key, entry)
	}
	return nil, errLeaseUnrenewable
}

// reserve takes a reference, reporting whether this caller must create the hold.
//
// A lease whose renewal has failed is replaced rather than joined. Its existing
// holders keep their reference and release it as they finish — release only
// deletes the map entry when it is still the current one — so a dead lease can
// never stall the callers that come after it.
func (h *SharedHolder) reserve(key string) (*sharedEntry, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if entry, ok := h.held[key]; ok && entry.live() {
		entry.refs++
		return entry, false
	}
	entry := &sharedEntry{ready: make(chan struct{}), refs: 1}
	h.held[key] = entry
	return entry, true
}

// live reports whether the entry is still usable: either still being acquired,
// or holding a lease that is still renewable.
func (e *sharedEntry) live() bool {
	select {
	case <-e.ready:
	default:
		return true // still acquiring; the waiter re-checks after ready closes
	}
	return e.err == nil && e.hold != nil && e.hold.Context().Err() == nil
}

// release drops one reference and closes the hold when the last user leaves.
func (h *SharedHolder) release(key string, entry *sharedEntry) error {
	h.mu.Lock()
	entry.refs--
	last := entry.refs <= 0
	if last && h.held[key] == entry {
		delete(h.held, key)
	}
	h.mu.Unlock()
	if !last || entry.hold == nil {
		return nil
	}
	// Close waits for a bounded release; never hold the map lock across it.
	return entry.hold.Close()
}

func (h *SharedHolder) attach(ctx context.Context, key string, entry *sharedEntry) *SharedHold {
	// Each caller gets its own context: cancelled by its own work, and also by
	// the shared lease dying. One caller finishing must never cancel a sibling,
	// which is why this is a per-caller child rather than the shared context.
	callerCtx, cancel := context.WithCancelCause(ctx)
	shared := &SharedHold{Lease: entry.hold.Lease, ctx: callerCtx, cancel: cancel, holder: h, key: key, entry: entry, stop: make(chan struct{})}
	leaseCtx := entry.hold.Context()
	go func() {
		select {
		case <-leaseCtx.Done():
			cancel(context.Cause(leaseCtx))
		case <-callerCtx.Done():
		case <-shared.stop:
		}
	}()
	return shared
}

// Len reports how many distinct leases are held. Tests use it to prove that a
// fan-out at one scenario takes one lease rather than one per request.
func (h *SharedHolder) Len() int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.held)
}

// SharedHold is one caller's reference to a shared lease.
type SharedHold struct {
	Lease Lease

	ctx    context.Context
	cancel context.CancelCauseFunc
	holder *SharedHolder
	key    string
	entry  *sharedEntry
	stop   chan struct{}
	once   sync.Once
	err    error
}

// Context is cancelled when this caller's work is cancelled or when the shared
// lease stops being renewable.
func (s *SharedHold) Context() context.Context {
	if s == nil {
		return context.Background()
	}
	return s.ctx
}

// Close drops this caller's reference. It is safe to call more than once, and
// only the last outstanding reference releases the underlying lease.
func (s *SharedHold) Close() error {
	if s == nil {
		return nil
	}
	s.once.Do(func() {
		close(s.stop)
		s.cancel(context.Canceled)
		s.err = s.holder.release(s.key, s.entry)
	})
	return s.err
}

type leaseError string

func (e leaseError) Error() string { return string(e) }

const errLeaseUnrenewable = leaseError("demand lease could not be kept renewable")

// IsContention reports whether err is the control plane refusing a lease
// because another writer held the runtime registry, rather than the target
// scenario being unavailable. Callers that report source health need the
// difference: contention is self-inflicted and retryable, an unreachable
// scenario is not.
func IsContention(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{"database is locked", "table in the database is locked", "sqlite_busy", "(517)"} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
