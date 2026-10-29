package providers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"storage-manager/hostfs"
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

// Disk-pressure recovery removed ~/.vrooli/logs/scenarios while every running
// scenario wrote its stdout there: the logs became unlinked-but-open files,
// `vrooli scenario logs` found nothing, and no blocks were freed. A held log
// keeps its top-level entry out of the plan, and an entry that becomes held
// after preview is skipped at apply.
func TestRuntimeHomeProviderNeverRemovesLogsHeldOpenUnderRecovery(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("open-handle discovery reads /proc")
	}
	root := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)
	held := filepath.Join(root, "scenarios", "live-app", "start-api.log")
	late := filepath.Join(root, "late", "start-api.log")
	stale := filepath.Join(root, "stopped", "start-api.log")
	for _, path := range []string{held, late, stale} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("log line\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, touched := range []string{path, filepath.Dir(path), filepath.Dir(filepath.Dir(path))} {
			if touched != root {
				_ = os.Chtimes(touched, old, old)
			}
		}
	}
	writer, err := os.OpenFile(held, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()

	provider := NewRuntimeHomeProvider(hostfs.New(hostfs.Options{}), cleanupfakes.Clock{Time: time.Now()}, FileProviderConfig{
		ID: "runtime-home-logs", Name: "Runtime home logs", Roots: []string{root},
		TopLevelEntries: true, RetentionMaxAge: 30 * 24 * time.Hour, RetentionMaxBytes: 10 << 30,
		ProtectActive: true, LeaseCheck: "open_handle", OpenHandleSkipsNonDumpable: true,
	})
	// Recovery waives the age floor, which is exactly when only the live
	// handle stands between a running scenario and its log.
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Policy: cleanup.ProviderPolicy{Enabled: true, AllowFreshReclaim: true, ApprovalMode: cleanup.ApprovalModeNone},
		Scope:  cleanup.ObservationScope{Now: time.Now(), Recovery: true},
	})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	planned := map[string]bool{}
	for _, item := range preview.Items {
		planned[item.Path] = true
	}
	if planned[filepath.Join(root, "scenarios")] {
		t.Fatalf("preview planned the directory of a held log: %#v", preview.Items)
	}
	if !planned[filepath.Join(root, "late")] || !planned[filepath.Join(root, "stopped")] {
		t.Fatalf("preview = %#v (warnings %v, blocked %q), want unheld entries planned", preview.Items, preview.Warnings, preview.BlockedReason)
	}

	// A scenario starts and opens its log between preview and apply.
	lateWriter, err := os.OpenFile(late, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lateWriter.Close()
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		Preview: preview, ProviderVersion: provider.Metadata().Version,
		IdempotencyKey: "held-log-regression", ApprovalMode: cleanup.ApprovalModeNone,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	for _, path := range []string{held, late} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("held log %s removed: %v (result %#v)", path, err, result)
		}
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("unheld stale log survived: %v", err)
	}
}
