package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestPlaywrightEngine_Run_DecodesScreenshotAndDOM(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "sess-123"})
	})
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	handler.HandleFunc("/session/sess-123/run", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		resp := map[string]any{
			"schema_version":        contracts.StepOutcomeSchemaVersion,
			"payload_version":       contracts.PayloadVersion,
			"step_index":            1,
			"step_type":             "navigate",
			"node_id":               "node-1",
			"success":               true,
			"final_url":             "https://example.com",
			"screenshot_base64":     "ZmFrZS1kYXRh",
			"screenshot_media_type": "image/png",
			"screenshot_width":      800,
			"screenshot_height":     600,
			"dom_html":              "<html></html>",
			"dom_preview":           "<html>",
			"video_path":            "/tmp/video.webm",
			"trace_path":            "/tmp/trace.zip",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	handler.HandleFunc("/session/sess-123/reset", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
	})
	handler.HandleFunc("/session/sess-123/close", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	log := logrus.New()
	engine, err := newPlaywrightEngine(server.URL, log)
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}

	session, err := engine.StartSession(context.Background(), SessionSpec{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	outcome, err := session.Run(context.Background(), contracts.CompiledInstruction{
		Index:  1,
		NodeID: "node-1",
		Type:   "navigate",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if outcome.Screenshot == nil || string(outcome.Screenshot.Data) != "fake-data" {
		t.Fatalf("expected screenshot data decoded, got %+v", outcome.Screenshot)
	}
	if outcome.DOMSnapshot == nil || outcome.DOMSnapshot.HTML != "<html></html>" {
		t.Fatalf("expected DOM snapshot, got %+v", outcome.DOMSnapshot)
	}
	if err := session.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestResolvePlaywrightDriverURL_Default(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	got := resolvePlaywrightDriverURL(log, true)
	if got == "" {
		t.Fatalf("expected default URL")
	}
	if got != "http://127.0.0.1:39400" {
		t.Fatalf("unexpected default URL: %s", got)
	}
}

func TestPlaywrightEngine_Capabilities(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	engine, err := newPlaywrightEngine(server.URL, nil)
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	caps, err := engine.Capabilities(ctx)
	if err != nil {
		t.Fatalf("caps: %v", err)
	}
	if caps.Engine != "playwright" {
		t.Fatalf("unexpected engine name: %s", caps.Engine)
	}
}
