package build

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"deployment-manager/bundles"

	"github.com/gorilla/mux"
)

func TestBuildHandlerRejectsInvalidRequests(t *testing.T) {
	h := NewHandler(nil, func(string, map[string]interface{}) {})
	for _, body := range []string{"{", `{ "dry_run": true }`} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/build", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		h.Build(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("body %q status=%d", body, rr.Code)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/build/id", nil)
	rr := httptest.NewRecorder()
	h.BuildStatus(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("BuildStatus status=%d", rr.Code)
	}
}

func TestBuildHandlerFiltersBuildableServicesAndResolvesPaths(t *testing.T) {
	h := NewHandler(nil, func(string, map[string]interface{}) {})
	services := []bundles.ServiceEntry{
		{ID: "api", Build: &bundles.BuildConfig{Type: "go"}},
		{ID: "ui"},
		{ID: "cli", Build: &bundles.BuildConfig{Type: "go"}},
	}
	if got := h.filterBuildableServices(services, nil); len(got) != 2 {
		t.Fatalf("all buildable = %#v", got)
	}
	if got := h.filterBuildableServices(services, []string{"cli"}); len(got) != 1 || got[0].ID != "cli" {
		t.Fatalf("filtered buildable = %#v", got)
	}
	if got := resolveScenarioDir("/repo", "demo"); got != "/repo/scenarios/demo" {
		t.Fatalf("scenario dir = %q", got)
	}
	if got := resolveRepoRoot(); got == "" {
		t.Fatal("repo root is empty")
	}
	if got, ok := canonicalRepoRootFromOverride("."); ok || got != "" {
		t.Fatalf("dot override = %q/%v", got, ok)
	}
}

func TestAutoBuildHandlerDryRunAndStatus(t *testing.T) {
	h := NewHandler(nil, func(string, map[string]interface{}) {})
	h.vrooli = filepath.Clean(filepath.Join("..", "..", "..", ".."))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/build/auto", bytes.NewBufferString(`{"scenario":"deployment-manager","platforms":["linux"],"dry_run":true}`))
	req.Host = "localhost:1234"
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	h.AutoBuild(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("AutoBuild status=%d body=%s", rr.Code, rr.Body.String())
	}
	var status AutoBuildStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Status != "dry_run" || status.BuildID == "" || len(status.Targets) == 0 {
		t.Fatalf("dry run status=%+v", status)
	}
	if status.StatusURL == "" || status.CheckCommand == "" {
		t.Fatalf("status URLs missing: %+v", status)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/build/auto/"+status.BuildID, nil)
	statusReq = mux.SetURLVars(statusReq, map[string]string{"build_id": status.BuildID})
	statusRR := httptest.NewRecorder()
	h.AutoBuildStatus(statusRR, statusReq)
	if statusRR.Code != http.StatusOK {
		t.Fatalf("AutoBuildStatus status=%d", statusRR.Code)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/api/v1/build/auto/missing", nil)
	missingReq = mux.SetURLVars(missingReq, map[string]string{"build_id": "missing"})
	missingRR := httptest.NewRecorder()
	h.AutoBuildStatus(missingRR, missingReq)
	if missingRR.Code != http.StatusNotFound {
		t.Fatalf("missing status=%d", missingRR.Code)
	}
}

func TestAutoBuildHandlerValidation(t *testing.T) {
	h := NewHandler(nil, func(string, map[string]interface{}) {})
	tests := []struct {
		body string
		want int
	}{
		{body: "{", want: http.StatusBadRequest},
		{body: `{ "scenario": "" }`, want: http.StatusBadRequest},
		{body: `{ "scenario": "does-not-exist" }`, want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/build/auto", bytes.NewBufferString(tt.body))
		rr := httptest.NewRecorder()
		h.AutoBuild(rr, req)
		if rr.Code != tt.want {
			t.Fatalf("body=%s status=%d want=%d", tt.body, rr.Code, tt.want)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/build/auto", nil)
	req = mux.SetURLVars(req, map[string]string{"build_id": ""})
	rr := httptest.NewRecorder()
	h.AutoBuildStatus(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("empty id status=%d", rr.Code)
	}
}

func TestAutoBuildSkipsScenarioWithoutGoTargetsAndCoversHelpers(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(nil, func(string, map[string]interface{}) {})
	h.vrooli = root
	req := httptest.NewRequest(http.MethodPost, "/api/v1/build/auto", bytes.NewBufferString(`{"scenario":"empty"}`))
	req.Host = ""
	rr := httptest.NewRecorder()
	h.AutoBuild(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("skip status=%d body=%s", rr.Code, rr.Body.String())
	}
	var status AutoBuildStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Status != "skipped" {
		t.Fatalf("status=%+v", status)
	}
	if got := buildAutoStatusURL(nil, "b1"); got != "" {
		t.Fatalf("nil status URL=%q", got)
	}
	noHost := httptest.NewRequest(http.MethodGet, "/", nil)
	noHost.Host = ""
	if got := buildAutoStatusURL(noHost, "b1"); got != "/api/v1/build/auto/b1" {
		t.Fatalf("no-host status URL=%q", got)
	}
	if got := buildAutoStatusURL(req, "b1"); got != "/api/v1/build/auto/b1" {
		t.Fatalf("empty-host status URL=%q", got)
	}
	plan := buildAutoTargetPlan(root, []buildTarget{{ID: "api", Folder: "api", Config: BuildConfig{Type: "go"}}}, []string{"linux-x64"})
	if len(plan) != 1 || len(plan[0].Platforms) != 1 {
		t.Fatalf("target plan=%+v", plan)
	}
	h.autoBuilds.Save(&status)
	h.appendAutoBuildLog(status.BuildID, "log")
	h.appendAutoBuildError(status.BuildID, "error")
	h.updateAutoBuildPlatform(status.BuildID, "missing", "linux-x64", func(*AutoBuildPlatformStatus) {})
}

func TestBuildHandlerDryRunUsesAnalyzerManifest(t *testing.T) {
	bundle := bundles.Manifest{
		SchemaVersion: "v0.1", Target: "desktop",
		App:       bundles.ManifestApp{Name: "demo", Version: "1.0.0"},
		IPC:       bundles.ManifestIPC{Mode: "loopback-http", Host: "127.0.0.1", Port: 47000, AuthTokenPath: "runtime/token"},
		Telemetry: bundles.ManifestTelemetry{File: "telemetry/events.jsonl"},
		Services: []bundles.ServiceEntry{{
			ID: "api", Type: "api-binary",
			Binaries:  map[string]bundles.ServiceBinary{"linux-x64": {Path: "bin/linux-x64/api"}},
			Build:     &bundles.BuildConfig{Type: "go", SourceDir: "api"},
			Health:    bundles.HealthCheck{Type: "http", Path: "/health", PortName: "http"},
			Readiness: bundles.ReadinessCheck{Type: "health_success", PortName: "http"},
		}},
	}
	manifest, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(append([]byte(`{"manifest":`), append(manifest, '}')...))
	}))
	defer analyzer.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", analyzer.URL)
	h := NewHandler(nil, func(string, map[string]interface{}) {})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/build", bytes.NewBufferString(`{"scenario":"demo","platforms":["linux-x64"],"service_ids":["api"],"dry_run":true}`))
	rr := httptest.NewRecorder()
	h.Build(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("dry run status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response BuildResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "dry_run" || len(response.Results) != 1 || response.Results[0].ServiceID != "api" {
		t.Fatalf("dry run response=%+v", response)
	}
}
