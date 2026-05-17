package ttschain

import (
	"context"
	"fmt"
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
	client VrooliClient
}

func NewVrooliProvider(client VrooliClient) *VrooliProvider { return &VrooliProvider{client: client} }

func (p *VrooliProvider) Type() ProviderTier { return TierVrooli }

func (p *VrooliProvider) IsAvailable(ctx context.Context) bool {
	if p == nil || p.client == nil {
		return false
	}
	return p.client.IsAvailable(ctx)
}

func (p *VrooliProvider) Synthesize(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("audio-tools/ttschain: vrooli client not configured")
	}
	if req.LPBSToken == "" {
		return nil, fmt.Errorf("audio-tools/ttschain: LPBS token required")
	}
	res, err := p.client.Synthesize(ctx, req.LPBSToken, req.UserIdentity, req)
	if err != nil {
		return nil, err
	}
	res.Tier = TierVrooli
	if res.ProviderID == "" {
		res.ProviderID = "lpbs"
	}
	if res.ModelID == "" {
		res.ModelID = p.client.Model()
	}
	return res, nil
}

func (p *VrooliProvider) Model() string {
	if p == nil || p.client == nil {
		return ""
	}
	return p.client.Model()
}

// StreamingCapability — LPBS audio-gateway streaming (PRD OT-P2-002)
// is out of scope here. Declared false; chain skips this tier during
// stream-start negotiation.
func (p *VrooliProvider) StreamingCapability() bool { return false }

func (p *VrooliProvider) SynthesizeStreaming(_ context.Context, _ Request) (<-chan AudioFrame, error) {
	return nil, nil
}
