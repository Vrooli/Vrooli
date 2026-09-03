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
	broker := &recordingBroker{changed: true}
	provider := NewDockerProvider(client, broker)
	policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}

	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: policy})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if estimate.EstimatedBytes != 100 || estimate.ItemCount != 1 {
		t.Fatalf("Estimate() = %#v, want 100 image bytes and one item", estimate)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: policy, Estimate: estimate})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("Preview() items = %d, want dangling images only", len(preview.Items))
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
	if !applied.Applied {
		t.Fatalf("Apply() with approval = %#v, want brokered apply", applied)
	}
	if len(client.Prunes) != 0 {
		t.Fatalf("Prunes = %#v, want no direct Docker prune", client.Prunes)
	}
	if len(broker.actions) != 1 || broker.actions[0] != "docker.prune.unused-images" {
		t.Fatalf("broker actions = %#v, want one brokered image prune", broker.actions)
	}
}
