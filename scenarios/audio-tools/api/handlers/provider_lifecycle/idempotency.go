package provider_lifecycle

import (
	"context"
	"errors"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

// idempotencyKeyHeader is the canonical idempotency request header. An
// empty value bypasses deduplication entirely (the pre-dedup behavior).
const idempotencyKeyHeader = "Idempotency-Key"

const (
	// defaultIdempotencyTTL is how long a completed result stays
	// replayable. Five minutes covers the realistic CLI/UI retry window
	// without retaining provider state observations for long.
	defaultIdempotencyTTL = 5 * time.Minute
	// defaultIdempotencyMaxEntries bounds the store's memory. Lifecycle
	// mutations are rare and operator-driven; 1024 cached responses is
	// far more than any burst a single scenario instance produces.
	defaultIdempotencyMaxEntries = 1024
)

// idempotencyKey scopes a caller-supplied Idempotency-Key to one
// operation and target. The same key reused for a different operation or
// target is a different request and must never collide. dryRun is part of
// the key as defense in depth even though dry runs currently bypass the
// store: any future path that caches them still cannot leak a dry-run
// response into a real run.
type idempotencyKey struct {
	key       string
	operation string
	target    string
	dryRun    bool
}

// idempotencyOutcome describes how a request was handled, for logging and
// tests. It is not part of the response contract.
type idempotencyOutcome string

const (
	// outcomeBypass means no cached result was used: empty key, dry run,
	// or the store was over capacity.
	outcomeBypass idempotencyOutcome = "bypass"
	// outcomeMiss means this request executed the action (leader).
	outcomeMiss idempotencyOutcome = "miss"
	// outcomeJoin means this request waited on an in-flight execution
	// started by another request with the same key.
	outcomeJoin idempotencyOutcome = "join"
	// outcomeHit means this request replayed a cached result.
	outcomeHit idempotencyOutcome = "hit"
)

// errIdempotentActionPanicked is published to joiners when the leader's
// action panics. The panic still propagates to the leader's caller.
var errIdempotentActionPanicked = errors.New("idempotent action panicked before completing")

// idempotencyEntry is one in-flight or completed execution. done is closed
// exactly once when the execution finishes; msg/err/completedAt are
// written before that close, so readers unblocked by it observe them
// without further synchronization.
type idempotencyEntry struct {
	done        chan struct{}
	msg         proto.Message
	err         error
	completedAt time.Time
}

func (e *idempotencyEntry) isDone() bool {
	select {
	case <-e.done:
		return true
	default:
		return false
	}
}

// IdempotencyOptions configures an IdempotencyStore. Zero values select
// the production defaults.
type IdempotencyOptions struct {
	// TTL bounds how long a completed result stays replayable.
	TTL time.Duration
	// MaxEntries is the hard cap on retained entries (in-flight plus
	// completed).
	MaxEntries int
	// Now is the clock seam; nil means time.Now.
	Now func() time.Time
}

// IdempotencyStore deduplicates mutating lifecycle actions. It is an
// in-memory, per-process cache: it protects against retries and concurrent
// duplicate submissions reaching this API instance, not across restarts or
// replicas.
//
// Semantics:
//   - key = (Idempotency-Key, operation, target, dryRun).
//   - single-flight: concurrent callers with the same key share the
//     leader's in-flight execution and receive a clone of its result.
//   - successes are cached for TTL; failures are never cached, so a
//     transient controller error is retryable with the same key. Joiners
//     that overlapped the failed leader still observe its error.
//   - cached messages are cloned on store and on read, so no caller can
//     mutate shared state.
type IdempotencyStore struct {
	mu         sync.Mutex
	ttl        time.Duration
	maxEntries int
	now        func() time.Time
	entries    map[idempotencyKey]*idempotencyEntry
}

// NewIdempotencyStore builds a store with the supplied options; zero
// values fall back to the production defaults (5m TTL, 1024 entries,
// wall clock).
func NewIdempotencyStore(opts IdempotencyOptions) *IdempotencyStore {
	if opts.TTL <= 0 {
		opts.TTL = defaultIdempotencyTTL
	}
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = defaultIdempotencyMaxEntries
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &IdempotencyStore{
		ttl:        opts.TTL,
		maxEntries: opts.MaxEntries,
		now:        opts.Now,
		entries:    make(map[idempotencyKey]*idempotencyEntry),
	}
}

// Do executes fn at most once per key while the entry is live.
//
// The first caller (leader) runs fn with its own context. Concurrent
// callers (joiners) block until the leader finishes, or until their own
// context is canceled, and receive a clone of the leader's message. The
// leader's cancellation is published to joiners as an error and the entry
// is released, so a later retry re-executes. Errors are never cached.
func (s *IdempotencyStore) Do(
	ctx context.Context,
	k idempotencyKey,
	fn func(context.Context) (proto.Message, error),
) (proto.Message, idempotencyOutcome, error) {
	s.mu.Lock()
	s.evictExpiredLocked(s.now())

	entry, ok := s.entries[k]
	if !ok {
		if len(s.entries) >= s.maxEntries && !s.evictOldestCompletedLocked() {
			// Everything retained is in flight. Run without registering
			// so the map stays hard-capped; duplicate requests in this
			// extreme case simply execute again.
			s.mu.Unlock()
			msg, err := fn(ctx)
			if err == nil {
				err = ctx.Err()
			}
			return cloneMessage(msg), outcomeBypass, err
		}
		entry = &idempotencyEntry{done: make(chan struct{})}
		s.entries[k] = entry
		s.mu.Unlock()
		return s.lead(ctx, k, entry, fn)
	}
	if entry.isDone() {
		msg, err := entry.msg, entry.err
		s.mu.Unlock()
		if err != nil {
			// Completed entries carrying errors are deleted; this is
			// defensive only.
			return nil, outcomeHit, err
		}
		return cloneMessage(msg), outcomeHit, nil
	}
	done := entry.done
	s.mu.Unlock()

	select {
	case <-done:
		if entry.err != nil {
			return nil, outcomeJoin, entry.err
		}
		return cloneMessage(entry.msg), outcomeJoin, nil
	case <-ctx.Done():
		// This caller gave up; the leader keeps running and the entry
		// stays reserved for callers that are still waiting.
		return nil, outcomeJoin, ctx.Err()
	}
}

// lead runs fn as the single owner of entry, then either caches the
// success or releases the entry on failure.
func (s *IdempotencyStore) lead(
	ctx context.Context,
	k idempotencyKey,
	entry *idempotencyEntry,
	fn func(context.Context) (proto.Message, error),
) (msg proto.Message, outcome idempotencyOutcome, err error) {
	completed := false
	defer func() {
		if completed {
			return
		}
		// fn panicked. Release the entry and unblock joiners before
		// letting the panic propagate to the leader's caller.
		s.mu.Lock()
		if s.entries[k] == entry {
			delete(s.entries, k)
		}
		s.mu.Unlock()
		entry.err = errIdempotentActionPanicked
		close(entry.done)
	}()

	msg, err = fn(ctx)
	if err == nil {
		// A canceled or timed-out leader must never publish a success:
		// the action may not have run to completion.
		err = ctx.Err()
	}
	if err != nil {
		s.mu.Lock()
		if s.entries[k] == entry {
			delete(s.entries, k)
		}
		s.mu.Unlock()
		entry.err = err
		completed = true
		close(entry.done)
		return nil, outcomeMiss, err
	}

	// Store a private copy so neither the leader's returned message nor a
	// later caller can mutate the cached state.
	s.mu.Lock()
	entry.msg = cloneMessage(msg)
	entry.completedAt = s.now()
	s.mu.Unlock()

	completed = true
	close(entry.done)
	return cloneMessage(entry.msg), outcomeMiss, nil
}

// evictExpiredLocked drops completed entries older than the TTL.
// In-flight entries are never evicted: joiners are waiting on them.
func (s *IdempotencyStore) evictExpiredLocked(now time.Time) {
	for k, entry := range s.entries {
		if !entry.isDone() {
			continue
		}
		if now.Sub(entry.completedAt) >= s.ttl {
			delete(s.entries, k)
		}
	}
}

// evictOldestCompletedLocked drops the oldest completed entry to make
// room, reporting whether it found one. Callers must hold s.mu.
func (s *IdempotencyStore) evictOldestCompletedLocked() bool {
	var (
		oldestKey idempotencyKey
		oldest    *idempotencyEntry
		found     bool
	)
	for k, entry := range s.entries {
		if !entry.isDone() {
			continue
		}
		if !found || entry.completedAt.Before(oldest.completedAt) {
			oldestKey, oldest, found = k, entry, true
		}
	}
	if !found {
		return false
	}
	delete(s.entries, oldestKey)
	return true
}

func cloneMessage(msg proto.Message) proto.Message {
	if msg == nil {
		return nil
	}
	return proto.Clone(msg)
}
