package providers

import (
	"context"
	"os"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
)

// TestLiveTestGenieCleanupDelegation is an opt-in destructive integration
// proof. It deletes only the exact terminal run named by
// TEST_GENIE_CLEANUP_ITEM_ID after that item appears in the owner's preview.
// The fleet-wide preview is never forwarded to Apply.
func TestLiveTestGenieCleanupDelegation(t *testing.T) {
	baseURL := os.Getenv("TEST_GENIE_CLEANUP_URL")
	itemID := os.Getenv("TEST_GENIE_CLEANUP_ITEM_ID")
	idempotencyKey := os.Getenv("TEST_GENIE_CLEANUP_IDEMPOTENCY_KEY")
	if baseURL == "" || itemID == "" || idempotencyKey == "" {
		t.Skip("set TEST_GENIE_CLEANUP_URL, TEST_GENIE_CLEANUP_ITEM_ID, and TEST_GENIE_CLEANUP_IDEMPOTENCY_KEY for the live owner-delegation proof")
	}

	client := &cleanup.HTTPScenarioProviderClient{
		ResolveURL: func(context.Context, string) (string, error) { return baseURL, nil },
	}
	provider := NewOwnerScenarioProvider(OwnerProviderConfig{
		ID:              "test-genie-run-retention",
		Name:            "Test Genie retained runs",
		OwnerScenario:   "test-genie",
		SafetyTier:      cleanup.SafetyTierSafeWithOwner,
		DefaultMode:     cleanup.ProviderModeDisabled,
		DefaultApproval: cleanup.ApprovalModeOwner,
	}, client)
	policy := cleanup.ProviderPolicy{Enabled: true, MinAge: 30 * 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOwner}

	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: policy})
	if err != nil || estimate.BlockedReason != "" {
		t.Fatalf("estimate = %#v, err = %v", estimate, err)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: policy, Estimate: estimate})
	if err != nil || preview.BlockedReason != "" {
		t.Fatalf("preview = %#v, err = %v", preview, err)
	}
	var selected cleanup.PreviewItem
	for _, item := range preview.Items {
		if item.ID == itemID {
			selected = item
			break
		}
	}
	if selected.ID == "" {
		t.Fatalf("explicit item %q is not an owner-approved cleanup candidate", itemID)
	}
	filtered := preview.Clone()
	filtered.Items = []cleanup.PreviewItem{selected}
	request := cleanup.ApplyRequest{ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeOwner, IdempotencyKey: idempotencyKey, Preview: filtered}

	first, err := provider.Apply(context.Background(), request)
	if err != nil || !first.Applied || first.ReclaimedBytes <= 0 || len(first.AppliedItems) != 1 || first.AppliedItems[0] != itemID {
		t.Fatalf("first apply = %#v, err = %v", first, err)
	}
	second, err := provider.Apply(context.Background(), request)
	if err != nil || !second.AlreadyDone || second.Applied || second.ReclaimedBytes != 0 || len(second.AppliedItems) != 0 {
		t.Fatalf("idempotent replay = %#v, err = %v", second, err)
	}
	if _, err := os.Stat(selected.Path); !os.IsNotExist(err) {
		t.Fatalf("owner reported deletion but artifact remains at %q: %v", selected.Path, err)
	}
	t.Logf("reclaimed %d bytes for %s through Storage Manager owner delegation", first.ReclaimedBytes, itemID)
}
