package uxcapture

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

type scenarioResolverFixture struct{ ui, api string }

func (f scenarioResolverFixture) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return f.ui, nil
}
func (f scenarioResolverFixture) ResolveScenarioURL(_ context.Context, _, port string) (string, error) {
	if port == "UI_PORT" {
		return f.ui, nil
	}
	return f.api, nil
}

type scenarioCaptureFixture struct {
	request *driver.ScenarioCaptureRequest
}

func (*scenarioCaptureFixture) CapturePublicPage(context.Context, *driver.PublicCaptureRequest) (json.RawMessage, error) {
	return nil, nil
}
func (*scenarioCaptureFixture) CaptureHTMLFile(context.Context, *driver.FileCaptureRequest) (json.RawMessage, error) {
	return nil, nil
}
func (f *scenarioCaptureFixture) CaptureScenarioPage(_ context.Context, request *driver.ScenarioCaptureRequest) (json.RawMessage, error) {
	f.request = request
	return json.RawMessage(`{"schema_version":"ux-capture.v1","id":"00000000-0000-4000-8000-000000000004","captured_at":"2026-11-14T05:00:00Z","source":{"kind":"scenario","scenario":"git-control-tower","route":"/"},"session":{"id":"session-test","browser":"chromium"},"viewport":{"width":1280,"height":720},"lenses":{"L0":{"status":"absent","reason":"fixture"},"L1":{"status":"absent","reason":"fixture"},"L2":{"status":"absent","reason":"fixture"},"L3":{"status":"absent","reason":"fixture"},"L4":{"status":"absent","reason":"fixture"},"L5":{"status":"absent","reason":"fixture"},"L6":{"status":"absent","reason":"fixture"}},"privacy":{"redaction_status":"complete"},"artifacts":{}}`), nil
}

func TestScenarioCaptureResolvesRunningAppRootWithoutRoute(t *testing.T) {
	client := &scenarioCaptureFixture{}
	router := chi.NewRouter()
	MountScenarioHTTP(router, client, scenarioResolverFixture{ui: "http://localhost:4321", api: "http://localhost:4322"}, filepath.Join(t.TempDir(), "captures"))
	request := httptest.NewRequest("POST", "/ux-capture/scenario", strings.NewReader(`{"scenario":"git-control-tower"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 201 {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "bundle_dir") {
		t.Fatal("BAS response leaked its private bundle directory")
	}
	if client.request == nil || client.request.URL != "http://localhost:4321/" || client.request.Route != "/" {
		t.Fatalf("scenario root was not resolved: %+v", client.request)
	}
	if len(client.request.AllowedOrigins) != 2 || client.request.AllowedOrigins[1] != "http://localhost:4322" {
		t.Fatalf("expected UI and API origins to be allowlisted, got %+v", client.request.AllowedOrigins)
	}
}

func TestScenarioCaptureForwardsBoundedWaitSelectorAndTheme(t *testing.T) {
	client := &scenarioCaptureFixture{}
	router := chi.NewRouter()
	MountScenarioHTTP(router, client, scenarioResolverFixture{ui: "http://localhost:4321", api: "http://localhost:4322"}, filepath.Join(t.TempDir(), "captures"))
	request := httptest.NewRequest("POST", "/ux-capture/scenario", strings.NewReader(`{"scenario":"git-control-tower","wait_selector":"[data-ready]","wait_hidden_selector":".animate-spin","theme":"dark","full_page_pixels":true}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 201 {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	if client.request == nil || client.request.WaitSelector != "[data-ready]" || client.request.WaitHiddenSelector != ".animate-spin" || client.request.Theme != "dark" || !client.request.FullPagePixels {
		t.Fatalf("capture options were not forwarded: %+v", client.request)
	}
}

func TestScenarioCaptureForwardsStrictTokenDiagnostic(t *testing.T) {
	client := &scenarioCaptureFixture{}
	router := chi.NewRouter()
	MountScenarioHTTP(router, client, scenarioResolverFixture{ui: "http://localhost:4321", api: "http://localhost:4322"}, filepath.Join(t.TempDir(), "captures"))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/ux-capture/scenario", strings.NewReader(`{"scenario":"git-control-tower","strict_tokens":true}`)))
	if response.Code != 201 {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	if client.request == nil || !client.request.StrictTokens {
		t.Fatalf("strict-token option was not forwarded: %+v", client.request)
	}
}

func TestScenarioCaptureForwardsComponentEvidenceProfile(t *testing.T) {
	client := &scenarioCaptureFixture{}
	router := chi.NewRouter()
	MountScenarioHTTP(router, client, scenarioResolverFixture{ui: "http://localhost:4321", api: "http://localhost:4322"}, t.TempDir())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/ux-capture/scenario", strings.NewReader(`{"scenario":"git-control-tower","evidence_profile":"componenttests","region_selector":"[data-story-root]"}`)))
	if response.Code != 201 {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	if client.request == nil || client.request.EvidenceProfile != "componenttests" || client.request.RegionSelector != "[data-story-root]" {
		t.Fatalf("component evidence was not forwarded: %+v", client.request)
	}
}

func TestScenarioCaptureRejectsUnknownEvidenceProfileAndLongRegionSelector(t *testing.T) {
	for _, body := range []string{`{"scenario":"git-control-tower","evidence_profile":"all"}`, `{"scenario":"git-control-tower","region_selector":"` + strings.Repeat("x", 257) + `"}`} {
		router := chi.NewRouter()
		MountScenarioHTTP(router, &scenarioCaptureFixture{}, scenarioResolverFixture{ui: "http://localhost:4321"}, t.TempDir())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/ux-capture/scenario", strings.NewReader(body)))
		if response.Code != 400 {
			t.Fatalf("body %s: status=%d", body[:min(len(body), 80)], response.Code)
		}
	}
}

func TestScenarioCaptureRejectsInvalidThemeAndOversizedWaitSelector(t *testing.T) {
	for _, body := range []string{
		`{"scenario":"git-control-tower","theme":"sepia"}`,
		`{"scenario":"git-control-tower","wait_selector":"` + strings.Repeat("x", 513) + `"}`,
	} {
		router := chi.NewRouter()
		MountScenarioHTTP(router, &scenarioCaptureFixture{}, scenarioResolverFixture{ui: "http://localhost:4321", api: "http://localhost:4322"}, t.TempDir())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/ux-capture/scenario", strings.NewReader(body)))
		if response.Code != 400 {
			t.Fatalf("body %s: status = %d; body=%s", body[:min(len(body), 80)], response.Code, response.Body.String())
		}
	}
}

func TestScenarioCaptureRejectsNonLoopbackResolver(t *testing.T) {
	client := &scenarioCaptureFixture{}
	router := chi.NewRouter()
	MountScenarioHTTP(router, client, scenarioResolverFixture{ui: "https://example.com", api: ""}, t.TempDir())
	request := httptest.NewRequest("POST", "/ux-capture/scenario", strings.NewReader(`{"scenario":"git-control-tower"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 424 {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	if client.request != nil {
		t.Fatalf("capture ran for non-loopback target: %+v", client.request)
	}
}

func TestCleanCaptureRouteAndRedaction(t *testing.T) {
	if got, err := cleanCaptureRoute(""); err != nil || got != "/" {
		t.Fatalf("default route = %q, err=%v", got, err)
	}
	if _, err := cleanCaptureRoute("//evil.example/"); err == nil {
		t.Fatal("protocol-relative route accepted")
	}
	if got := redactRouteQuery("/changes?owner=alice&token=secret"); got != "/changes?owner=%5BREDACTED%5D&token=%5BREDACTED%5D" {
		t.Fatalf("sensitive route query was not redacted: %q", got)
	}
}
