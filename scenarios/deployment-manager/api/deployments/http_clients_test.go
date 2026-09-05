package deployments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPCloudHealthClient(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantHealth bool
	}{
		{name: "healthy", status: http.StatusOK, body: `{"status":"ok"}`, wantHealth: true},
		{name: "unhealthy", status: http.StatusServiceUnavailable, body: `downstream unavailable`, wantHealth: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/deployments/landing-page-business-suite/health" {
					t.Errorf("path = %s", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := &HTTPCloudHealthClient{httpClient: srv.Client(), baseURL: srv.URL}
			result, err := client.CheckLPBSHealth(context.Background())
			if err != nil {
				t.Fatalf("CheckLPBSHealth() error = %v", err)
			}
			if result.Healthy != tt.wantHealth {
				t.Fatalf("Healthy = %v, want %v", result.Healthy, tt.wantHealth)
			}
			if !tt.wantHealth && result.Details == "" {
				t.Fatal("unhealthy result should retain the downstream response")
			}
			if tt.wantHealth && result.Details != "" {
				t.Fatalf("healthy Details = %q", result.Details)
			}
			if _, err := client.CheckLPBSHealth(context.Background()); err != nil {
				t.Fatalf("second health check error = %v", err)
			}
		})
	}
}

func TestNewHTTPCloudHealthClientUsesConfiguredURL(t *testing.T) {
	previous := os.Getenv("SCENARIO_TO_CLOUD_URL")
	t.Cleanup(func() { _ = os.Setenv("SCENARIO_TO_CLOUD_URL", previous) })
	if err := os.Setenv("SCENARIO_TO_CLOUD_URL", "http://cloud.example/"); err != nil {
		t.Fatal(err)
	}
	client, err := NewHTTPCloudHealthClient(nil)
	if err != nil {
		t.Fatalf("NewHTTPCloudHealthClient() error = %v", err)
	}
	if client.baseURL != "http://cloud.example" {
		t.Fatalf("baseURL = %q", client.baseURL)
	}
}

func TestDesktopPackagerClientHTTPWorkflow(t *testing.T) {
	var statusCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/desktop/generate/quick":
			var request QuickGenerateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode quick request: %v", err)
			}
			if request.ScenarioName != "demo" || request.DeploymentMode != "release" {
				t.Errorf("quick request = %+v", request)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"build_id":"b1","status":"building","scenario_name":"demo","desktop_path":"/tmp/demo","status_url":"/status/b1"}`))
		case "/api/v1/desktop/build/demo":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"build_id":"b2","status":"building","scenario":"demo","desktop_path":"/tmp/demo","platforms":["linux"],"status_url":"/status/b2"}`))
		case "/api/v1/desktop/status/b1":
			call := statusCalls.Add(1)
			status := "building"
			if call > 1 {
				status = "ready"
			}
			_, _ = w.Write([]byte(`{"build_id":"b1","scenario_name":"demo","status":"` + status + `","platforms":["linux"]}`))
		case "/api/v1/signing/demo/ready":
			_, _ = w.Write([]byte(`{"ready":true,"scenario":"demo","platforms":{"linux":{"ready":true}}}`))
		case "/health":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &DesktopPackagerClient{httpClient: srv.Client(), baseURL: srv.URL, log: func(string, map[string]interface{}) {}}
	generated, err := client.QuickGenerate(context.Background(), &QuickGenerateRequest{ScenarioName: "demo", DeploymentMode: "release", Platforms: []string{"linux"}})
	if err != nil || generated.BuildID != "b1" {
		t.Fatalf("QuickGenerate() = %+v, %v", generated, err)
	}
	built, err := client.BuildScenario(context.Background(), "demo", &ScenarioBuildRequest{Platforms: []string{"linux"}})
	if err != nil || built.BuildID != "b2" {
		t.Fatalf("BuildScenario() = %+v, %v", built, err)
	}
	ready, err := client.WaitForBuild(context.Background(), "b1", time.Millisecond)
	if err != nil || ready.Status != "ready" {
		t.Fatalf("WaitForBuild() = %+v, %v", ready, err)
	}
	if !client.IsAvailable(context.Background()) {
		t.Fatal("IsAvailable() = false")
	}
	signing, err := client.CheckSigningReadiness(context.Background(), "demo")
	if err != nil || !signing.Ready {
		t.Fatalf("CheckSigningReadiness() = %+v, %v", signing, err)
	}
}

func TestDesktopPackagerClientBuildFailureAndCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/desktop/status/fail":
			_, _ = w.Write([]byte(`{"build_id":"fail","status":"failed","error":"compiler failed"}`))
		case "/api/v1/signing/missing/ready":
			w.WriteHeader(http.StatusNotFound)
		case "/api/v1/desktop/status/bad":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("bad gateway"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := &DesktopPackagerClient{httpClient: srv.Client(), baseURL: srv.URL, log: func(string, map[string]interface{}) {}}

	failed, err := client.WaitForBuild(context.Background(), "fail", time.Millisecond)
	if failed == nil || err == nil {
		t.Fatalf("failed WaitForBuild() = %+v, %v", failed, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.WaitForBuild(ctx, "fail", time.Millisecond); err == nil {
		t.Fatal("cancelled WaitForBuild() returned nil error")
	}
	missing, err := client.CheckSigningReadiness(context.Background(), "missing")
	if err != nil || missing.Ready || len(missing.Issues) == 0 {
		t.Fatalf("missing signing readiness = %+v, %v", missing, err)
	}
	if _, err := client.GetBuildStatus(context.Background(), "bad"); err == nil {
		t.Fatal("non-200 GetBuildStatus() returned nil error")
	}
}
