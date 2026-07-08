package providers

import (
	"context"
	"testing"

	"cleanup-manager/internal/cleanup"
	cleanupfakes "cleanup-manager/internal/testutil/cleanup"
)

func TestOwnerScenarioProviderDelegatesOnlyAfterRequiredApproval(t *testing.T) {
	t.Parallel()

	client := &cleanupfakes.ScenarioProviderClient{
		EstimateResult: cleanup.Estimate{ProviderID: "web-console-sessions", ProviderVersion: "v1", EstimatedBytes: 300, ItemCount: 1},
		PreviewResult:  cleanup.Preview{ProviderID: "web-console-sessions", ProviderVersion: "v1", Items: []cleanup.PreviewItem{{ID: "session-1", Bytes: 300}}},
		ApplyResult:    cleanup.ApplyResult{ProviderID: "web-console-sessions", Applied: true, ReclaimedBytes: 300},
	}
	provider := NewOwnerScenarioProvider(OwnerProviderConfig{
		ID:              "web-console-sessions",
		Name:            "Web Console old sessions",
		OwnerScenario:   "web-console",
		SafetyTier:      cleanup.SafetyTierSafeWithOwner,
		DefaultMode:     cleanup.ProviderModeDisabled,
		DefaultApproval: cleanup.ApprovalModeOwner,
	}, client)
	policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOwner}

	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: policy})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: policy, Estimate: estimate})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	skipped, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeNone, IdempotencyKey: "owner-1", Preview: preview})
	if err != nil {
		t.Fatalf("Apply() without owner approval error = %v", err)
	}
	if skipped.Applied || len(client.Applies) != 0 {
		t.Fatalf("Apply() without approval = %#v applies=%#v, want no delegation", skipped, client.Applies)
	}
	applied, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeOwner, IdempotencyKey: "owner-2", Preview: preview})
	if err != nil {
		t.Fatalf("Apply() with owner approval error = %v", err)
	}
	if !applied.Applied || applied.ReclaimedBytes != 300 || len(client.Applies) != 1 {
		t.Fatalf("Apply() with owner approval = %#v applies=%#v, want delegated apply", applied, client.Applies)
	}
	if client.Applies[0].ScenarioID != "web-console" {
		t.Fatalf("delegated scenario = %q, want web-console", client.Applies[0].ScenarioID)
	}
}
