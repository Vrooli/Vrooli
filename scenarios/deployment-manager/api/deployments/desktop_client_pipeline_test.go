package deployments

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopPackagerPipelineClientOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/pipeline/run":
			if r.Method == http.MethodPost {
				body, _ := io.ReadAll(r.Body)
				if strings.Contains(string(body), "recording") {
					_, _ = w.Write([]byte(`{"smoke_test_id":"smoke-1","status":"passed","screen_recording":{"recorded":true}}`))
				} else {
					_, _ = w.Write([]byte(`{"pipeline_id":"pipe-1","status":"running"}`))
				}
				return
			}
		case "/api/v1/smoketest/smoke-1":
			_, _ = w.Write([]byte(`{"smoke_test_id":"smoke-1","status":"passed","screen_recording":{"recorded":true}}`))
			return
		case "/api/v1/smoketest/smoke-1/video":
			_, _ = w.Write([]byte("video"))
			return
		case "/api/v1/pipeline/pipe-1":
			_, _ = w.Write([]byte(`{"current_state":"completed","stages":{"deploy":{"details":{"artifacts":[]}}}}`))
			return
		case "/api/v1/signing/demo":
			w.WriteHeader(http.StatusCreated)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client := &DesktopPackagerClient{httpClient: server.Client(), baseURL: server.URL, log: func(string, map[string]interface{}) {}}
	ctx := context.Background()
	smoke, err := client.RunSmokeTest(ctx, &SmokeTestRequest{ScenarioName: "demo", Platform: "linux", Recording: &ScreenRecordingConfig{Enabled: true}})
	if err != nil || smoke.Status != "passed" {
		t.Fatalf("smoke = %#v, %v", smoke, err)
	}
	if status, err := client.GetSmokeTestStatus(ctx, "smoke-1"); err != nil || status.Status != "passed" {
		t.Fatalf("smoke status = %#v, %v", status, err)
	}
	video := filepath.Join(t.TempDir(), "smoke.mp4")
	if err := client.DownloadVideo(ctx, "smoke-1", video); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(video); err != nil || string(data) != "video" {
		t.Fatalf("video = %q, %v", data, err)
	}
	pipeline, err := client.RunPublishPipeline(ctx, &PublishPipelineRequest{ScenarioName: "demo", Publish: true})
	if err != nil || pipeline.PipelineID != "pipe-1" {
		t.Fatalf("pipeline = %#v, %v", pipeline, err)
	}
	if status, err := client.GetPipelineStatus(ctx, "pipe-1"); err != nil || status.CurrentState != "completed" {
		t.Fatalf("pipeline status = %#v, %v", status, err)
	}
	if err := client.SetSigningConfig(ctx, "demo", map[string]interface{}{"platform": "linux"}); err != nil {
		t.Fatal(err)
	}
}

func TestDesktopPackagerPipelineCancellationAndHTTPFailures(t *testing.T) {
	client := &DesktopPackagerClient{httpClient: http.DefaultClient, baseURL: "http://127.0.0.1:1", log: func(string, map[string]interface{}) {}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.WaitForSmokeTest(ctx, "smoke", 1); err == nil {
		t.Fatal("cancelled smoke wait returned nil error")
	}
	if _, err := client.WaitForPipeline(ctx, "pipeline"); err == nil {
		t.Fatal("cancelled pipeline wait returned nil error")
	}
}
