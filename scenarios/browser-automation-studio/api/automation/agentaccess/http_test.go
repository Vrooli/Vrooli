package agentaccess

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

func TestParseRequiredCapabilitiesIncludesOwnerGatedModes(t *testing.T) {
	got, err := parseRequiredCapabilities("approval_receipt, credit_budget, byte_budget, history_retention, credential_use, accessibility_tree, app_monitor_metrics, app_monitor_trace")
	if err != nil {
		t.Fatal(err)
	}
	want := []sessionpermission.Capability{sessionpermission.CapabilityApprovalReceipt, sessionpermission.CapabilityCreditBudget, sessionpermission.CapabilityByteBudget, sessionpermission.CapabilityHistoryRetention, sessionpermission.CapabilityCredentialUse, sessionpermission.CapabilityAccessibilityTree, sessionpermission.CapabilityAppMonitorMetrics, sessionpermission.CapabilityAppMonitorTrace}
	if len(got) != len(want) {
		t.Fatalf("capabilities=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("capabilities=%v want=%v", got, want)
		}
	}
	if _, err := parseRequiredCapabilities("credential_use,not-a-capability"); err == nil {
		t.Fatal("unknown capability was accepted")
	}
}

func TestBrowserRoutesRequireSingleOpaqueAgentCredentials(t *testing.T) {
	router := chi.NewRouter()
	(&HTTPHandler{}).RegisterRoutes(router)
	for _, path := range []string{"/browser/capabilities", "/browser/capabilities/profile"} {
		for name, headers := range map[string][]string{
			"missing":         nil,
			"duplicate-token": {"token-a", "token-b"},
			"missing-grant":   {"token-a"},
		} {
			t.Run(strings.TrimPrefix(path, "/browser/")+"/"+name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				for _, value := range headers {
					req.Header.Add("X-Agent-Identity-Token", value)
				}
				if name != "missing-grant" && len(headers) > 0 {
					req.Header.Set("X-Vrooli-Browser-Grant-Receipt", "browser:run-1")
				}
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)
				if resp.Code != http.StatusUnauthorized {
					t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
				}
			})
		}
	}
}

func TestNavigationHistoryRouteRequiresTrustedAgentContext(t *testing.T) {
	router := chi.NewRouter()
	(&HTTPHandler{}).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodGet, "/browser/sessions/session-1/history", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want unauthorized before broker access", resp.Code, resp.Body.String())
	}
}

func TestAccessibilityTreeRouteRequiresTrustedAgentContext(t *testing.T) {
	router := chi.NewRouter()
	(&HTTPHandler{}).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodGet, "/browser/sessions/session-1/accessibility-tree", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want unauthorized before broker access", resp.Code, resp.Body.String())
	}
}

func TestExtensionLifecyclePreflightRouteRequiresTrustedAgentContext(t *testing.T) {
	router := chi.NewRouter()
	(&HTTPHandler{}).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/browser/sessions/session-1/extensions/preflight", strings.NewReader(`{"artifact_ref":"build-1","expected_digest":"sha256:artifact"}`))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want unauthorized before broker access", resp.Code, resp.Body.String())
	}
}

func TestExtensionLifecycleOwnerUnavailableMapsToServiceUnavailable(t *testing.T) {
	resp := httptest.NewRecorder()
	writeError(resp, ErrExtensionOwnerUnavailable)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing extension lifecycle owner status=%d body=%s, want 503", resp.Code, resp.Body.String())
	}
}

func TestHarnessTaskRouteRequiresAgentCredentialsAndRejectsScriptFields(t *testing.T) {
	router := chi.NewRouter()
	(&HTTPHandler{}).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/browser/sessions/session-1/task", strings.NewReader(`{"operation":"click","selector":"#go","script":"document.cookie"}`))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("missing credentials status=%d body=%s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/browser/sessions/session-1/task", strings.NewReader(`{"operation":"click","selector":"#go","script":"document.cookie"}`))
	req.Header.Set("X-Agent-Identity-Token", "opaque-token")
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", "opaque-grant")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), "unknown field") {
		t.Fatalf("script-bearing request status=%d body=%s, want strict decode rejection", resp.Code, resp.Body.String())
	}
}

func TestTraceDraftRouteUsesExistingSessionAndRejectsProjectAuthorityFields(t *testing.T) {
	router := chi.NewRouter()
	(&HTTPHandler{Broker: &Broker{}}).RegisterRoutes(router)
	request := func(body string, authenticated bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/browser/sessions/session-1/trace/draft", strings.NewReader(body))
		if authenticated {
			req.Header.Set("X-Agent-Identity-Token", "opaque-token")
			req.Header.Set("X-Vrooli-Browser-Grant-Receipt", "opaque-grant")
		}
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}
	if got := request(`{"name":"preview","assertions":[{}]}`, false); got.Code != http.StatusUnauthorized {
		t.Fatalf("missing session identity status=%d body=%s", got.Code, got.Body.String())
	}
	if got := request(`{"project_id":"project-1","name":"preview","assertions":[{}]}`, true); got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), "unknown field") {
		t.Fatalf("draft must not accept project write authority status=%d body=%s", got.Code, got.Body.String())
	}
	if got := request(`{"name":"preview"}`, true); got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), "typed assertion") {
		t.Fatalf("draft without assertions status=%d body=%s", got.Code, got.Body.String())
	}
	if got := request(`{"name":"preview","assertions":[{}]}`, true); got.Code == http.StatusServiceUnavailable || strings.Contains(got.Body.String(), "workflow catalog") {
		t.Fatalf("draft must not require workflow persistence owner status=%d body=%s", got.Code, got.Body.String())
	}
}

func TestAppMonitorCaptureRouteRequiresTrustedContextAndStrictPhaseFields(t *testing.T) {
	router := chi.NewRouter()
	(&HTTPHandler{}).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/browser/sessions/session-1/app-monitor-capture", strings.NewReader(`{"phase":"begin"}`))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("missing credentials status=%d body=%s", resp.Code, resp.Body.String())
	}

	request := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/browser/sessions/session-1/app-monitor-capture", strings.NewReader(body))
		req.Header.Set("X-Agent-Identity-Token", "opaque-token")
		req.Header.Set("X-Vrooli-Browser-Grant-Receipt", "opaque-grant")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}
	if got := request(`{"phase":"begin","capture_id":"550e8400-e29b-41d4-a716-446655440000"}`); got.Code != http.StatusBadRequest {
		t.Fatalf("caller-selected capture ID status=%d body=%s", got.Code, got.Body.String())
	}
	if got := request(`{"phase":"finish","capture_id":"550e8400-e29b-41d4-a716-446655440000","url":"https://attacker.test"}`); got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), "unknown field") {
		t.Fatalf("arbitrary URL field status=%d body=%s", got.Code, got.Body.String())
	}
}
