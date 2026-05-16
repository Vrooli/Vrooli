package sttchain

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Chain composes Local + BYOK + Vrooli providers under the fixed precedence
// BYOK -> Vrooli -> Local. ErrInsufficientCredits from Vrooli short-circuits.
type Chain struct {
	local  *LocalProvider
	byok   *BYOKProvider
	vrooli *VrooliProvider

	enableLocal  bool
	enableBYOK   bool
	enableVrooli bool

	// per-tier availability cache
	availTTLByOK   time.Duration
	availTTLVrooli time.Duration

	mu          sync.Mutex
	byokOK      cachedAvail
	vrooliOK    cachedAvail
	localOK     cachedAvail
}

type cachedAvail struct {
	value     bool
	checkedAt time.Time
}

// Options configures a chain.
type Options struct {
	Local  *LocalProvider
	BYOK   *BYOKProvider
	Vrooli *VrooliProvider

	EnableLocal  bool
	EnableBYOK   bool
	EnableVrooli bool

	AvailTTLByOK   time.Duration
	AvailTTLVrooli time.Duration
}

func NewChain(opts Options) *Chain {
	if opts.AvailTTLByOK == 0 {
		opts.AvailTTLByOK = 5 * time.Minute
	}
	if opts.AvailTTLVrooli == 0 {
		opts.AvailTTLVrooli = 30 * time.Second
	}
	return &Chain{
		local:          opts.Local,
		byok:           opts.BYOK,
		vrooli:         opts.Vrooli,
		enableLocal:    opts.EnableLocal,
		enableBYOK:     opts.EnableBYOK,
		enableVrooli:   opts.EnableVrooli,
		availTTLByOK:   opts.AvailTTLByOK,
		availTTLVrooli: opts.AvailTTLVrooli,
	}
}

// Execute runs the request through the chain.
func (c *Chain) Execute(ctx context.Context, req Request) (*Result, error) {
	var lastErr error

	if c.enableBYOK && req.BYOKKey != "" {
		if c.byok != nil && c.availFor(ctx, TierBYOK) {
			res, err := c.byok.Transcribe(ctx, req)
			if err == nil {
				return res, nil
			}
			// Provider-resolution errors are terminal — no silent fallback.
			if errors.Is(err, ErrUnknownBYOKProvider) || errors.Is(err, ErrMissingBYOKProvider) {
				return nil, err
			}
			lastErr = err
		}
	}

	if c.enableVrooli && req.LPBSToken != "" {
		if c.vrooli != nil && c.availFor(ctx, TierVrooli) {
			res, err := c.vrooli.Transcribe(ctx, req)
			if err == nil {
				return res, nil
			}
			if errors.Is(err, ErrInsufficientCredits) {
				// Hard short-circuit: do NOT fall through to Local.
				return nil, err
			}
			lastErr = err
		}
	}

	if c.enableLocal && c.local != nil {
		if c.availFor(ctx, TierLocal) {
			res, err := c.local.Transcribe(ctx, req)
			if err == nil {
				return res, nil
			}
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrAllProvidersFailed
}

func (c *Chain) availFor(ctx context.Context, tier ProviderTier) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	switch tier {
	case TierBYOK:
		if now.Sub(c.byokOK.checkedAt) < c.availTTLByOK && !c.byokOK.checkedAt.IsZero() {
			return c.byokOK.value
		}
		c.byokOK = cachedAvail{value: c.byok != nil && c.byok.IsAvailable(ctx), checkedAt: now}
		return c.byokOK.value
	case TierVrooli:
		if now.Sub(c.vrooliOK.checkedAt) < c.availTTLVrooli && !c.vrooliOK.checkedAt.IsZero() {
			return c.vrooliOK.value
		}
		c.vrooliOK = cachedAvail{value: c.vrooli != nil && c.vrooli.IsAvailable(ctx), checkedAt: now}
		return c.vrooliOK.value
	case TierLocal:
		// Local availability is cheap — probe each time for fresh signal.
		return c.local != nil && c.local.IsAvailable(ctx)
	}
	return false
}
