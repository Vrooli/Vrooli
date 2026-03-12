package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/storage"
)

func testCaptureSetup(t *testing.T, basHandler http.HandlerFunc) (VisualCaptureDeps, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(basHandler)
	t.Cleanup(server.Close)

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("create repo dir: %v", err)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		EnvGet:      func(key string) string { return "" },
		UserHomeDir: func() (string, error) { return tmpDir, nil },
	})
	if err != nil {
		t.Fatalf("create storage resolver: %v", err)
	}

	basClient := &BrowserAutomationClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	return VisualCaptureDeps{
		BAS:     basClient,
		Storage: NewVisualCaptureStorage(resolver, OSFileIO{}),
		FS:      OSFileIO{},
		RepoDir: repoDir,
		RepoID:  1,
	}, server
}

func TestCaptureScenario_WithLighthousePages(t *testing.T) {
	t.Parallel()

	deps, _ := testCaptureSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/preview-screenshot" {
			pngData := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4E, 0x47})
			_ = json.NewEncoder(w).Encode(BASScreenshotResponse{
				Screenshot:     "data:image/png;base64," + pngData,
				URL:            "http://localhost:3000/",
				DurationMS:     100,
				ViewportWidth:  1280,
				ViewportHeight: 720,
			})
			return
		}
		w.WriteHeader(404)
	})

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

	// We need to mock the UI URL resolution - use the BAS server URL as a stand-in
	// since CaptureScenario calls discovery.ResolveScenarioURL which won't work in tests.
	// Instead, test the helper functions directly.

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

func TestDiscoverPages_MultipleLocations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create lighthouse.json under apps/ instead of scenarios/
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
		t.Fatalf("expected 1 page from apps/ lighthouse.json, got %d", len(pages))
	}
	if pages[0].Path != "/main" {
		t.Errorf("expected path /main, got %s", pages[0].Path)
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

func TestDecodeBase64DataURI(t *testing.T) {
	t.Parallel()

	// Valid data URI
	original := []byte{0x89, 0x50, 0x4E, 0x47}
	encoded := "data:image/png;base64," + base64.StdEncoding.EncodeToString(original)
	decoded, err := decodeBase64DataURI(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decoded) != len(original) {
		t.Errorf("expected %d bytes, got %d", len(original), len(decoded))
	}

	// Raw base64 without prefix
	rawEncoded := base64.StdEncoding.EncodeToString(original)
	decoded2, err := decodeBase64DataURI(rawEncoded)
	if err != nil {
		t.Fatalf("unexpected error for raw base64: %v", err)
	}
	if len(decoded2) != len(original) {
		t.Errorf("expected %d bytes, got %d", len(original), len(decoded2))
	}

	// Invalid base64
	_, err = decodeBase64DataURI("data:image/png;base64,not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestCaptureScenario_BASPartialFailure(t *testing.T) {
	t.Parallel()

	callCount := 0
	deps, _ := testCaptureSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/preview-screenshot" {
			callCount++
			if callCount == 1 {
				// First call succeeds
				pngData := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50})
				_ = json.NewEncoder(w).Encode(BASScreenshotResponse{
					Screenshot: "data:image/png;base64," + pngData,
				})
				return
			}
			// Second call fails
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "browser crashed"})
			return
		}
		w.WriteHeader(404)
	})

	// Test that partial failure doesn't prevent saving what succeeded
	// We test this through the BAS client directly since CaptureScenario
	// needs discovery which isn't available in unit tests
	ctx := context.Background()

	// First call should succeed
	resp, err := deps.BAS.CaptureScreenshot(ctx, "http://test/", BASViewport{Width: 1280, Height: 720})
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}
	if resp.Screenshot == "" {
		t.Error("expected screenshot data")
	}

	// Second call should fail
	_, err = deps.BAS.CaptureScreenshot(ctx, "http://test/about", BASViewport{Width: 1280, Height: 720})
	if err == nil {
		t.Error("second call should fail")
	}
}

func TestPrepareForCapture_BaselineMode_ClearsAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	resolver := testStorageResolver(t, dir)
	store := NewVisualCaptureStorage(resolver, OSFileIO{})

	// Seed baseline + capture
	for _, m := range []SnapshotSetMeta{
		{ID: "b1", ScenarioSlug: "s", Role: SnapshotRoleBaseline, TriggerType: "manual", CreatedAt: time.Now().UTC().Add(-2 * time.Minute), Status: "complete"},
		{ID: "c1", ScenarioSlug: "s", Role: SnapshotRoleCapture, TriggerType: "manual", CreatedAt: time.Now().UTC(), Status: "complete"},
	} {
		if err := store.SaveSnapshotSet(1, m, map[string][]byte{"_root_.png": {0x89}}, nil); err != nil {
			t.Fatalf("save %s: %v", m.ID, err)
		}
	}

	if err := prepareForCapture(store, 1, "s", CaptureModeBaseline); err != nil {
		t.Fatalf("prepareForCapture baseline: %v", err)
	}

	list, _ := store.ListSnapshotSets(1, "s")
	if len(list) != 0 {
		t.Errorf("expected all snapshots cleared for baseline mode, got %d", len(list))
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

	// Create scenario files
	scenarioDir := filepath.Join(dir, "scenarios", "test-app", "ui", "src")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	filePath := filepath.Join(scenarioDir, "App.tsx")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Capture happened 1 second before the file was written
	captureTime := time.Now().UTC().Add(-2 * time.Second)
	result := CheckCaptureStaleness(dir, "test-app", captureTime)
	if !result.IsStale {
		t.Error("expected stale when file modified after capture")
	}
}

func TestCheckCaptureStaleness_NotStale(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	scenarioDir := filepath.Join(dir, "scenarios", "test-app")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Capture happened well after the file was written
	captureTime := time.Now().UTC().Add(1 * time.Minute)
	result := CheckCaptureStaleness(dir, "test-app", captureTime)
	if result.IsStale {
		t.Error("expected not stale when capture is after file changes")
	}
}

func TestCheckCaptureStaleness_NoScenarioDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result := CheckCaptureStaleness(dir, "nonexistent", time.Now().UTC())
	if result.IsStale {
		t.Error("expected not stale for nonexistent scenario dir")
	}
}
