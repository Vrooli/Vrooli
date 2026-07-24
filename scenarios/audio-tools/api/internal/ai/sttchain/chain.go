package sttchain

import (
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
	c.Coordinator = tiered.NewCredentialChain(tiered.CredentialSet[Request, *Result]{
		BYOK:        tiered.TierFor(c.byok, (*BYOKProvider).Transcribe, (*BYOKProvider).IsAvailable),
		Vrooli:      tiered.TierFor(c.vrooli, (*VrooliProvider).Transcribe, (*VrooliProvider).IsAvailable),
		Local:       tiered.TierFor(c.local, (*LocalProvider).Transcribe, (*LocalProvider).IsAvailable),
		HasBYOK:     func(req Request) bool { return req.BYOKKey != "" },
		HasVrooli:   func(req Request) bool { return req.LPBSToken != "" },
		UnknownBYOK: ErrUnknownBYOKProvider, MissingBYOK: ErrMissingBYOKProvider,
		InsufficientCredits: ErrInsufficientCredits, AllFailed: ErrAllProvidersFailed,
	}, tiered.ChainOptions{
		EnableBYOK:   opts.EnableBYOK,
		EnableVrooli: opts.EnableVrooli,
		EnableLocal:  opts.EnableLocal,
		TTLByOK:      opts.AvailTTLByOK,
		TTLVrooli:    opts.AvailTTLVrooli,
		Clock:        opts.Clock,
		OnFallback:   tiered.FallbackLogger("stt", opts.Logx),
	})
	return c
}
