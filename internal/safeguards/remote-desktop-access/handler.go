// Package remotedesktopaccess declares remote-desktop intent and reports what

// the host can honestly provide. It never forces a display-manager policy.
package remotedesktopaccess

import (
	"context"
	"fmt"
	"io"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	handlerGnomeHeadless      = "gnome-headless"
	handlerWindowsTermservice = "windows-termservice"
)

const (
	handlerGnomeSystem        = "gnome-system"
	handlerGnomeUserShared    = "gnome-user-shared"
	handlerMacosScreenSharing = "macos-screen-sharing"
	handlerXrdp               = "xrdp"
)

const (
	experienceDirect  = "direct-desktop"
	experienceLogin   = "login-screen"
	experienceObserve = "observe-only"
	providerAuto      = "auto"
	remoteDesktopID   = "vrooli/remote-desktop"
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
	resolveCredentialFn = resolveRemoteDesktopCredential
	credentialsReadyFn  = remoteDesktopCredentialsReady
)

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}
func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

//nolint:gocyclo // remote desktop inspection is a provider, display, user, and capability state matrix.
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
	if experience == experienceDirect && hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformLinux && selected == handlerGnomeHeadless && !facts.Wayland.Attainable {
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
	if experience == experienceLogin && selected == handlerGnomeSystem && observed.active {
		if !credentialsReadyFn() {
			status.Notes = append(status.Notes, "system-mode GNOME Remote Desktop is active but its credentials are not configured in Vrooli's encrypted authority")
			return status
		}
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "system-mode GNOME Remote Desktop is enabled and active")
		return status
	}
	if experience == experienceDirect && selected == handlerGnomeSystem && observed.active {
		if !credentialsReadyFn() {
			status.Notes = append(status.Notes, "system-mode GNOME Remote Desktop is active but its credentials are not configured in Vrooli's encrypted authority")
			return status
		}
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "system-mode GNOME Remote Desktop is active and delivers direct-desktop through the autologin session")
		return status
	}
	if experience == experienceDirect && observed.active && selected != handlerGnomeSystem {
		if selected != handlerGnomeUserShared || status.CredentialStoreState == "ready" {
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
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	switchingFromSystem := hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformLinux && status.SelectedProvider == handlerGnomeUserShared && status.ObservedMode == "system"
	if (status.ObservedActive || switchingFromSystem) && !opts.MaintenanceWindow {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingNeedsMaintenanceWindow
		status.Notes = append(status.Notes, "an active remote-desktop session may be interrupted; rerun with --maintenance-window")
		return status, nil
	}
	plan, ready := prepareRemoteDesktopApply(host, &status, switchingFromSystem)
	if !ready {
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would enable the declared remote-desktop provider")
		return status, nil
	}
	return executeRemoteDesktopApply(host, status, opts, plan)
}

type remoteDesktopApplyPlan struct {
	switchingFromSystem bool
	systemUsername      string
	systemPassword      string
}

func requirePermission(status *hostreqkit.ItemStatus, key, command string) bool {
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

func prepareRemoteDesktopApply(host hostreqkit.Host, status *hostreqkit.ItemStatus, switchingFromSystem bool) (remoteDesktopApplyPlan, bool) {
	plan := remoteDesktopApplyPlan{switchingFromSystem: switchingFromSystem}
	platform := hostreqspec.PlatformFromGOOS(host.OS)
	switch {
	case platform == hostreqspec.PlatformLinux && status.SelectedProvider == handlerGnomeSystem:
		return prepareSystemProvider(status, plan)
	case platform == hostreqspec.PlatformLinux && status.SelectedProvider == handlerGnomeUserShared:
		return plan, prepareUserSharedProvider(status, switchingFromSystem)
	case platform == hostreqspec.PlatformWindows && status.SelectedProvider == handlerWindowsTermservice:
		return plan, requirePermission(status, "allow_enable_native_provider", setupCommand())
	case platform == hostreqspec.PlatformMacOS && status.SelectedProvider == handlerMacosScreenSharing:
		return plan, requirePermission(status, "allow_enable_native_provider", setupCommand())
	default:
		return plan, true
	}
}

func prepareSystemProvider(status *hostreqkit.ItemStatus, plan remoteDesktopApplyPlan) (remoteDesktopApplyPlan, bool) {
	if !status.ObservedActive && !requirePermission(status, "allow_enable_system", setupCommand()) {
		return plan, false
	}
	if !requirePermission(status, "allow_provision_credentials", credentialProvisionCommand("username")) {
		return plan, false
	}
	var err error
	if plan.systemUsername, err = resolveCredentialFn(remoteDesktopID, "username"); err != nil {
		*status = credentialProvisionRequired(status, "username")
		return plan, false
	}
	if plan.systemPassword, err = resolveCredentialFn(remoteDesktopID, "password"); err != nil {
		*status = credentialProvisionRequired(status, "password")
		return plan, false
	}
	return plan, true
}

func prepareUserSharedProvider(status *hostreqkit.ItemStatus, switchingFromSystem bool) bool {
	if switchingFromSystem && !requirePermission(status, "allow_disable_system_unit", setupCommand()) {
		return false
	}
	if !requirePermission(status, "allow_switch_provider", setupCommand()) || !requirePermission(status, "allow_enable_user_unit", setupCommand()) {
		return false
	}
	if status.CredentialStoreState == "ready" {
		return true
	}
	if credentialStoreBlock(status) || !requirePermission(status, "allow_provision_credentials", credentialCommand()) {
		return false
	}
	status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	status.BlockingReason = hostreqkit.BlockingManual
	status.Command = credentialCommand()
	status.Notes = append(status.Notes, "credential store is empty; enter credentials with "+credentialCommand())
	return false
}

func executeRemoteDesktopApply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions, plan remoteDesktopApplyPlan) (hostreqkit.ItemStatus, error) {
	platform := hostreqspec.PlatformFromGOOS(host.OS)
	switch {
	case platform == hostreqspec.PlatformLinux && status.SelectedProvider == handlerGnomeSystem:
		return applySystemProvider(status, opts, plan)
	case platform == hostreqspec.PlatformLinux && status.SelectedProvider == handlerGnomeUserShared:
		return applyUserSharedProvider(status, opts, plan.switchingFromSystem)
	case platform == hostreqspec.PlatformWindows && status.SelectedProvider == handlerWindowsTermservice:
		return applyWindowsProvider(status, opts)
	case platform == hostreqspec.PlatformMacOS:
		return applyMacOSProvider(status, opts)
	default:
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "no safe apply operation is implemented for the selected provider")
		return status, nil
	}
}

func applySystemProvider(status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions, plan remoteDesktopApplyPlan) (hostreqkit.ItemStatus, error) {
	provisionOpts := opts
	provisionOpts.Stdout = io.Discard
	provisionOpts.Stderr = io.Discard
	if err := runFn(opts.SudoMode, "grdctl", []string{"--system", "rdp", "set-credentials", plan.systemUsername, plan.systemPassword}, provisionOpts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "system-mode GNOME Remote Desktop credential provisioning failed; rerun the Vrooli credential doctor")
		return status, nil
	}
	if !status.ObservedActive {
		if err := runFn(opts.SudoMode, "systemctl", []string{"enable", "--now", "gnome-remote-desktop.service"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "enable gnome-remote-desktop.service failed: "+err.Error())
			return status, nil
		}
		status.Notes = append(status.Notes, "system-mode GNOME Remote Desktop enabled")
	} else {
		status.Notes = append(status.Notes, "system-mode GNOME Remote Desktop credentials refreshed")
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	return status, nil
}

func applyUserSharedProvider(status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions, switchingFromSystem bool) (hostreqkit.ItemStatus, error) {
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

func applyWindowsProvider(status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
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

func applyMacOSProvider(status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
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

func supportedOS(osName string) bool {
	switch hostreqspec.PlatformFromGOOS(osName) {
	case hostreqspec.PlatformLinux, hostreqspec.PlatformMacOS, hostreqspec.PlatformWindows:
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
	case providerAuto, handlerGnomeSystem, handlerGnomeUserShared, handlerGnomeHeadless, handlerXrdp, handlerWindowsTermservice, handlerMacosScreenSharing:
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
	ordered := []string{handlerGnomeSystem, handlerGnomeUserShared, handlerGnomeHeadless, handlerXrdp, handlerWindowsTermservice, handlerMacosScreenSharing}
	for _, candidate := range ordered {
		observation := observeProvider(candidate, osName, facts)
		if (experience == experienceObserve && observation.present) ||
			(experience != experienceObserve && (observation.present || observation.available) && providerDelivers(candidate, experience, facts)) {
			return candidate, observation
		}
	}
	return "", providerObservation{reason: "no provider detected"}
}

//nolint:gocyclo // provider observation maps platform, display, session, and evidence combinations.
func observeProvider(provider, osName string, facts hostinventory.Snapshot) providerObservation {
	osName = strings.ToLower(strings.TrimSpace(osName))
	hostOS := hostreqspec.PlatformFromGOOS(osName)
	observed, found := facts.RemoteDesktop.Provider(provider)
	if found {
		return providerObservation{present: observed.Present, active: observed.Active, available: observed.Present || providerAvailable(provider, osName, facts), reason: provider}
	}
	switch provider {
	case handlerGnomeSystem:
		if hostOS != hostreqspec.PlatformLinux || !facts.SupportsSystemd {
			return providerObservation{reason: "systemd is unavailable"}
		}
		return providerObservation{available: true, reason: "system-mode GNOME Remote Desktop"}
	case handlerGnomeHeadless:
		if hostOS != hostreqspec.PlatformLinux || !facts.SupportsSystemd || !facts.Wayland.Attainable {
			return providerObservation{reason: "GNOME headless provider is Linux-only"}
		}
		return providerObservation{available: true, reason: "GNOME headless provider"}
	case handlerGnomeUserShared:
		if hostOS != hostreqspec.PlatformLinux || !facts.SupportsSystemd || !facts.DisplayAttached {
			return providerObservation{reason: "GNOME user-shared provider needs Linux systemd and an attached display"}
		}
		return providerObservation{available: true, reason: "GNOME user-shared provider"}
	case handlerXrdp:
		if hostOS != hostreqspec.PlatformLinux {
			return providerObservation{reason: "xrdp is Linux-only"}
		}
		return providerObservation{reason: "xrdp provider"}
	case handlerWindowsTermservice:
		if hostOS != hostreqspec.PlatformWindows {
			return providerObservation{reason: "TermService is Windows-only"}
		}
		return providerObservation{available: true, reason: "Windows TermService"}
	case handlerMacosScreenSharing:
		if hostOS != hostreqspec.PlatformMacOS {
			return providerObservation{reason: "macOS Screen Sharing is macOS-only"}
		}
		return providerObservation{available: true, reason: "macOS Screen Sharing"}
	default:
		return providerObservation{reason: "unknown provider"}
	}
}

func providerAvailable(provider, osName string, facts hostinventory.Snapshot) bool {
	hostOS := hostreqspec.PlatformFromGOOS(osName)
	switch provider {
	case handlerGnomeSystem:
		return hostOS == hostreqspec.PlatformLinux && facts.SupportsSystemd
	case handlerGnomeUserShared:
		return hostOS == hostreqspec.PlatformLinux && facts.SupportsSystemd && facts.DisplayAttached
	case handlerGnomeHeadless:
		return hostOS == hostreqspec.PlatformLinux && facts.SupportsSystemd && facts.Wayland.Attainable
	case handlerWindowsTermservice:
		return hostOS == hostreqspec.PlatformWindows
	case handlerMacosScreenSharing:
		return hostOS == hostreqspec.PlatformMacOS
	default:
		return false
	}
}

func providerDelivers(provider, experience string, facts hostinventory.Snapshot) bool {
	switch provider {
	case handlerGnomeSystem:
		return experience == experienceLogin || experience == experienceDirect
	case handlerGnomeUserShared:
		return experience == experienceDirect && facts.DisplayAttached
	case handlerGnomeHeadless:
		return experience == experienceDirect && facts.Wayland.Attainable
	case handlerXrdp, handlerWindowsTermservice, handlerMacosScreenSharing:
		return experience == experienceDirect
	default:
		return false
	}
}

func observedExperience(provider string, facts hostinventory.Snapshot) string {
	switch provider {
	case handlerGnomeSystem:
		return experienceLogin
	case handlerGnomeUserShared:
		return experienceDirect
	case handlerGnomeHeadless:
		if facts.Wayland.Attainable {
			return experienceDirect
		}
	}
	return "unknown"
}
