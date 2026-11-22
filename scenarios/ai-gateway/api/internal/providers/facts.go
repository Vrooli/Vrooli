package providers

import (
	"context"
	"strings"
	"sync"
	"time"
)

// FactLifetime bounds how a resource's policy and inventory facts are reused.
// Facts are read-only answers (`policy roles`, `policy resolve`,
// `models inventory`) that cost one resource process each; without reuse every
// routed request paid several of them before the provider was even called.
//
// Reuse cannot widen authority. Every execution command carries the exact
// model, artifact digest, rights receipt and policy-file hash the gateway
// planned with, and the resource re-verifies all of them against its current
// state before it calls the provider. A stale fact therefore makes a call fail
// closed; callers drop the provider's facts after a failed execution so the
// next request plans from fresh ones.
type FactLifetime struct {
	// RefreshAfter is the age past which a served fact is re-read in the
	// background, so the next request sees the resource's current answer.
	RefreshAfter time.Duration
	// MaxAge is the age past which a fact is no longer served and a request
	// waits for a fresh read. It bounds how long a resource change can go
	// unseen by planning when no execution fails to reveal it.
	MaxAge time.Duration
	// KeepWarmFor is how long after its last use a fact is re-read in the
	// background before it reaches MaxAge (see KeepWarm), so a caller that
	// returns after an idle gap still plans without waiting on resources.
	KeepWarmFor time.Duration
}

// DefaultFactLifetime refreshes facts at most every 15 seconds of traffic,
// never plans from one older than five minutes, and keeps a fact used in the
// last hour from expiring. Callers such as Source Ledger send an embedding
// every minute or so, with longer quiet spells; without keep-warm the first
// call after one paid every resource read again (2-5 s on a loaded host). A
// kept-warm fact costs one resource process per MaxAge.
var DefaultFactLifetime = FactLifetime{RefreshAfter: 15 * time.Second, MaxAge: 5 * time.Minute, KeepWarmFor: time.Hour}

// factFailureTTL is how long a failed fact read is remembered. A resource that
// hangs or is missing would otherwise cost every request its full command
// timeout; remembering the failure briefly bounds that to one wait per window
// while still noticing recovery within seconds.
const factFailureTTL = 5 * time.Second

// FactCache is a CommandRunner that reuses the results of fact commands within
// a FactLifetime. Commands not marked Fact always run. Concurrent requests for
// the same missing fact share one resource process, and a fact past its
// refresh age is re-read in the background while the current value keeps
// serving, so traffic only waits on a fact read when none is held.
type FactCache struct {
	inner    CommandRunner
	lifetime FactLifetime
	clock    func() time.Time

	mu       sync.Mutex
	entries  map[string]factEntry
	inflight map[string]*factCall
	// generations counts invalidations per resource command, so a fetch that
	// started before an invalidation cannot store what it read.
	generations map[string]uint64
	// reads counts stored fact reads per resource command; it is what a
	// FactMark measures.
	reads map[string]FactMark
	// commands and used record, per fact, the command that reads it and when
	// a caller last relied on it; KeepWarm re-reads recently used facts.
	commands map[string]Command
	used     map[string]time.Time
}

type factWarmingKey struct{}

// WithFactWarming marks reads made only to fill the cache, such as start-up
// warming. They store facts but do not count as use, so a fact nobody asks
// for is not kept warm.
func WithFactWarming(ctx context.Context) context.Context {
	return context.WithValue(ctx, factWarmingKey{}, true)
}

func isFactWarming(ctx context.Context) bool {
	warming, _ := ctx.Value(factWarmingKey{}).(bool)
	return warming
}

// FactMark is how far one resource's fact reads had progressed at some
// moment. Taken before a request plans, it tells whether the request planned
// from facts read before it began (reused) or only from facts it read itself.
type FactMark uint64

// AnyFact is a mark later than every read.
const AnyFact = FactMark(^uint64(0))

type factEntry struct {
	result  Result
	err     error
	fetched time.Time
	read    FactMark
}

// servable is how long an entry may be served after it was read.
func (c *FactCache) servable(entry factEntry) time.Duration {
	if entry.err != nil {
		return min(factFailureTTL, c.lifetime.MaxAge)
	}
	return c.lifetime.MaxAge
}

type factCall struct {
	done   chan struct{}
	result Result
	err    error
}

var _ CommandRunner = (*FactCache)(nil)

// FactCacheOption configures a FactCache.
type FactCacheOption func(*FactCache)

// WithFactClock sets the time source fact lifetimes are measured against.
func WithFactClock(clock func() time.Time) FactCacheOption {
	return func(c *FactCache) {
		if clock != nil {
			c.clock = clock
		}
	}
}

// NewFactCache wraps inner. A non-positive MaxAge disables reuse; a
// RefreshAfter outside (0, MaxAge) refreshes only on expiry.
func NewFactCache(inner CommandRunner, lifetime FactLifetime, opts ...FactCacheOption) *FactCache {
	if inner == nil {
		inner = ExecRunner{}
	}
	if lifetime.RefreshAfter <= 0 || lifetime.RefreshAfter > lifetime.MaxAge {
		lifetime.RefreshAfter = lifetime.MaxAge
	}
	c := &FactCache{inner: inner, lifetime: lifetime, clock: time.Now, entries: map[string]factEntry{}, inflight: map[string]*factCall{}, generations: map[string]uint64{}, reads: map[string]FactMark{}, commands: map[string]Command{}, used: map[string]time.Time{}}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func factKey(command Command) string {
	return command.Name + "\x00" + strings.Join(command.Args, "\x00") + "\x00" + command.Stdin
}

func (c *FactCache) Run(ctx context.Context, command Command) (Result, error) {
	if !command.Fact || c.lifetime.MaxAge <= 0 {
		return c.inner.Run(ctx, command)
	}
	key := factKey(command)
	now := c.clock()
	c.mu.Lock()
	c.commands[key] = command
	if !isFactWarming(ctx) {
		c.used[key] = now
	}
	entry, cached := c.entries[key]
	age := now.Sub(entry.fetched)
	if cached && age < c.servable(entry) {
		if entry.err == nil && age >= c.lifetime.RefreshAfter {
			c.startLocked(ctx, key, command)
		}
		c.mu.Unlock()
		return entry.result, entry.err
	}
	call := c.startLocked(ctx, key, command)
	c.mu.Unlock()
	select {
	case <-call.done:
		return call.result, call.err
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

// keepWarmInterval is how often KeepWarm looks for facts nearing MaxAge.
func (c *FactCache) keepWarmInterval() time.Duration {
	return max(c.lifetime.MaxAge/10, time.Second)
}

// KeepWarm re-reads, in the background, every fact used within KeepWarmFor
// that is about to reach MaxAge, until ctx ends. Facts nobody has used for
// KeepWarmFor are left to expire.
func (c *FactCache) KeepWarm(ctx context.Context) {
	if c.lifetime.MaxAge <= 0 || c.lifetime.KeepWarmFor <= 0 {
		return
	}
	ticker := time.NewTicker(c.keepWarmInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.refreshNearingExpiry()
		}
	}
}

// refreshNearingExpiry starts a re-read of every recently used good fact that
// would reach MaxAge before the next two KeepWarm passes.
func (c *FactCache) refreshNearingExpiry() {
	now := c.clock()
	horizon := c.lifetime.MaxAge - 2*c.keepWarmInterval()
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.entries {
		if entry.err != nil || now.Sub(entry.fetched) < horizon || now.Sub(c.used[key]) >= c.lifetime.KeepWarmFor {
			continue
		}
		if command, ok := c.commands[key]; ok {
			c.startLocked(WithFactWarming(context.Background()), key, command)
		}
	}
}

// startLocked joins the in-flight fetch for key or starts one. The fetch runs
// detached from any one caller's cancellation, bounded by the command's own
// timeout, so a caller that gives up never fails the others waiting on it.
func (c *FactCache) startLocked(ctx context.Context, key string, command Command) *factCall {
	if call, ok := c.inflight[key]; ok {
		return call
	}
	call := &factCall{done: make(chan struct{})}
	c.inflight[key] = call
	generation := c.generations[command.Name]
	go func() {
		result, err := c.inner.Run(context.WithoutCancel(ctx), command)
		c.mu.Lock()
		delete(c.inflight, key)
		if c.generations[command.Name] == generation {
			c.entries[key] = c.storedEntry(command.Name, c.entries[key], result, err)
		}
		c.mu.Unlock()
		call.result, call.err = result, err
		close(call.done)
	}()
	return call
}

// storedEntry is what a finished fetch leaves behind. A failed refresh of a
// fact that may still be served keeps the good value; a failure with nothing
// good to keep is remembered for factFailureTTL.
func (c *FactCache) storedEntry(commandName string, previous factEntry, result Result, err error) factEntry {
	now := c.clock()
	if err == nil {
		c.reads[commandName]++
		return factEntry{result: result, fetched: now, read: c.reads[commandName]}
	}
	if previous.err == nil && !previous.fetched.IsZero() && now.Sub(previous.fetched) < c.lifetime.MaxAge {
		return previous
	}
	return factEntry{result: result, err: err, fetched: now}
}

// Mark returns how far one resource command's fact reads have progressed.
func (c *FactCache) Mark(commandName string) FactMark {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads[commandName]
}

// Invalidate drops every fact of one resource command and reports whether a
// dropped good fact had been read at or before mark, that is whether a caller
// that took mark before planning may have planned from a reused fact. A fetch
// already in flight finishes for its waiters but is not stored.
func (c *FactCache) Invalidate(commandName string, mark FactMark) bool {
	prefix := commandName + "\x00"
	c.mu.Lock()
	defer c.mu.Unlock()
	reused := false
	for key, entry := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
			reused = reused || (entry.err == nil && entry.read <= mark)
		}
	}
	c.generations[commandName]++
	return reused
}

// factHolder is implemented by runners that hold reusable facts.
type factHolder interface {
	Mark(commandName string) FactMark
	Invalidate(commandName string, mark FactMark) bool
}

func (a Adapter) factHolder() (factHolder, bool) {
	holder, ok := a.Runner.(factHolder)
	return holder, ok && strings.TrimSpace(a.CommandName) != ""
}

// FactMark marks how far this adapter's resource fact reads have progressed;
// take it before planning a request.
func (a Adapter) FactMark() FactMark {
	if holder, ok := a.factHolder(); ok {
		return holder.Mark(a.CommandName)
	}
	return 0
}

// InvalidateFacts drops the reusable facts this adapter's resource reported
// and reports whether any had been read at or before mark. Callers use it
// after an execution failure: the failure may be the resource refusing
// evidence planned from a fact that has since changed, which is only possible
// when the plan reused a fact read before the request began.
func (a Adapter) InvalidateFacts(mark FactMark) bool {
	holder, ok := a.factHolder()
	if !ok {
		return false
	}
	return holder.Invalidate(a.CommandName, mark)
}

// warmParallelism bounds the resource processes start-up warming runs at once,
// so warming a cold gateway does not crowd the host it shares.
const warmParallelism = 4

// WarmFacts reads the facts a first request would otherwise wait for: every
// adapter's role inventory and the resolution of each local role. Reads run in
// parallel and count as warming, not use, so facts nobody needs still expire.
// Adapters without a fact cache are skipped.
func WarmFacts(ctx context.Context, adapters []Adapter) {
	ctx = WithFactWarming(ctx)
	slots := make(chan struct{}, warmParallelism)
	var wg sync.WaitGroup
	run := func(read func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			read()
		}()
	}
	for _, adapter := range adapters {
		if _, ok := adapter.factHolder(); !ok {
			continue
		}
		run(func() {
			inventory, err := adapter.ListRoles(ctx)
			if err != nil {
				return
			}
			for _, role := range inventory.Roles {
				if strings.EqualFold(role.Locality, "local") {
					run(func() { _, _ = adapter.ResolveRole(ctx, role.Role) })
				}
			}
		})
	}
	wg.Wait()
}
