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
	if !strings.Contains(request.WaitFor.Selector, "component-harness") {
		t.Fatalf("wait selector = %q", request.WaitFor.Selector)
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
