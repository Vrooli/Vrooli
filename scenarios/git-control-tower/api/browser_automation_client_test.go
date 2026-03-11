package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
)

func TestBASClient_CaptureScreenshot(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/preview-screenshot", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req BASScreenshotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.URL != "http://localhost:3000/" {
			t.Errorf("unexpected URL: %s", req.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(BASScreenshotResponse{
			Screenshot:     "data:image/png;base64,iVBORw0KGgo=",
			URL:            req.URL,
			DurationMS:     150,
			ViewportWidth:  1280,
			ViewportHeight: 720,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &BrowserAutomationClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	resp, err := client.CaptureScreenshot(context.Background(), "http://localhost:3000/", BASViewport{Width: 1280, Height: 720})
	if err != nil {
		t.Fatalf("CaptureScreenshot returned error: %v", err)
	}
	if resp.Screenshot == "" {
		t.Error("expected non-empty screenshot data")
	}
	if resp.ViewportWidth != 1280 {
		t.Errorf("expected viewport width 1280, got %d", resp.ViewportWidth)
	}
}

func TestBASClient_CaptureScreenshot_ServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/preview-screenshot", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "browser crashed"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &BrowserAutomationClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	_, err := client.CaptureScreenshot(context.Background(), "http://localhost:3000/", BASViewport{Width: 1280, Height: 720})
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
		json.NewEncoder(w).Encode(BASExecuteResponse{
			ExecutionID: "exec-123",
			Status:      "completed",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &BrowserAutomationClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	resp, err := client.ExecuteAdhocWorkflow(context.Background(), BASExecuteAdhocRequest{
		FlowDefinition:    json.RawMessage(`{"steps":[]}`),
		WaitForCompletion: true,
	})
	if err != nil {
		t.Fatalf("ExecuteAdhocWorkflow returned error: %v", err)
	}
	if resp.ExecutionID != "exec-123" {
		t.Errorf("expected execution ID exec-123, got %s", resp.ExecutionID)
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
		json.NewEncoder(w).Encode(BASScreenshotsResponse{
			Screenshots: []BASExecutionScreenshot{
				{StepIndex: 0, StepLabel: "Navigate"},
				{StepIndex: 1, StepLabel: "Click button"},
			},
			Total: 2,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &BrowserAutomationClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
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
