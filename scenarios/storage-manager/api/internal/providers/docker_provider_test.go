package providers

import (
	"context"
	"testing"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

func TestDockerProviderExcludesVolumesAndRequiresOperatorApproval(t *testing.T) {
	t.Parallel()

	client := &cleanupfakes.DockerClient{Usage: cleanup.DockerUsage{ImagesBytes: 100, BuildCacheBytes: 25, VolumesBytes: 900}}
	provider := NewDockerProvider(client)
	policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}

	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: policy})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if estimate.EstimatedBytes != 125 {
		t.Fatalf("Estimate() bytes = %d, want 125", estimate.EstimatedBytes)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: policy, Estimate: estimate})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if len(preview.Items) != 2 {
		t.Fatalf("Preview() items = %d, want dangling images and build cache", len(preview.Items))
	}
	if len(preview.Warnings) == 0 {
		t.Fatal("Preview() expected volume exclusion warning")
	}
	skipped, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeNone, IdempotencyKey: "apply-1", Preview: preview})
	if err != nil {
		t.Fatalf("Apply() without approval error = %v", err)
	}
	if skipped.Applied || len(client.Prunes) != 0 {
		t.Fatalf("Apply() without approval = %#v prunes=%#v, want no mutation", skipped, client.Prunes)
	}
	applied, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeOperator, IdempotencyKey: "apply-2", Preview: preview})
	if err != nil {
		t.Fatalf("Apply() with approval error = %v", err)
	}
	if !applied.Applied || applied.ReclaimedBytes != 125 {
		t.Fatalf("Apply() with approval = %#v, want 125 reclaimed", applied)
	}
	if len(client.Prunes) != 1 || client.Prunes[0].Volumes {
		t.Fatalf("Prunes = %#v, want exactly one non-volume prune", client.Prunes)
	}
}
