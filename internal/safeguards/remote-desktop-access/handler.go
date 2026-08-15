// Package remotedesktopaccess declares remote-desktop intent and reports what
// the host can honestly provide. It never forces a display-manager policy.
package remotedesktopaccess

import (
	"context"
	"fmt"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	experienceDirect  = "direct-desktop"
	experienceLogin   = "login-screen"
	experienceObserve = "observe-only"
	providerAuto      = "auto"
)

type providerObservation struct {
	present   bool
	active    bool
	available bool
	reason    string
}

var (
	collectFactsFn = func() hostinventory.Snapshot {
		return hostinventory.CollectPlatformFacts(context.Background())
	}
	runFn     = hostreqkit.RunPrivilegedCommand
	runUserFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return platform.RunAsInvokingUserInSession(context.Background(), name, args, platform.IdentityCommandOptions{
			Stdout: opts.Stdout,
			Stderr: opts.Stderr,
		})
	}
)

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}
func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	experience := configValue(requirement.Config, "experience", experienceObserve)
	provider := configValue(requirement.Config, "provider", providerAuto)
	if !validExperience(experience) || !validProvider(provider) {
		return hostreqkit.InvalidConfigStatus(requirement, fmt.Sprintf("remote_desktop_access has invalid experience/provider values %q/%q", experience, provider))
	}
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if !supportedOS(host.OS) {
		return hostreqkit.UnsupportedRequirementStatus(requirement, "remote desktop capability is declared only for Linux, macOS and Windows")
	}

	facts := collectFactsFn()
	selected, observed := selectProvider(host.OS, provider, experience, facts)
	observedProvider := facts.RemoteDesktop.SelectedProvider
	status.SelectedProvider = selected
	status.ObservedProvider = observedProvider
	status.ObservedMode = facts.RemoteDesktop.Mode
	status.ObservedLive = facts.RemoteDesktop.Observed
	status.ObservedActive = facts.RemoteDesktop.Active
	status.CredentialStoreState = facts.RemoteDesktop.CredentialStore.State
	status.Notes = append(status.Notes, "observed remote-desktop provider: "+observedProvider)
	status.Notes = append(status.Notes, "selected remote-desktop provider: "+selected)
	status.Notes = append(status.Notes, "observed remote-desktop mode: "+facts.RemoteDesktop.Mode)
	status.Notes = append(status.Notes, "observed experience: "+observedExperience(observedProvider, facts))
	if experience == experienceObserve {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "observe-only: no host state will be changed")
		return status
	}
	// Keep the explicit Wayland policy diagnostic ahead of the generic
	// availability check. A headless GNOME provider may be detected but still
	// be unusable when the display manager explicitly disables Wayland.
	if experience == experienceDirect && host.OS == "linux" && selected == "gnome-headless" && !facts.Wayland.Attainable {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "direct-desktop is unsupported: "+facts.Wayland.Reason)
		return status
	}
	if selected == "" || (!observed.present && !observed.available) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "no declared remote-desktop provider can deliver "+experience)
		return status
	}
	if !providerDelivers(selected, experience, facts) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, fmt.Sprintf("provider %q is detection-only and cannot deliver %s", selected, experience))
		return status
	}
	if !observed.present {
		status.Notes = append(status.Notes, fmt.Sprintf("provider %q is available for %s but is not currently active", selected, experience))
		return status
	}
	if experience == experienceLogin && selected == "gnome-system" && observed.active {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "system-mode GNOME Remote Desktop is enabled and active")
		return status
	}
	if experience == experienceDirect && observed.active && selected != "gnome-system" {
		if selected != "gnome-user-shared" || status.CredentialStoreState == "ready" {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			status.Notes = append(status.Notes, fmt.Sprintf("provider %q is active and delivers direct-desktop", selected))
			return status
		}
		status.Notes = append(status.Notes, "user-shared provider is active but credential-store readiness is "+status.CredentialStoreState)
	}
	status.Notes = append(status.Notes, fmt.Sprintf("provider %q can deliver %s but is not currently active", selected, experience))
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.ExecutionState != hostreqkit.ExecutionPending {
		return status, nil
	}
	permission := func(key, command string) bool {
		allowed, _ := status.Config[key].(bool)
		if allowed {
			return true
		}
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingManual
		status.Command = command
		status.Notes = append(status.Notes, fmt.Sprintf("permission %q is false; run %s manually or opt in to this host change", key, command))
		return false
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	switchingFromSystem := host.OS == "linux" && status.SelectedProvider == "gnome-user-shared" && status.ObservedMode == "system"
	if (status.ObservedActive || switchingFromSystem) && !opts.MaintenanceWindow {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingNeedsMaintenanceWindow
		status.Notes = append(status.Notes, "an active remote-desktop session may be interrupted; rerun with --maintenance-window")
		return status, nil
	}
	// Evaluate every permission and host-state gate before the dry-run branch.
	// This keeps a preview from claiming an apply that a real run would refuse.
	if host.OS == "linux" && status.SelectedProvider == "gnome-system" {
		if !permission("allow_enable_system", "sudo systemctl enable --now gnome-remote-desktop.service") {
			return status, nil
		}
	} else if host.OS == "linux" && status.SelectedProvider == "gnome-user-shared" {
		if switchingFromSystem && !permission("allow_disable_system_unit", "sudo systemctl disable --now gnome-remote-desktop.service") {
			return status, nil
		}
		if !permission("allow_switch_provider", "systemctl --user enable --now gnome-remote-desktop.service") {
			return status, nil
		}
		if !permission("allow_enable_user_unit", "systemctl --user enable --now gnome-remote-desktop.service") {
			return status, nil
		}
		if status.CredentialStoreState != "ready" {
			if blocked := credentialStoreBlock(&status); blocked {
				return status, nil
			}
			if !permission("allow_provision_credentials", credentialCommand()) {
				return status, nil
			}
			status.ExecutionState = hostreqkit.ExecutionManualActionRequired
			status.BlockingReason = hostreqkit.BlockingManual
			status.Command = credentialCommand()
			status.Notes = append(status.Notes, "credential store is empty; enter credentials with "+credentialCommand())
			return status, nil
		}
	} else if host.OS == "windows" && status.SelectedProvider == "windows-termservice" {
		if !permission("allow_enable_native_provider", "sc.exe config TermService start= auto && sc.exe start TermService") {
			return status, nil
		}
	} else if (host.OS == "darwin" || host.OS == "macos") && status.SelectedProvider == "macos-screen-sharing" {
		if !permission("allow_enable_native_provider", "sudo launchctl enable system/com.apple.screensharing && sudo launchctl kickstart -k system/com.apple.screensharing") {
			return status, nil
		}
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would enable the declared remote-desktop provider")
		return status, nil
	}
	if host.OS == "linux" && status.SelectedProvider == "gnome-system" {
		if err := runFn(opts.SudoMode, "systemctl", []string{"enable", "--now", "gnome-remote-desktop.service"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "enable gnome-remote-desktop.service failed: "+err.Error())
			return status, nil
		}
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionApplied
		status.Notes = append(status.Notes, "system-mode GNOME Remote Desktop enabled")
		return status, nil
	}
	if host.OS == "linux" && status.SelectedProvider == "gnome-user-shared" {
		if switchingFromSystem {
			if err := runFn(opts.SudoMode, "systemctl", []string{"disable", "--now", "gnome-remote-desktop.service"}, opts); err != nil {
				status.ExecutionState = hostreqkit.ExecutionFailed
				status.Notes = append(status.Notes, "disable system-mode GNOME Remote Desktop failed: "+err.Error())
				return status, nil
			}
		}
		if err := runUserFn("systemctl", []string{"--user", "enable", "--now", "gnome-remote-desktop.service"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "enable user-mode GNOME Remote Desktop failed: "+err.Error())
			return status, nil
		}
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionApplied
		status.Notes = append(status.Notes, "user-shared GNOME Remote Desktop enabled")
		return status, nil
	}
	if host.OS == "windows" && status.SelectedProvider == "windows-termservice" {
		if err := runFn(opts.SudoMode, "sc.exe", []string{"config", "TermService", "start=", "auto"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "configure Windows TermService failed: "+err.Error())
			return status, nil
		}
		if err := runFn(opts.SudoMode, "sc.exe", []string{"start", "TermService"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "start Windows TermService failed: "+err.Error())
			return status, nil
		}
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionApplied
		status.Notes = append(status.Notes, "Windows TermService enabled")
		return status, nil
	}
	if host.OS == "darwin" || host.OS == "macos" {
		if err := runFn(opts.SudoMode, "launchctl", []string{"enable", "system/com.apple.screensharing"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "enable macOS Screen Sharing failed: "+err.Error())
			return status, nil
		}
		if err := runFn(opts.SudoMode, "launchctl", []string{"kickstart", "-k", "system/com.apple.screensharing"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "start macOS Screen Sharing failed: "+err.Error())
			return status, nil
		}
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionApplied
		status.Notes = append(status.Notes, "macOS Screen Sharing enabled")
		return status, nil
	}
	status.SupportClass = hostreqkit.SupportUnsupported
	status.ExecutionState = hostreqkit.ExecutionUnsupported
	status.Notes = append(status.Notes, "no safe apply operation is implemented for the selected provider")
	return status, nil
}

func supportedOS(osName string) bool {
	switch strings.ToLower(strings.TrimSpace(osName)) {
	case "linux", "darwin", "macos", "windows":
		return true
	default:
		return false
	}
}

func validExperience(value string) bool {
	return value == experienceDirect || value == experienceLogin || value == experienceObserve
}

func validProvider(value string) bool {
	switch value {
	case providerAuto, "gnome-system", "gnome-user-shared", "gnome-headless", "xrdp", "windows-termservice", "macos-screen-sharing":
		return true
	default:
		return false
	}
}

func configValue(config map[string]any, key, fallback string) string {
	if value, ok := config[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func selectProvider(osName, requested, experience string, facts hostinventory.Snapshot) (string, providerObservation) {
	if requested != providerAuto {
		return requested, observeProvider(requested, osName, facts)
	}
	ordered := []string{"gnome-system", "gnome-user-shared", "gnome-headless", "xrdp", "windows-termservice", "macos-screen-sharing"}
	for _, candidate := range ordered {
		observation := observeProvider(candidate, osName, facts)
		if (experience == experienceObserve && observation.present) ||
			(experience != experienceObserve && (observation.present || observation.available) && providerDelivers(candidate, experience, facts)) {
			return candidate, observation
		}
	}
	return "", providerObservation{reason: "no provider detected"}
}

func observeProvider(provider, osName string, facts hostinventory.Snapshot) providerObservation {
	osName = strings.ToLower(strings.TrimSpace(osName))
	observed, found := facts.RemoteDesktop.Provider(provider)
	if found {
		return providerObservation{present: observed.Present, active: observed.Active, available: observed.Present || providerAvailable(provider, osName, facts), reason: provider}
	}
	switch provider {
	case "gnome-system":
		if osName != "linux" || !facts.SupportsSystemd {
			return providerObservation{reason: "systemd is unavailable"}
		}
		return providerObservation{available: true, reason: "system-mode GNOME Remote Desktop"}
	case "gnome-headless":
		if osName != "linux" || !facts.SupportsSystemd || !facts.Wayland.Attainable {
			return providerObservation{reason: "GNOME headless provider is Linux-only"}
		}
		return providerObservation{available: true, reason: "GNOME headless provider"}
	case "gnome-user-shared":
		if osName != "linux" || !facts.SupportsSystemd || !facts.DisplayAttached {
			return providerObservation{reason: "GNOME user-shared provider needs Linux systemd and an attached display"}
		}
		return providerObservation{available: true, reason: "GNOME user-shared provider"}
	case "xrdp":
		if osName != "linux" {
			return providerObservation{reason: "xrdp is Linux-only"}
		}
		return providerObservation{reason: "xrdp provider"}
	case "windows-termservice":
		if osName != "windows" {
			return providerObservation{reason: "TermService is Windows-only"}
		}
		return providerObservation{available: true, reason: "Windows TermService"}
	case "macos-screen-sharing":
		if osName != "darwin" && osName != "macos" {
			return providerObservation{reason: "macOS Screen Sharing is macOS-only"}
		}
		return providerObservation{available: true, reason: "macOS Screen Sharing"}
	default:
		return providerObservation{reason: "unknown provider"}
	}
}

func providerAvailable(provider, osName string, facts hostinventory.Snapshot) bool {
	switch provider {
	case "gnome-system":
		return osName == "linux" && facts.SupportsSystemd
	case "gnome-user-shared":
		return osName == "linux" && facts.SupportsSystemd && facts.DisplayAttached
	case "gnome-headless":
		return osName == "linux" && facts.SupportsSystemd && facts.Wayland.Attainable
	case "windows-termservice":
		return osName == "windows"
	case "macos-screen-sharing":
		return osName == "darwin" || osName == "macos"
	default:
		return false
	}
}

func providerDelivers(provider, experience string, facts hostinventory.Snapshot) bool {
	switch provider {
	case "gnome-system":
		return experience == experienceLogin
	case "gnome-user-shared":
		return experience == experienceDirect && facts.DisplayAttached
	case "gnome-headless":
		return experience == experienceDirect && facts.Wayland.Attainable
	case "xrdp", "windows-termservice", "macos-screen-sharing":
		return experience == experienceDirect
	default:
		return false
	}
}

func observedExperience(provider string, facts hostinventory.Snapshot) string {
	switch provider {
	case "gnome-system":
		return experienceLogin
	case "gnome-user-shared":
		return experienceDirect
	case "gnome-headless":
		if facts.Wayland.Attainable {
			return experienceDirect
		}
	}
	return "unknown"
}

func credentialCommand() string {
	return "grdctl rdp set-credentials <username> <password>"
}

func credentialStoreBlock(status *hostreqkit.ItemStatus) bool {
	var reason hostreqkit.BlockingReason
	var remedy string
	switch status.CredentialStoreState {
	case "locked":
		reason = hostreqkit.BlockingCredentialStoreLocked
		remedy = "run `vrooli credentials keyring unlock`; if the login keyring is intentionally password-protected, opt in to login_keyring_unlock before retrying"
	case "unresponsive":
		reason = hostreqkit.BlockingCredentialStoreUnresponsive
		remedy = "run `vrooli credentials keyring status` and restore the session bus before retrying"
	case "unavailable", "unsupported":
		reason = hostreqkit.BlockingCredentialStoreUnavailable
		remedy = "make the active user's credential store available, then rerun `vrooli credentials keyring status`"
	default:
		return false
	}
	status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	status.BlockingReason = reason
	status.Command = ""
	status.Notes = append(status.Notes, "credential store state is "+status.CredentialStoreState+"; "+remedy)
	return true
}
