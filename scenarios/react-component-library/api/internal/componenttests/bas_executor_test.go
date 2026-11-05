package componenttests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliutil"
)

func TestBASCaptureExecutorProbeRequiresHealthyBoundary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	executor := BASCaptureExecutor{BASBaseURL: server.URL, HTTPClient: server.Client()}
	require.Error(t, executor.Probe(context.Background()))
}

func TestAppendURLPathPreservesEscapedBasePathPrefix(t *testing.T) {
	got, err := appendURLPath("https://bas.example/api%2Fv1/", basCaptureServicePath)
	require.NoError(t, err)
	require.Equal(t, "https://bas.example/api%2Fv1"+basCaptureServicePath, got)
}

func TestBASInspectionUsesJoinedHealthRequestPath(t *testing.T) {
	for _, test := range []struct {
		name     string
		basePath string
		wantPath string
	}{
		{name: "root", wantPath: "/health"},
		{name: "slash prefix", basePath: "/api/v1/", wantPath: "/api/v1/health"},
		{name: "escaped prefix", basePath: "/api%2Fv1/", wantPath: "/api%2Fv1/health"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.RequestURI
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			executor := BASCaptureExecutor{BASBaseURL: server.URL + test.basePath, HTTPClient: server.Client()}
			_, err := executor.InspectProvider(context.Background())
			require.NoError(t, err)
			require.Equal(t, test.wantPath, gotPath)
		})
	}
}

func TestSanitizedBASBaseURLRedactsCredentialAndRequestComponents(t *testing.T) {
	got, exact := sanitizedBASBaseURL("https://operator:secret@example.test/prefix?token=hidden#private")
	require.Equal(t, "https://example.test/prefix", got)
	require.False(t, exact, "a redacted URL must not be claimed as the exact selected endpoint")
	require.NotContains(t, got, "secret")
	require.NotContains(t, got, "hidden")
	require.NotContains(t, got, "private")
}

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
	if request.ScreenshotSelector != `[data-preview-capture-boundary="component-sheet"]` {
		t.Fatalf("screenshot selector = %q", request.ScreenshotSelector)
	}
	if request.Dimensions != (basDimensions{Width: 1280, Height: 800}) {
		t.Fatalf("individual story dimensions = %+v", request.Dimensions)
	}
	if !strings.Contains(request.WaitFor.Selector, "data-preview-readiness-marker") || !strings.Contains(request.WaitFor.Selector, "data-preview-ready") || !strings.Contains(request.WaitFor.Selector, "data-rcl-story-status") {
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
	parsedURL, err := url.Parse(request.URL)
	require.NoError(t, err)
	if got := parsedURL.Query().Get("stories"); got != "primary,error-retry" {
		t.Fatalf("sheet URL = %q", request.URL)
	}
	if parsedURL.Query().Has("story") {
		t.Fatalf("sheet URL unexpectedly has a single-story selector: %q", request.URL)
	}
	if request.Dimensions != (basDimensions{Width: 1280, Height: 800, DeviceScaleFactor: basStorySheetDeviceScaleFactor}) {
		t.Fatalf("story-sheet dimensions = %+v", request.Dimensions)
	}
	if request.ScreenshotSelector != `[data-preview-capture-boundary="component-sheet"]` {
		t.Fatalf("story-sheet screenshot selector = %q", request.ScreenshotSelector)
	}
	require.Equal(t, []string{
		"CAPTURE_TYPE_SCREENSHOT", "CAPTURE_TYPE_DOM", "CAPTURE_TYPE_CONSOLE_LOGS",
		"CAPTURE_TYPE_NETWORK", "CAPTURE_TYPE_PERFORMANCE", "CAPTURE_TYPE_ACCESSIBILITY",
	}, request.Captures)
	require.True(t, request.InlineDOM && request.InlineAccessibility && request.InlineComputedStyle)
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

func TestPageStoryCaptureKeepsPageOwnedScreenshotBoundary(t *testing.T) {
	var request basCaptureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"domHtml":"<main data-preview-sheet><pre id=\"rcl-story-result\">{\"passed\":true}</pre></main>","artifacts":[]}`))
	}))
	defer server.Close()

	root := t.TempDir()
	dir := filepath.Join(root, "ui", "src", "pages")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Design.tsx"), []byte("export function Design() { return null }"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Design.story.json"), []byte(`{"schemaVersion":5,"kind":"page","route":"/design","environment":{"fixtures":[]},"stories":[{"id":"detail","name":"Detail","route":"/design/demo/page","role":"anatomy","expect":[{"kind":"text","value":"Design"}]}]}`), 0644))
	executor := BASCaptureExecutor{RCLBaseURL: "http://rcl.test", BASBaseURL: server.URL, HTTPClient: server.Client(), PageSourceRoot: root, PageUIBaseURL: "http://ui.test"}
	_, err := executor.ExecuteStory(context.Background(), "page:Design", "workspace", "detail")
	require.NoError(t, err)
	require.Equal(t, "[data-preview-sheet]", request.ScreenshotSelector)
}

func TestPageStoryURLUsesConcreteStoryRoute(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ui", "src", "pages")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Design.tsx"), []byte("export function Design() { return null }"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Design.story.json"), []byte(`{"schemaVersion":5,"kind":"page","route":"/design","environment":{"fixtures":[]},"stories":[{"id":"detail","name":"Detail","route":"/design/demo/page","role":"anatomy","expect":[{"kind":"text","value":"Design"}]}]}`), 0644))
	executor := BASCaptureExecutor{PageSourceRoot: root, PageUIBaseURL: "http://ui.test"}
	got, err := executor.storyURL("page:Design", "workspace", "detail")
	require.NoError(t, err)
	require.Equal(t, "http://ui.test/design/demo/page?__rcl_page_story=Design&__rcl_story=detail", got)
}

// Use the shared discovery seam, not a private CLI or browser implementation.
func TestNewBASCaptureExecutorUsesSelectedRuntimeAndExplicitOverrides(t *testing.T) {
	t.Setenv("VROOLI_PORT_LOOKUP_STATS_FILE", filepath.Join(t.TempDir(), "lookup-stats.jsonl"))
	t.Setenv("RCL_BAS_API_URL", "")
	t.Setenv("BAS_API_URL", "")
	t.Setenv(cliutil.EnvShadowScenarios, "")
	t.Setenv(cliutil.EnvInstanceVariant, "rcl-capture-fixture")
	t.Setenv(cliutil.EnvVariantDependencies, "browser-automation-studio")
	calls := 0
	restore := cliutil.SetPortLookupRunner(func(ctx context.Context, target, key string) cliutil.ScenarioPortOutcome {
		calls++
		require.Equal(t, "browser-automation-studio@rcl-capture-fixture", target)
		require.Equal(t, "API_PORT", key)
		return cliutil.ScenarioPortOutcome{Port: "19321", Output: "19321"}
	})
	defer restore()
	require.Equal(t, "http://localhost:19321", NewBASCaptureExecutor().BASBaseURL)
	require.Equal(t, 1, calls)
	t.Setenv("BAS_API_URL", " http://bas-explicit.test/ ")
	require.Equal(t, "http://bas-explicit.test", NewBASCaptureExecutor().BASBaseURL)
	t.Setenv("RCL_BAS_API_URL", " http://rcl-bas-explicit.test/ ")
	require.Equal(t, "http://rcl-bas-explicit.test", NewBASCaptureExecutor().BASBaseURL)
	require.Equal(t, 1, calls, "explicit endpoints must bypass runtime discovery")
}

func TestNewBASCaptureExecutorDoesNotGuessPortForMissingSelectedRuntime(t *testing.T) {
	t.Setenv("VROOLI_PORT_LOOKUP_STATS_FILE", filepath.Join(t.TempDir(), "lookup-stats.jsonl"))
	t.Setenv("RCL_BAS_API_URL", "")
	t.Setenv("BAS_API_URL", "")
	t.Setenv(cliutil.EnvShadowScenarios, "")
	t.Setenv(cliutil.EnvInstanceVariant, "rcl-capture-missing")
	t.Setenv(cliutil.EnvVariantDependencies, "browser-automation-studio")
	restore := cliutil.SetPortLookupRunner(func(ctx context.Context, target, key string) cliutil.ScenarioPortOutcome {
		require.Equal(t, "browser-automation-studio@rcl-capture-missing", target)
		return cliutil.ScenarioPortOutcome{Output: "scenario is not running", Err: errors.New("scenario is not running")}
	})
	defer restore()
	executor := NewBASCaptureExecutor()
	require.Empty(t, executor.BASBaseURL)
	require.ErrorContains(t, executor.Probe(context.Background()), "discovery")
}

func TestBASCaptureExecutorRecoversWhenSelectedRuntimeBecomesAvailable(t *testing.T) {
	t.Setenv("VROOLI_PORT_LOOKUP_STATS_FILE", filepath.Join(t.TempDir(), "lookup-stats.jsonl"))
	t.Setenv("RCL_BAS_API_URL", "")
	t.Setenv("BAS_API_URL", "")
	t.Setenv(cliutil.EnvShadowScenarios, "")
	t.Setenv(cliutil.EnvInstanceVariant, "rcl-capture-recovery")
	t.Setenv(cliutil.EnvVariantDependencies, "browser-automation-studio")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	parsed, err := url.Parse(server.URL)
	require.NoError(t, err)
	available := false
	clock := time.Now()
	cliutil.SetPortCacheNowForTest(func() time.Time { return clock })
	defer cliutil.SetPortCacheNowForTest(nil)
	restore := cliutil.SetPortLookupRunner(func(ctx context.Context, target, key string) cliutil.ScenarioPortOutcome {
		require.Equal(t, "browser-automation-studio@rcl-capture-recovery", target)
		if !available {
			return cliutil.ScenarioPortOutcome{Output: "scenario is not running", Err: errors.New("scenario is not running")}
		}
		return cliutil.ScenarioPortOutcome{Port: parsed.Port(), Output: parsed.Port()}
	})
	defer restore()
	executor := NewBASCaptureExecutor()
	require.Error(t, executor.Probe(context.Background()))
	available = true
	clock = clock.Add(5 * time.Second)
	require.NoError(t, executor.Probe(context.Background()), "same executor must retry discovery after the dependency recovers")
}
