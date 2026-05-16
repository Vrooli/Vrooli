package ttschain

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Chain struct {
	local  *LocalProvider
	byok   *BYOKProvider
	vrooli *VrooliProvider

	enableLocal  bool
	enableBYOK   bool
	enableVrooli bool

	availTTLByOK   time.Duration
	availTTLVrooli time.Duration

	mu       sync.Mutex
	byokOK   cachedAvail
	vrooliOK cachedAvail
}

type cachedAvail struct {
	value     bool
	checkedAt time.Time
}

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
		local: opts.Local, byok: opts.BYOK, vrooli: opts.Vrooli,
		enableLocal: opts.EnableLocal, enableBYOK: opts.EnableBYOK, enableVrooli: opts.EnableVrooli,
		availTTLByOK: opts.AvailTTLByOK, availTTLVrooli: opts.AvailTTLVrooli,
	}
}

func (c *Chain) Execute(ctx context.Context, req Request) (*Result, error) {
	var lastErr error
	if c.enableBYOK && req.BYOKKey != "" && c.byok != nil && c.availFor(ctx, TierBYOK) {
		res, err := c.byok.Synthesize(ctx, req)
		if err == nil {
			return res, nil
		}
		if errors.Is(err, ErrUnknownBYOKProvider) || errors.Is(err, ErrMissingBYOKProvider) {
			return nil, err
		}
		lastErr = err
	}
	if c.enableVrooli && req.LPBSToken != "" && c.vrooli != nil && c.availFor(ctx, TierVrooli) {
		res, err := c.vrooli.Synthesize(ctx, req)
		if err == nil {
			return res, nil
		}
		if errors.Is(err, ErrInsufficientCredits) {
			return nil, err
		}
		lastErr = err
	}
	if c.enableLocal && c.local != nil && c.local.IsAvailable(ctx) {
		res, err := c.local.Synthesize(ctx, req)
		if err == nil {
			return res, nil
		}
		lastErr = err
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
		if !c.byokOK.checkedAt.IsZero() && now.Sub(c.byokOK.checkedAt) < c.availTTLByOK {
			return c.byokOK.value
		}
		c.byokOK = cachedAvail{value: c.byok != nil && c.byok.IsAvailable(ctx), checkedAt: now}
		return c.byokOK.value
	case TierVrooli:
		if !c.vrooliOK.checkedAt.IsZero() && now.Sub(c.vrooliOK.checkedAt) < c.availTTLVrooli {
			return c.vrooliOK.value
		}
		c.vrooliOK = cachedAvail{value: c.vrooli != nil && c.vrooli.IsAvailable(ctx), checkedAt: now}
		return c.vrooliOK.value
	}
	return false
}
