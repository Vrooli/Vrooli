package providers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"storage-manager/internal/cleanup"
)

type DockerVolume struct {
	Name  string
	Bytes int64
	InUse bool
}

type DockerVolumeInventory interface {
	ListVolumes(context.Context) ([]DockerVolume, error)
}

type DockerUnusedVolumesProvider struct {
	client cleanup.DockerClient
	broker cleanup.BrokerActionClient
}

func NewDockerUnusedVolumesProvider(client cleanup.DockerClient, broker cleanup.BrokerActionClient) cleanup.Provider {
	return &DockerUnusedVolumesProvider{client: client, broker: broker}
}

func (p *DockerUnusedVolumesProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID: "docker-unused-volumes", Name: "Docker unused volumes", Version: "v1",
		OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierConditional,
		DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeOperator,
		SupportedPlatforms: []string{"linux", "darwin", "windows"}, RequiredPrivileges: []string{"privilege-broker"},
		IrreversibleEffects: []string{"explicitly named unused Docker volumes are removed"}, TestSubstitute: "fake-docker",
	}
}

func (p *DockerUnusedVolumesProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	out := cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: "v1", RequiresApproval: true, ObservedAt: req.Scope.Now}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	inventory, ok := p.client.(DockerVolumeInventory)
	if !ok {
		out.BlockedReason = "docker volume inventory unavailable"
		return out, nil
	}
	volumes, err := inventory.ListVolumes(ctx)
	if err != nil {
		return out, err
	}
	for _, volume := range volumes {
		if strings.TrimSpace(volume.Name) != "" && !volume.InUse {
			out.EstimatedBytes += volume.Bytes
			out.ItemCount++
		}
	}
	return out, nil
}

func (p *DockerUnusedVolumesProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: "v1"}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	inventory, ok := p.client.(DockerVolumeInventory)
	if !ok {
		out.BlockedReason = "docker volume inventory unavailable"
		return out, nil
	}
	volumes, err := inventory.ListVolumes(ctx)
	if err != nil {
		return out, err
	}
	for _, volume := range volumes {
		if strings.TrimSpace(volume.Name) != "" && !volume.InUse {
			out.Items = append(out.Items, cleanup.PreviewItem{ID: "docker-unused-volumes:" + volume.Name, Description: "Explicitly named unused Docker volume", Bytes: volume.Bytes, Action: "docker.prune.unused-volumes", SafetyTier: cleanup.SafetyTierConditional})
		}
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].ID < out.Items[j].ID })
	return out, nil
}

func (p *DockerUnusedVolumesProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != "v1" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("docker unused volumes requires version v1 and an idempotency key")
	}
	if p.broker == nil {
		return cleanup.ApplyResult{ProviderID: p.Metadata().ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"awaiting standing approval and privilege broker"}}, nil
	}
	names := make([]string, 0, len(req.Preview.Items))
	for _, item := range req.Preview.Items {
		const prefix = "docker-unused-volumes:"
		if strings.HasPrefix(item.ID, prefix) && len(item.ID) > len(prefix) {
			names = append(names, item.ID[len(prefix):])
		}
	}
	if len(names) == 0 {
		return cleanup.ApplyResult{ProviderID: p.Metadata().ID, AlreadyDone: true}, nil
	}
	before, err := p.client.SystemUsage(ctx)
	if err != nil {
		return cleanup.ApplyResult{}, err
	}
	brokerResult, err := p.broker.Do(ctx, "docker.prune.unused-volumes", map[string]any{"docker": map[string]any{"volume_names": names}})
	if err != nil {
		return cleanup.ApplyResult{}, err
	}
	after, afterErr := p.client.SystemUsage(ctx)
	reclaimed := int64(0)
	if afterErr == nil && before.VolumesBytes > after.VolumesBytes {
		reclaimed = before.VolumesBytes - after.VolumesBytes
	}
	return cleanup.ApplyResult{ProviderID: p.Metadata().ID, Applied: brokerResult.Changed, ReclaimedBytes: reclaimed}, nil
}

func (p *DockerUnusedVolumesProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "volume removal accepts only explicit broker-validated names"}, nil
}
