package ttschain

import (
	"context"
	"fmt"

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
	return &VrooliProvider{LPBSProvider: tiered.NewLPBSProvider(client,
		func(client VrooliClient) bool { return client == nil },
		func(client VrooliClient, ctx context.Context) bool { return client.IsAvailable(ctx) },
		func(client VrooliClient) string { return client.Model() })}
}

func (p *VrooliProvider) Type() ProviderTier { return TierVrooli }

func (p *VrooliProvider) IsAvailable(ctx context.Context) bool {
	return p != nil && p.LPBSProvider != nil && p.LPBSProvider.IsAvailable(ctx)
}

func (p *VrooliProvider) Model() string {
	if p == nil || p.LPBSProvider == nil {
		return ""
	}
	return p.LPBSProvider.Model()
}

func (p *VrooliProvider) Synthesize(ctx context.Context, req Request) (*Result, error) {
	return tiered.ExecuteLPBS(p != nil && p.LPBSProvider != nil && p.Configured(), req.LPBSToken,
		fmt.Errorf("audio-tools/ttschain: vrooli client not configured"),
		fmt.Errorf("audio-tools/ttschain: LPBS token required"),
		func() (*Result, error) { return p.Client.Synthesize(ctx, req.LPBSToken, req.UserIdentity, req) },
		func(result *Result) {
			result.Tier = TierVrooli
			if result.ProviderID == "" {
				result.ProviderID = "lpbs"
			}
			if result.ModelID == "" {
				result.ModelID = p.Model()
			}
		},
	)
}

// StreamingCapability — LPBS audio-gateway streaming (PRD OT-P2-002)
// is out of scope here. Declared false; chain skips this tier during
// stream-start negotiation.
func (p *VrooliProvider) StreamingCapability() bool { return false }

func (p *VrooliProvider) SynthesizeStreaming(_ context.Context, _ Request) (<-chan AudioFrame, error) {
	return nil, nil
}
