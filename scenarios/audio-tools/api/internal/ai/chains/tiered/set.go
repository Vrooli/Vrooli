package tiered

import (
	"context"
	"fmt"
	"time"

	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
)

// ResolveBYOKAdapter centralizes the credential and provider-name guard used
// by every BYOK capability chain. It preserves a typed adapter at the caller
// while keeping common user-input failures consistent across STT, TTS, and
// summarization.
func ResolveBYOKAdapter[Adapter any](registry map[string]Adapter, key, provider, capability string, missingProvider, unknownProvider error) (Adapter, error) {
	var zero Adapter
	if key == "" {
		return zero, fmt.Errorf("audio-tools/%s: BYOK key required", capability)
	}
	if provider == "" {
		return zero, missingProvider
	}
	adapter, ok := registry[provider]
	if !ok {
		return zero, fmt.Errorf("%w: %q", unknownProvider, provider)
	}
	return adapter, nil
}

// ExecuteBYOK performs the common credential dispatch sequence while leaving
// each capability's adapter method and result metadata at its domain seam.
func ExecuteBYOK[Adapter, Response any](registry map[string]Adapter, key, provider, capability string, missingProvider, unknownProvider error, call func(Adapter) (*Response, error), decorate func(*Response, Adapter)) (*Response, error) {
	adapter, err := ResolveBYOKAdapter(registry, key, provider, capability, missingProvider, unknownProvider)
	if err != nil {
		return nil, err
	}
	response, err := call(adapter)
	if err != nil {
		return nil, err
	}
	decorate(response, adapter)
	return response, nil
}

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

// CredentialSet is the common three-tier shape used by capability chains
// whose remote tiers are activated by BYOK and Vrooli credentials.
// Capability packages supply only their request readers and error identities.
type CredentialSet[Req, Resp any] struct {
	BYOK   *Tier[Req, Resp]
	Vrooli *Tier[Req, Resp]
	Local  *Tier[Req, Resp]

	HasBYOK   func(Req) bool
	HasVrooli func(Req) bool

	UnknownBYOK         error
	MissingBYOK         error
	InsufficientCredits error
	AllFailed           error
}

// CredentialOptions holds the common provider pointers and runtime knobs for
// capability chains using the BYOK -> Vrooli -> Local routing policy.
type CredentialOptions[Local, BYOK, Vrooli any] struct {
	Local  *Local
	BYOK   *BYOK
	Vrooli *Vrooli

	EnableLocal  bool
	EnableBYOK   bool
	EnableVrooli bool

	AvailTTLByOK   time.Duration
	AvailTTLVrooli time.Duration

	Clock clock.Clock
	Logx  logx.Logger
}

// CredentialProviderChain keeps the typed provider pointers needed by
// capability-specific streaming paths while embedding the shared coordinator.
type CredentialProviderChain[Req, Resp, Local, BYOK, Vrooli any] struct {
	*Coordinator[Req, Resp]
	Local  *Local
	BYOK   *BYOK
	Vrooli *Vrooli
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

// NewCredentialChain constructs the shared BYOK -> Vrooli -> Local policy.
// It keeps capability-specific provider methods and error values at the
// domain boundary while centralizing the routing and terminal-error rules.
func NewCredentialChain[Req, Resp any](set CredentialSet[Req, Resp], opts ChainOptions) *Coordinator[Req, Resp] {
	return NewChainFromSet(ProviderSet[Req, Resp]{
		BYOK:       set.BYOK,
		Vrooli:     set.Vrooli,
		Local:      set.Local,
		Route:      CredentialRoute(set.HasBYOK, set.HasVrooli),
		IsTerminal: CredentialTerminal(set.UnknownBYOK, set.MissingBYOK, set.InsufficientCredits),
		AllFailed:  set.AllFailed,
	}, opts)
}

// NewCredentialCoordinator adapts concrete provider pointers and constructs
// the common credential-routed coordinator in one place.
func NewCredentialCoordinator[Req, Resp, Local, BYOK, Vrooli any](opts CredentialOptions[Local, BYOK, Vrooli], localTier func(*Local) *Tier[Req, Resp], byokTier func(*BYOK) *Tier[Req, Resp], vrooliTier func(*Vrooli) *Tier[Req, Resp], hasBYOK, hasVrooli func(Req) bool, unknownBYOK, missingBYOK, insufficientCredits, allFailed error, capability string) *Coordinator[Req, Resp] {
	return NewCredentialChain(CredentialSet[Req, Resp]{
		BYOK:                byokTier(opts.BYOK),
		Vrooli:              vrooliTier(opts.Vrooli),
		Local:               localTier(opts.Local),
		HasBYOK:             hasBYOK,
		HasVrooli:           hasVrooli,
		UnknownBYOK:         unknownBYOK,
		MissingBYOK:         missingBYOK,
		InsufficientCredits: insufficientCredits,
		AllFailed:           allFailed,
	}, ChainOptions{
		EnableBYOK:   opts.EnableBYOK,
		EnableVrooli: opts.EnableVrooli,
		EnableLocal:  opts.EnableLocal,
		TTLByOK:      opts.AvailTTLByOK,
		TTLVrooli:    opts.AvailTTLVrooli,
		Clock:        opts.Clock,
		OnFallback:   FallbackLogger(capability, opts.Logx),
	})
}

// NewCredentialProviderChain builds the common coordinator and retains the
// typed providers for capability-specific paths such as TTS streaming. The
// caller supplies only its concrete execute and availability methods.
func NewCredentialProviderChain[Req, Resp, Local, BYOK, Vrooli any](opts CredentialOptions[Local, BYOK, Vrooli], localExecute func(*Local, context.Context, Req) (Resp, error), localAvailable func(*Local, context.Context) bool, byokExecute func(*BYOK, context.Context, Req) (Resp, error), byokAvailable func(*BYOK, context.Context) bool, vrooliExecute func(*Vrooli, context.Context, Req) (Resp, error), vrooliAvailable func(*Vrooli, context.Context) bool, hasBYOK, hasVrooli func(Req) bool, unknownBYOK, missingBYOK, insufficientCredits, allFailed error, capability string) *CredentialProviderChain[Req, Resp, Local, BYOK, Vrooli] {
	return &CredentialProviderChain[Req, Resp, Local, BYOK, Vrooli]{
		Coordinator: NewCredentialCoordinator(opts,
			func(provider *Local) *Tier[Req, Resp] { return TierFor(provider, localExecute, localAvailable) },
			func(provider *BYOK) *Tier[Req, Resp] { return TierFor(provider, byokExecute, byokAvailable) },
			func(provider *Vrooli) *Tier[Req, Resp] { return TierFor(provider, vrooliExecute, vrooliAvailable) },
			hasBYOK, hasVrooli, unknownBYOK, missingBYOK, insufficientCredits, allFailed, capability),
		Local: opts.Local, BYOK: opts.BYOK, Vrooli: opts.Vrooli,
	}
}
