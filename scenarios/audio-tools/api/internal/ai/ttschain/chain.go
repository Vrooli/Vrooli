package ttschain

import "audio-tools/internal/ai/chains/tiered"

// Chain composes Local -> BYOK -> Vrooli TTS providers when LocalFirst is
// enabled (the production speech policy). Streaming remains
// capability-specific in stream.go and uses the embedded typed providers.
type Chain struct {
	*tiered.CredentialProviderChain[Request, *Result, LocalProvider, BYOKProvider, VrooliProvider]
	localFirst bool
}

type Options = tiered.CredentialOptions[Request, LocalProvider, BYOKProvider, VrooliProvider]

// LocalCacheBackends returns cache backend keys in local provider preference
// order, including any configured local failover provider.
func (c *Chain) LocalCacheBackends() []string {
	if c == nil || c.CredentialProviderChain == nil || c.Local == nil {
		return nil
	}
	return c.Local.CacheBackends()
}

// LocalProviderID returns the preferred local provider's runtime identity.
// The configured chain order is the same declaration-backed order synthesis
// uses, so status can describe Kale or another future local provider without
// hardcoding a model-specific name.
func (c *Chain) LocalProviderID() string {
	if c == nil || c.CredentialProviderChain == nil || c.Local == nil {
		return ""
	}
	return c.Local.ProviderID()
}

// BYOKAdapterConfigured reports adapter registration independently of the
// routing toggle and of request-scoped credential availability.
func (c *Chain) BYOKAdapterConfigured() bool {
	return c != nil && c.CredentialProviderChain != nil && c.BYOK != nil && c.BYOK.AdapterConfigured()
}

func NewChain(opts Options) *Chain {
	if opts.Redact == nil {
		opts.Redact = func(req Request) []string {
			return tiered.RequestCredentialValues(req.BYOKKey, req.LPBSToken)
		}
	}
	return &Chain{CredentialProviderChain: tiered.NewCredentialProviderChain(opts,
		(*LocalProvider).Synthesize, (*LocalProvider).IsAvailable,
		(*BYOKProvider).Synthesize, (*BYOKProvider).IsAvailable,
		(*VrooliProvider).Synthesize, (*VrooliProvider).IsAvailable,
		func(req Request) bool { return req.BYOKKey != "" }, func(req Request) bool { return req.LPBSToken != "" },
		ErrUnknownBYOKProvider, ErrMissingBYOKProvider, ErrInsufficientCredits, ErrAllProvidersFailed, "tts"), localFirst: opts.LocalFirst}
}
