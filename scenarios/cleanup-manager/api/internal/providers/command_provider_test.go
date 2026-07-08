package providers

import (
	"context"
	"testing"

	"cleanup-manager/internal/cleanup"
	cleanupfakes "cleanup-manager/internal/testutil/cleanup"
)

func TestJournalProviderUsesJournalClientAndSkipsWithoutOperatorApproval(t *testing.T) {
	t.Parallel()

	client := &cleanupfakes.JournalClient{Usage: 2048}
	provider := NewJournalProvider(client)
	policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}

	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: policy})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if estimate.EstimatedBytes != 2048 {
		t.Fatalf("Estimate() bytes = %d, want 2048", estimate.EstimatedBytes)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: policy, Estimate: estimate})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	skipped, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeNone, IdempotencyKey: "journal-1", Preview: preview})
	if err != nil {
		t.Fatalf("Apply() without approval error = %v", err)
	}
	if skipped.Applied || len(client.Vacuums) != 0 {
		t.Fatalf("Apply() without approval = %#v vacuums=%#v, want no vacuum", skipped, client.Vacuums)
	}
	applied, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeOperator, IdempotencyKey: "journal-2", Preview: preview})
	if err != nil {
		t.Fatalf("Apply() with approval error = %v", err)
	}
	if !applied.Applied || applied.ReclaimedBytes != 2048 || len(client.Vacuums) != 1 {
		t.Fatalf("Apply() with approval = %#v vacuums=%#v, want one fake vacuum", applied, client.Vacuums)
	}
}

func TestCommandMetadataProviderPreviewOnly(t *testing.T) {
	t.Parallel()

	runner := &cleanupfakes.ProcessRunner{Forbidden: []string{"apt clean"}, Result: cleanup.ProcessResult{Stdout: "4096 bytes"}}
	provider := NewCommandMetadataProvider(CommandProviderConfig{
		ID:                 "apt-metadata",
		Name:               "APT metadata",
		SafetyTier:         cleanup.SafetyTierConditional,
		DefaultMode:        cleanup.ProviderModeDisabled,
		DefaultApproval:    cleanup.ApprovalModeOperator,
		RequiredPrivileges: []string{"sudo"},
		TestSubstitute:     "fake-process-runner",
		EstimateCommand:    cleanup.ProcessCommand{Name: "apt-cache", Args: []string{"stats"}},
		PreviewAction:      "apt-metadata-clean",
	}, runner)
	policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}

	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: policy})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if estimate.EstimatedBytes != 4096 {
		t.Fatalf("Estimate() bytes = %d, want 4096", estimate.EstimatedBytes)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: policy, Estimate: estimate})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if len(preview.Items) != 1 || len(preview.Warnings) == 0 {
		t.Fatalf("Preview() = %#v, want item plus privilege warning", preview)
	}
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeOperator, IdempotencyKey: "apt-1", Preview: preview})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Applied {
		t.Fatalf("Apply() = %#v, command metadata provider should remain preview-only", result)
	}
}
