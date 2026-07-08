package providers

import (
	"context"
	"fmt"

	"cleanup-manager/internal/cleanup"
)

type DockerProvider struct {
	client cleanup.DockerClient
}

func NewDockerProvider(client cleanup.DockerClient) *DockerProvider {
	return &DockerProvider{client: client}
}

func (p *DockerProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID:                  "docker",
		Name:                "Docker pruneable data",
		Version:             "v1",
		OwnerScenario:       "cleanup-manager",
		SafetyTier:          cleanup.SafetyTierConditional,
		DefaultMode:         cleanup.ProviderModeDisabled,
		DefaultApproval:     cleanup.ApprovalModeOperator,
		SupportedPlatforms:  []string{"linux", "darwin"},
		RequiredPrivileges:  []string{"docker-daemon"},
		IrreversibleEffects: []string{"dangling images and build cache may be removed; volumes are never pruned by this provider"},
		TestSubstitute:      "fake-docker",
	}
}

func (p *DockerProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	if !req.Policy.Enabled {
		return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, RequiresApproval: true, BlockedReason: "provider disabled by policy", ObservedAt: req.Scope.Now}, nil
	}
	if p.client == nil {
		return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, RequiresApproval: true, BlockedReason: "docker client unavailable", ObservedAt: req.Scope.Now}, nil
	}
	usage, err := p.client.SystemUsage(ctx)
	if err != nil {
		return cleanup.Estimate{}, err
	}
	bytes := usage.ImagesBytes + usage.BuildCacheBytes
	return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, EstimatedBytes: bytes, ItemCount: 2, RequiresApproval: true, ObservedAt: req.Scope.Now}, nil
}

func (p *DockerProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if p.client == nil {
		out.BlockedReason = "docker client unavailable"
		return out, nil
	}
	usage, err := p.client.SystemUsage(ctx)
	if err != nil {
		return cleanup.Preview{}, err
	}
	if usage.ImagesBytes > 0 {
		out.Items = append(out.Items, cleanup.PreviewItem{ID: "docker:dangling-images", Description: "Dangling Docker images", Bytes: usage.ImagesBytes, Action: "docker-prune-dangling-images", SafetyTier: cleanup.SafetyTierConditional})
	}
	if usage.BuildCacheBytes > 0 {
		out.Items = append(out.Items, cleanup.PreviewItem{ID: "docker:build-cache", Description: "Docker build cache", Bytes: usage.BuildCacheBytes, Action: "docker-prune-build-cache", SafetyTier: cleanup.SafetyTierConditional})
	}
	if usage.VolumesBytes > 0 {
		out.Warnings = append(out.Warnings, "docker volumes are excluded by conservative provider defaults")
	}
	return out, nil
}

func (p *DockerProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.Metadata().Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider docker version mismatch: got %q want %q", req.ProviderVersion, p.Metadata().Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("docker apply requires idempotency key")
	}
	if req.ApprovalMode != cleanup.ApprovalModeOperator {
		return cleanup.ApplyResult{ProviderID: "docker", SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"operator approval required"}}, nil
	}
	result, err := p.client.Prune(ctx, cleanup.DockerPruneRequest{DanglingImages: true, BuildCache: true, StoppedOnly: true, Volumes: false})
	if err != nil {
		return cleanup.ApplyResult{}, err
	}
	return cleanup.ApplyResult{ProviderID: "docker", Applied: result.ReclaimedBytes > 0, ReclaimedBytes: result.ReclaimedBytes, Warnings: result.Warnings}, nil
}

func (p *DockerProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "docker provider excludes volumes"}, nil
}
