package componenttests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBASCaptureExecutorParsesStoryAndEvidence(t *testing.T) {
	var request basCaptureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
      "executionId":"capture-1",
      "durationMs":42,
      "domHtml":"<div id=\"root\"></div><pre id=\"rcl-story-result\" hidden>{\"passed\":true,\"failures\":[],\"performance\":{\"mountMs\":12.5,\"commitCount\":2,\"nodeCount\":8}}</pre>",
      "accessibilityJson":"{\"contract\":\"bas-accessibility-snapshot/v1\"}",
      "artifacts":[{"type":"CAPTURE_TYPE_SCREENSHOT","reference":"artifact://screenshot-1","sizeBytes":1234}]
    }`))
	}))
	defer server.Close()

	executor := BASCaptureExecutor{RCLBaseURL: "http://rcl.test", BASBaseURL: server.URL, HTTPClient: server.Client()}
	result, err := executor.ExecuteStory(context.Background(), "rcl:Button", "2.2.0", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.Performance.MountMS != 12.5 || len(result.Artifacts) != 1 {
		t.Fatalf("unexpected BAS result: %+v", result)
	}
	if request.URL != "http://rcl.test/preview/rcl:Button/harness.html?motion=reduce&runner=1&story=primary&version=2.2.0" {
		t.Fatalf("story URL = %q", request.URL)
	}
	if !request.InlineDOM || !request.InlineAccessibility || !request.InlineComputedStyle {
		t.Fatalf("capture request did not request structured evidence: %+v", request)
	}
	if request.ScreenshotSelector != "[data-preview-sheet]" {
		t.Fatalf("screenshot selector = %q", request.ScreenshotSelector)
	}
	if !strings.Contains(request.WaitFor.Selector, "component-harness") {
		t.Fatalf("wait selector = %q", request.WaitFor.Selector)
	}
}

func TestBASCaptureExecutorCapturesBoundedStorySheet(t *testing.T) {
	var request basCaptureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"durationMs":7,"domHtml":"<main data-preview-sheet=\"story-gallery\"><pre id=\"rcl-story-result\">{\"passed\":true,\"failures\":[]}</pre></main>","accessibilityJson":"{\"contract\":\"bas-accessibility-snapshot/v1\"}","artifacts":[{"type":"CAPTURE_TYPE_SCREENSHOT","reference":"artifact://sheet","primary":true}]}`))
	}))
	defer server.Close()

	executor := BASCaptureExecutor{RCLBaseURL: "http://rcl.test", BASBaseURL: server.URL, HTTPClient: server.Client()}
	result, err := executor.ExecuteStorySheet(context.Background(), "rcl:Button", "2.2.0", []string{"primary", "error-retry"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || len(result.Artifacts) != 1 {
		t.Fatalf("unexpected sheet result: %+v", result)
	}
	if result.Artifacts[0].Kind != "bas-story-sheet" || result.Artifacts[0].StoryID != "review-sheet:primary,error-retry" {
		t.Fatalf("sheet artifact identity = %+v", result.Artifacts[0])
	}
	if !strings.Contains(request.URL, "stories=primary%2Cerror-retry") || strings.Contains(request.URL, "story=") {
		t.Fatalf("sheet URL = %q", request.URL)
	}
}

func TestBrowserVisibleBASArtifactPathUsesRCLEmbeddedProxy(t *testing.T) {
	if got := browserVisibleBASArtifactPath("/api/v1/artifacts/capture.png"); got != "/embedded/browser-automation-studio/api/v1/artifacts/capture.png" {
		t.Fatalf("artifact path = %q", got)
	}
	if got := browserVisibleBASArtifactPath("http://127.0.0.1:17116/api/v1/artifacts/capture.png?download=0"); got != "/embedded/browser-automation-studio/api/v1/artifacts/capture.png?download=0" {
		t.Fatalf("absolute artifact path = %q", got)
	}
}

func TestBASArtifactsPreferPrimaryScreenshot(t *testing.T) {
	artifacts := basArtifacts("rcl:Dialog", "1.2.0", "details-open", "http://bas.test", []basArtifact{
		{Type: "CAPTURE_TYPE_SCREENSHOT", Reference: "bas-capture://capture/screenshot", Metadata: map[string]string{"view_url": "/api/v1/artifacts/intermediate.png"}},
		{Type: "CAPTURE_TYPE_SCREENSHOT", Reference: "bas-capture://capture/screenshot", Primary: true, Metadata: map[string]string{"view_url": "/api/v1/artifacts/primary.png"}},
	})
	if len(artifacts) != 1 {
		t.Fatalf("artifacts = %+v", artifacts)
	}
	if got := artifacts[0].Reference; got != "/embedded/browser-automation-studio/api/v1/artifacts/primary.png" {
		t.Fatalf("selected screenshot = %q", got)
	}
}

func TestBASArtifactsKeepStorySheetBoundToItsScreenshot(t *testing.T) {
	artifacts := basArtifacts("rcl:Button", "2.2.0", "review-sheet:primary,error-retry", "http://bas.test", []basArtifact{
		{Type: "CAPTURE_TYPE_DOM", Reference: "bas-capture://capture/dom"},
		{Type: "CAPTURE_TYPE_SCREENSHOT", Reference: "bas-capture://capture/screenshot", Primary: true, Metadata: map[string]string{"view_url": "/api/v1/artifacts/sheet.png"}},
		{Type: "CAPTURE_TYPE_PERFORMANCE", Reference: "bas-capture://capture/performance"},
	})

	byKind := make(map[string]Artifact, len(artifacts))
	for _, artifact := range artifacts {
		byKind[artifact.Kind] = artifact
	}
	for _, kind := range []string{"bas-dom", "bas-story-sheet", "bas-performance"} {
		if _, ok := byKind[kind]; !ok {
			t.Fatalf("missing %s artifact in %+v", kind, artifacts)
		}
	}
	if got := byKind["bas-story-sheet"].Reference; got != "/embedded/browser-automation-studio/api/v1/artifacts/sheet.png" {
		t.Fatalf("story sheet reference = %q", got)
	}
}
