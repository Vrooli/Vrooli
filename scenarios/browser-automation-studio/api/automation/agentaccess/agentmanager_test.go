package agentaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

type fixedOpenContext TrustedBrowserOpenContext

type staticCapabilityReporter []sessionpermission.BackendReport

func (r staticCapabilityReporter) BackendReports(context.Context) ([]sessionpermission.BackendReport, error) {
	return append([]sessionpermission.BackendReport(nil), r...), nil
}

func (f fixedOpenContext) ResolveBrowserOpen(context.Context) (TrustedBrowserOpenContext, error) {
	return TrustedBrowserOpenContext(f), nil
}

func TestResolveBrowserOpenBindsProfileToAMRunContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/browser/resolve-context" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-Agent-Identity-Token") != "signed-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "receipt-1" {
			http.Error(w, "missing trusted credentials", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"caller_id":"run:run-1","run_id":"run-1","grant_receipt":"receipt-1","harness":"codex","capability":{"enabled":true,"permissions":{"profileId":"audio-profile","audioMode":"pipewire_qualification"}}}`))
	}))
	defer server.Close()
	resolver := &AgentManagerRunContext{BaseURL: server.URL, Client: server.Client()}
	ctx := WithBrowserAgentCredentials(context.Background(), "signed-token", "receipt-1")
	trusted, err := resolver.ResolveBrowserOpen(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if trusted.CallerID != "run:run-1" || trusted.RunID != "run-1" || trusted.PermissionRef != "audio-profile" || trusted.GrantReceipt != "receipt-1" || trusted.IdentityToken != "signed-token" {
		t.Fatalf("trusted open context = %#v", trusted)
	}
	ctx = WithBrowserAgentCredentials(context.Background(), "signed-token", "wrong-receipt")
	if _, err := resolver.ResolveBrowserOpen(ctx); err == nil {
		t.Fatalf("mismatched caller receipt should fail closed, got %v", err)
	}
}

func TestAgentManagerOpenAuthorizerUsesTrustedRunGrant(t *testing.T) {
	var gotHeader string
	var gotReceiptHeader string
	var got map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/browser/authorize-open" {
			t.Errorf("path = %s", r.URL.Path)
		}
		gotHeader = r.Header.Get("X-Agent-Identity-Token")
		gotReceiptHeader = r.Header.Get("X-Vrooli-Browser-Grant-Receipt")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authorized":true,"grant_receipt":"browser:run-1"}`))
	}))
	defer server.Close()
	trusted := TrustedBrowserOpenContext{CallerID: "caller-1", GrantReceipt: "browser:run-1", IdentityToken: "signed-agent-token", PermissionRef: "profile-1"}
	authorizer := AgentManagerOpenAuthorizer{BaseURL: server.URL, Context: fixedOpenContext(trusted)}
	err := authorizer.AuthorizeOpen(context.Background(), sessionpermission.OpenRequest{CallerID: "caller-1", GrantReceipt: "browser:run-1", Backend: sessionpermission.BackendManaged, ProfileID: "observe", InitialURL: "https://example.test"})
	if err != nil {
		t.Fatalf("AuthorizeOpen: %v", err)
	}
	if gotHeader != trusted.IdentityToken {
		t.Fatalf("identity header = %q", gotHeader)
	}
	if gotReceiptHeader != trusted.GrantReceipt {
		t.Fatalf("grant receipt header = %q", gotReceiptHeader)
	}
	if got["profile_id"] != "observe" || got["initial_url"] != "https://example.test" || got["mode"] != "managed" {
		t.Fatalf("authorization payload = %#v", got)
	}
	for _, field := range []string{"caller_id", "owner_id", "run_id", "grant_receipt"} {
		if _, ok := got[field]; ok {
			t.Fatalf("request must not contain caller/run field %q", field)
		}
	}
}

func TestAgentManagerOpenAuthorizerRejectsUnboundReceipt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"authorized":true,"grant_receipt":"browser:other-run"}`))
	}))
	defer server.Close()
	trusted := TrustedBrowserOpenContext{CallerID: "caller-1", GrantReceipt: "browser:run-1", IdentityToken: "signed-agent-token"}
	authorizer := AgentManagerOpenAuthorizer{BaseURL: server.URL, Context: fixedOpenContext(trusted)}
	err := authorizer.AuthorizeOpen(context.Background(), sessionpermission.OpenRequest{CallerID: "caller-1", GrantReceipt: "browser:run-1", Backend: sessionpermission.BackendManaged, ProfileID: "observe", InitialURL: "https://example.test"})
	if err == nil {
		t.Fatal("expected mismatched grant receipt to fail")
	}
}

func TestAgentManagerProfileResolverMapsOnlySupportedPermissionFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/browser/resolve-profile" {
			t.Errorf("path=%s", r.URL.Path)
		}
		if r.Header.Get("X-Agent-Identity-Token") != "signed-agent-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "browser:run-1" {
			t.Errorf("headers=%v", r.Header)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["profile_ref"] != "profile-1" {
			t.Errorf("body=%v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"active","run_id":"run-1","grant_receipt":"browser:run-1","mode":"managed","permission_profile":{"profileId":"profile-1","profileMode":"fresh","level":"act","view":true,"control":true,"allowedOrigins":["https://example.test"],"credentialRefs":["vault-1"],"extensionIds":["ext-1"],"allowHistory":true,"appMonitorMetrics":true,"appMonitorTrace":true,"appMonitorTraceMaxDurationMs":30000,"appMonitorTraceMaxBytes":8388608,"requireApprovalFor":[],"identity_profile":{"preset":"balanced","viewport_width":1440,"viewport_height":900,"device_scale_factor":1.25,"locale":"en-GB","timezone_id":"Europe/London","color_scheme":"dark"}},"session_permission_mapping":{"view":true,"control":true,"profile_mode":"fresh","extension_access":true,"extension_permissions":{"ext-1":[]}},"budget":{"sessionMinutes":5,"actions":7,"bytes":4096,"inferenceTokens":0},"unsupported_sessionpermission_fields":[]}`))
	}))
	defer server.Close()
	resolver := AgentManagerProfileResolver{RunContext: &AgentManagerRunContext{BaseURL: server.URL, Client: server.Client()}}
	ctx := WithBrowserAgentCredentials(context.Background(), "signed-agent-token", "browser:run-1")
	profile, authorizer, err := resolver.Resolve(ctx, "profile-1")
	if err != nil {
		t.Fatal(err)
	}
	if authorizer == nil || profile.ID != "profile-1" || profile.Effect != sessionpermission.EffectAct || profile.MaxActions != 7 || profile.MaxBytes != 4096 || profile.MaxDuration != 5*time.Minute || !profile.HistoryRead || !profile.AllowsURL("https://example.test/path") {
		t.Fatalf("profile=%#v authorizer=%T", profile, authorizer)
	}
	if !profile.AppMonitorMetrics || !profile.AppMonitorTrace || profile.AppMonitorTraceMaxDurationMS != 30_000 || profile.AppMonitorTraceMaxBytes != 8<<20 {
		t.Fatalf("App Monitor capture grant did not preserve its explicit bounded dimensions: %#v", profile)
	}
	if !profile.View || !profile.Control || !profile.ExtensionAccess || profile.ExtensionInstall || profile.ProfileMode != "fresh" || len(profile.ExtensionPermissions) != 1 {
		t.Fatalf("unsupported permissions were inferred: %#v", profile)
	}
	if permissions, ok := profile.ExtensionPermissions["ext-1"]; !ok || len(permissions) != 0 {
		t.Fatalf("extension access was not restricted to the explicit empty-permission allowlist: %#v", profile.ExtensionPermissions)
	}
	if profile.IdentityProfile == nil || profile.IdentityProfile.Preset != "balanced" || profile.IdentityProfile.ViewportWidth != 1440 || profile.IdentityProfile.Locale != "en-GB" || profile.IdentityProfile.TimezoneID != "Europe/London" || profile.IdentityProfile.ColorScheme != "dark" {
		t.Fatalf("trusted identity profile was not preserved: %#v", profile.IdentityProfile)
	}
}

func TestAgentManagerProfileResolverMapsExtensionLifecycleOnlyFromExactManagedOptIns(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mode        string
		install     *bool
		load        *bool
		ids         []string
		permissions map[string][]string
		unsupported []string
		wantError   bool
	}{
		{name: "explicit install and load", mode: "managed", install: boolPointer(true), load: boolPointer(true), ids: []string{"ext-1"}, permissions: map[string][]string{"ext-1": {}}},
		{name: "legacy absent opt-ins default false", mode: "managed", ids: []string{"ext-1"}, permissions: map[string][]string{"ext-1": {}}},
		{name: "install without allowlist", mode: "managed", install: boolPointer(true), wantError: true},
		{name: "load without allowlist", mode: "managed", load: boolPointer(true), wantError: true},
		{name: "attached install mismatch", mode: "user-attached", install: boolPointer(true), ids: []string{"ext-1"}, permissions: map[string][]string{"ext-1": {}}, wantError: true},
		{name: "unsupported marker with forged install", mode: "managed", install: boolPointer(true), ids: []string{"ext-1"}, permissions: map[string][]string{"ext-1": {}}, unsupported: []string{"ExtensionInstall"}, wantError: true},
		{name: "mapping cannot add extension ID", mode: "managed", install: boolPointer(true), ids: []string{"ext-1"}, permissions: map[string][]string{"ext-1": {}, "forged": {}}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			permissionProfile := map[string]any{
				"profileId": "profile-1", "profileMode": map[string]string{"managed": "fresh", "user-attached": "user_attached"}[tc.mode], "level": "act",
				"view": true, "control": true, "extensionIds": tc.ids,
			}
			if tc.install != nil {
				permissionProfile["extensionInstall"] = *tc.install
			}
			if tc.load != nil {
				permissionProfile["extensionLoad"] = *tc.load
			}
			payload := map[string]any{
				"status": "active", "run_id": "run-1", "grant_receipt": "browser:run-1", "mode": tc.mode,
				"permission_profile": permissionProfile,
				"session_permission_mapping": map[string]any{
					"view": true, "control": true, "profile_mode": permissionProfile["profileMode"],
					"extension_access": len(tc.ids) > 0, "extension_permissions": tc.permissions,
				},
				"budget":                               map[string]any{"sessionMinutes": 5, "actions": 7, "bytes": 4096, "inferenceTokens": 0},
				"unsupported_sessionpermission_fields": tc.unsupported,
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
			defer server.Close()
			resolver := AgentManagerProfileResolver{RunContext: &AgentManagerRunContext{BaseURL: server.URL, Client: server.Client()}}
			profile, _, err := resolver.Resolve(WithBrowserAgentCredentials(context.Background(), "signed-agent-token", "browser:run-1"), "profile-1")
			if tc.wantError {
				if err == nil {
					t.Fatal("malformed or broader extension permission response was accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			wantInstall, wantLoad := false, false
			if tc.install != nil {
				wantInstall = *tc.install
			}
			if tc.load != nil {
				wantLoad = *tc.load
			}
			if profile.ExtensionInstall != wantInstall || profile.ExtensionLoad != wantLoad {
				t.Fatalf("extension opt-ins install=%v load=%v, want %v/%v", profile.ExtensionInstall, profile.ExtensionLoad, wantInstall, wantLoad)
			}
			if profile.AppMonitorMetrics || profile.AppMonitorTrace || profile.AppMonitorTraceMaxDurationMS != 0 || profile.AppMonitorTraceMaxBytes != 0 {
				t.Fatalf("legacy profile defaulted App Monitor permissions or budgets open: %#v", profile)
			}
		})
	}
}

func boolPointer(value bool) *bool { return &value }

func TestAgentManagerProfileResolverRejectsUnsupportedOrUnboundedAppMonitorTrace(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mode        string
		profileMode string
		trace       bool
		durationMS  int64
		maxBytes    int64
		wantErr     bool
	}{
		{name: "legacy defaults closed", mode: "managed", profileMode: "fresh"},
		{name: "bounded managed grant", mode: "managed", profileMode: "fresh", trace: true, durationMS: 30_000, maxBytes: 8 << 20},
		{name: "attached grant rejected", mode: "user-attached", profileMode: "user_attached", trace: true, durationMS: 30_000, maxBytes: 8 << 20, wantErr: true},
		{name: "missing budget rejected", mode: "managed", profileMode: "fresh", trace: true, wantErr: true},
		{name: "oversized budget rejected", mode: "managed", profileMode: "fresh", trace: true, durationMS: 60_001, maxBytes: 8 << 20, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]any{
				"status": "active", "run_id": "run-1", "grant_receipt": "browser:run-1", "mode": tc.mode,
				"permission_profile": map[string]any{
					"profileId": "profile-1", "profileMode": tc.profileMode, "level": "observe", "view": true,
					"allowedOrigins": []string{"https://example.test"}, "appMonitorTrace": tc.trace,
					"appMonitorTraceMaxDurationMs": tc.durationMS, "appMonitorTraceMaxBytes": tc.maxBytes,
				},
				"session_permission_mapping":           map[string]any{"view": true, "control": false, "profile_mode": tc.profileMode},
				"budget":                               map[string]any{"sessionMinutes": 5, "actions": 7, "bytes": 4096, "inferenceTokens": 0},
				"unsupported_sessionpermission_fields": []string{},
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
			defer server.Close()
			resolver := AgentManagerProfileResolver{RunContext: &AgentManagerRunContext{BaseURL: server.URL, Client: server.Client()}}
			profile, _, err := resolver.Resolve(WithBrowserAgentCredentials(context.Background(), "signed-agent-token", "browser:run-1"), "profile-1")
			if tc.wantErr {
				if err == nil {
					t.Fatal("unsupported App Monitor trace grant accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if profile.AppMonitorTrace != tc.trace || profile.AppMonitorTraceMaxDurationMS != tc.durationMS || profile.AppMonitorTraceMaxBytes != tc.maxBytes {
				t.Fatalf("mapped profile = %#v", profile)
			}
		})
	}
}

func TestBrokerRejectsForgedIdentityAndBackendMismatchBeforeDriverStart(t *testing.T) {
	for _, tc := range []struct {
		name        string
		backend     string
		identity    map[string]any
		wantMessage string
	}{
		{name: "malformed locale", backend: "managed", identity: map[string]any{"locale": "en_US"}, wantMessage: "identity locale"},
		{name: "host-local timezone", backend: "managed", identity: map[string]any{"timezone_id": "Local"}, wantMessage: "identity timezone"},
		{name: "attached backend mismatch", backend: string(sessionpermission.BackendUserAttached), identity: map[string]any{"locale": "en-GB"}, wantMessage: "matching Agent Manager profile mode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			started := 0
			amServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/browser/resolve-profile" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status": "active", "run_id": "run-1", "grant_receipt": "browser:run-1", "mode": "managed",
					"permission_profile": map[string]any{
						"profileId": "profile-1", "profileMode": "fresh", "level": "act", "view": true, "control": true,
						"allowedOrigins": []string{"https://example.test"}, "identity_profile": tc.identity,
					},
					"session_permission_mapping":           map[string]any{"view": true, "control": true, "profile_mode": "fresh"},
					"budget":                               map[string]any{"sessionMinutes": 5, "actions": 7, "bytes": 4096, "inferenceTokens": 0},
					"unsupported_sessionpermission_fields": []string{},
				})
			}))
			defer amServer.Close()
			driverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/session/start" {
					started++
				}
				http.Error(w, "driver must not start", http.StatusInternalServerError)
			}))
			defer driverServer.Close()
			client, err := driver.NewClientWithURL(driverServer.URL, driver.WithoutCircuitBreaker())
			if err != nil {
				t.Fatal(err)
			}
			manager := session.NewManagerWithClient(client)
			trusted := TrustedBrowserOpenContext{CallerID: "caller-1", GrantReceipt: "browser:run-1", IdentityToken: "signed-token", PermissionRef: "profile-1"}
			resolver := &AgentManagerProfileResolver{RunContext: &AgentManagerRunContext{BaseURL: amServer.URL, Client: amServer.Client()}}
			openOwner := &bridgeOpenRecorder{}
			broker, err := NewBroker(manager, resolver, openOwner, bridgeCaller(trusted.CallerID))
			if err != nil {
				t.Fatal(err)
			}
			broker.SetTrustedOpenContextResolver(fixedOpenContext(trusted))
			ctx := WithBrowserAgentCredentials(context.Background(), trusted.IdentityToken, trusted.GrantReceipt)
			_, err = broker.Open(ctx, OpenRequest{PermissionRef: "profile-1", Backend: tc.backend, StartURL: "https://example.test/start", ReuseMode: "fresh"})
			if err == nil || !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("Broker.Open error=%v, want reason %q", err, tc.wantMessage)
			}
			if started != 0 || len(openOwner.requests) != 0 {
				t.Fatalf("rejected identity reached open owners: driver starts=%d open authorization calls=%d", started, len(openOwner.requests))
			}
		})
	}
}

func TestAgentManagerProfileResolverDoesNotInferUnsupportedExtensionAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"active","run_id":"run-1","grant_receipt":"browser:run-1","mode":"managed","permission_profile":{"profileId":"profile-1","profileMode":"fresh","level":"act","extensionIds":["ext-1"]},"session_permission_mapping":{"profile_mode":"fresh","extension_access":true,"extension_permissions":{"ext-1":[]}},"budget":{"sessionMinutes":5,"actions":7},"unsupported_sessionpermission_fields":["ExtensionAccess"]}`))
	}))
	defer server.Close()
	resolver := AgentManagerProfileResolver{RunContext: &AgentManagerRunContext{BaseURL: server.URL, Client: server.Client()}}
	profile, _, err := resolver.Resolve(WithBrowserAgentCredentials(context.Background(), "signed-agent-token", "browser:run-1"), "profile-1")
	if err != nil {
		t.Fatal(err)
	}
	if profile.ExtensionAccess || len(profile.ExtensionPermissions) != 0 {
		t.Fatalf("unsupported extension access was inferred: %#v", profile)
	}
}

type identityEvidenceRecorder struct{}

func (identityEvidenceRecorder) RecordBrowserSession(context.Context, string, sessionpermission.Backend, string, string) error {
	return nil
}
func (identityEvidenceRecorder) RecordBrowserAction(context.Context, string, sessionpermission.Backend, string, string, string) error {
	return nil
}

func TestTrustedIdentityProfileFlowsIntoExistingManagedSessionOwner(t *testing.T) {
	identityJSON := `"identity_profile":{"preset":"balanced","viewport_width":1440,"viewport_height":900,"device_scale_factor":1.25,"locale":"en-GB","timezone_id":"Europe/London","color_scheme":"dark"}`
	amServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/browser/resolve-profile":
			_, _ = w.Write([]byte(`{"status":"active","run_id":"run-1","grant_receipt":"browser:run-1","mode":"managed","permission_profile":{"profileId":"profile-1","profileMode":"fresh","level":"act","view":true,"control":true,"allowedOrigins":["https://example.test"],` + identityJSON + `},"session_permission_mapping":{"view":true,"control":true,"profile_mode":"fresh"},"budget":{"sessionMinutes":5,"actions":7,"inferenceTokens":0},"unsupported_sessionpermission_fields":[]}`))
		case "/api/v1/browser/resolve-caller":
			if r.Header.Get("X-Agent-Identity-Token") != "signed-agent-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "browser:run-1" {
				http.Error(w, "missing trusted browser identity", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"caller_id":"run:00000000-0000-0000-0000-000000000001","grant_receipt":"browser:run-1"}`))
		case "/api/v1/browser/authorize-open":
			var req struct {
				Mode       string `json:"mode"`
				ProfileID  string `json:"profile_id"`
				InitialURL string `json:"initial_url"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Mode != "managed" || req.ProfileID != "profile-1" || req.InitialURL != "https://example.test/start" {
				t.Errorf("authorize-open request = %#v err=%v", req, err)
				http.Error(w, "wrong open binding", http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`{"authorized":true,"grant_receipt":"browser:run-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer amServer.Close()

	var driverProfile *struct {
		Preset      string `json:"preset"`
		Fingerprint struct {
			ViewportWidth  int     `json:"viewport_width"`
			ViewportHeight int     `json:"viewport_height"`
			Scale          float64 `json:"device_scale_factor"`
			Locale         string  `json:"locale"`
			Timezone       string  `json:"timezone_id"`
			ColorScheme    string  `json:"color_scheme"`
		} `json:"fingerprint"`
	}
	var started, closed bool
	driverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/session/start":
			var request driver.CreateSessionRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode driver open request: %v", err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			started = true
			if request.Options.BrowserProfile != nil {
				driverProfile = &struct {
					Preset      string `json:"preset"`
					Fingerprint struct {
						ViewportWidth  int     `json:"viewport_width"`
						ViewportHeight int     `json:"viewport_height"`
						Scale          float64 `json:"device_scale_factor"`
						Locale         string  `json:"locale"`
						Timezone       string  `json:"timezone_id"`
						ColorScheme    string  `json:"color_scheme"`
					} `json:"fingerprint"`
				}{Preset: request.Options.BrowserProfile.Preset}
				driverProfile.Fingerprint.ViewportWidth = request.Options.BrowserProfile.Fingerprint.ViewportWidth
				driverProfile.Fingerprint.ViewportHeight = request.Options.BrowserProfile.Fingerprint.ViewportHeight
				driverProfile.Fingerprint.Scale = request.Options.BrowserProfile.Fingerprint.DeviceScaleFactor
				driverProfile.Fingerprint.Locale = request.Options.BrowserProfile.Fingerprint.Locale
				driverProfile.Fingerprint.Timezone = request.Options.BrowserProfile.Fingerprint.TimezoneID
				driverProfile.Fingerprint.ColorScheme = request.Options.BrowserProfile.Fingerprint.ColorScheme
			}
			_ = json.NewEncoder(w).Encode(driver.CreateSessionResponse{
				SessionID: "managed-identity-session", LeaseID: "lease-1", ActivePageID: "page-1",
				ActualViewport: &driver.ActualViewport{Width: 1440, Height: 900, Source: driver.ViewportSourceFingerprint, Reason: "identity profile"},
			})
		case "/session/managed-identity-session/record/navigate":
			_ = json.NewEncoder(w).Encode(driver.NavigateResponse{DriverPageID: "page-1", URL: "https://example.test/start", Title: "start"})
		case "/session/managed-identity-session/close":
			closed = true
			_ = json.NewEncoder(w).Encode(driver.CloseSessionResponse{Success: true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer driverServer.Close()

	client, err := driver.NewClientWithURL(driverServer.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	manager := session.NewManagerWithClient(client)
	trusted := TrustedBrowserOpenContext{CallerID: "run:00000000-0000-0000-0000-000000000001", GrantReceipt: "browser:run-1", IdentityToken: "signed-agent-token", PermissionRef: "profile-1"}
	resolver := &AgentManagerProfileResolver{RunContext: &AgentManagerRunContext{BaseURL: amServer.URL, Client: amServer.Client()}}
	openAuthorizer := &AgentManagerOpenAuthorizer{BaseURL: amServer.URL, Client: amServer.Client(), Context: fixedOpenContext(trusted)}
	broker, err := NewBroker(manager, resolver, openAuthorizer, bridgeCaller(trusted.CallerID))
	if err != nil {
		t.Fatal(err)
	}
	broker.SetTrustedOpenContextResolver(fixedOpenContext(trusted))
	broker.SetEvidenceRecorder(identityEvidenceRecorder{})
	broker.SetCapabilityReporter(staticCapabilityReporter{{
		Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "managed-identity-test",
		Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityControl, sessionpermission.CapabilityIdentityCustomization},
	}})

	openCtx := WithBrowserAgentCredentials(context.Background(), trusted.IdentityToken, trusted.GrantReceipt)
	info, err := broker.Open(openCtx, OpenRequest{
		Backend: "managed", StartURL: "https://example.test/start", ReuseMode: "fresh",
	})
	if err != nil {
		t.Fatalf("trusted managed open failed: %v", err)
	}
	if !started || driverProfile == nil || info.IdentityProfile == nil || info.ActualViewport == nil {
		t.Fatalf("identity profile did not reach the managed owner or report the effective viewport: started=%v driver=%#v info=%#v", started, driverProfile, info)
	}
	binding, err := broker.ResolveInferenceBinding(openCtx, info.SessionID)
	if err != nil {
		t.Fatalf("resolve trusted active browser inference binding: %v", err)
	}
	if binding.RunID != "00000000-0000-0000-0000-000000000001" || binding.SessionID != info.SessionID || binding.GrantReceipt != trusted.GrantReceipt {
		t.Fatalf("inference binding was not resolved from the active broker session: %+v", binding)
	}
	if _, err := broker.ResolveInferenceBinding(WithBrowserAgentCredentials(context.Background(), trusted.IdentityToken, "wrong-grant"), info.SessionID); err == nil {
		t.Fatal("inference binding must reject a different Agent Manager grant")
	}
	if info.IdentityProfile.Preset != "balanced" || info.IdentityProfile.ViewportWidth != 1440 || info.IdentityProfile.Locale != "en-GB" {
		t.Fatalf("session receipt did not echo the trusted identity profile: %#v", info.IdentityProfile)
	}
	if driverProfile.Preset != "balanced" || driverProfile.Fingerprint.ViewportWidth != 1440 || driverProfile.Fingerprint.ViewportHeight != 900 || driverProfile.Fingerprint.Scale != 1.25 || driverProfile.Fingerprint.Locale != "en-GB" || driverProfile.Fingerprint.Timezone != "Europe/London" || driverProfile.Fingerprint.ColorScheme != "dark" {
		t.Fatalf("driver received wrong identity context profile: %#v", driverProfile)
	}
	if info.ActualViewport.Width != 1440 || info.ActualViewport.Height != 900 || info.ActualViewport.Source != driver.ViewportSourceFingerprint {
		t.Fatalf("session receipt did not report the owner-observed viewport: %#v", info.ActualViewport)
	}
	entry, err := broker.get(info.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	entry.profile.View = false
	deniedRead := broker.info(entry)
	if deniedRead.IdentityProfile != nil || deniedRead.ActualViewport != nil || deniedRead.IdentityEvidence != nil {
		t.Fatalf("session without view authority received identity evidence: %#v", deniedRead)
	}
	entry.profile.View = true
	if err := broker.Close(WithBrowserAgentCredentials(context.Background(), trusted.IdentityToken, trusted.GrantReceipt), info.SessionID); err != nil {
		t.Fatalf("close managed fake-owner session: %v", err)
	}
	if !closed {
		t.Fatal("managed session close did not reach the existing driver owner")
	}
}

func TestAgentManagerBackendModeAliases(t *testing.T) {
	cases := map[sessionpermission.Backend]string{
		sessionpermission.BackendManaged:      "managed",
		sessionpermission.BackendInstalled:    "installed-binary",
		sessionpermission.BackendUserAttached: "user-attached",
		sessionpermission.BackendRemote:       "remote",
	}
	for backend, want := range cases {
		got, err := agentManagerMode(backend)
		if err != nil || got != want {
			t.Fatalf("agentManagerMode(%q) = %q, %v; want %q", backend, got, err, want)
		}
	}
}

func TestAgentManagerEvidenceRecorderSendsOnlyBoundedRunMetadata(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/api/v1/browser/record-evidence" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("X-Agent-Identity-Token") != "signed-agent-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "browser-grant:opaque" {
			t.Errorf("credentials not forwarded: %v", r.Header)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		for _, forbidden := range []string{"url", "title", "text", "typed_value", "caller_id", "owner_id", "run_id", "grant_receipt", "input"} {
			if _, found := payload[forbidden]; found {
				t.Errorf("evidence payload contains forbidden field %q: %#v", forbidden, payload)
			}
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	recorder := AgentManagerEvidenceRecorder{RunContext: &AgentManagerRunContext{BaseURL: server.URL, Client: server.Client()}}
	ctx := WithBrowserAgentCredentials(context.Background(), "signed-agent-token", "browser-grant:opaque")
	if err := recorder.RecordBrowserSession(ctx, "session-1", sessionpermission.BackendManaged, "profile-1", "opened"); err != nil {
		t.Fatal(err)
	}
	if err := recorder.RecordBrowserAction(ctx, "session-1", sessionpermission.BackendManaged, "profile-1", "type", "completed"); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("recorded %d evidence events; want 2", requests)
	}
}

func TestAgentManagerProfileResolverUsesTrustedModeAndDoesNotInferUnsupportedMode(t *testing.T) {
	cases := []struct {
		name, mode, profileMode string
		unsupported             []string
		want                    string
		wantError               bool
	}{
		{name: "managed fresh", mode: "managed", profileMode: "fresh", want: "fresh"},
		{name: "installed isolated", mode: "installed-binary", profileMode: "isolated", want: "isolated"},
		{name: "user attached", mode: "user-attached", profileMode: "user_attached", want: "user_attached"},
		{name: "remote", mode: "remote", profileMode: "remote", want: "remote"},
		{name: "missing legacy mode", mode: "managed", wantError: true},
		{name: "mismatched persisted mode", mode: "managed", profileMode: "isolated", wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unsupported, _ := json.Marshal(tc.unsupported)
			body := `{"status":"active","run_id":"run-1","grant_receipt":"browser:run-1","mode":"` + tc.mode + `","permission_profile":{"profileId":"profile-1","profileMode":"` + tc.profileMode + `","level":"observe"},"session_permission_mapping":{"profile_mode":"` + tc.profileMode + `"},"budget":{"sessionMinutes":5,"actions":7},"unsupported_sessionpermission_fields":` + string(unsupported) + `}`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			resolver := AgentManagerProfileResolver{RunContext: &AgentManagerRunContext{BaseURL: server.URL, Client: server.Client()}}
			profile, _, err := resolver.Resolve(WithBrowserAgentCredentials(context.Background(), "signed-agent-token", "browser:run-1"), "profile-1")
			if tc.wantError {
				if err == nil {
					t.Fatal("expected missing or mismatched persisted mode to fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if profile.ProfileMode != tc.want {
				t.Fatalf("profile mode = %q, want %q", profile.ProfileMode, tc.want)
			}
		})
	}
}

func TestOpenFailsClosedForUnqualifiedManagedAndInstalledBackends(t *testing.T) {
	trusted := TrustedBrowserOpenContext{CallerID: "caller-1", GrantReceipt: "browser:run-1", IdentityToken: "signed-agent-token", PermissionRef: "profile-1"}
	for _, tc := range []struct {
		name, backend, reuse, profileMode string
		want                              string
	}{
		{name: "managed has no qualification receipt", backend: "managed", reuse: "fresh", profileMode: "fresh", want: "not capability-qualified"},
		{name: "installed has no qualification receipt", backend: "installed", reuse: "fresh", profileMode: "isolated", want: "not capability-qualified"},
		{name: "remote has no qualification receipt", backend: "remote", reuse: "fresh", profileMode: "remote", want: "not capability-qualified"},
		{name: "installed refuses reuse", backend: "installed", reuse: "reuse", profileMode: "isolated", want: "fresh isolated"},
		{name: "installed rejects managed profile", backend: "installed", reuse: "fresh", profileMode: "fresh", want: "matching Agent Manager profile mode"},
		{name: "managed rejects installed profile", backend: "managed", reuse: "fresh", profileMode: "isolated", want: "matching Agent Manager profile mode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := sessionpermission.Profile{ID: "profile-1", ProfileMode: tc.profileMode, Effect: sessionpermission.EffectObserve, Approval: sessionpermission.ApprovalNone, View: true, URLAllow: []string{"https://example.test/*"}}
			policy := &bridgePolicyRecorder{profile: profile}
			broker, err := NewBroker(&session.Manager{}, bridgeProfileResolver{profile: profile, authorizer: policy}, &bridgeOpenRecorder{}, bridgeCaller("caller-1"))
			if err != nil {
				t.Fatal(err)
			}
			broker.SetTrustedOpenContextResolver(fixedOpenContext(trusted))
			_, err = broker.Open(context.Background(), OpenRequest{PermissionRef: "profile-1", Backend: tc.backend, ReuseMode: tc.reuse, StartURL: "https://example.test/"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Open error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCapabilityPreflightKeepsEveryBackendVisibleForPartialOwnerReports(t *testing.T) {
	broker := &Broker{}
	broker.SetCapabilityReporter(staticCapabilityReporter{{
		Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "managed-receipt",
		Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve},
	}})
	rows, err := broker.Capabilities(context.Background(), []sessionpermission.Capability{sessionpermission.CapabilityObserve})
	if err != nil {
		t.Fatal(err)
	}
	want := []sessionpermission.Backend{
		sessionpermission.BackendManaged,
		sessionpermission.BackendInstalled,
		sessionpermission.BackendUserAttached,
		sessionpermission.BackendRemote,
	}
	if len(rows) != len(want) {
		t.Fatalf("partial owner report hid modes: %#v", rows)
	}
	for i, backend := range want {
		if rows[i].Backend != backend {
			t.Fatalf("backend row %d = %q, want stable order %q: %#v", i, rows[i].Backend, backend, rows)
		}
		if rows[i].Identity == nil {
			t.Fatalf("backend %q omitted identity customization limits", backend)
		}
	}
	wantConfigured := []string{"preset", "viewport", "device_scale_factor", "locale", "timezone", "color_scheme"}
	if len(rows[0].Identity.Configured) != len(wantConfigured) {
		t.Fatalf("managed configurable identity dimensions = %#v", rows[0].Identity.Configured)
	}
	for i := range wantConfigured {
		if rows[0].Identity.Configured[i] != wantConfigured[i] {
			t.Fatalf("managed configurable identity dimensions = %#v", rows[0].Identity.Configured)
		}
	}
	if len(rows[0].Identity.Observed) != 8 || rows[0].Identity.Observed[0] != "engine" {
		t.Fatalf("managed owner-observed identity dimensions = %#v", rows[0].Identity.Observed)
	}
	if len(rows[1].Identity.Configured) != 1 || rows[1].Identity.Configured[0] != "viewport" || rows[1].Identity.Unsupported["locale"] == "" || len(rows[1].Identity.Observed) != 8 {
		t.Fatalf("installed identity support matrix = %#v", rows[1].Identity)
	}
	if reason := rows[0].Identity.Unsupported["user_agent"]; reason != "" {
		t.Fatalf("managed preflight marked owner-observed user-agent unsupported: %q", reason)
	}
	if len(rows[2].Identity.Configured) != 0 || rows[2].Identity.Unsupported["engine_version"] == "" {
		t.Fatalf("attached mode did not preserve identity or state its version limit: %#v", rows[2].Identity)
	}
	if !rows[0].Accepted || rows[0].Available[0] != sessionpermission.CapabilityObserve {
		t.Fatalf("exactly reported managed capability was not accepted: %#v", rows[0])
	}
	for _, row := range rows[1:] {
		if row.Accepted || row.Qualified || len(row.Available) != 0 {
			t.Fatalf("unreported backend inherited another mode's evidence: %#v", row)
		}
		if _, exists := row.Reasons[sessionpermission.CapabilityObserve]; !exists {
			t.Fatalf("unreported backend omitted its preflight limit: %#v", row)
		}
	}
}

func TestCapabilityPreflightRejectsUnknownAndDuplicateBackendEvidence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		reports []sessionpermission.BackendReport
		want    string
	}{
		{name: "unknown backend", reports: []sessionpermission.BackendReport{{Backend: "browser-ish", Qualified: true, EvidenceID: "receipt"}}, want: "unsupported backend"},
		{name: "duplicate backend", reports: []sessionpermission.BackendReport{{Backend: sessionpermission.BackendManaged}, {Backend: sessionpermission.BackendManaged}}, want: "duplicate backend"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := mergeBackendQualificationReports([]sessionpermission.BackendReport{{Backend: sessionpermission.BackendManaged}}, tc.reports)
			if err == nil || !strings.Contains(err.Error(), tc.want) || rows != nil {
				t.Fatalf("merge result rows=%#v err=%v, want closed failure %q", rows, err, tc.want)
			}
		})
	}
}

func TestQualifiedRemoteBackendNeverFallsBackToLocalSession(t *testing.T) {
	profile := sessionpermission.Profile{ID: "remote-profile", ProfileMode: "remote", Effect: sessionpermission.EffectObserve, Approval: sessionpermission.ApprovalNone, View: true, URLAllow: []string{"https://example.test/*"}}
	policy := &bridgePolicyRecorder{profile: profile}
	localOpen := &bridgeOpenRecorder{}
	broker, err := NewBroker(&session.Manager{}, bridgeProfileResolver{profile: profile, authorizer: policy}, localOpen, bridgeCaller("caller-1"))
	if err != nil {
		t.Fatal(err)
	}
	broker.SetTrustedOpenContextResolver(fixedOpenContext(TrustedBrowserOpenContext{CallerID: "caller-1", GrantReceipt: "browser:run-1", IdentityToken: "signed-agent-token", PermissionRef: profile.ID}))
	broker.SetCapabilityReporter(staticCapabilityReporter{{Backend: sessionpermission.BackendRemote, Qualified: true, EvidenceID: "remote-capability-test", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve}}})
	_, err = broker.Open(context.Background(), OpenRequest{PermissionRef: profile.ID, Backend: string(sessionpermission.BackendRemote), ReuseMode: "fresh", StartURL: "https://example.test/"})
	if err == nil || !strings.Contains(err.Error(), "refusing local-session fallback") {
		t.Fatalf("qualified remote open error = %v, want explicit unavailable adapter denial", err)
	}
	if len(localOpen.requests) != 0 {
		t.Fatalf("remote open reached local session owner: %+v", localOpen.requests)
	}
}

func TestOpenRejectsCallerSelectedProfileThatDiffersFromRunGrant(t *testing.T) {
	profile := sessionpermission.Profile{ID: "persisted-profile", ProfileID: "persisted-profile", ProfileMode: "fresh", Effect: sessionpermission.EffectObserve, Approval: sessionpermission.ApprovalNone, View: true, URLAllow: []string{"https://example.test/*"}}
	policy := &bridgePolicyRecorder{profile: profile}
	broker, err := NewBroker(&session.Manager{}, bridgeProfileResolver{profile: profile, authorizer: policy}, &bridgeOpenRecorder{}, bridgeCaller("caller-1"))
	if err != nil {
		t.Fatal(err)
	}
	broker.SetTrustedOpenContextResolver(fixedOpenContext(TrustedBrowserOpenContext{CallerID: "caller-1", GrantReceipt: "browser:run-1", IdentityToken: "signed-agent-token", PermissionRef: profile.ID}))
	_, err = broker.Open(context.Background(), OpenRequest{PermissionRef: "caller-chosen-profile", Backend: "managed", ReuseMode: "fresh", StartURL: "https://example.test/"})
	if err == nil || !strings.Contains(err.Error(), "does not match the trusted Agent Manager run context") {
		t.Fatalf("caller-selected profile should be rejected, got %v", err)
	}
}

func TestOpenWithoutTrustedAMContextFailsBeforeOwnerCreation(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile-1", ProfileMode: "fresh", Effect: sessionpermission.EffectObserve, Approval: sessionpermission.ApprovalNone, View: true, URLAllow: []string{"https://example.test/*"}}
	policy := &bridgePolicyRecorder{profile: profile}
	openOwner := &bridgeOpenRecorder{}
	broker, err := NewBroker(&session.Manager{}, bridgeProfileResolver{profile: profile, authorizer: policy}, openOwner, bridgeCaller("caller-1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = broker.Open(context.Background(), OpenRequest{StartURL: "https://example.test/start", Backend: "managed", ReuseMode: "fresh"})
	if err == nil || !strings.Contains(err.Error(), "trusted Agent Manager open context resolver is required") {
		t.Fatalf("untrusted open error=%v", err)
	}
	if len(openOwner.requests) != 0 {
		t.Fatalf("untrusted profile-less open reached owner: %+v", openOwner.requests)
	}
}

func TestInstalledBackendRejectsUnsupportedIdentityFields(t *testing.T) {
	identity := &sessionpermission.BrowserIdentityProfile{Preset: "balanced", Locale: "en-GB"}
	profile := sessionpermission.Profile{ID: "installed-profile", ProfileMode: "isolated", Effect: sessionpermission.EffectObserve, Approval: sessionpermission.ApprovalNone, View: true, IdentityProfile: identity, URLAllow: []string{"https://example.test/*"}}
	policy := &bridgePolicyRecorder{profile: profile}
	openOwner := &bridgeOpenRecorder{}
	broker, err := NewBroker(&session.Manager{}, bridgeProfileResolver{profile: profile, authorizer: policy}, openOwner, bridgeCaller("caller-1"))
	if err != nil {
		t.Fatal(err)
	}
	broker.SetTrustedOpenContextResolver(fixedOpenContext(TrustedBrowserOpenContext{CallerID: "caller-1", GrantReceipt: "browser:run-1", IdentityToken: "signed-agent-token", PermissionRef: profile.ID}))
	_, err = broker.Open(context.Background(), OpenRequest{PermissionRef: profile.ID, Backend: string(sessionpermission.BackendInstalled), ReuseMode: "fresh", StartURL: "https://example.test/"})
	if err == nil || !strings.Contains(err.Error(), "viewport only") {
		t.Fatalf("installed backend profile controls error = %v, want explicit unsupported reason", err)
	}
	if len(openOwner.requests) != 0 || len(policy.requests) != 0 {
		t.Fatalf("unsupported identity combination reached authorization effects: open=%v policy=%v", openOwner.requests, policy.requests)
	}
}

func TestInstalledTrustedViewportFlowsIntoIsolatedSessionOptions(t *testing.T) {
	amServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/browser/resolve-profile" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"active","run_id":"installed-run","grant_receipt":"browser:installed-run","mode":"installed-binary","permission_profile":{"profileId":"installed-viewport","profileMode":"isolated","level":"observe","view":true,"allowedOrigins":["https://example.test"],"identity_profile":{"viewport_width":1365,"viewport_height":768}},"session_permission_mapping":{"view":true,"profile_mode":"isolated"},"budget":{"sessionMinutes":5,"actions":7,"bytes":4096,"inferenceTokens":0},"unsupported_sessionpermission_fields":[]}`))
	}))
	defer amServer.Close()
	resolver := &AgentManagerProfileResolver{RunContext: &AgentManagerRunContext{BaseURL: amServer.URL, Client: amServer.Client()}}
	profile, _, err := resolver.Resolve(WithBrowserAgentCredentials(context.Background(), "signed-token", "browser:installed-run"), "installed-viewport")
	if err != nil {
		t.Fatalf("resolve persisted installed viewport capability: %v", err)
	}
	if profile.ProfileMode != "isolated" || profile.IdentityProfile == nil || profile.IdentityProfile.ViewportWidth != 1365 || profile.IdentityProfile.ViewportHeight != 768 {
		t.Fatalf("resolver did not preserve the trusted installed viewport grant: %+v", profile)
	}
	policy := &bridgePolicyRecorder{profile: profile}
	var started, closed bool
	var gotRequest driver.CreateSessionRequest
	driverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/session/start":
			if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
				t.Errorf("decode installed start request: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			started = true
			_ = json.NewEncoder(w).Encode(driver.CreateSessionResponse{SessionID: "installed-viewport-session", LeaseID: "lease-viewport", ActivePageID: "page-viewport", ActualViewport: &driver.ActualViewport{Width: 1365, Height: 768, Source: driver.ViewportSourceRequested}})
		case "/session/installed-viewport-session/record/navigate":
			_ = json.NewEncoder(w).Encode(driver.NavigateResponse{DriverPageID: "page-viewport", URL: "https://example.test/start", Title: "start"})
		case "/session/installed-viewport-session/close":
			closed = true
			_ = json.NewEncoder(w).Encode(driver.CloseSessionResponse{Success: true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer driverServer.Close()
	client, err := driver.NewClientWithURL(driverServer.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	manager := session.NewManagerWithClient(client)
	openOwner := &bridgeOpenRecorder{}
	trustedCaller := "run:00000000-0000-0000-0000-000000000001"
	broker, err := NewBroker(manager, bridgeProfileResolver{profile: profile, authorizer: policy}, openOwner, bridgeCaller(trustedCaller))
	if err != nil {
		t.Fatal(err)
	}
	trusted := TrustedBrowserOpenContext{CallerID: trustedCaller, GrantReceipt: "browser:installed-run", IdentityToken: "signed-token", PermissionRef: profile.ID}
	broker.SetTrustedOpenContextResolver(fixedOpenContext(trusted))
	broker.SetEvidenceRecorder(identityEvidenceRecorder{})
	broker.SetCapabilityReporter(staticCapabilityReporter{{Backend: sessionpermission.BackendInstalled, Qualified: true, EvidenceID: "fake-installed-owner", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityByteBudget, sessionpermission.CapabilityIdentityCustomization}}})
	preflight, preflightErr := broker.Capabilities(context.Background(), requiredCapabilities(profile, string(sessionpermission.BackendInstalled)))
	if preflightErr != nil || len(preflight) < 2 || !preflight[1].Accepted {
		t.Fatalf("installed identity test preflight = %#v err=%v required=%v", preflight, preflightErr, requiredCapabilities(profile, string(sessionpermission.BackendInstalled)))
	}
	info, err := broker.Open(WithBrowserAgentCredentials(context.Background(), trusted.IdentityToken, trusted.GrantReceipt), OpenRequest{
		PermissionRef: profile.ID, Backend: string(sessionpermission.BackendInstalled), ReuseMode: "fresh", StartURL: "https://example.test/start", ViewportWidth: 800, ViewportHeight: 600,
	})
	if err != nil {
		t.Fatalf("open installed viewport profile: %v", err)
	}
	if !started || gotRequest.Options.BrowserBackend != string(sessionpermission.BackendInstalled) || gotRequest.Options.ReuseMode != "fresh" {
		t.Fatalf("installed context did not reach its backend-specific owner: started=%v request=%+v", started, gotRequest.Options)
	}
	if gotRequest.Options.Viewport != (driver.Viewport{Width: 1365, Height: 768}) || gotRequest.Options.BrowserProfile != nil {
		t.Fatalf("driver request did not use trusted viewport-only isolated config: viewport=%+v profile=%+v", gotRequest.Options.Viewport, gotRequest.Options.BrowserProfile)
	}
	if info.IdentityProfile == nil || info.IdentityProfile.ViewportWidth != 1365 || info.ActualViewport == nil || info.ActualViewport.Width != 1365 || info.ActualViewport.Height != 768 {
		t.Fatalf("view-gated owner receipt omitted configured/effective viewport: %#v", info)
	}
	if err := broker.Close(WithBrowserAgentCredentials(context.Background(), trusted.IdentityToken, trusted.GrantReceipt), info.SessionID); err != nil {
		t.Fatalf("close installed fake owner session: %v", err)
	}
	if !closed {
		t.Fatal("installed session close did not reach the isolated owner")
	}
}

func TestCapabilitiesHideUnconfiguredExtensionAuthorityAndPreserveObserveOnly(t *testing.T) {
	claims := []sessionpermission.Capability{
		sessionpermission.CapabilityObserve,
		sessionpermission.CapabilityIdentityCustomization,
		sessionpermission.CapabilityExtensionInstall,
		sessionpermission.CapabilityExtensionLoad,
		sessionpermission.CapabilityExtensionInventory,
		sessionpermission.CapabilityExtensionHostPermissions,
	}
	broker := &Broker{}
	broker.SetCapabilityReporter(staticCapabilityReporter{
		{Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "managed-fixture", Capabilities: claims},
		{Backend: sessionpermission.BackendUserAttached, Qualified: true, EvidenceID: "attached-fixture", Capabilities: claims},
	})
	rows, err := broker.Capabilities(context.Background(), claims)
	if err != nil {
		t.Fatal(err)
	}
	byBackend := map[sessionpermission.Backend]sessionpermission.CapabilityNegotiation{}
	for _, row := range rows {
		byBackend[row.Backend] = row
	}
	managed := byBackend[sessionpermission.BackendManaged]
	if managed.Accepted || len(managed.Available) != 2 || !containsCapability(managed.Available, sessionpermission.CapabilityObserve) || !containsCapability(managed.Available, sessionpermission.CapabilityIdentityCustomization) {
		t.Fatalf("managed extension capabilities were advertised without owners: %#v", managed)
	}
	if managed.Reasons[sessionpermission.CapabilityExtensionInstall] != "BAS extension controller is not configured" || managed.Reasons[sessionpermission.CapabilityExtensionInventory] == "" {
		t.Fatalf("managed missing-owner reasons = %#v", managed.Reasons)
	}
	installed := byBackend[sessionpermission.BackendInstalled]
	for capability, reason := range map[sessionpermission.Capability]string{
		sessionpermission.CapabilityExtensionInventory:       "installed-binary extension inventory adapter is unavailable",
		sessionpermission.CapabilityExtensionHostPermissions: "installed-binary extension permission evidence is unavailable",
		sessionpermission.CapabilityExtensionInstall:         "extension installation is supported only by the managed BAS session adapter",
		sessionpermission.CapabilityExtensionLoad:            "extension loading is supported only by the managed BAS session adapter",
	} {
		if installed.Reasons[capability] != reason || containsCapability(installed.Available, capability) {
			t.Fatalf("installed extension capability %s must be explicitly unavailable: %#v", capability, installed)
		}
	}
	if installed.Identity == nil || len(installed.Identity.Configured) != 1 || installed.Identity.Configured[0] != "viewport" || len(installed.Identity.Observed) != 8 || installed.Identity.Observed[0] != "engine" {
		t.Fatalf("installed mode must expose only its supported viewport provenance: %#v", installed.Identity)
	}
	for _, field := range []string{"preset", "device_scale_factor", "locale", "timezone", "color_scheme"} {
		if installed.Identity.Unsupported[field] == "" {
			t.Fatalf("installed mode omitted unsupported identity control %q: %#v", field, installed.Identity)
		}
	}
	attached := byBackend[sessionpermission.BackendUserAttached]
	if attached.Accepted || len(attached.Available) != 3 {
		t.Fatalf("selected-tab should preserve only observe and bridge inventory: %#v", attached)
	}
	if containsCapability(attached.Available, sessionpermission.CapabilityIdentityCustomization) || attached.Reasons[sessionpermission.CapabilityIdentityCustomization] != "fresh-session identity customization is unsupported for attached and remote modes" {
		t.Fatalf("selected-tab must preserve native identity and reject fresh-session customization: %#v", attached)
	}
	if attached.Reasons[sessionpermission.CapabilityExtensionInstall] == "" || attached.Reasons[sessionpermission.CapabilityExtensionLoad] == "" {
		t.Fatalf("selected-tab unsupported extension operation reasons = %#v", attached.Reasons)
	}
	broker.extensions = extensionControllerFixture{}
	controllerRows, err := broker.Capabilities(context.Background(), claims)
	if err != nil {
		t.Fatal(err)
	}
	var managedWithController sessionpermission.CapabilityNegotiation
	for _, row := range controllerRows {
		if row.Backend == sessionpermission.BackendManaged {
			managedWithController = row
		}
	}
	if managedWithController.Reasons[sessionpermission.CapabilityExtensionInstall] != "verified extension artifact authority is not configured" {
		t.Fatalf("install capability was not gated on artifact verifier: %#v", managedWithController)
	}
	if managedWithController.Reasons[sessionpermission.CapabilityExtensionLoad] != "verified extension artifact authority is not configured" || containsCapability(managedWithController.Available, sessionpermission.CapabilityExtensionLoad) {
		t.Fatalf("load capability was advertised without the artifact authority: %#v", managedWithController)
	}
	broker.SetExtensionArtifactVerifier(extensionArtifactVerifierFunc(func(context.Context, string, string) (ExtensionArtifact, error) {
		return ExtensionArtifact{}, nil
	}))
	configuredRows, err := broker.Capabilities(context.Background(), claims)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range configuredRows {
		if row.Backend != sessionpermission.BackendManaged {
			continue
		}
		for _, capability := range []sessionpermission.Capability{sessionpermission.CapabilityExtensionInstall, sessionpermission.CapabilityExtensionLoad, sessionpermission.CapabilityExtensionInventory, sessionpermission.CapabilityExtensionHostPermissions} {
			if !containsCapability(row.Available, capability) {
				t.Fatalf("configured managed controller/verifier did not expose %s: %#v", capability, row)
			}
		}
	}
	observerRows, err := broker.Capabilities(context.Background(), []sessionpermission.Capability{sessionpermission.CapabilityObserve})
	if err != nil {
		t.Fatal(err)
	}
	if len(observerRows) != 4 {
		t.Fatalf("observer-only preflight omitted backend rows: %#v", observerRows)
	}
	for _, row := range observerRows {
		if row.Backend == sessionpermission.BackendManaged || row.Backend == sessionpermission.BackendUserAttached {
			if !row.Accepted {
				t.Fatalf("observer-only profile became unavailable because extension authority is absent: %#v", row)
			}
			continue
		}
		if row.Accepted || row.Qualified {
			t.Fatalf("unreported backend inherited observer qualification: %#v", row)
		}
	}
}
