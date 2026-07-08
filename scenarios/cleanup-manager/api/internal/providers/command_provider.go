package providers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cleanup-manager/internal/cleanup"
)

type CommandProviderConfig struct {
	ID                 string
	Name               string
	SafetyTier         cleanup.SafetyTier
	DefaultMode        cleanup.ProviderMode
	DefaultApproval    cleanup.ApprovalMode
	RequiredPrivileges []string
	TestSubstitute     string
	EstimateCommand    cleanup.ProcessCommand
	PreviewAction      string
}

type CommandMetadataProvider struct {
	cfg    CommandProviderConfig
	runner cleanup.ProcessRunner
}

func NewCommandMetadataProvider(cfg CommandProviderConfig, runner cleanup.ProcessRunner) *CommandMetadataProvider {
	return &CommandMetadataProvider{cfg: cfg, runner: runner}
}

func (p *CommandMetadataProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID:                  p.cfg.ID,
		Name:                p.cfg.Name,
		Version:             "v1",
		OwnerScenario:       "cleanup-manager",
		SafetyTier:          p.cfg.SafetyTier,
		DefaultMode:         p.cfg.DefaultMode,
		DefaultApproval:     p.cfg.DefaultApproval,
		SupportedPlatforms:  []string{"linux"},
		RequiredPrivileges:  p.cfg.RequiredPrivileges,
		IrreversibleEffects: []string{"host package or runtime metadata may be removed when operator-approved"},
		TestSubstitute:      p.cfg.TestSubstitute,
	}
}

func (p *CommandMetadataProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	meta := p.Metadata()
	out := cleanup.Estimate{ProviderID: meta.ID, ProviderVersion: meta.Version, RequiresApproval: true, ObservedAt: req.Scope.Now}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if p.runner == nil {
		out.BlockedReason = "process runner unavailable"
		return out, nil
	}
	result, err := p.runner.Run(ctx, p.cfg.EstimateCommand)
	if err != nil {
		return cleanup.Estimate{}, err
	}
	out.EstimatedBytes = parseFirstInt64(result.Stdout)
	if out.EstimatedBytes > 0 {
		out.ItemCount = 1
	}
	return out, nil
}

func (p *CommandMetadataProvider) Preview(_ context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	meta := p.Metadata()
	out := cleanup.Preview{ProviderID: meta.ID, ProviderVersion: meta.Version}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	out.Items = append(out.Items, cleanup.PreviewItem{ID: meta.ID + ":metadata", Description: meta.Name, Bytes: req.Estimate.EstimatedBytes, Action: p.cfg.PreviewAction, SafetyTier: meta.SafetyTier})
	if len(meta.RequiredPrivileges) > 0 {
		out.Warnings = append(out.Warnings, "requires privileges: "+strings.Join(meta.RequiredPrivileges, ","))
	}
	return out, nil
}

func (p *CommandMetadataProvider) Apply(_ context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.Metadata().Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", p.cfg.ID, req.ProviderVersion, p.Metadata().Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("command metadata provider %s apply requires idempotency key", p.cfg.ID)
	}
	return cleanup.ApplyResult{ProviderID: p.Metadata().ID, Applied: false, Warnings: []string{"command cleanup apply is disabled until a typed allowlisted executor is wired"}}, nil
}

func (p *CommandMetadataProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "command metadata provider is preview-only"}, nil
}

func parseFirstInt64(text string) int64 {
	for _, field := range strings.Fields(text) {
		cleaned := strings.Trim(field, " ,:")
		if n, err := strconv.ParseInt(cleaned, 10, 64); err == nil {
			return n
		}
	}
	return 0
}

type JournalProvider struct {
	client cleanup.JournalClient
}

func NewJournalProvider(client cleanup.JournalClient) *JournalProvider {
	return &JournalProvider{client: client}
}

func (p *JournalProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID:                  "journald",
		Name:                "systemd journal",
		Version:             "v1",
		OwnerScenario:       "cleanup-manager",
		SafetyTier:          cleanup.SafetyTierConditional,
		DefaultMode:         cleanup.ProviderModeDisabled,
		DefaultApproval:     cleanup.ApprovalModeOperator,
		SupportedPlatforms:  []string{"linux"},
		RequiredPrivileges:  []string{"sudo", "systemd-journald"},
		IrreversibleEffects: []string{"journal entries older than policy retention may be vacuumed"},
		TestSubstitute:      "fake-journal",
	}
}

func (p *JournalProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	out := cleanup.Estimate{ProviderID: "journald", ProviderVersion: "v1", RequiresApproval: true, ObservedAt: req.Scope.Now}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if p.client == nil {
		out.BlockedReason = "journal client unavailable"
		return out, nil
	}
	bytes, err := p.client.DiskUsage(ctx)
	if err != nil {
		return cleanup.Estimate{}, err
	}
	out.EstimatedBytes = bytes
	if bytes > 0 {
		out.ItemCount = 1
	}
	return out, nil
}

func (p *JournalProvider) Preview(_ context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: "journald", ProviderVersion: "v1"}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	out.Items = append(out.Items, cleanup.PreviewItem{ID: "journald:vacuum", Description: "systemd journal vacuum by age", Bytes: req.Estimate.EstimatedBytes, Action: "journal-vacuum", SafetyTier: cleanup.SafetyTierConditional})
	out.Warnings = append(out.Warnings, "requires privileges: sudo,systemd-journald")
	return out, nil
}

func (p *JournalProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.Metadata().Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider journald version mismatch: got %q want %q", req.ProviderVersion, p.Metadata().Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("journald apply requires idempotency key")
	}
	if req.ApprovalMode != cleanup.ApprovalModeOperator {
		return cleanup.ApplyResult{ProviderID: "journald", SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"operator approval required"}}, nil
	}
	result, err := p.client.Vacuum(ctx, cleanup.JournalVacuumRequest{})
	if err != nil {
		return cleanup.ApplyResult{}, err
	}
	return cleanup.ApplyResult{ProviderID: "journald", Applied: result.ReclaimedBytes > 0, ReclaimedBytes: result.ReclaimedBytes}, nil
}

func (p *JournalProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "journal provider uses JournalClient seam"}, nil
}
