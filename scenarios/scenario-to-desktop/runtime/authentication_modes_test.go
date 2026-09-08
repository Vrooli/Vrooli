package bundleruntime

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/testutil"
)

func testAuthenticationProfile() *manifest.AuthenticationProfile {
	return &manifest.AuthenticationProfile{
		Version:     1,
		Mode:        "personal_local",
		HumanSignIn: "disabled",
		Offline:     true,
		ModeProfiles: map[string]manifest.AuthenticationModeProfile{
			"local_multi_user": {
				Provider:              "scenario-authenticator",
				Resource:              "demo",
				Audience:              "scenario:demo",
				ProviderServiceID:     "scenario-authenticator",
				HumanSignIn:           "required",
				RequiresAuthenticator: true,
			},
			"remote_vrooli": {
				Provider:         "scenario-authenticator",
				Resource:         "demo",
				Audience:         "scenario:demo",
				ProviderEndpoint: "https://auth.example.test",
				HumanSignIn:      "required",
			},
			"shared_provider": {
				Provider:         "landing-page-business-suite",
				Resource:         "demo",
				Audience:         "scenario:demo",
				ProviderEndpoint: "https://provider.example.test",
				HumanSignIn:      "required",
				LeasePath:        "runtime/provider-lease.json",
			},
		},
	}
}

func TestAuthenticationModeManager_SelectionAndRollback(t *testing.T) {
	fs := testutil.NewMockFileSystem()
	clock := testutil.NewMockClock(time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))
	profile := testAuthenticationProfile()
	manager, err := newAuthenticationModeManager(fs, clock, "/app", profile)
	if err != nil {
		t.Fatalf("newAuthenticationModeManager() error = %v", err)
	}

	status, err := manager.selectMode(context.Background(), "local_multi_user")
	if err != nil {
		t.Fatalf("selectMode(local_multi_user) error = %v", err)
	}
	if status["mode"] != "local_multi_user" || status["state"] != "setup_required" {
		t.Fatalf("local multi-user status = %#v", status)
	}
	if _, err := manager.selectMode(context.Background(), "not_declared"); err == nil {
		t.Fatal("selectMode(not_declared) succeeded")
	}

	status, err = manager.rollback(context.Background())
	if err != nil {
		t.Fatalf("rollback() error = %v", err)
	}
	if status["mode"] != "personal_local" || status["state"] != "offline_ready" {
		t.Fatalf("rollback status = %#v", status)
	}

	data, err := fs.ReadFile("/app/runtime/authentication-mode.json")
	if err != nil {
		t.Fatalf("read persisted mode = %v", err)
	}
	if strings.Contains(string(data), "token") || strings.Contains(string(data), "secret") {
		t.Fatalf("mode state contains credential-like material: %s", data)
	}
}

func TestAuthenticationModeManager_ReportsDistinctModeContracts(t *testing.T) {
	fs := testutil.NewMockFileSystem()
	clock := testutil.NewMockClock(time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))
	profile := testAuthenticationProfile()
	manager, err := newAuthenticationModeManager(fs, clock, "/app", profile)
	if err != nil {
		t.Fatalf("newAuthenticationModeManager() error = %v", err)
	}

	options := manager.options()
	if len(options) != 4 {
		t.Fatalf("mode options = %#v, want four declared modes", options)
	}
	byMode := make(map[string]map[string]interface{}, len(options))
	for _, option := range options {
		mode, ok := option["mode"].(string)
		if !ok {
			t.Fatalf("mode option has no string mode: %#v", option)
		}
		byMode[mode] = option
	}
	checks := []struct {
		mode       string
		state      string
		offline    bool
		provider   string
		requiresID bool
	}{
		{mode: "personal_local", state: "offline_ready", offline: true, provider: "", requiresID: false},
		{mode: "local_multi_user", state: "setup_required", provider: "scenario-authenticator", requiresID: true},
		{mode: "remote_vrooli", state: "provider_required", provider: "scenario-authenticator", requiresID: false},
		{mode: "shared_provider", state: "lease_unavailable", provider: "landing-page-business-suite", requiresID: false},
	}
	for _, check := range checks {
		option, ok := byMode[check.mode]
		if !ok {
			t.Fatalf("missing %s option: %#v", check.mode, byMode)
		}
		if option["state"] != check.state || option["offline"] != check.offline || option["provider"] != check.provider || option["requires_authenticator"] != check.requiresID {
			t.Fatalf("%s option = %#v", check.mode, option)
		}
	}
}

func TestAuthenticationModeManager_SharedProviderFailsClosedAndRecovers(t *testing.T) {
	fs := testutil.NewMockFileSystem()
	clock := testutil.NewMockClock(time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))
	profile := testAuthenticationProfile()
	manager, err := newAuthenticationModeManager(fs, clock, "/app", profile)
	if err != nil {
		t.Fatalf("newAuthenticationModeManager() error = %v", err)
	}

	if _, err := manager.selectMode(context.Background(), "shared_provider"); err == nil || !strings.Contains(err.Error(), "lease_unavailable") {
		t.Fatalf("selectMode(shared_provider) error = %v, want lease refusal", err)
	}

	leasePath := "/app/runtime/provider-lease.json"
	if err := fs.WriteFile(leasePath, []byte(`{"expires_at":"2026-09-07T13:00:00Z"}`), 0o600); err != nil {
		t.Fatalf("write lease = %v", err)
	}
	status, err := manager.selectMode(context.Background(), "shared_provider")
	if err != nil {
		t.Fatalf("selectMode(shared_provider) with lease error = %v", err)
	}
	if status["state"] != "lease_ready" {
		t.Fatalf("shared provider status = %#v", status)
	}
	if status["lease_expires_at"] != "2026-09-07T13:00:00Z" {
		t.Fatalf("shared provider lease expiry = %v", status["lease_expires_at"])
	}

	clock.Advance(2 * time.Hour)
	if got := manager.status()["state"]; got != "lease_expired" {
		t.Fatalf("expired shared provider state = %v, want lease_expired", got)
	}
}

func TestAuthenticationModeManager_RejectsCredentialBearingLeaseMetadata(t *testing.T) {
	fs := testutil.NewMockFileSystem()
	clock := testutil.NewMockClock(time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))
	profile := testAuthenticationProfile()
	manager, err := newAuthenticationModeManager(fs, clock, "/app", profile)
	if err != nil {
		t.Fatalf("newAuthenticationModeManager() error = %v", err)
	}

	leasePath := "/app/runtime/provider-lease.json"
	if err := fs.WriteFile(leasePath, []byte(`{"expires_at":"2026-09-07T13:00:00Z","token":"do-not-persist"}`), 0o600); err != nil {
		t.Fatalf("write credential-bearing lease metadata = %v", err)
	}
	shared, _ := profile.ProfileForMode("shared_provider")
	if got := manager.sharedProviderState(shared); got != "lease_invalid" {
		t.Fatalf("credential-bearing lease state = %v, want lease_invalid", got)
	}
	if _, err := manager.selectMode(context.Background(), "shared_provider"); err == nil || !strings.Contains(err.Error(), "lease_invalid") {
		t.Fatalf("selectMode(shared_provider) error = %v, want credential-bearing metadata refusal", err)
	}
}

func TestAuthenticationModeManager_RejectsMismatchedLeaseMetadata(t *testing.T) {
	fs := testutil.NewMockFileSystem()
	clock := testutil.NewMockClock(time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))
	profile := testAuthenticationProfile()
	manager, err := newAuthenticationModeManager(fs, clock, "/app", profile)
	if err != nil {
		t.Fatalf("newAuthenticationModeManager() error = %v", err)
	}

	leasePath := "/app/runtime/provider-lease.json"
	if err := fs.WriteFile(leasePath, []byte(`{"expires_at":"2026-09-07T13:00:00Z","resource":"other-app","audience":"scenario:other"}`), 0o600); err != nil {
		t.Fatalf("write mismatched lease metadata = %v", err)
	}
	shared, _ := profile.ProfileForMode("shared_provider")
	if got := manager.sharedProviderState(shared); got != "lease_invalid" {
		t.Fatalf("mismatched lease state = %v, want lease_invalid", got)
	}
}

func TestAuthenticationModeManager_PersistsAcrossRuntimeRestart(t *testing.T) {
	fs := testutil.NewMockFileSystem()
	clock := testutil.NewMockClock(time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))
	profile := testAuthenticationProfile()
	manager, err := newAuthenticationModeManager(fs, clock, "/app", profile)
	if err != nil {
		t.Fatalf("newAuthenticationModeManager() error = %v", err)
	}
	if _, err := manager.selectMode(context.Background(), "remote_vrooli"); err != nil {
		t.Fatalf("selectMode(remote_vrooli) error = %v", err)
	}

	restarted, err := newAuthenticationModeManager(fs, clock, "/app", profile)
	if err != nil {
		t.Fatalf("restart manager error = %v", err)
	}
	status := restarted.status()
	if status["mode"] != "remote_vrooli" || status["state"] != "provider_required" {
		t.Fatalf("restarted status = %#v", status)
	}
}

func TestSupervisorAuthenticationServiceFollowsSelectedMode(t *testing.T) {
	fs := testutil.NewMockFileSystem()
	clock := testutil.NewMockClock(time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))
	profile := testAuthenticationProfile()
	manager, err := newAuthenticationModeManager(fs, clock, "/app", profile)
	if err != nil {
		t.Fatalf("newAuthenticationModeManager() error = %v", err)
	}

	supervisor := &Supervisor{
		authModes: manager,
		opts: Options{Manifest: &manifest.Manifest{
			Authentication: profile,
			Services: []manifest.Service{
				{ID: "api"},
				{ID: "scenario-authenticator"},
			},
		}},
	}

	if skip, reason := supervisor.shouldSkipAuthenticationService(manifest.Service{ID: "scenario-authenticator"}); !skip || !strings.Contains(reason, "personal_local") {
		t.Fatalf("personal_local authenticator decision = skip %v, reason %q; want skipped", skip, reason)
	}
	if skip, reason := supervisor.shouldSkipAuthenticationService(manifest.Service{ID: "api"}); skip || reason != "" {
		t.Fatalf("application service decision = skip %v, reason %q; want allowed", skip, reason)
	}

	if _, err := manager.selectMode(context.Background(), "local_multi_user"); err != nil {
		t.Fatalf("selectMode(local_multi_user) error = %v", err)
	}
	if skip, reason := supervisor.shouldSkipAuthenticationService(manifest.Service{ID: "scenario-authenticator"}); skip || reason != "" {
		t.Fatalf("local_multi_user authenticator decision = skip %v, reason %q; want allowed", skip, reason)
	}

	if _, err := manager.selectMode(context.Background(), "remote_vrooli"); err != nil {
		t.Fatalf("selectMode(remote_vrooli) error = %v", err)
	}
	if skip, reason := supervisor.shouldSkipAuthenticationService(manifest.Service{ID: "scenario-authenticator"}); !skip || !strings.Contains(reason, "remote_vrooli") {
		t.Fatalf("remote_vrooli authenticator decision = skip %v, reason %q; want skipped", skip, reason)
	}
}
