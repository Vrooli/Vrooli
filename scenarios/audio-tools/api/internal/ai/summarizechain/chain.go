package summarizechain

import (
	"context"
	"errors"
	"time"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
)

// Chain composes BYOK -> Vrooli -> Local summarization providers. Unary
// Execute, Reconfigure, Probe, Eligible, and availability caching are
// inherited from the embedded *tiered.Coordinator.
type Chain struct {
	*tiered.Coordinator[Request, *Result]

	local  *LocalProvider
	byok   *BYOKProvider
	vrooli *VrooliProvider
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

	Clock clock.Clock

	// Logx, when set, receives an `event=tier_fallback` line each time
	// a request is served from a non-primary tier. See sttchain for
	// the same seam.
	Logx logx.Logger
}

func NewChain(opts Options) *Chain {
	c := &Chain{local: opts.Local, byok: opts.BYOK, vrooli: opts.Vrooli}
	c.Coordinator = tiered.NewChainFromSet(tiered.ProviderSet[Request, *Result]{
		BYOK:       sumTier(c.byok),
		Vrooli:     sumTier(c.vrooli),
		Local:      sumTier(c.local),
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
		OnFallback:   fallbackLogger("summarize", opts.Logx),
	})
	return c
}

// fallbackLogger mirrors sttchain.fallbackLogger; capability="summarize".
func fallbackLogger(capability string, lg logx.Logger) func(ctx context.Context, ev tiered.FallbackEvent) {
	if lg == nil {
		return nil
	}
	return func(_ context.Context, ev tiered.FallbackEvent) {
		lg.Printf("event=tier_fallback capability=%s from_tier=%s to_tier=%s reason=%q",
			capability, ev.From.String(), ev.To.String(), ev.Reason)
	}
}

// sumTier wraps a concrete provider as a tiered.Tier. The pointer-shaped
// type parameter avoids the interface-typed-nil pitfall.
func sumTier[T any, P interface {
	*T
	Provider
}](p P) *tiered.Tier[Request, *Result] {
	if p == nil {
		return nil
	}
	return &tiered.Tier[Request, *Result]{Execute: p.Summarize, IsAvailable: p.IsAvailable}
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
