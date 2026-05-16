package summarizechain

import (
	"context"
	"fmt"
)

type VrooliClient interface {
	Summarize(ctx context.Context, lpbsToken, userIdentity string, req Request) (*Result, error)
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

func (p *VrooliProvider) Summarize(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("audio-tools/summarizechain: vrooli client not configured")
	}
	if req.LPBSToken == "" {
		return nil, fmt.Errorf("audio-tools/summarizechain: LPBS token required")
	}
	res, err := p.client.Summarize(ctx, req.LPBSToken, req.UserIdentity, req)
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
