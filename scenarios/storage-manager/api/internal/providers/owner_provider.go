package providers

import (
	"context"
	"fmt"

	"storage-manager/internal/cleanup"
)

type OwnerProviderConfig struct {
	ID              string
	Name            string
	OwnerScenario   string
	SafetyTier      cleanup.SafetyTier
	DefaultMode     cleanup.ProviderMode
	DefaultApproval cleanup.ApprovalMode
}

type OwnerScenarioProvider struct {
	cfg    OwnerProviderConfig
	client cleanup.ScenarioProviderClient
}

func NewOwnerScenarioProvider(cfg OwnerProviderConfig, client cleanup.ScenarioProviderClient) *OwnerScenarioProvider {
	return &OwnerScenarioProvider{cfg: cfg, client: client}
}

func (p *OwnerScenarioProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID:                  p.cfg.ID,
		Name:                p.cfg.Name,
		Version:             "v1",
		OwnerScenario:       p.cfg.OwnerScenario,
		SafetyTier:          p.cfg.SafetyTier,
		DefaultMode:         p.cfg.DefaultMode,
		DefaultApproval:     p.cfg.DefaultApproval,
		SupportedPlatforms:  []string{"linux", "darwin"},
		IrreversibleEffects: []string{"owner scenario deletes private data after preview approval"},
		TestSubstitute:      "fake-owner-provider",
	}
}

func (p *OwnerScenarioProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	if !req.Policy.Enabled {
		meta := p.Metadata()
		return cleanup.Estimate{ProviderID: meta.ID, ProviderVersion: meta.Version, BlockedReason: "provider disabled by policy", ObservedAt: req.Scope.Now}, nil
	}
	if p.client == nil {
		meta := p.Metadata()
		return cleanup.Estimate{ProviderID: meta.ID, ProviderVersion: meta.Version, BlockedReason: "owner scenario client unavailable", ObservedAt: req.Scope.Now}, nil
	}
	return p.client.Estimate(ctx, p.cfg.OwnerScenario, req.Policy)
}

func (p *OwnerScenarioProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	if !req.Policy.Enabled {
		meta := p.Metadata()
		return cleanup.Preview{ProviderID: meta.ID, ProviderVersion: meta.Version, BlockedReason: "provider disabled by policy"}, nil
	}
	if p.client == nil {
		meta := p.Metadata()
		return cleanup.Preview{ProviderID: meta.ID, ProviderVersion: meta.Version, BlockedReason: "owner scenario client unavailable"}, nil
	}
	return p.client.Preview(ctx, p.cfg.OwnerScenario, req.Estimate)
}

func (p *OwnerScenarioProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.Metadata().Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", p.cfg.ID, req.ProviderVersion, p.Metadata().Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("owner scenario provider %s apply requires idempotency key", p.cfg.ID)
	}
	if p.cfg.SafetyTier == cleanup.SafetyTierSafeWithOwner && req.ApprovalMode != cleanup.ApprovalModeOwner && req.ApprovalMode != cleanup.ApprovalModeOperator {
		return cleanup.ApplyResult{ProviderID: p.cfg.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"owner or operator approval required"}}, nil
	}
	if p.cfg.SafetyTier == cleanup.SafetyTierConditional && req.ApprovalMode != cleanup.ApprovalModeOperator {
		return cleanup.ApplyResult{ProviderID: p.cfg.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"operator approval required"}}, nil
	}
	if p.client == nil {
		return cleanup.ApplyResult{}, fmt.Errorf("owner scenario client unavailable")
	}
	return p.client.Apply(ctx, cleanup.ScenarioCleanupRequest{ScenarioID: p.cfg.OwnerScenario, ProviderID: p.cfg.ID, Preview: req.Preview})
}

func (p *OwnerScenarioProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "owner scenario provider delegates private deletion"}, nil
}
