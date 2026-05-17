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

	"audio-tools/internal/clock"
)

// Slot identifies which of the three tier slots is being addressed.
type Slot int

const (
	SlotBYOK Slot = iota
	SlotVrooli
	SlotLocal
)

// Tier wraps the per-tier execute + availability functions.
type Tier[Req, Resp any] struct {
	Execute     func(ctx context.Context, req Req) (Resp, error)
	IsAvailable func(ctx context.Context) bool
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

	// TTLs for the per-tier availability cache. Local is probed every
	// call (cheap, no cache); BYOK/Vrooli are cached.
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

	// Clock seam for TTL comparisons. Defaults to clock.System{}.
	Clock clock.Clock
}

// Coordinator orchestrates execution across BYOK -> Vrooli -> Local.
type Coordinator[Req, Resp any] struct {
	byok   *Tier[Req, Resp]
	vrooli *Tier[Req, Resp]
	local  *Tier[Req, Resp]

	route      func(slot Slot, req Req) bool
	isTerminal func(slot Slot, err error) bool
	allFailed  error

	clk clock.Clock

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
		opts.Clock = clock.System{}
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
		clk:          opts.Clock,
	}
}

// Execute walks BYOK -> Vrooli -> Local. Returns the first non-error
// result; a terminal error short-circuits; otherwise records the error
// and falls through. Returns AllFailed if no tier was attempted.
func (c *Coordinator[Req, Resp]) Execute(ctx context.Context, req Req) (Resp, error) {
	var zero Resp
	var lastErr error

	for _, slot := range []Slot{SlotBYOK, SlotVrooli, SlotLocal} {
		if !c.Eligible(ctx, slot, req) {
			continue
		}
		r, err := c.tier(slot).Execute(ctx, req)
		if err == nil {
			return r, nil
		}
		if c.isTerminal(slot, err) {
			return zero, err
		}
		lastErr = err
	}

	if lastErr != nil {
		return zero, lastErr
	}
	return zero, c.allFailed
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
		// Cheap probe; matches pre-consolidation ttschain/summarizechain.
		return t.IsAvailable(ctx)
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
