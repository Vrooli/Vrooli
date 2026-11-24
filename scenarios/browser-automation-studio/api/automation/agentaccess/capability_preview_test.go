package agentaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

func TestCapabilityPreviewUsesCurrentAgentManagerGrantAcrossModes(t *testing.T) {
	cases := []struct {
		name, mode, profileMode string
		profile                 map[string]any
		mapping                 map[string]any
		identity                *sessionpermission.BrowserIdentityProfile
		wantIdentityBehavior    string
		wantPermissionID        string
	}{
		{
			name: "managed effective identity and permission summary", mode: "managed", profileMode: "fresh",
			profile: map[string]any{
				"profileId": "managed-profile", "profileMode": "fresh", "level": "act", "view": true, "control": true,
				"allowedOrigins": []string{"https://safe.test"}, "credentialRefs": []string{"vault-ref-1"}, "allowHistory": true,
				"requireApprovalFor": []string{"control"},
				"identity_profile":   map[string]any{"preset": "balanced", "viewport_width": 1440, "viewport_height": 900, "device_scale_factor": 1.25, "locale": "en-GB", "timezone_id": "Europe/London", "color_scheme": "dark"},
			},
			mapping:              map[string]any{"view": true, "control": true, "profile_mode": "fresh"},
			identity:             &sessionpermission.BrowserIdentityProfile{Preset: "balanced", ViewportWidth: 1440, ViewportHeight: 900, DeviceScaleFactor: 1.25, Locale: "en-GB", TimezoneID: "Europe/London", ColorScheme: "dark"},
			wantIdentityBehavior: "managed fresh-context identity", wantPermissionID: "managed-profile",
		},
		{
			name: "installed viewport-only profile", mode: "installed-binary", profileMode: "isolated",
			profile: map[string]any{
				"profileId": "installed-profile", "profileMode": "isolated", "level": "observe", "view": true,
				"allowedOrigins": []string{"https://safe.test"}, "identity_profile": map[string]any{"viewport_width": 1365, "viewport_height": 768},
			},
			mapping:              map[string]any{"view": true, "profile_mode": "isolated"},
			identity:             &sessionpermission.BrowserIdentityProfile{ViewportWidth: 1365, ViewportHeight: 768},
			wantIdentityBehavior: "installed binary keeps native defaults", wantPermissionID: "installed-profile",
		},
		{
			name: "attached mode preserves the real tab and declared extension permissions", mode: "user-attached", profileMode: "user_attached",
			profile: map[string]any{
				"profileId": "attached-profile", "profileMode": "user_attached", "level": "act", "view": true, "control": true,
				"allowedOrigins": []string{"https://safe.test"}, "extensionIds": []string{"ext-safe"},
			},
			mapping:              map[string]any{"view": true, "control": true, "profile_mode": "user_attached", "extension_access": true, "extension_permissions": map[string][]string{"ext-safe": {"storage", "https://safe.test/*"}}},
			wantIdentityBehavior: "preserve the selected real tab identity", wantPermissionID: "attached-profile",
		},
		{
			name: "remote remains fail closed", mode: "remote", profileMode: "remote",
			profile:              map[string]any{"profileId": "remote-profile", "profileMode": "remote", "level": "observe", "view": true, "allowedOrigins": []string{"https://safe.test"}},
			mapping:              map[string]any{"view": true, "profile_mode": "remote"},
			wantIdentityBehavior: "remote identity behavior is unsupported", wantPermissionID: "remote-profile",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profileBody, _ := json.Marshal(tc.profile)
			mappingBody, _ := json.Marshal(tc.mapping)
			var starts, openAuthorizationCalls int
			am := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/browser/resolve-context":
					_, _ = w.Write([]byte(`{"caller_id":"run:capability-test","run_id":"run-1","grant_receipt":"grant-1","capability":{"permissions":{"profileId":"` + tc.wantPermissionID + `"}}}`))
				case "/api/v1/browser/resolve-profile":
					var request map[string]string
					_ = json.NewDecoder(r.Body).Decode(&request)
					if request["profile_ref"] != tc.wantPermissionID {
						t.Errorf("profile_ref=%q", request["profile_ref"])
					}
					response := map[string]any{
						"status": "active", "run_id": "run-1", "grant_receipt": "grant-1", "mode": tc.mode,
						"permission_profile": json.RawMessage(profileBody), "session_permission_mapping": json.RawMessage(mappingBody),
						"budget":                               map[string]any{"sessionMinutes": 5, "actions": 25, "bytes": 4096, "inferenceTokens": 0},
						"unsupported_sessionpermission_fields": []string{},
					}
					_ = json.NewEncoder(w).Encode(response)
				case "/api/v1/browser/authorize-open":
					openAuthorizationCalls++
					_, _ = w.Write([]byte(`{"authorized":true,"grant_receipt":"grant-1"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer am.Close()

			runContext := &AgentManagerRunContext{BaseURL: am.URL, Client: am.Client()}
			profileResolver := &AgentManagerProfileResolver{RunContext: runContext}
			profileForRequirements := sessionpermission.Profile{
				ID: tc.wantPermissionID, ProfileID: tc.wantPermissionID, ProfileMode: tc.profileMode,
				Effect: sessionpermission.EffectObserve, Approval: sessionpermission.ApprovalNone, View: true, MaxBytes: 4096,
				URLAllow: []string{"https://safe.test/*"}, IdentityProfile: tc.identity,
			}
			if tc.mode == "managed" {
				profileForRequirements.Control = true
				profileForRequirements.Approval = sessionpermission.ApprovalEveryAction
				profileForRequirements.CredentialIDs = []string{"vault-ref-1"}
				profileForRequirements.HistoryRead = true
			}
			if tc.mode == "user-attached" {
				profileForRequirements.Control = true
				profileForRequirements.ExtensionAccess = true
				profileForRequirements.ExtensionPermissions = map[string][]string{"ext-safe": {"storage", "https://safe.test/*"}}
			}
			backend, ok := backendForProfileMode(tc.profileMode)
			if !ok {
				t.Fatalf("profile mode %q has no backend", tc.profileMode)
			}
			report := sessionpermission.BackendReport{Backend: backend, Qualified: tc.mode != "remote", EvidenceID: "fake-source-receipt"}
			if report.Qualified {
				report.Capabilities = requiredCapabilities(profileForRequirements, string(backend))
			}
			broker, err := NewBroker(&session.Manager{}, profileResolver, &bridgeOpenRecorder{}, bridgeCaller("run:capability-test"))
			if err != nil {
				t.Fatal(err)
			}
			broker.SetTrustedOpenContextResolver(runContext)
			broker.SetCapabilityReporter(staticCapabilityReporter{report})
			ctx := WithBrowserAgentCredentials(context.Background(), "signed-token", "grant-1")
			preview, err := broker.CapabilityPreview(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if preview.GrantedBackend != backend || preview.PermissionProfile.ProfileID != tc.wantPermissionID {
				t.Fatalf("grant binding = %q / %q, want %q / %q", preview.GrantedBackend, preview.PermissionProfile.ProfileID, backend, tc.wantPermissionID)
			}
			if preview.PermissionProfile.ProfileMode != tc.profileMode || len(preview.Modes) != 4 {
				t.Fatalf("preview mode/profile rows = %q / %d, want %q / 4", preview.PermissionProfile.ProfileMode, len(preview.Modes), tc.profileMode)
			}
			selected := modeCapability(preview.Modes, backend)
			if !selected.ProfileCompatible || !strings.Contains(selected.IdentityBehavior, tc.wantIdentityBehavior) {
				t.Fatalf("selected mode did not expose exact profile behavior: %#v", selected)
			}
			if tc.identity != nil && (selected.ConfiguredIdentity == nil || *selected.ConfiguredIdentity != *tc.identity) {
				t.Fatalf("configured identity = %#v, want %#v", selected.ConfiguredIdentity, tc.identity)
			}
			if tc.identity == nil && selected.ConfiguredIdentity != nil {
				t.Fatalf("backend invented fresh-session identity: %#v", selected.ConfiguredIdentity)
			}
			if tc.mode == "managed" {
				if !selected.OpenRequestAllowed || !preview.PermissionProfile.HistoryRead || preview.PermissionProfile.Approval != sessionpermission.ApprovalEveryAction || preview.PermissionProfile.CredentialRefs[0] != "vault-ref-1" {
					t.Fatalf("managed grant summary/preflight = %#v / %#v", preview.PermissionProfile, selected)
				}
				if modeCapability(preview.Modes, sessionpermission.BackendInstalled).ProfileCompatible {
					t.Fatal("installed backend inherited the managed grant")
				}
			}
			if tc.mode == "installed-binary" && (!selected.OpenRequestAllowed || selected.ConfiguredIdentity.ViewportWidth != 1365 || preview.PermissionProfile.IdentityProfile.ViewportHeight != 768) {
				t.Fatalf("installed viewport preflight = %#v / %#v", selected, preview.PermissionProfile)
			}
			if tc.mode == "user-attached" {
				if !selected.OpenRequestAllowed || !strings.Contains(selected.NextStep, "popup consent") || preview.PermissionProfile.ExtensionPermissions["ext-safe"][0] != "storage" {
					t.Fatalf("attached preservation/extension permission summary = %#v / %#v", selected, preview.PermissionProfile)
				}
			}
			if tc.mode == "remote" {
				if selected.OpenRequestAllowed || len(selected.Missing) == 0 || !strings.Contains(selected.ProfileReason, "adapter is unavailable") {
					t.Fatalf("remote preflight failed to stay closed: %#v", selected)
				}
			}
			if openAuthorizationCalls != 0 || starts != 0 {
				t.Fatalf("pre-open negotiation caused effects: AM open authorization=%d driver starts=%d", openAuthorizationCalls, starts)
			}
		})
	}
}

func modeCapability(rows []ModeCapabilityPreview, backend sessionpermission.Backend) ModeCapabilityPreview {
	for _, row := range rows {
		if row.Backend == backend {
			return row
		}
	}
	return ModeCapabilityPreview{}
}

func TestCapabilityPreviewReportsRequestedPermissionDenialsBeforeOpen(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile", ProfileID: "profile", ProfileMode: "fresh", Effect: sessionpermission.EffectObserve, Approval: sessionpermission.ApprovalNone, View: true, Control: false, URLAllow: []string{"https://safe.test/*"}}
	policy, err := sessionpermission.New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	broker, err := NewBroker(&session.Manager{}, bridgeProfileResolver{profile: profile, authorizer: policy}, &bridgeOpenRecorder{}, bridgeCaller("caller"))
	if err != nil {
		t.Fatal(err)
	}
	broker.SetTrustedOpenContextResolver(fixedOpenContext(TrustedBrowserOpenContext{CallerID: "caller", GrantReceipt: "grant", IdentityToken: "signed", PermissionRef: "profile"}))
	broker.SetCapabilityReporter(staticCapabilityReporter{{Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "fake", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityControl}}})
	preview, err := broker.CapabilityPreview(context.Background(), []sessionpermission.Capability{sessionpermission.CapabilityControl})
	if err != nil {
		t.Fatal(err)
	}
	managed := modeCapability(preview.Modes, sessionpermission.BackendManaged)
	if managed.OpenRequestAllowed || len(managed.PermissionDenied) != 1 || managed.PermissionDenied[0] != sessionpermission.CapabilityControl {
		t.Fatalf("profile-denied control was not distinguished from runtime support: %#v", managed)
	}
	if modeCapability(preview.Modes, sessionpermission.BackendInstalled).ProfileCompatible {
		t.Fatal("installed backend was treated as compatible with managed profile")
	}
}

func TestRemoteProfileOpenRemainsFailClosedBeforeDriverEffect(t *testing.T) {
	profile := sessionpermission.Profile{ID: "remote-profile", ProfileID: "remote-profile", ProfileMode: "remote", Effect: sessionpermission.EffectObserve, Approval: sessionpermission.ApprovalNone, View: true, URLAllow: []string{"https://safe.test/*"}}
	policy, err := sessionpermission.New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	var starts int
	driverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/start" {
			starts++
		}
		http.NotFound(w, r)
	}))
	defer driverServer.Close()
	driverClient, err := driver.NewClientWithURL(driverServer.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	manager := session.NewManagerWithClient(driverClient)
	openAuthorizer := &bridgeOpenRecorder{}
	broker, err := NewBroker(manager, bridgeProfileResolver{profile: profile, authorizer: policy}, openAuthorizer, bridgeCaller("caller"))
	if err != nil {
		t.Fatal(err)
	}
	broker.SetTrustedOpenContextResolver(fixedOpenContext(TrustedBrowserOpenContext{CallerID: "caller", GrantReceipt: "grant", IdentityToken: "signed", PermissionRef: "remote-profile"}))
	broker.SetCapabilityReporter(staticCapabilityReporter{{Backend: sessionpermission.BackendRemote, Qualified: true, EvidenceID: "synthetic-only", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve}}})
	_, err = broker.Open(context.Background(), OpenRequest{PermissionRef: "remote-profile", Backend: string(sessionpermission.BackendRemote), ReuseMode: "fresh", StartURL: "https://safe.test/"})
	if err == nil || !strings.Contains(err.Error(), "remote browser backend adapter is unavailable") {
		t.Fatalf("remote open error=%v, want adapter denial", err)
	}
	if starts != 0 || len(openAuthorizer.requests) != 0 {
		t.Fatalf("remote fallback caused effects: driver starts=%d open-owner calls=%d", starts, len(openAuthorizer.requests))
	}
}
