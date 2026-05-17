package tiered

import (
	"context"
	"time"

	"audio-tools/internal/clock"
)

// ProviderSet bundles the three tiers and the routing / terminal-error /
// AllFailed decisions that vary per domain. Paired with ChainOptions
// (runtime knobs that do NOT vary per domain) it lets each chain
// package describe itself in one declarative call to NewChainFromSet,
// replacing the per-package Options + NewCoordinator boilerplate.
//
// seam: ProviderSet is the chain-construction seam shared by sttchain,
// ttschain, and summarizechain (SEAMS.md row "chains/tiered.Coordinator").
type ProviderSet[Req, Resp any] struct {
	BYOK   *Tier[Req, Resp]
	Vrooli *Tier[Req, Resp]
	Local  *Tier[Req, Resp]

	Route      func(slot Slot, req Req) bool
	IsTerminal func(slot Slot, err error) bool
	AllFailed  error
}

// ChainOptions holds the runtime-tunable knobs shared by every chain.
type ChainOptions struct {
	EnableBYOK   bool
	EnableVrooli bool
	EnableLocal  bool

	TTLByOK   time.Duration
	TTLVrooli time.Duration

	Clock clock.Clock

	// OnFallback, when non-nil, is forwarded to the Coordinator and fires
	// whenever a successful response originates from a non-primary tier.
	// Per-request callbacks attached via WithOnFallback also fire.
	OnFallback func(ctx context.Context, ev FallbackEvent)
}

// NewChainFromSet builds a Coordinator from a domain ProviderSet plus
// the shared runtime ChainOptions.
func NewChainFromSet[Req, Resp any](set ProviderSet[Req, Resp], opts ChainOptions) *Coordinator[Req, Resp] {
	return NewCoordinator(Options[Req, Resp]{
		BYOK:         set.BYOK,
		Vrooli:       set.Vrooli,
		Local:        set.Local,
		EnableBYOK:   opts.EnableBYOK,
		EnableVrooli: opts.EnableVrooli,
		EnableLocal:  opts.EnableLocal,
		TTLByOK:      opts.TTLByOK,
		TTLVrooli:    opts.TTLVrooli,
		Route:        set.Route,
		IsTerminal:   set.IsTerminal,
		AllFailed:    set.AllFailed,
		OnFallback:   opts.OnFallback,
		Clock:        opts.Clock,
	})
}
