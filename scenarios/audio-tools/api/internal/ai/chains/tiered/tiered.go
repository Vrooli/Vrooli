// Package tiered hosts a generic three-tier (BYOK -> Vrooli -> Local)
// orchestrator shared by sttchain, ttschain, and summarizechain. Chains
// keep their tier-specific request/response shapes and streaming
// entrypoints; the unary Execute path, availability cache, Reconfigure,
// and Probe are delegated to Coordinator.
//
// seam: Coordinator is the generic chain-orchestrator seam (SEAMS.md
// row "chains/tiered.Coordinator").
package tiered

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// Slot identifies which of the three tier slots is being addressed.
type Slot int

const (
	SlotBYOK Slot = iota
	SlotVrooli
	SlotLocal
)

// String returns the canonical lowercase tier name ("byok", "vrooli", "local")
// used in fallback events, logs, and the x-audio-tools-fallback header.
func (s Slot) String() string {
	switch s {
	case SlotBYOK:
		return "byok"
	case SlotVrooli:
		return "vrooli"
	case SlotLocal:
		return "local"
	}
	return "unknown"
}

// FallbackEvent is emitted when a request is served from a tier OTHER than
// the user's first-priority (first-eligible) tier. From identifies the
// first-priority tier that was attempted but failed/declined; To identifies
// the tier that ultimately succeeded; Reason is a short error class string
// extracted from the From-tier error (e.g. "provider_unavailable", or the
// underlying error.Error() if no class is known).
type FallbackEvent struct {
	From   Slot
	To     Slot
	Reason string
}

// Tier wraps the per-tier execute + availability functions.
type Tier[Req, Resp any] struct {
	Execute     func(ctx context.Context, req Req) (Resp, error)
	IsAvailable func(ctx context.Context) bool
}

// TierFor adapts a concrete pointer provider to a tier. Keeping the provider
// pointer in the signature preserves nil semantics: a typed-nil provider is
// absent rather than a non-nil interface that fails later at execution time.
func TierFor[Provider, Req, Resp any](provider *Provider, execute func(*Provider, context.Context, Req) (Resp, error), available func(*Provider, context.Context) bool) *Tier[Req, Resp] {
	if provider == nil {
		return nil
	}
	return &Tier[Req, Resp]{
		Execute:     func(ctx context.Context, req Req) (Resp, error) { return execute(provider, ctx, req) },
		IsAvailable: func(ctx context.Context) bool { return available(provider, ctx) },
	}
}

// Options configures a Coordinator. Any nil Tier slot is skipped at
// Execute time.
type Options[Req, Resp any] struct {
	BYOK   *Tier[Req, Resp]
	Vrooli *Tier[Req, Resp]
	Local  *Tier[Req, Resp]

	EnableBYOK   bool
	EnableVrooli bool
	EnableLocal  bool

	// TTLs for the per-tier availability cache. BYOK and Vrooli are
	// cached because their IsAvailable probes hit paid/remote providers.
	// Local is never pre-probed during Execute — the actual call is the
	// source of truth (see Coordinator.available).
	TTLByOK   time.Duration
	TTLVrooli time.Duration

	// Route answers "does this request opt into the given tier?".
	// Nil means every enabled tier is tried.
	Route func(slot Slot, req Req) bool

	// IsTerminal classifies an error from the given slot as terminal
	// (no fallback to subsequent tiers). Nil means all errors fall through.
	IsTerminal func(slot Slot, err error) bool

	// AllFailed is returned by Execute when no tier was attempted.
	AllFailed error

	// Clock seam for TTL comparisons. Defaults to schedule.System().
	Clock schedule.Clock

	// OnFallback, when non-nil, is invoked whenever Execute returns a
	// successful result from a tier OTHER than the first-eligible tier
	// for the request. The callback is also invoked when a per-request
	// callback is set in the context via WithOnFallback — both fire.
	// Invocation is synchronous; do not block.
	OnFallback func(ctx context.Context, ev FallbackEvent)
}

// Coordinator orchestrates execution across BYOK -> Vrooli -> Local.
type Coordinator[Req, Resp any] struct {
	byok   *Tier[Req, Resp]
	vrooli *Tier[Req, Resp]
	local  *Tier[Req, Resp]

	route      func(slot Slot, req Req) bool
	isTerminal func(slot Slot, err error) bool
	allFailed  error
	onFallback func(ctx context.Context, ev FallbackEvent)

	clk schedule.Clock

	mu           sync.Mutex
	enableBYOK   bool
	enableVrooli bool
	enableLocal  bool
	ttlByOK      time.Duration
	ttlVrooli    time.Duration
	byokAvail    cachedAvail
	vrooliAvail  cachedAvail
}

type cachedAvail struct {
	value     bool
	checkedAt time.Time
}

func NewCoordinator[Req, Resp any](opts Options[Req, Resp]) *Coordinator[Req, Resp] {
	if opts.TTLByOK == 0 {
		opts.TTLByOK = 5 * time.Minute
	}
	if opts.TTLVrooli == 0 {
		opts.TTLVrooli = 30 * time.Second
	}
	if opts.Clock == nil {
		opts.Clock = schedule.System()
	}
	if opts.Route == nil {
		opts.Route = func(Slot, Req) bool { return true }
	}
	if opts.IsTerminal == nil {
		opts.IsTerminal = func(Slot, error) bool { return false }
	}
	if opts.AllFailed == nil {
		opts.AllFailed = errors.New("tiered: all providers failed")
	}
	return &Coordinator[Req, Resp]{
		byok:         opts.BYOK,
		vrooli:       opts.Vrooli,
		local:        opts.Local,
		enableBYOK:   opts.EnableBYOK,
		enableVrooli: opts.EnableVrooli,
		enableLocal:  opts.EnableLocal,
		ttlByOK:      opts.TTLByOK,
		ttlVrooli:    opts.TTLVrooli,
		route:        opts.Route,
		isTerminal:   opts.IsTerminal,
		allFailed:    opts.AllFailed,
		onFallback:   opts.OnFallback,
		clk:          opts.Clock,
	}
}

// Execute walks BYOK -> Vrooli -> Local. Returns the first non-error
// result; a terminal error short-circuits; otherwise records the error
// and falls through. Returns AllFailed if no tier was attempted.
func (c *Coordinator[Req, Resp]) Execute(ctx context.Context, req Req) (Resp, error) {
	var zero Resp
	var lastErr error

	// Track the first-eligible (first-priority) tier and the reason the
	// chain had to skip past it. Used to emit a fallback event when a
	// lower-priority tier succeeds.
	var (
		firstSlot   Slot
		firstReason string
		haveFirst   bool
	)

	for _, slot := range []Slot{SlotBYOK, SlotVrooli, SlotLocal} {
		if !c.Eligible(ctx, slot, req) {
			continue
		}
		if !haveFirst {
			firstSlot = slot
			haveFirst = true
		}
		r, err := c.tier(slot).Execute(ctx, req)
		if err == nil {
			if haveFirst && slot != firstSlot {
				c.emitFallback(ctx, FallbackEvent{From: firstSlot, To: slot, Reason: firstReason})
			}
			return r, nil
		}
		if c.isTerminal(slot, err) {
			return zero, err
		}
		if slot == firstSlot && firstReason == "" {
			firstReason = classifyReason(err)
		}
		lastErr = err
	}

	if lastErr != nil {
		return zero, lastErr
	}
	return zero, c.allFailed
}

// emitFallback fires the per-coordinator and per-request OnFallback hooks,
// in that order. Safe to call with nil callbacks.
func (c *Coordinator[Req, Resp]) emitFallback(ctx context.Context, ev FallbackEvent) {
	if c.onFallback != nil {
		c.onFallback(ctx, ev)
	}
	if cb := onFallbackFromContext(ctx); cb != nil {
		cb(ev)
	}
}

// classifyReason maps an error to a short stable code used as the
// `reason` field in fallback events. Currently a best-effort string;
// callers that need typed reasons should wrap their errors with codes.
func classifyReason(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Eligible reports whether the given tier should be tried for this
// request: enabled, non-nil, route-true, and currently-available.
// Exposed so chains can reuse the gate from their bespoke Stream paths.
func (c *Coordinator[Req, Resp]) Eligible(ctx context.Context, slot Slot, req Req) bool {
	c.mu.Lock()
	enabled := c.enabled(slot)
	c.mu.Unlock()
	if !enabled {
		return false
	}
	t := c.tier(slot)
	if t == nil {
		return false
	}
	if !c.route(slot, req) {
		return false
	}
	return c.available(ctx, slot)
}

// Reconfigure swaps runtime toggles + TTLs; invalidates availability caches.
func (c *Coordinator[Req, Resp]) Reconfigure(enableBYOK, enableVrooli, enableLocal bool, ttlBYOK, ttlVrooli time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enableBYOK = enableBYOK
	c.enableVrooli = enableVrooli
	c.enableLocal = enableLocal
	if ttlBYOK > 0 {
		c.ttlByOK = ttlBYOK
	}
	if ttlVrooli > 0 {
		c.ttlVrooli = ttlVrooli
	}
	c.byokAvail = cachedAvail{}
	c.vrooliAvail = cachedAvail{}
}

// ProbeResult is the per-tier availability snapshot.
type ProbeResult struct {
	BYOK   bool
	Vrooli bool
	Local  bool
}

// Probe runs a fresh availability check on every enabled, non-nil tier.
// Bypasses the cache.
func (c *Coordinator[Req, Resp]) Probe(ctx context.Context) ProbeResult {
	c.mu.Lock()
	eb, ev, el := c.enableBYOK, c.enableVrooli, c.enableLocal
	c.mu.Unlock()
	return ProbeResult{
		BYOK:   eb && c.byok != nil && c.byok.IsAvailable(ctx),
		Vrooli: ev && c.vrooli != nil && c.vrooli.IsAvailable(ctx),
		Local:  el && c.local != nil && c.local.IsAvailable(ctx),
	}
}

func (c *Coordinator[Req, Resp]) tier(slot Slot) *Tier[Req, Resp] {
	switch slot {
	case SlotBYOK:
		return c.byok
	case SlotVrooli:
		return c.vrooli
	case SlotLocal:
		return c.local
	}
	return nil
}

func (c *Coordinator[Req, Resp]) enabled(slot Slot) bool {
	switch slot {
	case SlotBYOK:
		return c.enableBYOK
	case SlotVrooli:
		return c.enableVrooli
	case SlotLocal:
		return c.enableLocal
	}
	return false
}

func (c *Coordinator[Req, Resp]) available(ctx context.Context, slot Slot) bool {
	t := c.tier(slot)
	if t == nil {
		return false
	}
	switch slot {
	case SlotLocal:
		// Local is never pre-probed: the actual Execute call is the
		// source of truth. A pre-probe adds a separate failure mode
		// (e.g. a stale capability checker disagreeing with reality)
		// that masks the underlying provider error as
		// ErrAllProvidersFailed with 0 ms latency. The real call is
		// free locally, so just attempt it and surface whatever the
		// provider returns. Tier.IsAvailable is still consulted by
		// Probe() for the health-status surface.
		return true
	case SlotBYOK:
		return c.cachedAvailability(ctx, t, &c.byokAvail, c.ttlByOK)
	case SlotVrooli:
		return c.cachedAvailability(ctx, t, &c.vrooliAvail, c.ttlVrooli)
	}
	return false
}

func (c *Coordinator[Req, Resp]) cachedAvailability(ctx context.Context, t *Tier[Req, Resp], slot *cachedAvail, ttl time.Duration) bool {
	c.mu.Lock()
	now := c.clk.Now()
	if !slot.checkedAt.IsZero() && now.Sub(slot.checkedAt) < ttl {
		v := slot.value
		c.mu.Unlock()
		return v
	}
	c.mu.Unlock()

	// IsAvailable may do I/O — call outside the lock.
	ok := t.IsAvailable(ctx)

	c.mu.Lock()
	slot.value = ok
	slot.checkedAt = now
	c.mu.Unlock()
	return ok
}
