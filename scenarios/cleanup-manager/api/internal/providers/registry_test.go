package providers

import (
	"context"
	"testing"
	"time"

	"cleanup-manager/internal/cleanup"
	cleanupfakes "cleanup-manager/internal/testutil/cleanup"
)

func TestRegistryRejectsDuplicateProviders(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	fsys := &cleanupfakes.FileSystem{Root: "/fake", Files: map[string]cleanup.FileInfo{}}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{}, FileProviderConfig{ID: "tmp", Name: "Temporary files", Roots: []string{"/fake/tmp"}})
	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register() first error = %v", err)
	}
	if err := registry.Register(provider); err == nil {
		t.Fatal("Register() duplicate expected error")
	}
}

func TestConservativeBuiltInsValidateAndSortCatalog(t *testing.T) {
	t.Parallel()

	providers, err := ConservativeBuiltIns(BuiltInDeps{
		FileSystem:           &cleanupfakes.FileSystem{Root: "/fake", Files: map[string]cleanup.FileInfo{}},
		ProcessRunner:        &cleanupfakes.ProcessRunner{Result: cleanup.ProcessResult{Stdout: "1024"}},
		Docker:               &cleanupfakes.DockerClient{},
		Journal:              &cleanupfakes.JournalClient{},
		OwnerScenarioClient:  &cleanupfakes.ScenarioProviderClient{},
		Clock:                cleanupfakes.Clock{Time: time.Unix(10, 0)},
		TrashRoots:           []string{"/fake/trash"},
		TmpRoots:             []string{"/fake/tmp"},
		GoBuildCacheRoots:    []string{"/fake/go-build"},
		PlaywrightCacheRoots: []string{"/fake/playwright"},
	})
	if err != nil {
		t.Fatalf("ConservativeBuiltIns() error = %v", err)
	}
	registry, err := NewRegistry(providers...)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	got := registry.List()
	if len(got) != 10 {
		t.Fatalf("List() len = %d, want 10", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].ID > got[i].ID {
			t.Fatalf("List() not sorted: %q before %q", got[i-1].ID, got[i].ID)
		}
	}
	if _, ok := registry.Get("docker"); !ok {
		t.Fatal("Get(\"docker\") missing built-in provider")
	}
	for _, id := range []string{"workspace-sandbox-retention", "test-genie-run-retention", "web-console-sessions"} {
		provider, ok := registry.Get(id)
		if !ok {
			t.Fatalf("Get(%q) missing owner-scenario provider", id)
		}
		meta := provider.Metadata()
		if meta.SafetyTier != cleanup.SafetyTierSafeWithOwner || meta.DefaultMode != cleanup.ProviderModeDisabled || meta.DefaultApproval != cleanup.ApprovalModeOwner {
			t.Fatalf("%s metadata = %#v, want disabled safe_with_owner owner approval", id, meta)
		}
	}
}

func TestOwnerScenarioBuiltInsDelegateThroughOwnerClientOnly(t *testing.T) {
	t.Parallel()

	client := &cleanupfakes.ScenarioProviderClient{
		EstimateResult: cleanup.Estimate{ProviderID: "workspace-sandbox-retention", ProviderVersion: "v1", EstimatedBytes: 4096, ItemCount: 2},
		PreviewResult: cleanup.Preview{ProviderID: "workspace-sandbox-retention", ProviderVersion: "v1", Items: []cleanup.PreviewItem{
			{ID: "sandbox-old", Description: "expired sandbox", Bytes: 4096, SafetyTier: cleanup.SafetyTierSafeWithOwner},
		}},
		ApplyResult: cleanup.ApplyResult{ProviderID: "workspace-sandbox-retention", Applied: true, ReclaimedBytes: 4096},
	}
	registry, err := NewRegistry(OwnerScenarioBuiltIns(client)...)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	provider, ok := registry.Get("workspace-sandbox-retention")
	if !ok {
		t.Fatal("workspace-sandbox-retention provider missing")
	}
	policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOwner}
	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: policy})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: policy, Estimate: estimate})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: "v1",
		ApprovalMode:    cleanup.ApprovalModeOwner,
		IdempotencyKey:  "sandbox-retention-1",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !result.Applied || len(client.Applies) != 1 {
		t.Fatalf("Apply() = %#v applies=%#v, want one delegated owner apply", result, client.Applies)
	}
	if got := client.Applies[0]; got.ScenarioID != "workspace-sandbox" || got.ProviderID != "workspace-sandbox-retention" {
		t.Fatalf("delegated request = %#v, want workspace-sandbox owner hook", got)
	}
}

func TestFileProviderEstimatePreviewAndApplyUseFakeRootOnly(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	fsys := &cleanupfakes.FileSystem{
		Root:        "/fake",
		AllowRemove: true,
		Files: map[string]cleanup.FileInfo{
			"/fake/tmp/old.log":    {Path: "/fake/tmp/old.log", Size: 64, ModTime: now.Add(-48 * time.Hour)},
			"/fake/tmp/new.log":    {Path: "/fake/tmp/new.log", Size: 128, ModTime: now.Add(-time.Hour)},
			"/fake/tmp/live.lock":  {Path: "/fake/tmp/live.lock", Size: 256, ModTime: now.Add(-72 * time.Hour)},
			"/fake/other/old.log":  {Path: "/fake/other/old.log", Size: 512, ModTime: now.Add(-72 * time.Hour)},
			"/outside/tmp/old.log": {Path: "/outside/tmp/old.log", Size: 1024, ModTime: now.Add(-72 * time.Hour)},
		},
	}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "tmp", Name: "Temporary files", Roots: []string{"/fake/tmp"}})
	policy := cleanup.ProviderPolicy{Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOperator}

	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: policy})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if estimate.EstimatedBytes != 64 || estimate.ItemCount != 1 {
		t.Fatalf("Estimate() = bytes %d count %d, want 64/1", estimate.EstimatedBytes, estimate.ItemCount)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: policy, Estimate: estimate})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if len(preview.Items) != 1 || preview.Items[0].Path != "/fake/tmp/old.log" {
		t.Fatalf("Preview() items = %#v, want only old fake tmp file", preview.Items)
	}
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		PlanID:          "plan-1",
		ProviderVersion: "v1",
		ApprovalMode:    cleanup.ApprovalModeOperator,
		IdempotencyKey:  "apply-1",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !result.Applied || result.ReclaimedBytes != 64 {
		t.Fatalf("Apply() = %#v, want applied 64 bytes", result)
	}
	if len(fsys.Removed) != 1 || fsys.Removed[0] != "/fake/tmp/old.log" {
		t.Fatalf("Removed = %#v, want fake tmp old file", fsys.Removed)
	}
}

func TestFileProviderDisabledPolicyDoesNotPreviewOrApply(t *testing.T) {
	t.Parallel()

	fsys := &cleanupfakes.FileSystem{Root: "/fake", Files: map[string]cleanup.FileInfo{"/fake/tmp/old.log": {Path: "/fake/tmp/old.log", Size: 64}}}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{}, FileProviderConfig{ID: "tmp", Name: "Temporary files", Roots: []string{"/fake/tmp"}})
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: false}})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if preview.BlockedReason == "" || len(preview.Items) != 0 {
		t.Fatalf("Preview() = %#v, want blocked with no items", preview)
	}
}
