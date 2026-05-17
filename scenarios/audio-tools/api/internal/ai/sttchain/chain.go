package sttchain

import (
	"context"
	"errors"
	"time"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/clock"
)

// Chain composes Local + BYOK + Vrooli providers under the fixed precedence
// BYOK -> Vrooli -> Local. ErrInsufficientCredits from Vrooli short-circuits.
//
// Unary Execute + Reconfigure + Probe + availability caching are delegated
// to *tiered.Coordinator. Streaming (Stream, StreamCandidates) lives in
// stream.go because it touches per-provider Traits.
type Chain struct {
	coord *tiered.Coordinator[Request, *Result]

	local  *LocalProvider
	byok   *BYOKProvider
	vrooli *VrooliProvider
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

	// Clock is the wall-clock seam used for availability-cache TTL
	// comparisons. Defaults to clock.System{}.
	Clock clock.Clock
}

func NewChain(opts Options) *Chain {
	c := &Chain{local: opts.Local, byok: opts.BYOK, vrooli: opts.Vrooli}
	c.coord = tiered.NewCoordinator(tiered.Options[Request, *Result]{
		BYOK:         byokTier(opts.BYOK),
		Vrooli:       vrooliTier(opts.Vrooli),
		Local:        localTier(opts.Local),
		EnableBYOK:   opts.EnableBYOK,
		EnableVrooli: opts.EnableVrooli,
		EnableLocal:  opts.EnableLocal,
		TTLByOK:      opts.AvailTTLByOK,
		TTLVrooli:    opts.AvailTTLVrooli,
		Route:        routeFn,
		IsTerminal:   terminalFn,
		AllFailed:    ErrAllProvidersFailed,
		Clock:        opts.Clock,
	})
	return c
}

func byokTier(p *BYOKProvider) *tiered.Tier[Request, *Result] {
	if p == nil {
		return nil
	}
	return &tiered.Tier[Request, *Result]{Execute: p.Transcribe, IsAvailable: p.IsAvailable}
}

func vrooliTier(p *VrooliProvider) *tiered.Tier[Request, *Result] {
	if p == nil {
		return nil
	}
	return &tiered.Tier[Request, *Result]{Execute: p.Transcribe, IsAvailable: p.IsAvailable}
}

func localTier(p *LocalProvider) *tiered.Tier[Request, *Result] {
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

// Execute runs the request through the chain.
func (c *Chain) Execute(ctx context.Context, req Request) (*Result, error) {
	return c.coord.Execute(ctx, req)
}

// Reconfigure swaps runtime toggles + TTLs; invalidates availability caches.
func (c *Chain) Reconfigure(enableBYOK, enableVrooli, enableLocal bool, ttlBYOK, ttlVrooli time.Duration) {
	c.coord.Reconfigure(enableBYOK, enableVrooli, enableLocal, ttlBYOK, ttlVrooli)
}

// ProbeResult is the per-tier availability snapshot.
type ProbeResult struct {
	Local  bool
	BYOK   bool
	Vrooli bool
}

func (c *Chain) Probe(ctx context.Context) ProbeResult {
	r := c.coord.Probe(ctx)
	return ProbeResult{Local: r.Local, BYOK: r.BYOK, Vrooli: r.Vrooli}
}
