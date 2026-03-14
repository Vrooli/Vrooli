package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

	deps, _ := testCaptureSetup(t, adhocWorkflowHandler(t, nil))

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

func TestCaptureScenario_BASPartialFailure(t *testing.T) {
	t.Parallel()

	var execCount int32
	deps, _ := testCaptureSetup(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workflows/execute-adhoc" && r.Method == http.MethodPost:
			n := atomic.AddInt32(&execCount, 1)
			_ = json.NewEncoder(w).Encode(BASExecuteResponse{
				ExecutionID: fmt.Sprintf("exec-%d", n),
				Status:      "running",
			})

		case strings.HasPrefix(r.URL.Path, "/api/v1/executions/") && strings.HasSuffix(r.URL.Path, "/screenshots"):
			execID := strings.TrimPrefix(r.URL.Path, "/api/v1/executions/")
			execID = strings.TrimSuffix(execID, "/screenshots")
			if execID == "exec-2" {
				// Second page returns no screenshots (simulate failure)
				_ = json.NewEncoder(w).Encode(BASScreenshotsResponse{Total: 0})
				return
			}
			_ = json.NewEncoder(w).Encode(BASScreenshotsResponse{
				Screenshots: []BASExecutionScreenshot{
					{Screenshot: struct {
						ArtifactID   string `json:"artifact_id"`
						Url          string `json:"url"`
						ThumbnailUrl string `json:"thumbnail_url"`
						ContentType  string `json:"content_type"`
						Width        int    `json:"width"`
						Height       int    `json:"height"`
					}{Url: "/api/v1/screenshots/artifacts/ss-1.png", ContentType: "image/png", Width: 1280, Height: 720}},
				},
				Total: 1,
			})

		case strings.HasPrefix(r.URL.Path, "/api/v1/executions/"):
			_ = json.NewEncoder(w).Encode(BASExecutionDetail{
				ExecutionID: strings.TrimPrefix(r.URL.Path, "/api/v1/executions/"),
				Status:      "EXECUTION_STATUS_COMPLETED",
			})

		case r.URL.Path == "/api/v1/screenshots/artifacts/ss-1.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47})

		default:
			w.WriteHeader(404)
		}
	})

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

	deps, _ := testCaptureSetup(t, adhocWorkflowHandler(t, nil))

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

func TestBuildScreenshotWorkflow(t *testing.T) {
	t.Parallel()

	wfJSON, err := buildScreenshotWorkflow("my-app", LighthousePage{Path: "/dashboard", Label: "Dashboard"}, CapturePreset{Name: "Desktop Dark", Width: 1920, Height: 1080, Theme: "dark"})
	if err != nil {
		t.Fatalf("buildScreenshotWorkflow returned error: %v", err)
	}

	var wf map[string]interface{}
	if err := json.Unmarshal(wfJSON, &wf); err != nil {
		t.Fatalf("failed to unmarshal workflow JSON: %v", err)
	}

	// Check settings
	settings := wf["settings"].(map[string]interface{})
	if int(settings["viewport_width"].(float64)) != 1920 {
		t.Errorf("expected viewport_width 1920, got %v", settings["viewport_width"])
	}
	bp := settings["browser_profile"].(map[string]interface{})
	fp := bp["fingerprint"].(map[string]interface{})
	if fp["color_scheme"] != "dark" {
		t.Errorf("expected color_scheme dark, got %v", fp["color_scheme"])
	}

	// 4 nodes: navigate → wait-settled → set-theme → screenshot
	nodes := wf["nodes"].([]interface{})
	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(nodes))
	}

	// Navigate node
	navNode := nodes[0].(map[string]interface{})
	nav := navNode["action"].(map[string]interface{})["navigate"].(map[string]interface{})
	if nav["scenario"] != "my-app" {
		t.Errorf("expected scenario my-app, got %v", nav["scenario"])
	}
	if nav["scenario_path"] != "/dashboard" {
		t.Errorf("expected scenario_path /dashboard, got %v", nav["scenario_path"])
	}

	// Settle node (evaluate with spinner poll)
	settleNode := nodes[1].(map[string]interface{})
	settleAction := settleNode["action"].(map[string]interface{})
	if settleAction["type"] != "ACTION_TYPE_EVALUATE" {
		t.Errorf("expected ACTION_TYPE_EVALUATE, got %v", settleAction["type"])
	}
	expr := settleAction["evaluate"].(map[string]interface{})["expression"].(string)
	if !strings.Contains(expr, "animate-spin") {
		t.Errorf("settle expression should check for animate-spin spinners")
	}

	// Theme node
	themeNode := nodes[2].(map[string]interface{})
	themeAction := themeNode["action"].(map[string]interface{})
	themeExpr := themeAction["evaluate"].(map[string]interface{})["expression"].(string)
	if !strings.Contains(themeExpr, "colorScheme") {
		t.Errorf("theme expression should set colorScheme, got: %s", themeExpr)
	}

	// Screenshot node
	ssNode := nodes[3].(map[string]interface{})
	ssAction := ssNode["action"].(map[string]interface{})
	if ssAction["type"] != "ACTION_TYPE_SCREENSHOT" {
		t.Errorf("expected ACTION_TYPE_SCREENSHOT, got %v", ssAction["type"])
	}

	// Check edges: navigate→wait-settled→set-theme→screenshot
	edges := wf["edges"].([]interface{})
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(edges))
	}
	e1 := edges[0].(map[string]interface{})
	if e1["source"] != "navigate" || e1["target"] != "wait-settled" {
		t.Errorf("expected edge navigate->wait-settled, got %v->%v", e1["source"], e1["target"])
	}
	e2 := edges[1].(map[string]interface{})
	if e2["source"] != "wait-settled" || e2["target"] != "set-theme" {
		t.Errorf("expected edge wait-settled->set-theme, got %v->%v", e2["source"], e2["target"])
	}
	e3 := edges[2].(map[string]interface{})
	if e3["source"] != "set-theme" || e3["target"] != "screenshot" {
		t.Errorf("expected edge set-theme->screenshot, got %v->%v", e3["source"], e3["target"])
	}
}

func TestBuildScreenshotWorkflow_WithWaitForSelector(t *testing.T) {
	t.Parallel()

	wfJSON, err := buildScreenshotWorkflow("my-app", LighthousePage{Path: "/", Label: "Home", WaitForSelector: "[data-testid=\"main\"]"}, CapturePreset{Name: "Desktop Light", Width: 1280, Height: 720, Theme: "light"})
	if err != nil {
		t.Fatalf("buildScreenshotWorkflow returned error: %v", err)
	}

	var wf map[string]interface{}
	if err := json.Unmarshal(wfJSON, &wf); err != nil {
		t.Fatalf("failed to unmarshal workflow JSON: %v", err)
	}

	// Settle expression should include the selector check
	nodes := wf["nodes"].([]interface{})
	settleAction := nodes[1].(map[string]interface{})["action"].(map[string]interface{})
	expr := settleAction["evaluate"].(map[string]interface{})["expression"].(string)
	if !strings.Contains(expr, `data-testid`) {
		t.Errorf("settle expression should include waitForSelector, got: %s", expr)
	}
}

func TestSettleExpression(t *testing.T) {
	t.Parallel()

	// Without selector: just spinner check
	expr := settleExpression("")
	if !strings.Contains(expr, "animate-spin") {
		t.Error("expected animate-spin check in expression")
	}
	if strings.Contains(expr, "querySelector") && strings.Contains(expr, "offsetParent") {
		t.Error("expected no selector check when waitForSelector is empty")
	}

	// With selector: spinner check + selector visibility
	expr = settleExpression("[data-testid=\"app\"]")
	if !strings.Contains(expr, "animate-spin") {
		t.Error("expected animate-spin check in expression")
	}
	if !strings.Contains(expr, `data-testid`) {
		t.Error("expected selector in expression")
	}
	if !strings.Contains(expr, "offsetParent") {
		t.Error("expected visibility check in expression")
	}
}

// adhocWorkflowHandler returns an http.HandlerFunc that mocks all BAS endpoints
// needed for CaptureScenario's adhoc workflow path. The optional failExecIDs set
// causes those execution IDs to return FAILED status.
func adhocWorkflowHandler(t *testing.T, failExecIDs map[string]bool) http.HandlerFunc {
	t.Helper()
	var execCount int32

	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workflows/execute-adhoc" && r.Method == http.MethodPost:
			n := atomic.AddInt32(&execCount, 1)
			_ = json.NewEncoder(w).Encode(BASExecuteResponse{
				ExecutionID: fmt.Sprintf("exec-%d", n),
				Status:      "running",
			})

		case strings.HasPrefix(r.URL.Path, "/api/v1/executions/") && strings.HasSuffix(r.URL.Path, "/screenshots"):
			_ = json.NewEncoder(w).Encode(BASScreenshotsResponse{
				Screenshots: []BASExecutionScreenshot{
					{Screenshot: struct {
						ArtifactID   string `json:"artifact_id"`
						Url          string `json:"url"`
						ThumbnailUrl string `json:"thumbnail_url"`
						ContentType  string `json:"content_type"`
						Width        int    `json:"width"`
						Height       int    `json:"height"`
					}{Url: "/api/v1/screenshots/artifacts/ss.png", ContentType: "image/png", Width: 1280, Height: 720}},
				},
				Total: 1,
			})

		case strings.HasPrefix(r.URL.Path, "/api/v1/executions/"):
			execID := strings.TrimPrefix(r.URL.Path, "/api/v1/executions/")
			status := "EXECUTION_STATUS_COMPLETED"
			errMsg := ""
			if failExecIDs != nil && failExecIDs[execID] {
				status = "EXECUTION_STATUS_FAILED"
				errMsg = "step timed out"
			}
			_ = json.NewEncoder(w).Encode(BASExecutionDetail{
				ExecutionID: execID,
				Status:      status,
				Error:       errMsg,
			})

		case r.URL.Path == "/api/v1/screenshots/artifacts/ss.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47})

		default:
			w.WriteHeader(404)
		}
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

	deps, _ := testCaptureSetup(t, adhocWorkflowHandler(t, nil))

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
