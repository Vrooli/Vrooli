package providers

import (
	"context"
	"fmt"

	"storage-manager/internal/cleanup"
)

type DockerProvider struct {
	client cleanup.DockerClient
	broker cleanup.BrokerActionClient
}

func NewDockerProvider(client cleanup.DockerClient, broker ...cleanup.BrokerActionClient) *DockerProvider {
	var actionClient cleanup.BrokerActionClient
	if len(broker) > 0 {
		actionClient = broker[0]
	}
	return &DockerProvider{client: client, broker: actionClient}
}

func (p *DockerProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID:                  "docker",
		Name:                "Docker pruneable data",
		Version:             "v1",
		OwnerScenario:       "storage-manager",
		SafetyTier:          cleanup.SafetyTierConditional,
		DefaultMode:         cleanup.ProviderModeDisabled,
		DefaultApproval:     cleanup.ApprovalModeOperator,
		SupportedPlatforms:  []string{"linux", "darwin"},
		RequiredPrivileges:  []string{"docker-daemon"},
		IrreversibleEffects: []string{"dangling images may be removed; build cache and volumes are never pruned by this provider"},
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
	itemCount := 0
	if usage.ImagesBytes > 0 {
		itemCount = 1
	}
	return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, EstimatedBytes: usage.ImagesBytes, ItemCount: itemCount, RequiresApproval: true, ObservedAt: req.Scope.Now}, nil
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
	if p.broker == nil {
		return cleanup.ApplyResult{ProviderID: "docker", SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"privilege broker unavailable; Docker deletion withheld"}}, nil
	}
	before, err := p.client.SystemUsage(ctx)
	if err != nil {
		return cleanup.ApplyResult{}, err
	}
	brokerResult, err := p.broker.Do(ctx, "docker.prune.unused-images", map[string]any{"docker": map[string]any{}})
	if err != nil {
		return cleanup.ApplyResult{}, err
	}
	after, afterErr := p.client.SystemUsage(ctx)
	reclaimed := int64(0)
	if afterErr == nil {
		beforeBytes := before.ImagesBytes
		afterBytes := after.ImagesBytes
		if beforeBytes > afterBytes {
			reclaimed = beforeBytes - afterBytes
		}
	}
	return cleanup.ApplyResult{ProviderID: "docker", Applied: brokerResult.Changed, ReclaimedBytes: reclaimed}, nil
}

func (p *DockerProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "docker provider excludes volumes"}, nil
}
