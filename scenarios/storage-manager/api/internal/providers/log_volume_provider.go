package providers

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"storage-manager/internal/cleanup"
)

type LogVolumeProvider struct {
	files  cleanup.FileSystem
	broker cleanup.BrokerActionClient
}

func NewLogVolumeProvider(files cleanup.FileSystem, broker cleanup.BrokerActionClient) cleanup.Provider {
	return &LogVolumeProvider{files: files, broker: broker}
}

func (p *LogVolumeProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID: "log-volume-force-rotate", Name: "Managed log volume force-rotate", Version: "v1",
		OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierConditional,
		DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeOperator,
		SupportedPlatforms: []string{"linux"}, RequiredPrivileges: []string{"privilege-broker"},
		IrreversibleEffects: []string{"managed log files are rotated according to the installed safeguard stanza"},
		TestSubstitute:      "fake-filesystem",
	}
}

func (p *LogVolumeProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	out := cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: "v1", RequiresApproval: true, ObservedAt: req.Scope.Now}
	if runtime.GOOS != "linux" {
		out.BlockedReason = "managed flat log store is not applicable on this platform"
		return out, nil
	}
	if !req.Policy.Enabled || p.files == nil {
		out.BlockedReason = "provider disabled or filesystem unavailable"
		return out, nil
	}
	for _, path := range []string{"/var/log/syslog", "/var/log/auth.log"} {
		info, err := p.files.Stat(ctx, path)
		if err == nil {
			out.EstimatedBytes += info.Size
			out.ItemCount++
		}
	}
	return out, nil
}

func (p *LogVolumeProvider) Preview(_ context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: "v1"}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if req.Estimate.EstimatedBytes > 0 {
		out.Items = []cleanup.PreviewItem{{ID: "log-volume-force-rotate", Description: "Force-rotate managed system logs", Bytes: req.Estimate.EstimatedBytes, Action: "log.rotate.force", SafetyTier: cleanup.SafetyTierConditional}}
	}
	return out, nil
}

func (p *LogVolumeProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != "v1" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("log volume provider requires version v1 and an idempotency key")
	}
	if p.broker == nil || p.files == nil || runtime.GOOS != "linux" {
		return cleanup.ApplyResult{ProviderID: p.Metadata().ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"awaiting standing approval and privilege broker"}}, nil
	}
	var before int64
	for _, item := range req.Preview.Items {
		before += item.Bytes
	}
	result, err := p.broker.Do(ctx, "log.rotate.force", map[string]any{"log": map[string]any{"stanza": "vrooli-log-volume-bounds"}})
	if err != nil {
		return cleanup.ApplyResult{}, err
	}
	var after int64
	for _, path := range []string{"/var/log/syslog", "/var/log/auth.log"} {
		if info, statErr := p.files.Stat(ctx, path); statErr == nil {
			after += info.Size
		}
	}
	reclaimed := int64(0)
	if before > after {
		reclaimed = before - after
	}
	return cleanup.ApplyResult{ProviderID: p.Metadata().ID, Applied: result.Changed, ReclaimedBytes: reclaimed}, nil
}

func (p *LogVolumeProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: runtime.GOOS == "linux", Message: "force-rotation is delegated to the managed broker stanza"}, nil
}
