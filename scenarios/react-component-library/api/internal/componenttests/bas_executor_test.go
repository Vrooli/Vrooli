package componenttests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	"react-component-library/internal/components"
)

func TestStoryViewportUsesDeclaredPhoneGeometry(t *testing.T) {
	require.Equal(t, basDimensions{Width: 390, Height: 240}, storyViewport(&components.StoryGeometry{Width: "390px", Height: "240px"}))
	require.Equal(t, basDimensions{Width: 1280, Height: 800}, storyViewport(&components.StoryGeometry{Width: "min(100vw, 390px)", Height: "auto"}))
}

func TestBASCaptureExecutorUsesCanonicalComponentEvidenceBundle(t *testing.T) {
	contents := componentCaptureArtifacts()
	var request uxScenarioCaptureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			require.Equal(t, uxScenarioCapturePath, r.URL.Path)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, componentCaptureResponse(contents))
			return
		}
		artifactPath := strings.TrimPrefix(r.URL.Path, uxArtifactPathPrefix+"44444444-4444-4444-8444-444444444444/artifacts/")
		content, ok := contents[artifactPath]
		require.True(t, ok, "unexpected bundle artifact %q", artifactPath)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	executor := BASCaptureExecutor{RCLUIBaseURL: "http://rcl-ui.test", BASBaseURL: server.URL, HTTPClient: server.Client()}
	result, err := executor.ExecuteStoryWithGeometry(context.Background(), "rcl:OverlayActionBar", "1.0.0", "phone", &components.StoryGeometry{Width: "390px", Height: "844px"})
	require.NoError(t, err)
	require.True(t, result.Passed)
	require.Equal(t, 12.5, result.Performance.MountMS)
	require.Equal(t, "bas-accessibility-snapshot/v1", mustJSONField(t, result.AccessibilityJSON, "schema"))
	require.Equal(t, "rcl-axe-results/v1", mustJSONNestedField(t, result.AccessibilityJSON, "axe", "schema"))
	require.Equal(t, "4.11.4", mustJSONNestedField(t, result.AccessibilityJSON, "axe", "version"))
	require.Empty(t, mustJSONNestedField(t, result.AccessibilityJSON, "axe", "violations"))
	require.Len(t, result.Artifacts, 6)
	require.Equal(t, uxScenarioCaptureRequest{
		Scenario:     "react-component-library",
		Route:        "/preview/rcl:OverlayActionBar/harness.html?motion=reduce&runner=1&story=phone&version=1.0.0",
		WaitSelector: `[data-preview-readiness-marker][data-preview-ready="true"][data-rcl-story-status="passed"]`,
		Width:        390, Height: 844, EvidenceProfile: "componenttests", RegionSelector: `[data-preview-capture-boundary="component-sheet"]`,
	}, request)
	for _, artifact := range result.Artifacts {
		require.Contains(t, artifact.Reference, "/embedded/browser-automation-studio/ux-capture/bundles/44444444-4444-4444-8444-444444444444/artifacts/")
	}
}

func TestDecodeBASStoryResultRetainsCompleteUnavailableAndMissingAxe(t *testing.T) {
	tests := []struct {
		name        string
		axe         string
		wantStatus  string
		wantVersion string
	}{
		{name: "complete violation", axe: `,"axe":{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":[{"id":"button-name","impact":"serious","description":"Button needs a name.","targets":["button"]}]}`, wantStatus: "complete", wantVersion: "4.11.4"},
		{name: "unavailable", axe: `,"axe":{"schema":"rcl-axe-results/v1","status":"unavailable","reason":"axe-core-not-loaded"}`, wantStatus: "unavailable"},
		{name: "missing", wantStatus: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := decodeBASStoryResult(`<pre id="rcl-story-result">{"passed":true,"failures":[],"performance":{}` + test.axe + `}</pre>`)
			require.NoError(t, err)
			if test.wantStatus == "" {
				require.Nil(t, result.AxeResults)
				return
			}
			require.NotNil(t, result.AxeResults)
			require.Equal(t, test.wantStatus, result.AxeResults.Status)
			require.Equal(t, test.wantVersion, result.AxeResults.Version)
			if test.wantStatus == "complete" {
				require.Equal(t, "button-name", result.AxeResults.Violations[0].ID)
				require.Equal(t, []string{"button"}, result.AxeResults.Violations[0].Targets)
			}
		})
	}

	for _, invalid := range []string{
		`{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":null}`,
		`{"schema":"rcl-axe-results/v1","status":"unavailable"}`,
	} {
		_, err := decodeBASStoryResult(`<pre id="rcl-story-result">{"passed":true,"failures":[],"axe":` + invalid + `}</pre>`)
		require.Error(t, err)
	}
}

func TestBASCaptureExecutorRequiresCompleteRedactionAndDeclaredArtifactIntegrity(t *testing.T) {
	contents := componentCaptureArtifacts()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = fmt.Fprint(w, componentCaptureResponse(contents))
			return
		}
		_, _ = w.Write([]byte("tampered"))
	}))
	defer server.Close()
	executor := BASCaptureExecutor{RCLUIBaseURL: "http://rcl.test", BASBaseURL: server.URL, HTTPClient: server.Client()}
	_, err := executor.ExecuteStory(context.Background(), "rcl:Button", "1.0.0", "anatomy")
	require.ErrorContains(t, err, "size does not match")

	badManifest := strings.Replace(componentCaptureResponse(contents), `"redaction_status":"complete"`, `"redaction_status":"partial"`, 1)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = fmt.Fprint(w, badManifest)
			return
		}
		_, _ = w.Write(contents[strings.TrimPrefix(r.URL.Path, uxArtifactPathPrefix+"44444444-4444-4444-8444-444444444444/artifacts/")])
	})
	_, err = executor.ExecuteStory(context.Background(), "rcl:Button", "1.0.0", "anatomy")
	require.ErrorContains(t, err, "incompletely redacted")
}

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
	got, err := appendURLPath("https://bas.example/api%2Fv1/", uxScenarioCapturePath)
	require.NoError(t, err)
	require.Equal(t, "https://bas.example/api%2Fv1"+uxScenarioCapturePath, got)
}

func TestBASInspectionUsesCanonicalCapturePath(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { gotPath = r.URL.Path; w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	executor := BASCaptureExecutor{BASBaseURL: server.URL, HTTPClient: server.Client()}
	inspection, err := executor.InspectProvider(context.Background())
	require.NoError(t, err)
	require.Equal(t, uxScenarioCapturePath, inspection.CapturePath)
	require.Equal(t, "/health", gotPath)
}

func TestSanitizedBASBaseURLRedactsCredentialAndRequestComponents(t *testing.T) {
	got, exact := sanitizedBASBaseURL("https://operator:secret@example.test/prefix?token=hidden#private")
	require.Equal(t, "https://example.test/prefix", got)
	require.False(t, exact)
	require.NotContains(t, got, "secret")
	require.NotContains(t, got, "hidden")
	require.NotContains(t, got, "private")
}

func TestCaptureScenarioURLUsesScenarioBundleAndStyleSample(t *testing.T) {
	contents := componentCaptureArtifacts()
	var request uxScenarioCaptureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			_, _ = fmt.Fprint(w, componentCaptureResponseFor(contents, "rcl-fixture-positive"))
			return
		}
		_, _ = w.Write(contents[strings.TrimPrefix(r.URL.Path, uxArtifactPathPrefix+"44444444-4444-4444-8444-444444444444/artifacts/")])
	}))
	defer server.Close()
	executor := BASCaptureExecutor{BASBaseURL: server.URL, HTTPClient: server.Client()}
	capture, err := executor.CaptureScenarioURL(context.Background(), "rcl-fixture-positive", "/", `[data-rcl-control="true"]`, `[data-rcl-control="true"]`)
	require.NoError(t, err)
	require.Equal(t, "rcl-fixture-positive", request.Scenario)
	require.Equal(t, "/", request.Route)
	require.Equal(t, "componenttests", request.EvidenceProfile)
	require.Contains(t, capture.RegionStyle, "bas-region-style-sample/v1")
}

func TestBrowserVisibleBASArtifactPathUsesEmbeddedProxy(t *testing.T) {
	got := browserVisibleBASArtifactPath("http://127.0.0.1:17116/ux-capture/bundles/id/artifacts/pixels/region.png")
	require.Equal(t, "/embedded/browser-automation-studio/ux-capture/bundles/id/artifacts/pixels/region.png", got)
}

func TestPageStoryCaptureUsesConcreteStoryRoute(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ui", "src", "pages")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Design.tsx"), []byte("export function Design() { return null }"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Design.story.json"), []byte(`{"schemaVersion":5,"kind":"page","route":"/design","environment":{"fixtures":[]},"stories":[{"id":"detail","name":"Detail","route":"/design/demo/page","role":"anatomy","expect":[{"kind":"text","value":"Design"}]}]}`), 0o644))
	executor := BASCaptureExecutor{RCLUIBaseURL: "http://ui.test", PageSourceRoot: root, PageUIBaseURL: "http://ui.test"}
	got, err := executor.storyURL("page:Design", "workspace", "detail")
	require.NoError(t, err)
	require.Equal(t, "http://ui.test/design/demo/page?__rcl_page_story=Design&__rcl_story=detail", got)
}

func TestNewBASCaptureExecutorUsesSelectedRuntimeAndExplicitOverrides(t *testing.T) {
	t.Setenv("RCL_BAS_API_URL", "")
	t.Setenv("RCL_UI_BASE_URL", "http://rcl-ui.test")
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
	require.Equal(t, 1, calls)
}

func TestNewBASCaptureExecutorDoesNotGuessPortForMissingSelectedRuntime(t *testing.T) {
	t.Setenv("RCL_BAS_API_URL", "")
	t.Setenv("RCL_UI_BASE_URL", "http://rcl-ui.test")
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
	t.Setenv("RCL_BAS_API_URL", "")
	t.Setenv("RCL_UI_BASE_URL", "http://rcl-ui.test")
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
	require.NoError(t, executor.Probe(context.Background()))
}

func componentCaptureArtifacts() map[string][]byte {
	frozen := []byte(`<main><pre id="rcl-story-result" hidden>{"passed":true,"failures":[],"performance":{"mountMs":12.5,"commitCount":2,"nodeCount":8},"axe":{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":[],"violationCount":0}}</pre></main>`)
	accessibility := []byte(`{"schema":"bas-accessibility-snapshot/v1","snapshot":"main\n  button \"Save\""}`)
	console := []byte(`{"schema":"bas-console-summary/v1","error_count":0,"warning_count":0}`)
	performance := []byte(`{"schema":"bas-performance-metrics/v1","dom_content_loaded_ms":20,"load_ms":30,"resource_count":4}`)
	region := []byte("crop-png")
	style := []byte(`{"schema":"bas-region-style-sample/v1","computed_style":{"color":"rgb(0, 0, 0)","background_color":"rgb(255, 255, 255)","border_color":"rgb(0, 0, 0)","border_style":"solid","border_width":"1px","visibility":"visible","display":"block","opacity":"1"}}`)
	return map[string][]byte{"page/frozen.html": frozen, "accessibility/snapshot.json": accessibility, "console/summary.json": console, "performance/metrics.json": performance, "pixels/region.png": region, "style/region-sample.json": style}
}

func mustJSONNestedField(t *testing.T, raw, parent, key string) any {
	t.Helper()
	var object map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &object))
	nested, ok := object[parent].(map[string]any)
	require.True(t, ok, "expected %s object in %s", parent, raw)
	return nested[key]
}

func componentCaptureResponse(contents map[string][]byte) string {
	return componentCaptureResponseFor(contents, "react-component-library")
}

func componentCaptureResponseFor(contents map[string][]byte, scenario string) string {
	artifacts := make([]uxCaptureArtifact, 0, len(contents))
	kinds := map[string]string{"page/frozen.html": "frozen-page", "accessibility/snapshot.json": "accessibility-snapshot", "console/summary.json": "console-summary", "performance/metrics.json": "performance-metrics", "pixels/region.png": "region-pixels", "style/region-sample.json": "region-style-sample"}
	for path, content := range contents {
		digest := sha256.Sum256(content)
		artifacts = append(artifacts, uxCaptureArtifact{Kind: kinds[path], Path: path, SizeBytes: int64(len(content)), SHA256: hex.EncodeToString(digest[:])})
	}
	body, _ := json.Marshal(map[string]any{"bundle_dir": "/private/bas/captures/ux/44444444-4444-4444-8444-444444444444", "manifest": uxCaptureBundle{SchemaVersion: "ux-capture.v1", ID: "44444444-4444-4444-8444-444444444444", Source: struct {
		Kind     string `json:"kind"`
		Scenario string `json:"scenario"`
		Route    string `json:"route"`
	}{Kind: "scenario", Scenario: scenario, Route: "/preview/[REDACTED]"}, Privacy: struct {
		RedactionStatus string `json:"redaction_status"`
	}{RedactionStatus: "complete"}, Lenses: map[string]struct {
		Status string `json:"status"`
	}{}, Artifacts: artifacts}})
	return string(body)
}

func mustJSONField(t *testing.T, raw, field string) string {
	t.Helper()
	var value map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &value))
	text, _ := value[field].(string)
	return text
}
