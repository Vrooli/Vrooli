package deployments

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCommandsCoverDeploymentAndBuildPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/logs/"):
			_, _ = io.WriteString(w, `[{"timestamp":"now","level":"info","message":"started"}]`)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/cost-estimate"):
			_, _ = io.WriteString(w, `{"total":12}`)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/validate"):
			_, _ = io.WriteString(w, `{"valid":true}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/build":
			_, _ = io.WriteString(w, `{"status":"partial","scenario":"demo","duration":"1s","message":"built","results":[{"service_id":"api","all_succeeded":false,"results":[{"platform":"linux-x64","output_path":"api","success":true},{"platform":"win-x64","output_path":"api.exe","success":false,"error":"compiler"}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/deploy/demo":
			_, _ = io.WriteString(w, `{"status":"started"}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/deployments/"):
			_, _ = io.WriteString(w, `{"id":"d1","status":"running"}`)
		default:
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer server.Close()
	cmd := New(testAPIClient(server.URL))

	if err := cmd.Deploy([]string{"demo", "--dry-run", "--async", "--format", "json"}); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if err := cmd.Deployment([]string{"status", "d1", "--format", "json"}); err != nil {
		t.Fatalf("deployment status: %v", err)
	}
	if err := cmd.Validate([]string{"demo", "--verbose", "--format", "json"}); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := cmd.EstimateCost([]string{"demo", "--verbose", "--format", "json"}); err != nil {
		t.Fatalf("estimate: %v", err)
	}
	if err := cmd.Logs([]string{"demo", "--format", "table"}); err != nil {
		t.Fatalf("logs table: %v", err)
	}
	if err := cmd.Build([]string{"--profile", "demo", "--platforms", "linux-x64, win-x64", "--services", "api, worker", "--format", "json"}); err != nil {
		t.Fatalf("build json: %v", err)
	}
	if err := cmd.Build([]string{"demo", "--dry-run"}); err != nil {
		t.Fatalf("build table: %v", err)
	}
}

func TestDeployDesktopCoversPayloadSigningAndPresentation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/deploy-desktop" {
			_, _ = io.WriteString(w, `{"status":"success","profile_id":"demo","scenario":"example","duration":"2s","manifest_path":"manifest.json","steps":[{"name":"build","status":"success","message":"ok"},{"name":"package","status":"failed","error":"warning"}],"build_results":{"results":[{"platform":"linux","success":true,"output_path":"app"},{"platform":"win","success":false,"output_path":"app.exe","error":"failed"}]},"desktop_path":"desktop.AppImage","installers":{"linux":"installer.AppImage"},"next_steps":["review evidence"]}`)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/profiles/") {
			_, _ = io.WriteString(w, `{"scenario":"example"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	cmd := New(testAPIClient(server.URL))
	config := filepath.Join(t.TempDir(), "signing.json")
	if err := os.WriteFile(config, []byte(`{"platform":"linux","key":"test"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.DeployDesktop([]string{"demo", "--platforms", "linux, win", "--output", "/tmp/bundle", "--skip-build", "--skip-validation", "--skip-packaging", "--skip-installers", "--mode", "external-server", "--dry-run", "--signing-config", config, "--format", "json", "--timeout", "2m"}); err != nil {
		t.Fatalf("deploy desktop: %v", err)
	}
	if got := buildDeployPayload("demo", "", "linux, win", true, false, true, false, "bundled", false, durationPtr(30)); got["platforms"].([]string)[1] != "win" {
		t.Fatalf("platforms were not normalized: %#v", got["platforms"])
	}
	if err := cmd.DeployDesktop(nil); err == nil {
		t.Fatal("expected missing profile error")
	}
}

func TestDeployDesktopRejectsSigningConfigAndPrintsFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `{"scenario":"does-not-exist"}`)
			return
		}
		http.Error(w, "failed", http.StatusInternalServerError)
	}))
	defer server.Close()
	cmd := New(testAPIClient(server.URL))
	if err := cmd.DeployDesktop([]string{"demo", "--signing-config", filepath.Join(t.TempDir(), "missing.json")}); err == nil {
		t.Fatal("expected missing signing config error")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.DeployDesktop([]string{"demo", "--signing-config", bad}); err == nil {
		t.Fatal("expected malformed signing config error")
	}
}

func durationPtr(value time.Duration) *time.Duration { return &value }
