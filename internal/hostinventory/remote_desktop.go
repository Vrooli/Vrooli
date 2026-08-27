package hostinventory

import (
	"context"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/shell"
)

const (
	remoteDesktopActive = "active"
)

const (
	remoteDesktopParameterA = 3
)

// RemoteDesktopCommandRunner is the minimal command seam needed by the
// shared provider classifier. It intentionally has no path lookup or mutation
// operations.
type RemoteDesktopCommandRunner = shell.Runner

// ProbeRemoteDesktopCredentials reads the provider's non-secret status output
// through the shared host-inventory command boundary. Callers may pass the
// session environment required by a user-mode GNOME daemon; no credential
// value is returned by the inventory layer's policy decisions.
func ProbeRemoteDesktopCredentials(ctx context.Context, commands RemoteDesktopCommandRunner, sessionEnv []string) ([]byte, error) {
	args := append(append([]string{}, sessionEnv...), "grdctl", "status")
	return boundedCommand(ctx, commands, "env", args...)
}

// ClassifyRemoteDesktop lets consumers reuse the same authority while keeping
// their own command-test seam. The caller supplies the already-known systemd
// fact; this function performs only provider observations.
func ClassifyRemoteDesktop(ctx context.Context, osName string, supportsSystemd bool, commands RemoteDesktopCommandRunner) RemoteDesktopCapability {
	return classifyRemoteDesktopWithDisplay(ctx, osName, supportsSystemd, false, commands)
}

// ClassifyRemoteDesktopWithDisplay adds the display attachment fact when
// distinguishing a GNOME user-shared desktop from a headless session.
func ClassifyRemoteDesktopWithDisplay(ctx context.Context, osName string, supportsSystemd, displayAttached bool, commands RemoteDesktopCommandRunner) RemoteDesktopCapability {
	return classifyRemoteDesktopWithDisplayAndUser(ctx, osName, supportsSystemd, displayAttached, "", commands)
}

// ClassifyRemoteDesktopWithDisplayAndUser supplies the active graphical user
// so Linux can inspect that user's systemd --user unit without trusting the
// caller's ambient XDG_RUNTIME_DIR. An empty user keeps the lightweight
// command-test seam useful for callers that do not have session identity.
func ClassifyRemoteDesktopWithDisplayAndUser(ctx context.Context, osName string, supportsSystemd, displayAttached bool, sessionUser string, commands RemoteDesktopCommandRunner) RemoteDesktopCapability {
	return classifyRemoteDesktopWithDisplayAndUser(ctx, osName, supportsSystemd, displayAttached, sessionUser, commands)
}

func classifyRemoteDesktopWithDisplay(ctx context.Context, osName string, systemd, displayAttached bool, commands RemoteDesktopCommandRunner) RemoteDesktopCapability {
	return classifyRemoteDesktopWithDisplayAndUser(ctx, osName, systemd, displayAttached, "", commands)
}

func classifyRemoteDesktopWithDisplayAndUser(ctx context.Context, osName string, systemd, displayAttached bool, sessionUser string, commands RemoteDesktopCommandRunner) RemoteDesktopCapability {
	osName = strings.ToLower(strings.TrimSpace(osName))
	var providers []RemoteDesktopProvider
	probe := remoteDesktopProbe{ctx: ctx, systemd: systemd, displayAttached: displayAttached, sessionUser: sessionUser, commands: commands}
	switch hostreqspec.PlatformFromGOOS(osName) {
	case hostreqspec.PlatformLinux:
		providers = probe.linuxProviders()
	case hostreqspec.PlatformWindows:
		providers = windowsRemoteDesktopProviders(ctx, commands)
	case hostreqspec.PlatformMacOS:
		providers = macOSRemoteDesktopProviders(ctx, commands)
	}
	return summarizeRemoteDesktopProviders(providers)
}

type remoteDesktopProbe struct {
	ctx             context.Context
	systemd         bool
	displayAttached bool
	sessionUser     string
	commands        RemoteDesktopCommandRunner
}

func (p remoteDesktopProbe) linuxProviders() []RemoteDesktopProvider {
	providers := make([]RemoteDesktopProvider, 0, remoteDesktopParameterA)
	daemonSystem, daemonUser := daemonModes(p.ctx, p.commands)
	if p.systemd {
		providers = append(providers, p.gnomeSystem(daemonSystem, daemonUser))
	}
	providers = append(providers, p.gnomeUser(daemonUser))
	return append(providers, p.xrdp())
}

func (p remoteDesktopProbe) gnomeSystem(daemonSystem, daemonUser bool) RemoteDesktopProvider {
	systemEnabled := boundedValue(p.ctx, p.commands, "systemctl", "is-enabled", "gnome-remote-desktop.service")
	systemActive := boundedValue(p.ctx, p.commands, "systemctl", "is-active", "gnome-remote-desktop.service")
	return RemoteDesktopProvider{Name: "gnome-system", Present: systemEnabled == "enabled" || systemEnabled == "static" || systemActive == remoteDesktopActive || daemonSystem, Active: systemActive == remoteDesktopActive || daemonSystem, ProbeSucceeded: systemEnabled != "" || systemActive != "" || daemonSystem || daemonUser}
}

func (p remoteDesktopProbe) gnomeUser(daemonUser bool) RemoteDesktopProvider {
	userEnabled, userActive := userUnitState(p.ctx, p.commands, p.sessionUser)
	output, err := remoteDesktopStatus(p.ctx, p.commands, p.sessionUser)
	statusEnabled := err == nil && strings.Contains(string(output), "Status: enabled")
	gnome := RemoteDesktopProvider{Name: "gnome-headless", Present: (statusEnabled && (userEnabled || userActive || daemonUser || strings.TrimSpace(p.sessionUser) == "")) || userActive || daemonUser, ProbeSucceeded: err == nil, UserSession: true}
	if gnome.Present && p.displayAttached && !strings.Contains(strings.ToLower(string(output)), "headless: true") {
		gnome.Name = "gnome-user-shared"
	}
	if gnome.Present {
		gnome.Active = userActive || daemonUser
	}
	return gnome
}

func (p remoteDesktopProbe) xrdp() RemoteDesktopProvider {
	xrdpPresent := false
	if p.systemd {
		output, err := boundedCommand(p.ctx, p.commands, "systemctl", "list-unit-files", "xrdp.service")
		xrdpPresent = err == nil && strings.Contains(string(output), "xrdp.service")
	}
	return RemoteDesktopProvider{Name: "xrdp", Present: xrdpPresent, Active: xrdpPresent && boundedValue(p.ctx, p.commands, "systemctl", "is-active", "xrdp.service") == remoteDesktopActive, ProbeSucceeded: xrdpPresent}
}

func windowsRemoteDesktopProviders(ctx context.Context, commands RemoteDesktopCommandRunner) []RemoteDesktopProvider {
	output, err := boundedCommand(ctx, commands, "sc.exe", "query", "TermService")
	text := string(output)
	return []RemoteDesktopProvider{{Name: "windows-termservice", Present: err == nil && strings.Contains(text, "TermService"), Active: err == nil && strings.Contains(text, "RUNNING"), ProbeSucceeded: err == nil}}
}

func macOSRemoteDesktopProviders(ctx context.Context, commands RemoteDesktopCommandRunner) []RemoteDesktopProvider {
	_, err := boundedCommand(ctx, commands, "launchctl", "print", "system/com.apple.screensharing")
	return []RemoteDesktopProvider{{Name: "macos-screen-sharing", Present: err == nil, Active: err == nil, ProbeSucceeded: err == nil}}
}

func summarizeRemoteDesktopProviders(providers []RemoteDesktopProvider) RemoteDesktopCapability {
	capability := RemoteDesktopCapability{Providers: providers}
	for _, provider := range providers {
		if provider.Present {
			capability.Supported = true
			if capability.SelectedProvider == "" {
				capability.SelectedProvider = provider.Name
				capability.Observed = true
				capability.Active = provider.Active
				capability.Mode = remoteDesktopMode(provider.Name)
				capability.ListeningPort = 3389
			}
		}
	}
	return capability
}

func daemonModes(ctx context.Context, commands RemoteDesktopCommandRunner) (system, user bool) {
	output, err := boundedCommand(ctx, commands, "pgrep", "-a", "-f", "gnome-remote-desktop-daemon")
	if err != nil || strings.TrimSpace(string(output)) == "" {
		// Older/minimal command seams may expose only the matching PID. It
		// cannot distinguish system from user mode, so use it only as a
		// user-mode presence fallback; complete rows above remain authoritative
		// whenever they are available.
		legacy, legacyErr := boundedCommand(ctx, commands, "pgrep", "-f", "gnome-remote-desktop-daemon")
		if legacyErr == nil && strings.TrimSpace(string(legacy)) != "" {
			return false, true
		}
		return false, false
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		// pgrep -f can include the probe shell itself because its command
		// line contains the search pattern. The executable column is the
		// stable discriminator for an actual daemon row.
		if len(fields) < 2 || !strings.HasSuffix(fields[1], "gnome-remote-desktop-daemon") {
			continue
		}
		isSystem := false
		for _, field := range fields[2:] {
			if field == "--system" {
				isSystem = true
				break
			}
		}
		if isSystem {
			system = true
		} else {
			user = true
		}
	}
	return system, user
}

func userUnitState(ctx context.Context, commands RemoteDesktopCommandRunner, sessionUser string) (enabled, active bool) {
	sessionUser = strings.TrimSpace(sessionUser)
	if sessionUser == "" {
		return false, false
	}
	uid := boundedValue(ctx, commands, "id", "-u", sessionUser)
	if uid == "" {
		return false, false
	}
	env := []string{
		"XDG_RUNTIME_DIR=/run/user/" + uid,
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/" + uid + "/bus",
		"systemctl", "--user",
	}
	enabled = boundedValue(ctx, commands, "env", append(append([]string{}, env...), "is-enabled", "gnome-remote-desktop.service")...) == "enabled"
	active = boundedValue(ctx, commands, "env", append(append([]string{}, env...), "is-active", "gnome-remote-desktop.service")...) == remoteDesktopActive
	return enabled, active
}

func remoteDesktopStatus(ctx context.Context, commands RemoteDesktopCommandRunner, sessionUser string) ([]byte, error) {
	if strings.TrimSpace(sessionUser) == "" {
		return boundedCommand(ctx, commands, "grdctl", "status")
	}
	uid := boundedValue(ctx, commands, "id", "-u", sessionUser)
	if uid == "" {
		return nil, context.Canceled
	}
	return boundedCommand(ctx, commands, "env",
		"XDG_RUNTIME_DIR=/run/user/"+uid,
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/"+uid+"/bus",
		"grdctl", "status")
}

func remoteDesktopMode(provider string) string {
	switch provider {
	case "gnome-system", "xrdp", "windows-termservice", "macos-screen-sharing":
		return "system"
	case "gnome-user-shared":
		return "user-shared"
	case "gnome-headless":
		return "headless"
	default:
		return "none"
	}
}

func boundedCommand(parent context.Context, commands RemoteDesktopCommandRunner, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, tuning.RemoteDesktopProbeTimeout())
	defer cancel()
	return commands.Run(ctx, name, args...)
}

func boundedValue(parent context.Context, commands RemoteDesktopCommandRunner, name string, args ...string) string {
	output, err := boundedCommand(parent, commands, name, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
