package summarizechain

import "audio-tools/internal/ai/chains/tiered"

// Chain composes BYOK -> Vrooli -> Local summarization providers.
type Chain = tiered.CredentialProviderChain[Request, *Result, LocalProvider, BYOKProvider, VrooliProvider]

type Options = tiered.CredentialOptions[Request, LocalProvider, BYOKProvider, VrooliProvider]

func NewChain(opts Options) *Chain {
	if opts.Redact == nil {
		opts.Redact = func(req Request) []string {
			return tiered.RequestCredentialValues(req.BYOKKey, req.LPBSToken)
		}
	}
	hasBYOK := func(req Request) bool { return req.BYOKKey != "" }
	hasVrooli := func(req Request) bool { return req.LPBSToken != "" }
	route := func(slot tiered.Slot, req Request) bool {
		if req.LocalModelRole != "" {
			return slot == tiered.SlotLocal
		}
		switch slot {
		case tiered.SlotBYOK:
			return hasBYOK(req)
		case tiered.SlotVrooli:
			return hasVrooli(req)
		default:
			return true
		}
	}
	coordinator := tiered.NewChainFromSet(tiered.ProviderSet[Request, *Result]{
		Local:      tiered.TierFor(opts.Local, (*LocalProvider).Summarize, (*LocalProvider).IsAvailable),
		BYOK:       tiered.TierFor(opts.BYOK, (*BYOKProvider).Summarize, (*BYOKProvider).IsAvailable),
		Vrooli:     tiered.TierFor(opts.Vrooli, (*VrooliProvider).Summarize, (*VrooliProvider).IsAvailable),
		Route:      route,
		IsTerminal: tiered.CredentialTerminal(ErrUnknownBYOKProvider, ErrMissingBYOKProvider, ErrInsufficientCredits),
		AllFailed:  ErrAllProvidersFailed,
		Redact:     opts.Redact,
	}, tiered.ChainOptions{
		EnableLocal: opts.EnableLocal, EnableBYOK: opts.EnableBYOK, EnableVrooli: opts.EnableVrooli,
		LocalFirst: opts.LocalFirst,
		TTLByOK:    opts.AvailTTLByOK, TTLVrooli: opts.AvailTTLVrooli,
		Clock: opts.Clock, OnFallback: tiered.FallbackLogger("summarize", opts.Logx),
	})
	return &tiered.CredentialProviderChain[Request, *Result, LocalProvider, BYOKProvider, VrooliProvider]{
		Coordinator: coordinator, Local: opts.Local, BYOK: opts.BYOK, Vrooli: opts.Vrooli,
	}
}
