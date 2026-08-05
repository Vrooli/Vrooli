package providers

import (
	"context"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
)

type ledgerDockerClient struct {
	images  []DockerImage
	removed []string
}

func (c *ledgerDockerClient) SystemUsage(context.Context) (cleanup.DockerUsage, error) {
	return cleanup.DockerUsage{}, nil
}
func (c *ledgerDockerClient) Prune(context.Context, cleanup.DockerPruneRequest) (cleanup.DockerPruneResult, error) {
	return cleanup.DockerPruneResult{}, nil
}
func (c *ledgerDockerClient) ListImages(context.Context) ([]DockerImage, error) { return c.images, nil }
func (c *ledgerDockerClient) RemoveImages(_ context.Context, ids []string) (int64, error) {
	c.removed = append(c.removed, ids...)
	return int64(len(ids)) * 100, nil
}

func TestDockerUnusedImagesUsesLastLedgerUseAndFullCoverageWindow(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	ledger := NewMemoryDockerUsageLedger()
	if err := ledger.Record(now.Add(-30*24*time.Hour), []DockerImage{{ID: "old", Bytes: 100, Running: true}, {ID: "recent", Bytes: 200, Running: true}}); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Record(now.Add(-24*time.Hour), []DockerImage{{ID: "recent", Bytes: 200, Running: true}}); err != nil {
		t.Fatal(err)
	}
	client := &ledgerDockerClient{images: []DockerImage{{ID: "old", Bytes: 100}, {ID: "recent", Bytes: 200}}}
	provider := NewDockerUnusedImagesProvider(client, ledger)
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 7 * 24 * time.Hour}, Scope: cleanup.ObservationScope{Now: now}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 || preview.Items[0].ID != "docker-unused-images:old" {
		t.Fatalf("preview = %#v, want only the ledger-aged old image", preview.Items)
	}
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeOperator, IdempotencyKey: "ledger-apply", Preview: preview})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || result.ReclaimedBytes != 100 || len(client.removed) != 1 || client.removed[0] != "old" {
		t.Fatalf("apply = %#v removed=%#v", result, client.removed)
	}
}

func TestDockerUnusedImagesRefusesShortLedgerWindow(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	ledger := NewMemoryDockerUsageLedger()
	client := &ledgerDockerClient{images: []DockerImage{{ID: "new", Bytes: 100}}}
	provider := NewDockerUnusedImagesProvider(client, ledger)
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 7 * 24 * time.Hour}, Scope: cleanup.ObservationScope{Now: now}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 0 || len(client.removed) != 0 {
		t.Fatalf("short ledger preview = %#v removed=%#v, want no candidates or mutation", preview.Items, client.removed)
	}
}
