package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"git-control-tower/internal/testutil/httpx"
	"github.com/vrooli/api-core/discovery"
)

func TestBASClient_GetScreenshotData(t *testing.T) {
	t.Parallel()

	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/screenshots/artifacts/ss-1.png", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	data, contentType, err := client.GetScreenshotData(context.Background(), "/api/v1/screenshots/artifacts/ss-1.png")
	if err != nil {
		t.Fatalf("GetScreenshotData returned error: %v", err)
	}
	if contentType != "image/png" {
		t.Errorf("expected content-type image/png, got %s", contentType)
	}
	if len(data) != len(pngBytes) {
		t.Errorf("expected %d bytes, got %d", len(pngBytes), len(data))
	}
}

func TestBASClient_GetScreenshotData_ServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/screenshots/artifacts/ss-1.png", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	_, _, err := client.GetScreenshotData(context.Background(), "/api/v1/screenshots/artifacts/ss-1.png")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestBASClient_ExecuteAdhocWorkflow(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/workflows/execute-adhoc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BASExecuteResponse{
			ExecutionID: "exec-123",
			Status:      "completed",
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	resp, err := client.ExecuteAdhocWorkflow(context.Background(), BASExecuteAdhocRequest{
		FlowDefinition: json.RawMessage(`{"steps":[]}`),
	}, false)
	if err != nil {
		t.Fatalf("ExecuteAdhocWorkflow returned error: %v", err)
	}
	if resp.ExecutionID != "exec-123" {
		t.Errorf("expected execution ID exec-123, got %s", resp.ExecutionID)
	}
}

func TestBASClient_ExecuteAdhocWorkflow_RequiresVideo(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/workflows/execute-adhoc", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("requires_video") != "true" {
			t.Errorf("expected requires_video=true query param, got %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BASExecuteResponse{
			ExecutionID: "exec-vid",
			Status:      "running",
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	resp, err := client.ExecuteAdhocWorkflow(context.Background(), BASExecuteAdhocRequest{
		FlowDefinition: json.RawMessage(`{"steps":[]}`),
	}, true)
	if err != nil {
		t.Fatalf("ExecuteAdhocWorkflow returned error: %v", err)
	}
	if resp.ExecutionID != "exec-vid" {
		t.Errorf("expected execution ID exec-vid, got %s", resp.ExecutionID)
	}
}

func TestBASClient_PollExecutionCompletion(t *testing.T) {
	t.Parallel()

	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/executions/exec-poll", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		status := "EXECUTION_STATUS_RUNNING"
		if callCount >= 3 {
			status = "EXECUTION_STATUS_COMPLETED"
		}
		_ = json.NewEncoder(w).Encode(BASExecutionDetail{
			ExecutionID: "exec-poll",
			Status:      status,
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	detail, err := client.PollExecutionCompletion(context.Background(), "exec-poll", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("PollExecutionCompletion returned error: %v", err)
	}
	if detail.Status != "EXECUTION_STATUS_COMPLETED" {
		t.Errorf("expected COMPLETED status, got %s", detail.Status)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 poll calls, got %d", callCount)
	}
}

func TestBASClient_PollExecutionCompletion_Failed(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/executions/exec-fail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BASExecutionDetail{
			ExecutionID: "exec-fail",
			Status:      "EXECUTION_STATUS_FAILED",
			Error:       "step timed out",
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	detail, err := client.PollExecutionCompletion(context.Background(), "exec-fail", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("PollExecutionCompletion returned error: %v", err)
	}
	if detail.Status != "EXECUTION_STATUS_FAILED" {
		t.Errorf("expected FAILED status, got %s", detail.Status)
	}
	if detail.Error != "step timed out" {
		t.Errorf("expected error 'step timed out', got %s", detail.Error)
	}
}

func TestBASClient_GetScreenshots(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/executions/exec-456/screenshots", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BASScreenshotsResponse{
			Screenshots: []BASExecutionScreenshot{
				{StepIndex: 0, StepLabel: "Navigate"},
				{StepIndex: 1, StepLabel: "Click button"},
			},
			Total: 2,
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	resp, err := client.GetScreenshots(context.Background(), "exec-456")
	if err != nil {
		t.Fatalf("GetScreenshots returned error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected 2 screenshots, got %d", resp.Total)
	}
	if len(resp.Screenshots) != 2 {
		t.Errorf("expected 2 screenshot entries, got %d", len(resp.Screenshots))
	}
}

func TestBASClient_GetRecordedVideos(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/executions/exec-789/recorded-videos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BASRecordedVideosResponse{
			ExecutionID: "exec-789",
			Videos: []BASVideoArtifact{
				{ArtifactID: "vid-1", ContentType: "video/webm", SizeBytes: 1024},
			},
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	resp, err := client.GetRecordedVideos(context.Background(), "exec-789")
	if err != nil {
		t.Fatalf("GetRecordedVideos returned error: %v", err)
	}
	if resp.ExecutionID != "exec-789" {
		t.Errorf("expected execution ID exec-789, got %s", resp.ExecutionID)
	}
	if len(resp.Videos) != 1 {
		t.Errorf("expected 1 video, got %d", len(resp.Videos))
	}
}

func TestBASClient_GetVideoData(t *testing.T) {
	t.Parallel()

	videoBytes := []byte("fake-video-data")
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/recordings/assets/exec-789/artifacts/videos/vid-1.webm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "video/webm")
		_, _ = w.Write(videoBytes)
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	data, contentType, err := client.GetVideoData(context.Background(), "/api/v1/recordings/assets/exec-789/artifacts/videos/vid-1.webm")
	if err != nil {
		t.Fatalf("GetVideoData returned error: %v", err)
	}
	if contentType != "video/webm" {
		t.Errorf("expected content-type video/webm, got %s", contentType)
	}
	if string(data) != "fake-video-data" {
		t.Errorf("unexpected video data: %s", string(data))
	}
}
