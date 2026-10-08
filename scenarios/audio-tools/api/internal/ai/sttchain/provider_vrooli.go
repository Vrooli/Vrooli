package sttchain

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/chains/tiered"
)

// seam: VrooliClient is the STT Vrooli-LPBS client seam (SEAMS.md row
// "sttchain.VrooliClient"). Production wires integrations/lpbs/clients;
// tests wire fakes.
//
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
	*tiered.LPBSProvider[VrooliClient]
}

func NewVrooliProvider(client VrooliClient) *VrooliProvider {
	return &VrooliProvider{LPBSProvider: tiered.NewLPBSProvider(client,
		func(client VrooliClient) bool { return client == nil },
		func(client VrooliClient, ctx context.Context) bool { return client.IsAvailable(ctx) },
		func(client VrooliClient) string { return client.Model() })}
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

func (p *VrooliProvider) Transcribe(ctx context.Context, req Request) (*Result, error) {
	return tiered.ExecuteLPBS(p != nil && p.LPBSProvider != nil && p.Configured(), req.LPBSToken,
		fmt.Errorf("audio-tools/sttchain: vrooli client not configured"),
		fmt.Errorf("audio-tools/sttchain: LPBS token required"),
		func() (*Result, error) { return p.Client.Transcribe(ctx, req.LPBSToken, req.UserIdentity, req) },
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

// Traits reports the Vrooli tier as batch-only today; LPBS audio-gateway
// streaming endpoints are tracked under PRD OT-P2-002 (out of scope for
// this plan). When that lands, Stream flips to true with
// Strategies=[passthrough].
func (p *VrooliProvider) Traits() ProviderTraits {
	return ProviderTraits{Batch: true, Stream: false}
}

// TranscribeStreaming declines streaming on the Vrooli tier. The chain
// falls through to the next tier or to buffered mode.
func (p *VrooliProvider) TranscribeStreaming(_ context.Context, _ StreamStart, _ <-chan AudioChunk) (<-chan StreamEvent, error) {
	return nil, nil
}
