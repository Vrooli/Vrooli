package sttchain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/logx"

	"github.com/vrooli/api-core/schedule"
)

var (
	ErrUnaryModelUnconfigured = errors.New("sttchain: unary model is not configured")
	ErrUnaryModelUnavailable  = errors.New("sttchain: unary model is not serviceable")
	ErrLocalModelDisabled     = errors.New("sttchain: local model is disabled")
)

// Chain composes Local + BYOK + Vrooli providers. Production speech routing
// prefers Local, then BYOK, then Vrooli. Unary Execute, Reconfigure, Probe,
// Eligible, and availability caching are inherited from the embedded
// *tiered.Coordinator; streaming (Stream, StreamCandidates) lives in
// stream.go and reaches the typed provider pointers directly.
type Chain struct {
	*tiered.Coordinator[Request, *Result]

	local  *LocalProvider
	byok   *BYOKProvider
	vrooli *VrooliProvider

	// localEngines maps an sttengine manifest engine id to the Local-tier
	// provider that serves it on the STREAMING path. Whisper, Kyutai, and
	// sherpa-onnx are Local-tier engines; StreamCandidates resolves the right one from
	// StreamStart.EngineID. The UNARY path (Execute) always uses `local`
	// (Whisper) — only Whisper is batch-capable. Empty/unknown ids fall back
	// to `local`. The map is supplied by bootstrap (which owns both the
	// providers and the engine ids) so this package never imports sttengine
	// — that would cycle via egress.
	localEngines map[string]Provider
	batchModels  map[string]Provider
	enableLocal  bool
	localFirst   bool
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
	// Test-only engines may also be supplied here. They are never selectable
	// from the production manifest; the transport must independently gate the
	// request before putting their engine id into StreamStart.
	LocalEngines map[string]Provider
	// BatchModels maps explicit unary model ids to batch-capable providers.
	BatchModels map[string]Provider

	EnableLocal  bool
	EnableBYOK   bool
	EnableVrooli bool
	// LocalFirst selects Local -> BYOK -> Vrooli for production speech
	// routing. It is explicit so test and evaluation chains can retain their
	// historical order when needed.
	LocalFirst bool

	AvailTTLByOK   time.Duration
	AvailTTLVrooli time.Duration

	Clock schedule.Clock

	// Logx, when set, receives a structured `event=tier_fallback` line
	// each time a request is served from a tier other than the first-
	// priority tier. Nil disables the chain-level log; per-request
	// callbacks attached via tiered.WithOnFallback still fire.
	Logx logx.Logger

	// Redact names the per-request credential values scrubbed from
	// fallback reasons before they are logged or returned. Nil uses the
	// chain defaults (BYOK key and LPBS token).
	Redact func(req Request) []string
}

func NewChain(opts Options) *Chain {
	if opts.Redact == nil {
		opts.Redact = func(req Request) []string {
			return tiered.RequestCredentialValues(req.BYOKKey, req.LPBSToken)
		}
	}
	c := &Chain{
		local:        opts.Local,
		byok:         opts.BYOK,
		vrooli:       opts.Vrooli,
		localEngines: opts.LocalEngines,
		batchModels:  opts.BatchModels,
		enableLocal:  opts.EnableLocal,
		localFirst:   opts.LocalFirst,
	}
	c.Coordinator = tiered.NewCredentialChain(tiered.CredentialSet[Request, *Result]{
		BYOK:        tiered.TierFor(c.byok, (*BYOKProvider).Transcribe, (*BYOKProvider).IsAvailable),
		Vrooli:      tiered.TierFor(c.vrooli, (*VrooliProvider).Transcribe, (*VrooliProvider).IsAvailable),
		Local:       tiered.TierFor(c.local, (*LocalProvider).Transcribe, (*LocalProvider).IsAvailable),
		HasBYOK:     func(req Request) bool { return req.BYOKKey != "" },
		HasVrooli:   func(req Request) bool { return req.LPBSToken != "" },
		UnknownBYOK: ErrUnknownBYOKProvider, MissingBYOK: ErrMissingBYOKProvider,
		InsufficientCredits: ErrInsufficientCredits, AllFailed: ErrAllProvidersFailed,
		Redact: opts.Redact,
	}, tiered.ChainOptions{
		EnableBYOK:   opts.EnableBYOK,
		EnableVrooli: opts.EnableVrooli,
		EnableLocal:  opts.EnableLocal,
		LocalFirst:   opts.LocalFirst,
		TTLByOK:      opts.AvailTTLByOK,
		TTLVrooli:    opts.AvailTTLVrooli,
		Clock:        opts.Clock,
		OnFallback:   tiered.FallbackLogger("stt", opts.Logx),
	})
	return c
}

// Execute routes an explicitly selected batch model directly. Empty selection
// retains the existing BYOK/Vrooli/local tier order with Whisper as Local.
// Explicit routes fail closed and never silently substitute another engine.
func (c *Chain) Execute(ctx context.Context, req Request) (*Result, error) {
	if req.Model == "" {
		return c.Coordinator.Execute(ctx, req)
	}
	if !c.enableLocal {
		return nil, fmt.Errorf("%w: %q", ErrLocalModelDisabled, req.Model)
	}
	provider, ok := c.batchModels[req.Model]
	if !ok || provider == nil {
		return nil, fmt.Errorf("%w: %q", ErrUnaryModelUnconfigured, req.Model)
	}
	if !provider.IsAvailable(ctx) {
		return nil, fmt.Errorf("%w: %q", ErrUnaryModelUnavailable, req.Model)
	}
	return provider.Transcribe(ctx, req)
}
