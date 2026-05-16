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

// Reconfigure swaps the runtime toggles + TTLs without dropping
// in-flight requests. Per-tier availability caches are invalidated so
// the new flags take effect immediately on the next call.
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
	c.localOK = cachedAvail{}
}

// Probe returns the current per-tier availability snapshot for the
// diagnostic surface (CLI + UI). Probes are short-lived but fresh.
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

// Stream runs a streaming transcription session through the chain.
// It negotiates a streaming-capable tier at stream-start (BYOK -> Vrooli
// -> Local precedence, filtered by StreamingCapability()=true). When no
// streaming-capable tier accepts, the chain falls back to the buffered
// unary path: it drains `chunks`, concatenates the audio bytes, runs
// Execute() with the buffered audio, and emits a synthetic Segment +
// Done event sequence so consumers see a consistent event shape.
//
// The locked tier is reported on the final Done event. Mid-stream
// failover is explicitly out of scope (see plan §5 Out of scope).
//
// The returned channel is closed by Stream after emitting the final
// Done event. The caller must drain it; abandoning it without context
// cancellation will leak the streaming goroutine.
func (c *Chain) Stream(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error) {
	// Try BYOK first if a key is present and the tier is enabled.
	if c.enableBYOK && start.BYOKKey != "" && c.byok != nil && c.availFor(ctx, TierBYOK) {
		if c.byok.StreamingCapability() {
			out, err := c.byok.TranscribeStreaming(ctx, start, chunks)
			if err != nil {
				// Hard errors from adapter selection are terminal.
				if errors.Is(err, ErrUnknownBYOKProvider) || errors.Is(err, ErrMissingBYOKProvider) {
					return nil, err
				}
				// Fall through to next tier on transport errors.
			} else if out != nil {
				return out, nil
			}
		}
	}

	// Vrooli tier (declared non-streaming today; kept for symmetry).
	if c.enableVrooli && start.LPBSToken != "" && c.vrooli != nil && c.availFor(ctx, TierVrooli) && c.vrooli.StreamingCapability() {
		out, err := c.vrooli.TranscribeStreaming(ctx, start, chunks)
		if err == nil && out != nil {
			return out, nil
		}
	}

	// Local tier.
	if c.enableLocal && c.local != nil && c.availFor(ctx, TierLocal) && c.local.StreamingCapability() {
		out, err := c.local.TranscribeStreaming(ctx, start, chunks)
		if err == nil && out != nil {
			return out, nil
		}
	}

	// Fallback: buffered unary mode. Drain the channel, concatenate the
	// bytes, and run the unary chain. Emit one Segment + one Done event.
	return c.bufferedFallback(ctx, start, chunks), nil
}

// bufferedFallback consumes the chunks channel, runs the unary chain on
// the concatenated audio, and emits a single Segment + Done event.
// Stamps DoneEvent.FellBackToUnary=true so consumers can identify the
// degraded path.
func (c *Chain) bufferedFallback(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) <-chan StreamEvent {
	out := make(chan StreamEvent, 4)
	go func() {
		defer close(out)
		var buf []byte
		for {
			select {
			case <-ctx.Done():
				out <- StreamEvent{Kind: StreamEventError, Error: ctx.Err()}
				return
			case ch, ok := <-chunks:
				if !ok {
					goto run
				}
				buf = append(buf, ch.Audio...)
			}
		}
	run:
		req := Request{
			Audio:                   buf,
			Language:                start.Language,
			InitialPrompt:           start.InitialPrompt,
			SkipSpeakerVerification: start.SkipSpeakerVerification,
			BYOKProvider:            start.BYOKProvider,
			BYOKKey:                 start.BYOKKey,
			LPBSToken:               start.LPBSToken,
			UserIdentity:            start.UserIdentity,
		}
		res, err := c.Execute(ctx, req)
		if err != nil {
			out <- StreamEvent{Kind: StreamEventError, Error: err}
			out <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{FellBackToUnary: true}}
			return
		}
		out <- StreamEvent{Kind: StreamEventSegment, Segment: &SegmentEvent{
			Text:             res.Text,
			DetectedLanguage: res.DetectedLanguage,
			ProviderTier:     res.Tier,
			ProviderID:       res.ProviderID,
			ModelID:          res.ModelID,
			LatencyMs:        float64(res.Latency.Milliseconds()),
		}}
		out <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{
			FinalText:       res.Text,
			LockedTier:      res.Tier,
			ProviderID:      res.ProviderID,
			ModelID:         res.ModelID,
			LatencyMs:       float64(res.Latency.Milliseconds()),
			FellBackToUnary: true,
		}}
	}()
	return out
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
