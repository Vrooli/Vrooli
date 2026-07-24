package ttschain

import "audio-tools/internal/ai/chains/tiered"

// Chain composes BYOK -> Vrooli -> Local TTS providers. Streaming remains
// capability-specific in stream.go and uses the embedded typed providers.
type Chain struct {
	*tiered.CredentialProviderChain[Request, *Result, LocalProvider, BYOKProvider, VrooliProvider]
}

type Options = tiered.CredentialOptions[LocalProvider, BYOKProvider, VrooliProvider]

func NewChain(opts Options) *Chain {
	return &Chain{CredentialProviderChain: tiered.NewCredentialProviderChain(opts,
		(*LocalProvider).Synthesize, (*LocalProvider).IsAvailable,
		(*BYOKProvider).Synthesize, (*BYOKProvider).IsAvailable,
		(*VrooliProvider).Synthesize, (*VrooliProvider).IsAvailable,
		func(req Request) bool { return req.BYOKKey != "" }, func(req Request) bool { return req.LPBSToken != "" },
		ErrUnknownBYOKProvider, ErrMissingBYOKProvider, ErrInsufficientCredits, ErrAllProvidersFailed, "tts")}
}
