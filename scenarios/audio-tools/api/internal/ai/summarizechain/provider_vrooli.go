package summarizechain

import (
	"context"

	"audio-tools/internal/ai/chains/tiered"
)

// seam: VrooliClient is the summarize Vrooli-LPBS client seam (SEAMS.md
// row "summarizechain.VrooliClient"). Production wires
// integrations/lpbs/clients; tests wire fakes.
type VrooliClient interface {
	Summarize(ctx context.Context, lpbsToken, userIdentity string, req Request) (*Result, error)
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

func (p *VrooliProvider) IsAvailable(ctx context.Context) bool { return tiered.SafeIsAvailable(ctx, p) }

func (p *VrooliProvider) Model() string { return tiered.SafeModel(p) }

func (p *VrooliProvider) Summarize(ctx context.Context, req Request) (*Result, error) {
	// Local policy selectors belong to the Ollama route; LPBS model selection
	// remains controlled by the existing provider configuration.
	req.LocalModelRole = ""
	call := func(client VrooliClient) (*Result, error) {
		return client.Summarize(ctx, req.LPBSToken, req.UserIdentity, req)
	}
	return tiered.ExecuteLPBSCarrier(p, req.LPBSToken,
		tiered.LPBSNotConfigured("summarizechain"), tiered.LPBSTokenRequired("summarizechain"), call, stampLPBSResult)
}

func stampLPBSResult(result *Result, state *tiered.LPBSProvider[VrooliClient]) {
	tiered.StampLPBSResult(&result.Tier, TierVrooli, &result.ProviderID, &result.ModelID, state.Model())
}
