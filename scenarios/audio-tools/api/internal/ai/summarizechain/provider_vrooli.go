package summarizechain

import (
	"context"
	"fmt"

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

func (p *VrooliProvider) Summarize(ctx context.Context, req Request) (*Result, error) {
	return tiered.ExecuteLPBS(p != nil && p.LPBSProvider != nil && p.Configured(), req.LPBSToken,
		fmt.Errorf("audio-tools/summarizechain: vrooli client not configured"),
		fmt.Errorf("audio-tools/summarizechain: LPBS token required"),
		func() (*Result, error) { return p.Client.Summarize(ctx, req.LPBSToken, req.UserIdentity, req) },
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
