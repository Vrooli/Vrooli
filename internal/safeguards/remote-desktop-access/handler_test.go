package remotedesktopaccess

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func stubRemoteDesktop(t *testing.T, snapshot hostinventory.Snapshot) {
	t.Helper()
	origFacts, origRun, origRunUser, origResolve, origReady := collectFactsFn, runFn, runUserFn, resolveCredentialFn, credentialsReadyFn
	collectFactsFn = func() hostinventory.Snapshot { return snapshot }
	runFn = func(string, string, []string, hostreqkit.EnsureOptions) error { return nil }
	runUserFn = func(string, []string, hostreqkit.EnsureOptions) error { return nil }
	resolveCredentialFn = func(_, field string) (string, error) {
		if field == "username" {
			return "rdp-user", nil
		}
		return "rdp-password", nil
	}
	credentialsReadyFn = func() bool { return true }
	t.Cleanup(func() {
		collectFactsFn, runFn, runUserFn, resolveCredentialFn, credentialsReadyFn = origFacts, origRun, origRunUser, origResolve, origReady
	})
}

func remoteRequest(config map[string]any) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "remote_desktop_access", Kind: hostreqspec.KindSafeguard, Required: true, Config: config}
}

func TestObserveOnlyIsAlreadyPresentAndNeverApplies(t *testing.T) {
	runCalled := false
	stubRemoteDesktop(t, hostinventory.Snapshot{OS: "linux"})
	runFn = func(string, string, []string, hostreqkit.EnsureOptions) error { runCalled = true; return nil }
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(map[string]any{"experience": experienceObserve, "provider": providerAuto}))
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent || !status.Applied {
		t.Fatalf("observe status = %#v", status)
	}
	if _, err := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{}); err != nil {
		t.Fatal(err)
	}
	if runCalled {
		t.Fatal("observe-only called an apply command")
	}
}

func TestApplyRequiresExplicitSystemPermission(t *testing.T) {
	status := hostreqkit.ItemStatus{
		ExecutionState:   hostreqkit.ExecutionPending,
		SelectedProvider: "gnome-system",
		Config:           map[string]any{"allow_enable_system": false},
	}
	got, err := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutionState != hostreqkit.ExecutionManualActionRequired || got.BlockingReason != hostreqkit.BlockingManual {
		t.Fatalf("permission-gated apply = %#v", got)
	}
	if got.Command != setupCommand() {
		t.Fatalf("manual command = %q", got.Command)
	}
}

func TestSystemProviderResolvesAndProvisionsAuthorityCredentials(t *testing.T) {
	var calls []string
	stubRemoteDesktop(t, hostinventory.Snapshot{
		OS:              "linux",
		SupportsSystemd: true,
		RemoteDesktop: hostinventory.RemoteDesktopCapability{
			Providers: []hostinventory.RemoteDesktopProvider{{Name: "gnome-system", Present: false}},
		},
	})
	runFn = func(_ string, command string, args []string, _ hostreqkit.EnsureOptions) error {
		calls = append(calls, command+" "+strings.Join(args, " "))
		return nil
	}
	config := map[string]any{
		"experience":                  experienceDirect,
		"provider":                    "gnome-system",
		"allow_enable_system":         true,
		"allow_provision_credentials": true,
	}
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(config))
	status, err := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{MaintenanceWindow: true})
	if err != nil || status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("system credential apply = %#v, err=%v", status, err)
	}
	want := []string{
		"grdctl --system rdp set-credentials rdp-user rdp-password",
		"systemctl enable --now gnome-remote-desktop.service",
	}
	if strings.Join(calls, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestActiveSystemProviderOnlyRefreshesCredentials(t *testing.T) {
	var calls []string
	stubRemoteDesktop(t, hostinventory.Snapshot{
		OS:              "linux",
		SupportsSystemd: true,
		RemoteDesktop: hostinventory.RemoteDesktopCapability{
			Mode:     "system",
			Observed: true,
			Active:   true,
			Providers: []hostinventory.RemoteDesktopProvider{{
				Name: "gnome-system", Present: true, Active: true,
			}},
		},
	})
	runFn = func(_ string, command string, args []string, _ hostreqkit.EnsureOptions) error {
		calls = append(calls, command+" "+strings.Join(args, " "))
		return nil
	}
	config := map[string]any{
		"experience":                  experienceDirect,
		"provider":                    "gnome-system",
		"allow_provision_credentials": true,
	}
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(config))
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("active system status = %#v", status)
	}
	// Simulate a refresh request after the authority values change: an active
	// unit must not be needlessly toggled, but the system credential store still
	// receives the current authority values.
	status.Applied = false
	status.ExecutionState = hostreqkit.ExecutionPending
	status.ObservedActive = true
	status, err := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{MaintenanceWindow: true})
	if err != nil || status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("active system refresh = %#v, err=%v", status, err)
	}
	if len(calls) != 1 || calls[0] != "grdctl --system rdp set-credentials rdp-user rdp-password" {
		t.Fatalf("calls = %v, want credential refresh only", calls)
	}
}

func TestLoginScreenUsesActiveSystemProvider(t *testing.T) {
	stubRemoteDesktop(t, hostinventory.Snapshot{
		OS:              "linux",
		SupportsSystemd: true,
		RemoteDesktop: hostinventory.RemoteDesktopCapability{
			Supported:        true,
			Observed:         true,
			SelectedProvider: "gnome-system",
			Providers:        []hostinventory.RemoteDesktopProvider{{Name: "gnome-system", Present: true, Active: true}},
		},
	})
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(map[string]any{"experience": experienceLogin, "provider": providerAuto}))
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent || !status.Applied {
		t.Fatalf("login-screen status = %#v", status)
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "system-mode GNOME Remote Desktop") {
		t.Fatalf("login-screen notes = %v", status.Notes)
	}
}

func TestDirectDesktopNamesWaylandPolicyBlock(t *testing.T) {
	stubRemoteDesktop(t, hostinventory.Snapshot{OS: "linux", SupportsSystemd: true, Wayland: hostinventory.WaylandCapability{Reason: "display manager explicitly disables Wayland"}, RemoteDesktop: hostinventory.RemoteDesktopCapability{Supported: true, Observed: true, SelectedProvider: "gnome-headless", Providers: []hostinventory.RemoteDesktopProvider{{Name: "gnome-headless", Present: true, Active: true}}}})
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(map[string]any{"experience": experienceDirect, "provider": "gnome-headless"}))
	if status.SupportClass != hostreqkit.SupportUnsupported || status.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("direct-desktop status = %#v", status)
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "explicitly disables Wayland") {
		t.Fatalf("direct-desktop notes = %v", status.Notes)
	}
}

func TestDirectDesktopSeparatesObservedAndSelectedProviders(t *testing.T) {
	stubRemoteDesktop(t, hostinventory.Snapshot{
		OS:              "linux",
		SupportsSystemd: true,
		DisplayAttached: true,
		RemoteDesktop: hostinventory.RemoteDesktopCapability{
			Supported:        true,
			Observed:         true,
			Active:           true,
			Mode:             "system",
			SelectedProvider: "gnome-system",
			Providers: []hostinventory.RemoteDesktopProvider{{
				Name:    "gnome-system",
				Present: true,
				Active:  true,
			}},
		},
	})
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(
		hostreqkit.Host{OS: "linux"},
		remoteRequest(map[string]any{"experience": experienceDirect, "provider": providerAuto}),
	)
	if status.ObservedProvider != "gnome-system" || status.SelectedProvider != "gnome-system" {
		t.Fatalf("provider roles = observed %q selected %q, want gnome-system/gnome-system", status.ObservedProvider, status.SelectedProvider)
	}
	if !status.ObservedLive || !status.ObservedActive || status.ObservedMode != "system" {
		t.Fatalf("observed state = live %t active %t mode %q, want true/true/system", status.ObservedLive, status.ObservedActive, status.ObservedMode)
	}
	joined := strings.Join(status.Notes, " ")
	if !strings.Contains(joined, "observed experience: login-screen") || !strings.Contains(joined, "selected remote-desktop provider: gnome-system") {
		t.Fatalf("notes do not distinguish observed target: %s", joined)
	}
}

func TestDirectDesktopAcceptsSystemProvider(t *testing.T) {
	stubRemoteDesktop(t, hostinventory.Snapshot{OS: "linux", SupportsSystemd: true, RemoteDesktop: hostinventory.RemoteDesktopCapability{Mode: "system", Providers: []hostinventory.RemoteDesktopProvider{{Name: "gnome-system", Present: true, Active: true}}}})
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(map[string]any{"experience": experienceDirect, "provider": "gnome-system"}))
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent || !status.Applied {
		t.Fatalf("system provider status = %#v", status)
	}
}

func TestNativeDirectDesktopDoesNotUseLinuxWaylandGate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		os       string
		provider string
	}{
		{name: "windows", os: "windows", provider: "windows-termservice"},
		{name: "macos", os: "darwin", provider: "macos-screen-sharing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stubRemoteDesktop(t, hostinventory.Snapshot{OS: tc.os, Wayland: hostinventory.WaylandCapability{Reason: "not a native platform fact"}, RemoteDesktop: hostinventory.RemoteDesktopCapability{Providers: []hostinventory.RemoteDesktopProvider{{Name: tc.provider, Present: true, Active: true}}}})
			status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: tc.os}, remoteRequest(map[string]any{"experience": experienceDirect, "provider": providerAuto}))
			if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent || !status.Applied || status.SelectedProvider != tc.provider {
				t.Fatalf("native direct status = %#v", status)
			}
		})
	}
}

func TestDeclaredProvidersReturnTypedUnsupportedInsteadOfPanicking(t *testing.T) {
	providers := []string{"gnome-system", "gnome-headless", "gnome-user-shared", "xrdp", "windows-termservice", "macos-screen-sharing"}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			stubRemoteDesktop(t, hostinventory.Snapshot{OS: "linux"})
			status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(map[string]any{"experience": experienceLogin, "provider": provider}))
			if status.ExecutionState != hostreqkit.ExecutionUnsupported || status.SupportClass != hostreqkit.SupportUnsupported {
				t.Fatalf("provider status = %#v", status)
			}
		})
	}
}

func TestUserSharedSwitchDisablesSystemBeforeEnablingUserUnit(t *testing.T) {
	var calls []string
	stubRemoteDesktop(t, hostinventory.Snapshot{OS: "linux", SupportsSystemd: true, DisplayAttached: true, RemoteDesktop: hostinventory.RemoteDesktopCapability{Mode: "system", Providers: []hostinventory.RemoteDesktopProvider{{Name: "gnome-user-shared", Present: false}}, CredentialStore: hostinventory.CredentialStoreCapability{State: "ready"}}})
	runFn = func(_ string, command string, args []string, _ hostreqkit.EnsureOptions) error {
		calls = append(calls, command+" "+strings.Join(args, " "))
		return nil
	}
	runUserFn = func(command string, args []string, _ hostreqkit.EnsureOptions) error {
		calls = append(calls, command+" "+strings.Join(args, " "))
		return nil
	}
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(map[string]any{"experience": experienceDirect, "provider": "gnome-user-shared", "allow_switch_provider": true, "allow_enable_user_unit": true, "allow_disable_system_unit": true}))
	status, err := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{MaintenanceWindow: true})
	if err != nil || status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("switch status = %#v, err=%v", status, err)
	}
	want := []string{"systemctl disable --now gnome-remote-desktop.service", "systemctl --user enable --now gnome-remote-desktop.service"}
	if strings.Join(calls, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestUserSharedSwitchDoesNotDisableSystemWhenCredentialsAreUnready(t *testing.T) {
	var calls []string
	stubRemoteDesktop(t, hostinventory.Snapshot{OS: "linux", SupportsSystemd: true, DisplayAttached: true, RemoteDesktop: hostinventory.RemoteDesktopCapability{Mode: "system", Providers: []hostinventory.RemoteDesktopProvider{{Name: "gnome-user-shared", Present: false}}, CredentialStore: hostinventory.CredentialStoreCapability{State: "unresponsive"}}})
	runFn = func(_ string, command string, args []string, _ hostreqkit.EnsureOptions) error {
		calls = append(calls, command+" "+strings.Join(args, " "))
		return nil
	}
	runUserFn = func(command string, args []string, _ hostreqkit.EnsureOptions) error {
		calls = append(calls, command+" "+strings.Join(args, " "))
		return nil
	}
	config := map[string]any{"experience": experienceDirect, "provider": "gnome-user-shared", "allow_switch_provider": true, "allow_enable_user_unit": true, "allow_disable_system_unit": true, "allow_provision_credentials": true}
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(config))
	status, err := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{MaintenanceWindow: true})
	if err != nil || status.ExecutionState != hostreqkit.ExecutionManualActionRequired || status.BlockingReason != hostreqkit.BlockingCredentialStoreUnresponsive || status.Command != "" {
		t.Fatalf("credential gate status = %#v, err=%v", status, err)
	}
	if len(calls) != 0 {
		t.Fatalf("credential gate mutated provider state: %v", calls)
	}
}

func TestUserSharedCredentialStoreStatesUseStateSpecificRemedies(t *testing.T) {
	for _, tc := range []struct {
		state   string
		reason  hostreqkit.BlockingReason
		remedy  string
		command string
	}{
		{state: "locked", reason: hostreqkit.BlockingCredentialStoreLocked, remedy: "credentials keyring unlock"},
		{state: "unresponsive", reason: hostreqkit.BlockingCredentialStoreUnresponsive, remedy: "credentials keyring status"},
		{state: "unavailable", reason: hostreqkit.BlockingCredentialStoreUnavailable, remedy: "credentials keyring status"},
		{state: "unsupported", reason: hostreqkit.BlockingCredentialStoreUnavailable, remedy: "credentials keyring status"},
		{state: "empty", reason: hostreqkit.BlockingManual, remedy: "vrooli credentials provision", command: credentialCommand()},
	} {
		t.Run(tc.state, func(t *testing.T) {
			stubRemoteDesktop(t, hostinventory.Snapshot{OS: "linux", SupportsSystemd: true, DisplayAttached: true, RemoteDesktop: hostinventory.RemoteDesktopCapability{Mode: "system", Providers: []hostinventory.RemoteDesktopProvider{{Name: "gnome-user-shared", Present: false}}, CredentialStore: hostinventory.CredentialStoreCapability{State: tc.state}}})
			config := map[string]any{"experience": experienceDirect, "provider": "gnome-user-shared", "allow_switch_provider": true, "allow_enable_user_unit": true, "allow_disable_system_unit": true, "allow_provision_credentials": true}
			status := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Inspect(hostreqkit.Host{OS: "linux"}, remoteRequest(config))
			status, err := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{MaintenanceWindow: true})
			if err != nil || status.BlockingReason != tc.reason {
				t.Fatalf("state=%s status=%#v err=%v", tc.state, status, err)
			}
			if !strings.Contains(strings.Join(status.Notes, " "), tc.remedy) {
				t.Fatalf("state=%s notes=%v want remedy containing %q", tc.state, status.Notes, tc.remedy)
			}
			if status.Command != tc.command {
				t.Fatalf("state=%s command=%q want %q", tc.state, status.Command, tc.command)
			}
		})
	}
}

func TestDryRunMatchesMaintenanceWindowGate(t *testing.T) {
	stubRemoteDesktop(t, hostinventory.Snapshot{OS: "linux"})
	status := hostreqkit.ItemStatus{
		ExecutionState:   hostreqkit.ExecutionPending,
		SelectedProvider: "gnome-user-shared",
		ObservedMode:     "system",
		ObservedActive:   true,
		Config: map[string]any{
			"allow_switch_provider":       true,
			"allow_enable_user_unit":      true,
			"allow_disable_system_unit":   true,
			"allow_provision_credentials": true,
		},
	}
	comparison, err := hostreqkit.CompareDryRunAndApply(
		NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}),
		hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.DryRun.ExecutionState == hostreqkit.ExecutionWouldApply {
		t.Fatalf("blocked dry-run reported would_apply: %#v", comparison.DryRun)
	}
	if comparison.DryRun.BlockingReason != hostreqkit.BlockingNeedsMaintenanceWindow || comparison.Apply.BlockingReason != hostreqkit.BlockingNeedsMaintenanceWindow {
		t.Fatalf("gate outcomes = dry %#v apply %#v", comparison.DryRun, comparison.Apply)
	}
}

func TestNativeProvidersArePermissionGatedAndApplied(t *testing.T) {
	for _, tc := range []struct {
		name string
		os   string
		want []string
	}{
		{name: "windows", os: "windows", want: []string{"sc.exe config TermService start= auto", "sc.exe start TermService"}},
		{name: "macos", os: "darwin", want: []string{"launchctl enable system/com.apple.screensharing", "launchctl kickstart -k system/com.apple.screensharing"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			stubRemoteDesktop(t, hostinventory.Snapshot{OS: tc.os})
			runFn = func(_ string, command string, args []string, _ hostreqkit.EnsureOptions) error {
				calls = append(calls, command+" "+strings.Join(args, " "))
				return nil
			}
			selected := "windows-termservice"
			if tc.os == "darwin" {
				selected = "macos-screen-sharing"
			}
			status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending, SelectedProvider: selected, Config: map[string]any{"allow_enable_native_provider": true}}
			got, err := NewHandler(hostreqkit.SafeguardManifest{Name: "remote_desktop_access"}).Apply(hostreqkit.Host{OS: tc.os}, status, hostreqkit.EnsureOptions{})
			if err != nil || got.ExecutionState != hostreqkit.ExecutionApplied || strings.Join(calls, "\x00") != strings.Join(tc.want, "\x00") {
				t.Fatalf("native apply = %#v, calls=%v, err=%v", got, calls, err)
			}
		})
	}
}
