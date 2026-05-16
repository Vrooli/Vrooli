package sttchain

import (
	"context"
	"fmt"
)

// VrooliClient is implemented by the LPBS audio-gateway client. Kept as an
// interface so the chain doesn't import the lpbs package directly (and so
// tests can fake it).
type VrooliClient interface {
	Transcribe(ctx context.Context, lpbsToken, userIdentity string, req Request) (*Result, error)
	IsAvailable(ctx context.Context) bool
	Model() string
}

// VrooliProvider routes through the LPBS audio gateway. Disabled by default
// (AUDIO_AI_ENABLE_VROOLI=false) until execute/lpbs-audio-gateway-endpoints
// lands.
type VrooliProvider struct {
	client VrooliClient
}

func NewVrooliProvider(client VrooliClient) *VrooliProvider {
	return &VrooliProvider{client: client}
}

func (p *VrooliProvider) Type() ProviderTier { return TierVrooli }

func (p *VrooliProvider) IsAvailable(ctx context.Context) bool {
	if p == nil || p.client == nil {
		return false
	}
	return p.client.IsAvailable(ctx)
}

func (p *VrooliProvider) Transcribe(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("audio-tools/sttchain: vrooli client not configured")
	}
	if req.LPBSToken == "" {
		return nil, fmt.Errorf("audio-tools/sttchain: LPBS token required")
	}
	res, err := p.client.Transcribe(ctx, req.LPBSToken, req.UserIdentity, req)
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

// StreamingCapability reports the Vrooli tier as non-streaming-capable
// today; LPBS audio-gateway streaming endpoints are tracked under PRD
// OT-P2-002 (out of scope for this plan).
func (p *VrooliProvider) StreamingCapability() bool { return false }

// TranscribeStreaming declines streaming on the Vrooli tier. The chain
// falls through to the next tier or to buffered mode.
func (p *VrooliProvider) TranscribeStreaming(_ context.Context, _ StreamStart, _ <-chan AudioChunk) (<-chan StreamEvent, error) {
	return nil, nil
}
