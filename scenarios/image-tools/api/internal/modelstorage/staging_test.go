package modelstorage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"image-tools/internal/fetch"
)

// stagingHarness is a service whose models root is a real temp directory.
func stagingHarness(t *testing.T, cfg Config) (*harness, string) {
	t.Helper()
	h := newHarness(t, 100*gib, cfg)
	root := t.TempDir()
	h.svc.deps.Root = root
	return h, root
}

// stageDir creates a staging directory holding size bytes whose newest write
// happened `silent` before the test clock.
func stageDir(t *testing.T, parent, name string, size int, silent time.Duration) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(dir, ".cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, ".cache", "blob.incomplete")
	if err := os.WriteFile(file, make([]byte, size), 0o600); err != nil {
		t.Fatal(err)
	}
	at := testNow.Add(-silent)
	for _, p := range []string{file, filepath.Join(dir, ".cache"), dir} {
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func stateOf(t *testing.T, entries []StagingEntry, dir string) StagingEntry {
	t.Helper()
	for _, e := range entries {
		if e.Path == dir {
			return e
		}
	}
	t.Fatalf("%s not listed in %v", dir, entries)
	return StagingEntry{}
}

func TestStagingAbandonmentDecision(t *testing.T) {
	h, root := stagingHarness(t, Config{StagingGrace: time.Hour})
	weights := filepath.Join(root, "models")
	models := filepath.Join(root, "staging", "models")
	adapters := filepath.Join(root, "staging", "adapters")
	if err := os.MkdirAll(weights, 0o755); err != nil {
		t.Fatal(err)
	}
	owned := stageDir(t, models, ".acquiring-owned-aaaaaaaaaaaa", 1<<20, 48*time.Hour)
	// Staging an earlier release left beside the weights is still reclaimed.
	stale := stageDir(t, weights, ".acquiring-858204562", 1<<20, 6*time.Hour)
	fresh := stageDir(t, models, ".acquiring-fresh-bbbbbbbbbbbb", 1<<20, 10*time.Minute)
	adapterStale := stageDir(t, adapters, ".acquiring-lora-cccccccccccc", 1<<20, 3*time.Hour)
	// Replaced installations: one whose destination is present, one without.
	covered := stageDir(t, models, ".acquiring-flux-dddddddddddd"+fetch.PreviousSuffix, 1<<20, 5*time.Hour)
	if err := os.MkdirAll(filepath.Join(weights, "flux"), 0o755); err != nil {
		t.Fatal(err)
	}
	sole := stageDir(t, models, ".acquiring-lost-eeeeeeeeeeee"+fetch.PreviousSuffix, 1<<20, 5*time.Hour)
	if err := os.WriteFile(filepath.Join(weights, "not-staging.bin"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !fetch.HoldStage(owned) {
		t.Fatal("hold failed")
	}
	defer fetch.ReleaseStage(owned)

	entries, err := h.svc.Staging(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Fatalf("listed %d entries, want 6 staging dirs only: %v", len(entries), entries)
	}
	want := map[string]string{
		owned: StageActive, stale: StageAbandoned, fresh: StageFresh, adapterStale: StageAbandoned,
		covered: StageAbandoned, sole: StageRecoverable,
	}
	for dir, state := range want {
		if got := stateOf(t, entries, dir); got.State != state || got.Reason == "" {
			t.Errorf("%s state=%q reason=%q, want %q", filepath.Base(dir), got.State, got.Reason, state)
		}
	}
	if got := stateOf(t, entries, stale); got.Bytes < 1<<20 {
		t.Errorf("bytes=%d, want at least the 1 MiB written", got.Bytes)
	}
}

func TestStagingNewestWriteAnywhereInTreeKeepsItFresh(t *testing.T) {
	h, root := stagingHarness(t, Config{StagingGrace: time.Hour})
	models := filepath.Join(root, "staging", "models")
	dir := stageDir(t, models, ".acquiring-busy-ffffffffffff", 1<<20, 10*time.Hour)
	recent := testNow.Add(-time.Minute)
	if err := os.Chtimes(filepath.Join(dir, ".cache", "blob.incomplete"), recent, recent); err != nil {
		t.Fatal(err)
	}
	entries, err := h.svc.Staging(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, entries, dir); got.State != StageFresh {
		t.Fatalf("state=%q: a recent write deep in the tree must keep the directory fresh", got.State)
	}
}

func TestSweepStagingReclaimsOnlyAbandonedAndCountsBytes(t *testing.T) {
	h, root := stagingHarness(t, Config{StagingGrace: time.Hour, AutoOffload: true})
	models := filepath.Join(root, "staging", "models")
	owned := stageDir(t, models, ".acquiring-owned-aaaaaaaaaaaa", 1<<20, 48*time.Hour)
	stale := stageDir(t, filepath.Join(root, "models"), ".acquiring-858204562", 2<<20, 6*time.Hour)
	fresh := stageDir(t, models, ".acquiring-fresh-bbbbbbbbbbbb", 1<<20, time.Minute)
	sole := stageDir(t, models, ".acquiring-lost-eeeeeeeeeeee"+fetch.PreviousSuffix, 1<<20, 9*time.Hour)
	if !fetch.HoldStage(owned) {
		t.Fatal("hold failed")
	}
	defer fetch.ReleaseStage(owned)

	report, err := h.svc.Plan(context.Background(), TriggerRequested, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Staging.AbandonedCount != 1 || report.Staging.AbandonedBytes < 2<<20 || report.Staging.TotalBytes < report.Staging.AbandonedBytes+3<<20 || report.Staging.GraceSeconds != 3600 {
		t.Fatalf("staging summary = %+v", report.Staging)
	}

	reclaimed, err := h.svc.SweepStaging(context.Background(), TriggerPeriodic, true)
	if err != nil || len(reclaimed) != 1 {
		t.Fatalf("dry run reclaimed=%v err=%v", reclaimed, err)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Fatalf("dry run removed the directory: %v", err)
	}

	reclaimed, err = h.svc.SweepStaging(context.Background(), TriggerPeriodic, false)
	if err != nil || len(reclaimed) != 1 || reclaimed[0].Path != stale {
		t.Fatalf("reclaimed=%v err=%v", reclaimed, err)
	}
	if _, err := os.Lstat(stale); !os.IsNotExist(err) {
		t.Fatalf("abandoned staging still present: %v", err)
	}
	for _, kept := range []string{owned, fresh, sole} {
		if _, err := os.Stat(kept); err != nil {
			t.Errorf("%s was removed: %v", filepath.Base(kept), err)
		}
	}
	if fetch.StageHeld(stale) {
		t.Error("sweep leaked its ownership hold")
	}
	again, err := h.svc.SweepStaging(context.Background(), TriggerPeriodic, false)
	if err != nil || len(again) != 0 {
		t.Fatalf("second sweep reclaimed %v err=%v", again, err)
	}
}

func TestSweepRechecksOwnershipUnderTheHold(t *testing.T) {
	h, root := stagingHarness(t, Config{StagingGrace: time.Hour, AutoOffload: true})
	stale := stageDir(t, filepath.Join(root, "staging", "models"), ".acquiring-race-aaaaaaaaaaaa", 1<<20, 6*time.Hour)
	// An acquisition claims and writes between the listing and the removal.
	h.svc.deps.Now = func() time.Time { return testNow }
	entries, err := h.svc.Staging(context.Background())
	if err != nil || stateOf(t, entries, stale).State != StageAbandoned {
		t.Fatalf("setup: %v %v", entries, err)
	}
	if err := os.WriteFile(filepath.Join(stale, "new.bin"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(stale, "new.bin"), testNow, testNow); err != nil {
		t.Fatal(err)
	}
	reclaimed, err := h.svc.SweepStaging(context.Background(), TriggerPeriodic, false)
	if err != nil || len(reclaimed) != 0 {
		t.Fatalf("a directory written since inspection was reclaimed: %v %v", reclaimed, err)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Fatal(err)
	}
}

func TestSweepPassOnlyReportsWhenAutoOffloadIsOff(t *testing.T) {
	h, root := stagingHarness(t, Config{StagingGrace: time.Hour, AutoOffload: false})
	stale := stageDir(t, filepath.Join(root, "models"), ".acquiring-858204562", 1<<20, 6*time.Hour)
	if err := h.svc.sweepPass(context.Background(), TriggerBoot); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Fatalf("auto-offload off must not delete: %v", err)
	}
	h.svc.cfg.AutoOffload = true
	if err := h.svc.sweepPass(context.Background(), TriggerBoot); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(stale); !os.IsNotExist(err) {
		t.Fatalf("auto-offload on must reclaim: %v", err)
	}
}

func TestStagingGraceConfig(t *testing.T) {
	cfg, err := ConfigFromEnv(func(string) string { return "" }, gib)
	if err != nil || cfg.StagingGrace != DefaultStagingGrace {
		t.Fatalf("default grace = %v err=%v", cfg.StagingGrace, err)
	}
	cfg, err = ConfigFromEnv(func(k string) string {
		if k == EnvStagingGrace {
			return "45m"
		}
		return ""
	}, gib)
	if err != nil || cfg.StagingGrace != 45*time.Minute {
		t.Fatalf("grace = %v err=%v", cfg.StagingGrace, err)
	}
	if _, err := ConfigFromEnv(func(k string) string {
		if k == EnvStagingGrace {
			return "-1h"
		}
		return ""
	}, gib); err == nil {
		t.Fatal("negative grace accepted")
	}
}
