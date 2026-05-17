package ttschain

import (
	"context"
	"errors"
	"sync"
	"time"

	"audio-tools/internal/clock"
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

	clk clock.Clock

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

	// Clock is the wall-clock seam used for TTL comparisons.
	Clock clock.Clock
}

func NewChain(opts Options) *Chain {
	if opts.AvailTTLByOK == 0 {
		opts.AvailTTLByOK = 5 * time.Minute
	}
	if opts.AvailTTLVrooli == 0 {
		opts.AvailTTLVrooli = 30 * time.Second
	}
	clk := opts.Clock
	if clk == nil {
		clk = clock.System{}
	}
	return &Chain{
		local: opts.Local, byok: opts.BYOK, vrooli: opts.Vrooli,
		enableLocal: opts.EnableLocal, enableBYOK: opts.EnableBYOK, enableVrooli: opts.EnableVrooli,
		availTTLByOK: opts.AvailTTLByOK, availTTLVrooli: opts.AvailTTLVrooli,
		clk: clk,
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

// Reconfigure swaps toggles + TTLs at runtime; invalidates availability caches.
func (c *Chain) Reconfigure(enableBYOK, enableVrooli, enableLocal bool, ttlBYOK, ttlVrooli time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enableBYOK = enableBYOK
	c.enableVrooli = enableVrooli
	c.enableLocal = enableLocal
	if ttlBYOK > 0 {
		c.availTTLByOK = ttlBYOK
	}
	if ttlVrooli > 0 {
		c.availTTLVrooli = ttlVrooli
	}
	c.byokOK = cachedAvail{}
	c.vrooliOK = cachedAvail{}
}

type ProbeResult struct {
	Local  bool
	BYOK   bool
	Vrooli bool
}

func (c *Chain) Probe(ctx context.Context) ProbeResult {
	return ProbeResult{
		Local:  c.enableLocal && c.local != nil && c.local.IsAvailable(ctx),
		BYOK:   c.enableBYOK && c.byok != nil && c.byok.IsAvailable(ctx),
		Vrooli: c.enableVrooli && c.vrooli != nil && c.vrooli.IsAvailable(ctx),
	}
}

// Stream runs a streaming synthesis session through the chain.
// Tier negotiation mirrors Execute()'s BYOK->Vrooli->Local precedence,
// filtered by StreamingCapability()=true. When no streaming-capable
// tier accepts, the chain falls back to Execute() and emits a single
// is_final=true frame carrying the full audio bytes.
//
// The returned channel is closed after the final frame. Caller must
// drain it; abandoning it without context cancellation will leak the
// streaming goroutine.
func (c *Chain) Stream(ctx context.Context, req Request) (<-chan AudioFrame, error) {
	if c.enableBYOK && req.BYOKKey != "" && c.byok != nil && c.availFor(ctx, TierBYOK) {
		if c.byok.StreamingCapability() {
			out, err := c.byok.SynthesizeStreaming(ctx, req)
			if err != nil {
				if errors.Is(err, ErrUnknownBYOKProvider) || errors.Is(err, ErrMissingBYOKProvider) {
					return nil, err
				}
			} else if out != nil {
				return out, nil
			}
		}
	}
	if c.enableVrooli && req.LPBSToken != "" && c.vrooli != nil && c.availFor(ctx, TierVrooli) && c.vrooli.StreamingCapability() {
		out, err := c.vrooli.SynthesizeStreaming(ctx, req)
		if err == nil && out != nil {
			return out, nil
		}
	}
	if c.enableLocal && c.local != nil && c.local.IsAvailable(ctx) && c.local.StreamingCapability() {
		out, err := c.local.SynthesizeStreaming(ctx, req)
		if err == nil && out != nil {
			return out, nil
		}
	}
	return c.bufferedFallback(ctx, req), nil
}

// bufferedFallback runs Execute() once and emits a single is_final=true
// frame carrying the full audio. Used when no tier declares streaming.
func (c *Chain) bufferedFallback(ctx context.Context, req Request) <-chan AudioFrame {
	out := make(chan AudioFrame, 1)
	go func() {
		defer close(out)
		res, err := c.Execute(ctx, req)
		if err != nil {
			out <- AudioFrame{IsFinal: true, Err: err}
			return
		}
		out <- AudioFrame{
			Audio:       res.Audio,
			ContentType: res.ContentType,
			IsFinal:     true,
			Tier:        res.Tier,
			ProviderID:  res.ProviderID,
			ModelID:     res.ModelID,
			VoiceUsed:   res.VoiceUsed,
			Latency:     res.Latency,
			ContentHash: res.ContentHash,
		}
	}()
	return out
}

func (c *Chain) availFor(ctx context.Context, tier ProviderTier) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	clk := c.clk
	if clk == nil {
		clk = clock.System{}
	}
	now := clk.Now()
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
