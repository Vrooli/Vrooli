package hostinventory

import (
	"context"
	"strings"
	"time"
)

const remoteDesktopProbeTimeout = 5 * time.Second

// RemoteDesktopCommandRunner is the minimal command seam needed by the
// shared provider classifier. It intentionally has no path lookup or mutation
// operations.
type RemoteDesktopCommandRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

// ProbeRemoteDesktopCredentials reads the provider's non-secret status output
// through the shared host-inventory command boundary. Callers may pass the
// session environment required by a user-mode GNOME daemon; no credential
// value is returned by the inventory layer's policy decisions.
func ProbeRemoteDesktopCredentials(ctx context.Context, commands RemoteDesktopCommandRunner, sessionEnv []string) ([]byte, error) {
	args := append(append([]string{}, sessionEnv...), "grdctl", "status")
	return boundedCommand(commands, ctx, "env", args...)
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

// classifyRemoteDesktop is the only local provider classifier. Its output is
// deliberately a complete observation set so callers can apply policy without
// knowing which command or service implements a provider.
func classifyRemoteDesktop(ctx context.Context, c Collector, osName string) RemoteDesktopCapability {
	return classifyRemoteDesktopWithDisplayAndUser(ctx, osName, detectSystemd(c), false, "", c.Commands)
}

func classifyRemoteDesktopWith(ctx context.Context, osName string, systemd bool, commands RemoteDesktopCommandRunner) RemoteDesktopCapability {
	return classifyRemoteDesktopWithDisplayAndUser(ctx, osName, systemd, false, "", commands)
}

func classifyRemoteDesktopWithDisplay(ctx context.Context, osName string, systemd, displayAttached bool, commands RemoteDesktopCommandRunner) RemoteDesktopCapability {
	return classifyRemoteDesktopWithDisplayAndUser(ctx, osName, systemd, displayAttached, "", commands)
}

func classifyRemoteDesktopWithDisplayAndUser(ctx context.Context, osName string, systemd, displayAttached bool, sessionUser string, commands RemoteDesktopCommandRunner) RemoteDesktopCapability {
	osName = strings.ToLower(strings.TrimSpace(osName))
	providers := make([]RemoteDesktopProvider, 0, 3)
	switch osName {
	case "linux":
		daemonSystem, daemonUser := daemonModes(ctx, commands)
		var systemEnabled, systemActive string
		if systemd {
			systemEnabled = boundedValue(commands, ctx, "systemctl", "is-enabled", "gnome-remote-desktop.service")
			systemActive = boundedValue(commands, ctx, "systemctl", "is-active", "gnome-remote-desktop.service")
			providers = append(providers, RemoteDesktopProvider{
				Name:           "gnome-system",
				Present:        systemEnabled == "enabled" || systemEnabled == "static" || systemActive == "active" || daemonSystem,
				Active:         systemActive == "active" || daemonSystem,
				ProbeSucceeded: systemEnabled != "" || systemActive != "" || daemonSystem || daemonUser,
			})
		}
		userEnabled, userActive := userUnitState(ctx, commands, sessionUser)
		output, err := remoteDesktopStatus(ctx, commands, sessionUser)
		statusEnabled := err == nil && strings.Contains(string(output), "Status: enabled")
		gnomeHeadless := RemoteDesktopProvider{
			Name:           "gnome-headless",
			Present:        (statusEnabled && (userEnabled || userActive || daemonUser || strings.TrimSpace(sessionUser) == "")) || userActive || daemonUser,
			ProbeSucceeded: err == nil,
			UserSession:    true,
		}
		if gnomeHeadless.Present && displayAttached && !strings.Contains(strings.ToLower(string(output)), "headless: true") {
			gnomeHeadless.Name = "gnome-user-shared"
		}
		if gnomeHeadless.Present {
			gnomeHeadless.Active = userActive || daemonUser
		}
		providers = append(providers, gnomeHeadless)
		xrdpPresent := false
		if systemd {
			output, err = boundedCommand(commands, ctx, "systemctl", "list-unit-files", "xrdp.service")
			xrdpPresent = err == nil && strings.Contains(string(output), "xrdp.service")
		}
		providers = append(providers, RemoteDesktopProvider{
			Name:           "xrdp",
			Present:        xrdpPresent,
			Active:         xrdpPresent && boundedValue(commands, ctx, "systemctl", "is-active", "xrdp.service") == "active",
			ProbeSucceeded: xrdpPresent,
		})
	case "windows":
		output, err := boundedCommand(commands, ctx, "sc.exe", "query", "TermService")
		text := string(output)
		providers = append(providers, RemoteDesktopProvider{
			Name:           "windows-termservice",
			Present:        err == nil && strings.Contains(text, "TermService"),
			Active:         err == nil && strings.Contains(text, "RUNNING"),
			ProbeSucceeded: err == nil,
		})
	case "darwin", "macos":
		_, err := boundedCommand(commands, ctx, "launchctl", "print", "system/com.apple.screensharing")
		providers = append(providers, RemoteDesktopProvider{
			Name:           "macos-screen-sharing",
			Present:        err == nil,
			Active:         err == nil,
			ProbeSucceeded: err == nil,
		})
	}

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
	output, err := boundedCommand(commands, ctx, "pgrep", "-a", "-f", "gnome-remote-desktop-daemon")
	if err != nil || strings.TrimSpace(string(output)) == "" {
		// Older/minimal command seams may expose only the matching PID. It
		// cannot distinguish system from user mode, so use it only as a
		// user-mode presence fallback; complete rows above remain authoritative
		// whenever they are available.
		legacy, legacyErr := boundedCommand(commands, ctx, "pgrep", "-f", "gnome-remote-desktop-daemon")
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
	uid := boundedValue(commands, ctx, "id", "-u", sessionUser)
	if uid == "" {
		return false, false
	}
	env := []string{
		"XDG_RUNTIME_DIR=/run/user/" + uid,
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/" + uid + "/bus",
		"systemctl", "--user",
	}
	enabled = boundedValue(commands, ctx, "env", append(append([]string{}, env...), "is-enabled", "gnome-remote-desktop.service")...) == "enabled"
	active = boundedValue(commands, ctx, "env", append(append([]string{}, env...), "is-active", "gnome-remote-desktop.service")...) == "active"
	return enabled, active
}

func remoteDesktopStatus(ctx context.Context, commands RemoteDesktopCommandRunner, sessionUser string) ([]byte, error) {
	if strings.TrimSpace(sessionUser) == "" {
		return boundedCommand(commands, ctx, "grdctl", "status")
	}
	uid := boundedValue(commands, ctx, "id", "-u", sessionUser)
	if uid == "" {
		return nil, context.Canceled
	}
	return boundedCommand(commands, ctx, "env",
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

func boundedCommand(commands RemoteDesktopCommandRunner, parent context.Context, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, remoteDesktopProbeTimeout)
	defer cancel()
	return commands.Run(ctx, name, args...)
}

func boundedValue(commands RemoteDesktopCommandRunner, parent context.Context, name string, args ...string) string {
	output, err := boundedCommand(commands, parent, name, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
