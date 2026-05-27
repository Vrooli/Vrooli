package sttchain

import (
	"context"
	"errors"
	"time"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
)

// Chain composes Local + BYOK + Vrooli providers under the fixed
// precedence BYOK -> Vrooli -> Local. Unary Execute, Reconfigure, Probe,
// Eligible, and availability caching are inherited from the embedded
// *tiered.Coordinator; streaming (Stream, StreamCandidates) lives in
// stream.go and reaches the typed provider pointers directly.
type Chain struct {
	*tiered.Coordinator[Request, *Result]

	local  *LocalProvider
	byok   *BYOKProvider
	vrooli *VrooliProvider

	// localEngines maps an sttengine manifest engine id to the Local-tier
	// provider that serves it on the STREAMING path. Whisper and Kyutai are
	// both Local-tier engines; StreamCandidates resolves the right one from
	// StreamStart.EngineID. The UNARY path (Execute) always uses `local`
	// (Whisper) — only Whisper is batch-capable. Empty/unknown ids fall back
	// to `local`. The map is supplied by bootstrap (which owns both the
	// providers and the engine ids) so this package never imports sttengine
	// — that would cycle via egress.
	localEngines map[string]Provider
	enableLocal  bool
}

// Options configures a chain.
type Options struct {
	Local  *LocalProvider
	BYOK   *BYOKProvider
	Vrooli *VrooliProvider

	// LocalEngines maps engine id -> Local-tier streaming provider (e.g.
	// {"whisper-local": Local, "kyutai": kyutaiProvider}). Optional; when nil
	// the streaming path uses Local for every engine id. The map values for
	// batch-only engines may point at the same *LocalProvider as Local.
	LocalEngines map[string]Provider

	EnableLocal  bool
	EnableBYOK   bool
	EnableVrooli bool

	AvailTTLByOK   time.Duration
	AvailTTLVrooli time.Duration

	Clock clock.Clock

	// Logx, when set, receives a structured `event=tier_fallback` line
	// each time a request is served from a tier other than the first-
	// priority tier. Nil disables the chain-level log; per-request
	// callbacks attached via tiered.WithOnFallback still fire.
	Logx logx.Logger
}

func NewChain(opts Options) *Chain {
	c := &Chain{
		local:        opts.Local,
		byok:         opts.BYOK,
		vrooli:       opts.Vrooli,
		localEngines: opts.LocalEngines,
		enableLocal:  opts.EnableLocal,
	}
	c.Coordinator = tiered.NewChainFromSet(tiered.ProviderSet[Request, *Result]{
		BYOK:       sttTier(c.byok),
		Vrooli:     sttTier(c.vrooli),
		Local:      sttTier(c.local),
		Route:      routeFn,
		IsTerminal: terminalFn,
		AllFailed:  ErrAllProvidersFailed,
	}, tiered.ChainOptions{
		EnableBYOK:   opts.EnableBYOK,
		EnableVrooli: opts.EnableVrooli,
		EnableLocal:  opts.EnableLocal,
		TTLByOK:      opts.AvailTTLByOK,
		TTLVrooli:    opts.AvailTTLVrooli,
		Clock:        opts.Clock,
		OnFallback:   fallbackLogger("stt", opts.Logx),
	})
	return c
}

// fallbackLogger returns a tiered.OnFallback hook that emits a structured
// log line tagged with the given capability. Returns nil when lg is nil
// so the coordinator skips invocation entirely.
func fallbackLogger(capability string, lg logx.Logger) func(ctx context.Context, ev tiered.FallbackEvent) {
	if lg == nil {
		return nil
	}
	return func(_ context.Context, ev tiered.FallbackEvent) {
		lg.Printf("event=tier_fallback capability=%s from_tier=%s to_tier=%s reason=%q",
			capability, ev.From.String(), ev.To.String(), ev.Reason)
	}
}

// sttTier wraps a concrete provider as a tiered.Tier. The pointer-shaped
// type parameter avoids the interface-typed-nil pitfall (a typed-nil
// *BYOKProvider would otherwise become a non-nil Provider interface).
func sttTier[T any, P interface {
	*T
	Provider
}](p P) *tiered.Tier[Request, *Result] {
	if p == nil {
		return nil
	}
	return &tiered.Tier[Request, *Result]{Execute: p.Transcribe, IsAvailable: p.IsAvailable}
}

func routeFn(slot tiered.Slot, req Request) bool {
	switch slot {
	case tiered.SlotBYOK:
		return req.BYOKKey != ""
	case tiered.SlotVrooli:
		return req.LPBSToken != ""
	}
	return true
}

func terminalFn(slot tiered.Slot, err error) bool {
	switch slot {
	case tiered.SlotBYOK:
		return errors.Is(err, ErrUnknownBYOKProvider) || errors.Is(err, ErrMissingBYOKProvider)
	case tiered.SlotVrooli:
		return errors.Is(err, ErrInsufficientCredits)
	}
	return false
}
