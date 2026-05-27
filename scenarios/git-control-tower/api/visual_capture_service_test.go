package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	"git-control-tower/internal/testutil/fixtures"
	"git-control-tower/internal/testutil/httpx"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/storage"

	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	bas_base "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bas_telemetry "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bas_execution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

// basStubOpts configures the in-memory BAS stub used by visual-capture tests.
// Each ExecuteAdhocWorkflow call produces a sequentially-numbered exec_id
// ("exec-1", "exec-2", ...); the same id is then expected by status and
// screenshot lookups.
type basStubOpts struct {
	failExecIDs            map[string]bool // executions that should report FAILED status
	emptyScreenshotExecIDs map[string]bool // executions whose screenshot list should be empty
	screenshotURL          string          // absolute URL to return for each screenshot artifact; required when binary fetches happen
}

type basStub struct {
	execCount int32
	opts      basStubOpts
}

func (s *basStub) ExecuteAdhocWorkflow(_ context.Context, _ *connect.Request[bas_execution.ExecuteAdhocRequest]) (*connect.Response[bas_execution.ExecuteAdhocResponse], error) {
	n := atomic.AddInt32(&s.execCount, 1)
	return connect.NewResponse(&bas_execution.ExecuteAdhocResponse{
		ExecutionId: fmt.Sprintf("exec-%d", n),
		Status:      bas_base.ExecutionStatus_EXECUTION_STATUS_RUNNING,
	}), nil
}

func (s *basStub) GetExecution(_ context.Context, req *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
	id := req.Msg.GetExecutionId()
	exec := &bas_execution.Execution{
		ExecutionId: id,
		Status:      bas_base.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
	}
	if s.opts.failExecIDs[id] {
		exec.Status = bas_base.ExecutionStatus_EXECUTION_STATUS_FAILED
		msg := "step timed out"
		exec.Error = &msg
	}
	return connect.NewResponse(&basapi.GetExecutionResponse{Execution: exec}), nil
}

func (s *basStub) GetExecutionScreenshots(_ context.Context, req *connect.Request[basapi.GetExecutionScreenshotsRequest]) (*connect.Response[bas_execution.GetScreenshotsResponse], error) {
	id := req.Msg.GetExecutionId()
	if s.opts.emptyScreenshotExecIDs[id] {
		return connect.NewResponse(&bas_execution.GetScreenshotsResponse{ExecutionId: id, Total: 0}), nil
	}
	return connect.NewResponse(&bas_execution.GetScreenshotsResponse{
		ExecutionId: id,
		Total:       1,
		Screenshots: []*bas_execution.ExecutionScreenshot{
			{
				StepIndex: 0,
				Screenshot: &bas_telemetry.TimelineScreenshot{
					ArtifactId:  "ss",
					Url:         s.opts.screenshotURL,
					ContentType: "image/png",
					Width:       1280,
					Height:      720,
				},
			},
		},
	}), nil
}

func (s *basStub) GetExecutionRecordedVideos(_ context.Context, _ *connect.Request[basapi.GetExecutionArtifactsRequest]) (*connect.Response[basapi.GetExecutionVideosResponse], error) {
	return connect.NewResponse(&basapi.GetExecutionVideosResponse{}), nil
}

func testCaptureSetup(t *testing.T, opts basStubOpts) (VisualCaptureDeps, string) {
	t.Helper()

	// Binary asset server: serves the screenshot PNG (and would serve videos
	// in future) at a stable URL the Connect stub points each screenshot at.
	assetMux := http.NewServeMux()
	assetMux.HandleFunc("/api/v1/screenshots/artifacts/ss.png", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	})
	server := httpx.NewHandlerServer(t, assetMux)

	if opts.screenshotURL == "" {
		// Path-only — BrowserAutomationClient.GetScreenshotData prepends the
		// resolved base URL when fetching the bytes.
		opts.screenshotURL = "/api/v1/screenshots/artifacts/ss.png"
	}

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	fixtures.WriteRepoContract(t, repoDir)
	fixtures.WriteScenarioServiceJSON(t, repoDir, "test-app", `{"service":{"name":"test-app"}}`)

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		EnvGet:      func(key string) string { return "" },
		UserHomeDir: func() (string, error) { return tmpDir, nil },
	})
	if err != nil {
		t.Fatalf("create storage resolver: %v", err)
	}

	stub := &basStub{opts: opts}
	basClient := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
		workflowsFactory:  func(string) basWorkflowsAPI { return stub },
		executionsFactory: func(string) basExecutionsAPI { return stub },
	}

	return VisualCaptureDeps{
		BAS:     basClient,
		Storage: NewVisualCaptureStorage(resolver, OSFileIO{}),
		FS:      OSFileIO{},
		RepoDir: repoDir,
		RepoID:  1,
	}, server.URL
}

func TestCaptureScenario_WithLighthousePages(t *testing.T) {
	t.Parallel()

	deps, _ := testCaptureSetup(t, basStubOpts{})

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

	deps, _ := testCaptureSetup(t, basStubOpts{emptyScreenshotExecIDs: map[string]bool{"exec-2": true}})

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

	deps, _ := testCaptureSetup(t, basStubOpts{})

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

// workflowMap parses buildScreenshotWorkflow output into a generic map for assertion helpers.
func workflowMap(t *testing.T, scenario string, page LighthousePage, preset CapturePreset) map[string]interface{} {
	t.Helper()
	wfJSON, err := buildScreenshotWorkflow(scenario, page, preset)
	if err != nil {
		t.Fatalf("buildScreenshotWorkflow returned error: %v", err)
	}
	var wf map[string]interface{}
	if err := json.Unmarshal(wfJSON, &wf); err != nil {
		t.Fatalf("failed to unmarshal workflow JSON: %v", err)
	}
	return wf
}

// workflowNodes extracts the nodes slice from a workflow map.
func workflowNodes(t *testing.T, wf map[string]interface{}) []map[string]interface{} {
	t.Helper()
	raw := wf["nodes"].([]interface{})
	out := make([]map[string]interface{}, len(raw))
	for i, r := range raw {
		out[i] = r.(map[string]interface{})
	}
	return out
}

// nodeAction returns the "action" sub-map for a workflow node.
func nodeAction(t *testing.T, node map[string]interface{}) map[string]interface{} {
	t.Helper()
	return node["action"].(map[string]interface{})
}

// workflowEdges extracts the edges as typed maps from a workflow map.
func workflowEdges(t *testing.T, wf map[string]interface{}) []map[string]interface{} {
	t.Helper()
	raw := wf["edges"].([]interface{})
	out := make([]map[string]interface{}, len(raw))
	for i, r := range raw {
		out[i] = r.(map[string]interface{})
	}
	return out
}

func TestBuildScreenshotWorkflow(t *testing.T) {
	t.Parallel()

	wf := workflowMap(t, "my-app", LighthousePage{Path: "/dashboard", Label: "Dashboard"}, CapturePreset{Name: "Desktop Dark", Width: 1920, Height: 1080, Theme: "dark"})
	nodes := workflowNodes(t, wf)
	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(nodes))
	}

	t.Run("settings", func(t *testing.T) {
		assertWorkflowSettings(t, wf, 1920, "dark")
	})

	t.Run("navigate node", func(t *testing.T) {
		assertNavigateNode(t, nodes[0], "my-app", "/dashboard")
	})

	t.Run("settle node", func(t *testing.T) {
		assertSettleNode(t, nodes[1])
	})

	t.Run("theme node", func(t *testing.T) {
		action := nodeAction(t, nodes[2])
		expr := action["evaluate"].(map[string]interface{})["expression"].(string)
		if !strings.Contains(expr, "colorScheme") {
			t.Errorf("theme expression should set colorScheme, got: %s", expr)
		}
	})

	t.Run("screenshot node", func(t *testing.T) {
		action := nodeAction(t, nodes[3])
		if action["type"] != "ACTION_TYPE_SCREENSHOT" {
			t.Errorf("expected ACTION_TYPE_SCREENSHOT, got %v", action["type"])
		}
	})

	t.Run("edges", func(t *testing.T) {
		assertWorkflowEdges(t, wf)
	})
}

func assertWorkflowSettings(t *testing.T, wf map[string]interface{}, wantWidth int, wantColorScheme string) {
	t.Helper()
	settings := wf["settings"].(map[string]interface{})
	if int(settings["viewport_width"].(float64)) != wantWidth {
		t.Errorf("expected viewport_width %d, got %v", wantWidth, settings["viewport_width"])
	}
	bp := settings["browser_profile"].(map[string]interface{})
	fp := bp["fingerprint"].(map[string]interface{})
	if fp["color_scheme"] != wantColorScheme {
		t.Errorf("expected color_scheme %s, got %v", wantColorScheme, fp["color_scheme"])
	}
}

func assertNavigateNode(t *testing.T, node map[string]interface{}, wantScenario, wantPath string) {
	t.Helper()
	nav := nodeAction(t, node)["navigate"].(map[string]interface{})
	if nav["scenario"] != wantScenario {
		t.Errorf("expected scenario %s, got %v", wantScenario, nav["scenario"])
	}
	if nav["scenario_path"] != wantPath {
		t.Errorf("expected scenario_path %s, got %v", wantPath, nav["scenario_path"])
	}
}

func assertSettleNode(t *testing.T, node map[string]interface{}) {
	t.Helper()
	action := nodeAction(t, node)
	if action["type"] != "ACTION_TYPE_EVALUATE" {
		t.Errorf("expected ACTION_TYPE_EVALUATE, got %v", action["type"])
	}
	expr := action["evaluate"].(map[string]interface{})["expression"].(string)
	if !strings.Contains(expr, "animate-spin") {
		t.Errorf("settle expression should check for animate-spin spinners")
	}
}

func assertWorkflowEdges(t *testing.T, wf map[string]interface{}) {
	t.Helper()
	edges := workflowEdges(t, wf)
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(edges))
	}
	wantEdges := []struct{ source, target string }{
		{"navigate", "wait-settled"},
		{"wait-settled", "set-theme"},
		{"set-theme", "screenshot"},
	}
	for i, want := range wantEdges {
		if edges[i]["source"] != want.source || edges[i]["target"] != want.target {
			t.Errorf("edge %d: got %v->%v, want %v->%v", i, edges[i]["source"], edges[i]["target"], want.source, want.target)
		}
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
	fixtures.WriteRepoContract(t, root)
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

	deps, _ := testCaptureSetup(t, basStubOpts{})

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
