package ttschain

import (
	"context"

	"audio-tools/internal/ai/chains/tiered"
)

// seam: VrooliClient is the TTS Vrooli-LPBS client seam (SEAMS.md row
// "ttschain.VrooliClient"). Production wires integrations/lpbs/clients;
// tests wire fakes.
type VrooliClient interface {
	Synthesize(ctx context.Context, lpbsToken, userIdentity string, req Request) (*Result, error)
	IsAvailable(ctx context.Context) bool
	Model() string
}

type VrooliProvider struct {
	*tiered.LPBSProvider[VrooliClient]
}

func NewVrooliProvider(client VrooliClient) *VrooliProvider {
	return &VrooliProvider{LPBSProvider: tiered.NewLPBSClientProvider(client,
		func(client VrooliClient) bool { return client == nil })}
}

func (p *VrooliProvider) Type() ProviderTier { return TierVrooli }

func (p *VrooliProvider) LPBSState() *tiered.LPBSProvider[VrooliClient] {
	if p == nil {
		return nil
	}
	return p.LPBSProvider
}

func (p *VrooliProvider) IsAvailable(ctx context.Context) bool { return tiered.SafeIsAvailable(p, ctx) }

func (p *VrooliProvider) Model() string { return tiered.SafeModel(p) }

func (p *VrooliProvider) Synthesize(ctx context.Context, req Request) (*Result, error) {
	call := func(client VrooliClient) (*Result, error) {
		return client.Synthesize(ctx, req.LPBSToken, req.UserIdentity, req)
	}
	return tiered.ExecuteLPBSCarrier(p, req.LPBSToken,
		tiered.LPBSNotConfigured("ttschain"), tiered.LPBSTokenRequired("ttschain"), call, stampLPBSResult)
}

func stampLPBSResult(result *Result, state *tiered.LPBSProvider[VrooliClient]) {
	tiered.StampLPBSResult(&result.Tier, TierVrooli, &result.ProviderID, &result.ModelID, state.Model())
}

// StreamingCapability — LPBS audio-gateway streaming (PRD OT-P2-002)
// is out of scope here. Declared false; chain skips this tier during
// stream-start negotiation.
func (p *VrooliProvider) StreamingCapability() bool { return false }

func (p *VrooliProvider) SynthesizeStreaming(_ context.Context, _ Request) (<-chan AudioFrame, error) {
	return nil, nil
}
