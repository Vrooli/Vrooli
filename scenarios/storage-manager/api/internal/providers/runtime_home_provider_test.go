package providers

import (
	"context"
	"errors"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

type fakeOwnershipRepairer struct {
	calls int
}

func (r *fakeOwnershipRepairer) Repair(context.Context, string) (cleanup.OwnershipRepairResult, error) {
	r.calls++
	return cleanup.OwnershipRepairResult{Repaired: 1}, nil
}

func TestRuntimeHomeProviderNeverBroadensContractRetention(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	fsys := &cleanupfakes.FileSystem{
		Root:        "/fake",
		AllowRemove: true,
		Files: map[string]cleanup.FileInfo{
			"/fake/runtime/old":    {Path: "/fake/runtime/old", Size: 100, ModTime: now.Add(-40 * 24 * time.Hour)},
			"/fake/runtime/recent": {Path: "/fake/runtime/recent", Size: 200, ModTime: now.Add(-10 * 24 * time.Hour)},
		},
	}
	provider := NewRuntimeHomeProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "runtime-home-artifacts", Name: "Runtime artifacts", Roots: []string{"/fake/runtime"},
		TopLevelEntries: true, RetentionMaxAge: 30 * 24 * time.Hour, RetentionMaxBytes: 150,
	})
	meta := provider.Metadata()
	if meta.SafetyTier != cleanup.SafetyTierRegenerable || !meta.NoLease {
		t.Fatalf("runtime-home metadata = %#v, want regenerable with NoLease proof", meta)
	}

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{
		Enabled: true, MinAge: 24 * time.Hour, MaxBytes: 1000, ApprovalMode: cleanup.ApprovalModeOwner,
	}, Scope: cleanup.ObservationScope{Now: now}})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(preview.Items) != 1 || preview.Items[0].Path != "/fake/runtime/old" {
		t.Fatalf("Preview items = %#v, want only contract-eligible old entry", preview.Items)
	}
	if preview.Items[0].Bytes != 100 {
		t.Fatalf("Preview bytes = %d, want 100", preview.Items[0].Bytes)
	}
}

func TestRuntimeHomeProviderProtectsActiveLeaseMarkers(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	fsys := &cleanupfakes.FileSystem{
		Root:        "/fake",
		AllowRemove: true,
		Files: map[string]cleanup.FileInfo{
			"/fake/runtime/old":          {Path: "/fake/runtime/old", Size: 100, ModTime: now.Add(-40 * 24 * time.Hour)},
			"/fake/runtime/job.active":   {Path: "/fake/runtime/job.active", Size: 200, ModTime: now.Add(-40 * 24 * time.Hour)},
			"/fake/runtime/job.finished": {Path: "/fake/runtime/job.finished", Size: 300, ModTime: now.Add(-40 * 24 * time.Hour)},
		},
	}
	provider := NewRuntimeHomeProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "runtime-home-artifacts", Name: "Runtime artifacts", Roots: []string{"/fake/runtime"},
		TopLevelEntries: true, RetentionMaxAge: 30 * 24 * time.Hour, ProtectActive: true,
	})

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{
		Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOwner,
	}, Scope: cleanup.ObservationScope{Now: now}})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(preview.Items) != 2 || preview.Items[0].Path != "/fake/runtime/job.finished" || preview.Items[1].Path != "/fake/runtime/old" {
		t.Fatalf("Preview items = %#v, want active lease excluded", preview.Items)
	}
}

func TestRuntimeHomeProviderRepairsOwnershipAndRetriesOnce(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	fsys := &cleanupfakes.FileSystem{
		Root: "/fake", AllowRemove: true,
		Files:        map[string]cleanup.FileInfo{"/fake/runtime/old": {Path: "/fake/runtime/old", Size: 100, ModTime: now.Add(-40 * 24 * time.Hour)}},
		RemoveErrors: []error{errors.New("permission denied")},
	}
	repairer := &fakeOwnershipRepairer{}
	provider := NewRuntimeHomeProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "runtime-home-artifacts", Name: "Runtime artifacts", Roots: []string{"/fake/runtime"},
		RepairClass: "artifacts", OwnershipRepairer: repairer,
	})

	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: "v1", ApprovalMode: cleanup.ApprovalModeOwner, IdempotencyKey: "retry-once",
		Preview: cleanup.Preview{ProviderID: "runtime-home-artifacts", ProviderVersion: "v1", Items: []cleanup.PreviewItem{{ID: "artifact-old", Path: "/fake/runtime/old", Bytes: 100}}},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if repairer.calls != 1 || result.RepairAttempts != 1 || result.Repairs != 1 || result.RetryAttempts != 1 || result.ReclaimedBytes != 100 {
		t.Fatalf("Apply result = %#v, repair calls = %d", result, repairer.calls)
	}
}
