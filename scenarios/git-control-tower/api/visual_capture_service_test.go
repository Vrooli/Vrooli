package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"git-control-tower/internal/testutil/fixtures"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

type visualCaptureTestOptions struct{ failOnCall map[int32]bool }

func testCaptureSetup(t *testing.T, opts visualCaptureTestOptions) VisualCaptureDeps {
	t.Helper()
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	repocontracttest.WriteRepoContract(t, repoDir, "scenarios")
	fixtures.WriteScenarioServiceJSON(t, repoDir, "test-app", `{"service":{"name":"test-app"}}`)
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		EnvGet:      func(string) string { return "" },
		UserHomeDir: func() (string, error) { return tmpDir, nil },
	})
	if err != nil {
		t.Fatalf("create storage resolver: %v", err)
	}
	var captureCount int32
	return VisualCaptureDeps{
		CaptureBundle: func(context.Context, string, string, CapturePreset, string) ([]byte, error) {
			n := atomic.AddInt32(&captureCount, 1)
			if opts.failOnCall[n] {
				return nil, fmt.Errorf("fixture capture failure")
			}
			return []byte{0x89, 0x50, 0x4E, 0x47}, nil
		},
		Storage: NewVisualCaptureStorage(resolver, OSFileIO{}), FS: OSFileIO{}, RepoDir: repoDir, RepoID: 1,
	}
}

func TestCaptureScenario_WithLighthousePages(t *testing.T) {
	t.Parallel()

	deps := testCaptureSetup(t, visualCaptureTestOptions{})

	// Create lighthouse.json
	lhDir := filepath.Join(deps.RepoDir, "scenarios", "test-app", ".vrooli")
	if err := os.MkdirAll(lhDir, 0o755); err != nil {
		t.Fatalf("create lighthouse dir: %v", err)
	}
	lhConfig := LighthouseConfig{
		Enabled: true,
		Pages: []LighthousePage{
			{ID: "home", Path: "/", Label: "Home"},
			{ID: "about", Path: "/about", Label: "About"},
		},
	}
	lhData, err := json.Marshal(lhConfig)
	if err != nil {
		t.Fatalf("marshal lighthouse config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(lhDir, "lighthouse.json"), lhData, 0o644); err != nil {
		t.Fatalf("write lighthouse.json: %v", err)
	}

	pages := discoverPages(deps.FS, deps.RepoDir, "test-app")
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages from lighthouse.json, got %d", len(pages))
	}
	if pages[0].Path != "/" {
		t.Errorf("expected first page path /, got %s", pages[0].Path)
	}
	if pages[1].Label != "About" {
		t.Errorf("expected second page label About, got %s", pages[1].Label)
	}
}

func TestCaptureScenario_NoLighthouseConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("create repo dir: %v", err)
	}

	pages := discoverPages(OSFileIO{}, repoDir, "nonexistent")
	if len(pages) != 1 {
		t.Fatalf("expected 1 fallback page, got %d", len(pages))
	}
	if pages[0].Path != "/" {
		t.Errorf("expected fallback path /, got %s", pages[0].Path)
	}
	if pages[0].Label != "Home" {
		t.Errorf("expected fallback label Home, got %s", pages[0].Label)
	}
}

func TestDiscoverPages_IgnoresLegacyAppsLocation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mkdirAllVisualCaptureRepo(t, tmpDir, "my-app")

	// Create lighthouse.json under apps/ instead of the contract-defined scenario path.
	appsDir := filepath.Join(tmpDir, "apps", "my-app", ".vrooli")
	if err := os.MkdirAll(appsDir, 0o755); err != nil {
		t.Fatalf("create apps dir: %v", err)
	}
	lhConfig := LighthouseConfig{
		Enabled: true,
		Pages:   []LighthousePage{{ID: "main", Path: "/main", Label: "Main"}},
	}
	lhData, err := json.Marshal(lhConfig)
	if err != nil {
		t.Fatalf("marshal lighthouse config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appsDir, "lighthouse.json"), lhData, 0o644); err != nil {
		t.Fatalf("write lighthouse.json: %v", err)
	}

	pages := discoverPages(OSFileIO{}, tmpDir, "my-app")
	if len(pages) != 1 {
		t.Fatalf("expected fallback page set, got %d", len(pages))
	}
	if pages[0].Path != "/" {
		t.Errorf("expected fallback path /, got %s", pages[0].Path)
	}
}

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"/", "_root_"},
		{"", "_root_"},
		{"/about", "_about_"},
		{"/workflow/new", "_workflow_new_"},
		{"/settings/profile/edit", "_settings_profile_edit_"},
	}

	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCaptureScenario_BASPartialFailure(t *testing.T) {
	t.Parallel()

	deps := testCaptureSetup(t, visualCaptureTestOptions{failOnCall: map[int32]bool{2: true}})

	meta, err := CaptureScenario(context.Background(), deps, VisualCaptureRequest{
		ScenarioSlug: "test-app",
		Pages:        []string{"/", "/about"},
	})
	if err != nil {
		t.Fatalf("CaptureScenario returned error: %v", err)
	}
	if meta.ScreenshotCount != 1 {
		t.Errorf("expected 1 screenshot (partial), got %d", meta.ScreenshotCount)
	}
	if meta.Status != "complete" {
		t.Errorf("expected status complete, got %s", meta.Status)
	}
}

func TestCaptureScenario_FullFlow(t *testing.T) {
	t.Parallel()

	deps := testCaptureSetup(t, visualCaptureTestOptions{})

	// Create lighthouse.json with two pages
	lhDir := filepath.Join(deps.RepoDir, "scenarios", "test-app", ".vrooli")
	if err := os.MkdirAll(lhDir, 0o755); err != nil {
		t.Fatalf("create lighthouse dir: %v", err)
	}
	lhConfig := LighthouseConfig{
		Enabled: true,
		Pages: []LighthousePage{
			{ID: "home", Path: "/", Label: "Home"},
			{ID: "settings", Path: "/settings", Label: "Settings"},
		},
	}
	lhData, _ := json.Marshal(lhConfig)
	if err := os.WriteFile(filepath.Join(lhDir, "lighthouse.json"), lhData, 0o644); err != nil {
		t.Fatalf("write lighthouse.json: %v", err)
	}

	meta, err := CaptureScenario(context.Background(), deps, VisualCaptureRequest{
		ScenarioSlug: "test-app",
		Mode:         CaptureModeBaseline,
	})
	if err != nil {
		t.Fatalf("CaptureScenario returned error: %v", err)
	}
	if meta.Status != "complete" {
		t.Errorf("expected status complete, got %s (error: %s)", meta.Status, meta.Error)
	}
	if meta.ScreenshotCount != 2 {
		t.Errorf("expected 2 screenshots, got %d", meta.ScreenshotCount)
	}
	if meta.Role != SnapshotRoleBaseline {
		t.Errorf("expected role baseline, got %s", meta.Role)
	}
	if meta.PageDiscoveryMethod != "lighthouse" {
		t.Errorf("expected discovery method lighthouse, got %s", meta.PageDiscoveryMethod)
	}
	if len(meta.Pages) != 2 {
		t.Errorf("expected 2 captured pages, got %d", len(meta.Pages))
	}
}

func TestPrepareForCapture_BaselineMode_IsAdditive(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	resolver := testStorageResolver(t, dir)
	store := NewVisualCaptureStorage(resolver, OSFileIO{})

	// Seed an existing baseline-pinned snapshot and a loose capture.
	for _, m := range []SnapshotSetMeta{
		{ID: "b1", ScenarioSlug: "s", Role: SnapshotRoleBaseline, TriggerType: "manual", CreatedAt: time.Now().UTC().Add(-2 * time.Minute), Status: "complete"},
		{ID: "c1", ScenarioSlug: "s", Role: SnapshotRoleCapture, TriggerType: "manual", CreatedAt: time.Now().UTC(), Status: "complete"},
	} {
		if err := store.SaveSnapshotSet(1, m, map[string][]byte{"_root_.png": {0x89}}, nil); err != nil {
			t.Fatalf("save %s: %v", m.ID, err)
		}
	}

	// Baseline mode is additive: a new baseline must not disturb existing
	// snapshots (other baselines' pins or the loose capture).
	if err := prepareForCapture(store, 1, "s", CaptureModeBaseline); err != nil {
		t.Fatalf("prepareForCapture baseline: %v", err)
	}

	list, _ := store.ListSnapshotSets(1, "s")
	if len(list) != 2 {
		t.Errorf("expected baseline mode to preserve all snapshots, got %d", len(list))
	}
}

func TestPrepareForCapture_CaptureMode_PreservesBaseline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	resolver := testStorageResolver(t, dir)
	store := NewVisualCaptureStorage(resolver, OSFileIO{})

	for _, m := range []SnapshotSetMeta{
		{ID: "b1", ScenarioSlug: "s", Role: SnapshotRoleBaseline, TriggerType: "manual", CreatedAt: time.Now().UTC().Add(-2 * time.Minute), Status: "complete"},
		{ID: "c1", ScenarioSlug: "s", Role: SnapshotRoleCapture, TriggerType: "manual", CreatedAt: time.Now().UTC(), Status: "complete"},
	} {
		if err := store.SaveSnapshotSet(1, m, map[string][]byte{"_root_.png": {0x89}}, nil); err != nil {
			t.Fatalf("save %s: %v", m.ID, err)
		}
	}

	if err := prepareForCapture(store, 1, "s", CaptureModeCapture); err != nil {
		t.Fatalf("prepareForCapture capture: %v", err)
	}

	list, _ := store.ListSnapshotSets(1, "s")
	if len(list) != 1 {
		t.Fatalf("expected 1 snapshot (baseline preserved), got %d", len(list))
	}
	if list[0].ID != "b1" {
		t.Errorf("expected baseline b1 preserved, got %s", list[0].ID)
	}
}

func TestCheckCaptureStaleness_DetectsChanges(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mkdirAllVisualCaptureRepo(t, dir, "test-app")

	// Create scenario files
	scenarioDir := filepath.Join(dir, "scenarios", "test-app", "ui", "src")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	filePath := filepath.Join(scenarioDir, "App.tsx")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	captureTime := time.Now().UTC().Add(-2 * time.Second)
	modTime := captureTime.Add(1 * time.Second)
	if err := os.Chtimes(filePath, modTime, modTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	result := CheckCaptureStaleness(dir, "test-app", captureTime)
	if !result.IsStale {
		t.Error("expected stale when file modified after capture")
	}
}

func TestCheckCaptureStaleness_NotStale(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mkdirAllVisualCaptureRepo(t, dir, "test-app")

	scenarioDir := filepath.Join(dir, "scenarios", "test-app")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	captureTime := time.Now().UTC().Add(1 * time.Minute)
	modTime := captureTime.Add(-1 * time.Minute)
	target := filepath.Join(scenarioDir, "main.go")
	if err := os.Chtimes(target, modTime, modTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	result := CheckCaptureStaleness(dir, "test-app", captureTime)
	if result.IsStale {
		t.Error("expected not stale when capture is after file changes")
	}
}

func TestCheckCaptureStaleness_NoScenarioDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mkdirAllVisualCaptureRepo(t, dir, "existing")

	result := CheckCaptureStaleness(dir, "nonexistent", time.Now().UTC())
	if result.IsStale {
		t.Error("expected not stale for nonexistent scenario dir")
	}
}

func mkdirAllVisualCaptureRepo(t *testing.T, root string, scenarios ...string) {
	t.Helper()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	for _, scenario := range scenarios {
		fixtures.WriteScenarioServiceJSON(t, root, scenario, `{"service":{"name":"`+scenario+`"}}`)
	}
}

func TestPresetSuffix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		preset CapturePreset
		want   string
	}{
		{CapturePreset{Name: "Desktop Light", Width: 1440, Height: 900, Theme: "light"}, "@1440x900_light"},
		{CapturePreset{Name: "Mobile Dark", Width: 390, Height: 844, Theme: "dark"}, "@390x844_dark"},
		{CapturePreset{Name: "Tablet Light", Width: 768, Height: 1024, Theme: "light"}, "@768x1024_light"},
	}
	for _, tc := range tests {
		got := presetSuffix(tc.preset)
		if got != tc.want {
			t.Errorf("presetSuffix(%v) = %q, want %q", tc.preset, got, tc.want)
		}
	}
}

func TestCaptureScenario_MultiplePresets(t *testing.T) {
	t.Parallel()

	deps := testCaptureSetup(t, visualCaptureTestOptions{})

	// Create lighthouse.json with 2 pages
	lhDir := filepath.Join(deps.RepoDir, "scenarios", "test-app", ".vrooli")
	if err := os.MkdirAll(lhDir, 0o755); err != nil {
		t.Fatalf("create lighthouse dir: %v", err)
	}
	lhConfig := LighthouseConfig{
		Enabled: true,
		Pages: []LighthousePage{
			{ID: "home", Path: "/", Label: "Home"},
			{ID: "about", Path: "/about", Label: "About"},
		},
	}
	lhData, _ := json.Marshal(lhConfig)
	if err := os.WriteFile(filepath.Join(lhDir, "lighthouse.json"), lhData, 0o644); err != nil {
		t.Fatalf("write lighthouse.json: %v", err)
	}

	presets := []CapturePreset{
		{Name: "Desktop Light", Width: 1440, Height: 900, Theme: "light"},
		{Name: "Mobile Dark", Width: 390, Height: 844, Theme: "dark"},
	}
	meta, err := CaptureScenario(context.Background(), deps, VisualCaptureRequest{
		ScenarioSlug: "test-app",
		Mode:         CaptureModeBaseline,
		Presets:      presets,
	})
	if err != nil {
		t.Fatalf("CaptureScenario returned error: %v", err)
	}
	if meta.Status != "complete" {
		t.Errorf("expected status complete, got %s (error: %s)", meta.Status, meta.Error)
	}
	// 2 pages × 2 presets = 4 screenshots
	if meta.ScreenshotCount != 4 {
		t.Errorf("expected 4 screenshots (2 pages × 2 presets), got %d", meta.ScreenshotCount)
	}
	if len(meta.Presets) != 2 {
		t.Errorf("expected 2 presets in metadata, got %d", len(meta.Presets))
	}
	if len(meta.Pages) != 2 {
		t.Errorf("expected 2 captured pages, got %d", len(meta.Pages))
	}
}
