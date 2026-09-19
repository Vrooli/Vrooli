package summarizechain

import "audio-tools/internal/ai/chains/tiered"

// Chain composes BYOK -> Vrooli -> Local summarization providers.
type Chain = tiered.CredentialProviderChain[Request, *Result, LocalProvider, BYOKProvider, VrooliProvider]

type Options = tiered.CredentialOptions[LocalProvider, BYOKProvider, VrooliProvider]

func NewChain(opts Options) *Chain {
	return tiered.NewCredentialProviderChain(opts,
		(*LocalProvider).Summarize, (*LocalProvider).IsAvailable,
		(*BYOKProvider).Summarize, (*BYOKProvider).IsAvailable,
		(*VrooliProvider).Summarize, (*VrooliProvider).IsAvailable,
		func(req Request) bool { return req.BYOKKey != "" }, func(req Request) bool { return req.LPBSToken != "" },
		ErrUnknownBYOKProvider, ErrMissingBYOKProvider, ErrInsufficientCredits, ErrAllProvidersFailed, "summarize")
}
