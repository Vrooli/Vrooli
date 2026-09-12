package providers

import (
	"context"
	"testing"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

type recordingBroker struct {
	actions  []string
	subjects []map[string]any
	changed  bool
}

func (b *recordingBroker) Do(_ context.Context, action string, subject map[string]any) (cleanup.BrokerActionResult, error) {
	b.actions = append(b.actions, action)
	b.subjects = append(b.subjects, subject)
	return cleanup.BrokerActionResult{Changed: b.changed}, nil
}

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

func TestJournalProviderUsesBrokerForStandingApprovalPath(t *testing.T) {
	client := &cleanupfakes.JournalClient{Usage: 2048}
	broker := &recordingBroker{changed: true}
	provider := NewJournalProvider(client, broker)
	preview := cleanup.Preview{ProviderID: "journald", ProviderVersion: "v1", Items: []cleanup.PreviewItem{{Bytes: 2048}}}
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeNone, IdempotencyKey: "standing-1", Preview: preview})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !result.Applied || len(broker.actions) != 1 || broker.actions[0] != "journald.vacuum" {
		t.Fatalf("result=%#v broker actions=%#v, want brokered vacuum", result, broker.actions)
	}
	journal, ok := broker.subjects[0]["journal"].(map[string]any)
	if !ok || journal["max_use_bytes"] != int64(1<<30) {
		t.Fatalf("broker subject=%#v, want one-gigabyte journal bound", broker.subjects[0])
	}
	if len(client.Vacuums) != 0 {
		t.Fatalf("journal client vacuum calls=%#v, standing approval must not bypass broker", client.Vacuums)
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
