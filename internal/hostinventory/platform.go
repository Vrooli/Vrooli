package hostinventory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CurrentPlatform returns the process host platform without running any
// probes. Callers that need capability facts should use CollectPlatformFacts;
// platform-branch logic should use this lightweight authority instead.
func CurrentPlatform() string {
	return runtime.GOOS
}

// CollectPlatformFacts performs the lightweight platform probe used by the
// runtime capability layer. It intentionally shares Collector seams so tests
// can model command and file state without a process-wide cache.
func CollectPlatformFacts(ctx context.Context) Snapshot {
	var cached Snapshot
	if raw, err := sharedFactsReader().Read(ctx, "platform"); err == nil && json.Unmarshal(raw, &cached) == nil {
		return cached
	}
	snapshot, _ := SystemCollector().CollectPlatformFacts(ctx)
	return snapshot
}

func (c Collector) CollectPlatformFacts(ctx context.Context) (Snapshot, error) {
	c = c.withDefaults()
	snapshot := Snapshot{OS: c.GOOS, Arch: c.GOARCH, FieldProvenance: map[string]Provenance{}, ProbeStatuses: map[string]string{}}
	c.collectPlatformFacts(ctx, &snapshot, c.Clock.Now())
	return snapshot, nil
}

func (c Collector) collectPlatformFacts(ctx context.Context, snapshot *Snapshot, observedAt time.Time) {
	snapshot.Elevation = currentElevation()
	setPlatformProvenance := func(field, source string, kind SourceKind, command, file string) {
		snapshot.FieldProvenance[field] = Provenance{SourceKind: kind, Source: source, ObservedAt: observedAt, Confidence: "high", Command: command, File: file}
	}

	switch snapshot.OS {
	case "linux":
		snapshot.SupportsSystemd = detectSystemd(c)
		snapshot.SupportsSysctl = isDirectoryWith(c, "/etc/sysctl.d")
		snapshot.InitSystem = "unknown"
		if snapshot.SupportsSystemd {
			snapshot.InitSystem = "systemd"
		} else if strings.TrimSpace(c.readString("/proc/1/comm")) != "" {
			snapshot.InitSystem = strings.TrimSpace(c.readString("/proc/1/comm"))
		}
		snapshot.SessionType = c.commandValue(ctx, "loginctl", "show-session", "self", "-p", "Type", "--value")
		snapshot.Seat = c.commandValue(ctx, "loginctl", "show-session", "self", "-p", "Seat", "--value")
		snapshot.ActiveSessionUser = ActiveSessionUser(ctx, c.Commands)
		snapshot.DisplayManager = c.displayManager(ctx)
		displayPolicy := c.displayPolicy()
		snapshot.DisplayServer = displayPolicy.Preference
		snapshot.DisplayAttached = c.displayAttached()
		snapshot.AutoLoginUser = displayPolicy.AutoLoginUser
		snapshot.Wayland = waylandCapability(displayPolicy)
		snapshot.RemoteDesktop = ClassifyRemoteDesktopWithDisplayAndUser(ctx, snapshot.OS, snapshot.SupportsSystemd, snapshot.DisplayAttached, snapshot.ActiveSessionUser, c.Commands)
		snapshot.RemoteDesktop.CredentialStore = probeCredentialStore(ctx, c, snapshot.ActiveSessionUser)
		snapshot.IsWSL = c.isWSL()
		snapshot.IsHeadless = snapshot.SessionType == "" && snapshot.DisplayManager == ""
		snapshot.SupportsRDP = snapshot.RemoteDesktop.Supported
		snapshot.SupportsCloudflared = c.commandAvailable("cloudflared")
		setPlatformProvenance("init_system", "linux init signals", SourceKindDerived, "systemctl /run/systemd/system /proc/1/comm", "")
		setPlatformProvenance("session_type", "loginctl", SourceKindCommand, "loginctl show-session self -p Type --value", "")
		setPlatformProvenance("seat", "loginctl", SourceKindCommand, "loginctl show-session self -p Seat --value", "")
		setPlatformProvenance("display_manager", "systemd display-manager unit", SourceKindCommand, "systemctl show display-manager.service -p Id --value", "")
		setPlatformProvenance("display_server", "display-manager configuration", SourceKindFile, "", displayPolicy.Path)
		setPlatformProvenance("display_attached", "DRM connector status", SourceKindFile, "", "/sys/class/drm/*/status")
		setPlatformProvenance("wayland", "display-manager policy", SourceKindDerived, "", displayPolicy.Path)
		setPlatformProvenance("remote_desktop", "remote-desktop provider classifier", SourceKindDerived, "systemctl pgrep grdctl", "")
	case "darwin":
		snapshot.InitSystem = "launchd"
		snapshot.SupportsSysctl = false
		snapshot.SupportsSystemd = false
		snapshot.SupportsLaunchd = c.commandAvailable("launchctl")
		snapshot.IsHeadless = strings.TrimSpace(c.envValue("DISPLAY")) == "" && strings.TrimSpace(c.envValue("WAYLAND_DISPLAY")) == ""
		snapshot.SupportsCloudflared = c.commandAvailable("cloudflared")
		snapshot.RemoteDesktop = ClassifyRemoteDesktop(ctx, snapshot.OS, snapshot.SupportsSystemd, c.Commands)
		snapshot.SupportsRDP = snapshot.RemoteDesktop.Supported
		setPlatformProvenance("remote_desktop", "remote-desktop provider classifier", SourceKindDerived, "launchctl", "")
		setPlatformProvenance("init_system", "launchctl", SourceKindCommand, "launchctl", "")
	case "windows":
		snapshot.InitSystem = "windows-service-manager"
		snapshot.SupportsSysctl = false
		snapshot.SupportsSystemd = false
		snapshot.SupportsWindowsServices = c.commandAvailable("sc.exe")
		snapshot.RemoteDesktop = ClassifyRemoteDesktop(ctx, snapshot.OS, snapshot.SupportsSystemd, c.Commands)
		snapshot.SupportsRDP = snapshot.RemoteDesktop.Supported
		snapshot.IsHeadless = !c.commandAvailable("explorer.exe")
		snapshot.SupportsCloudflared = c.commandAvailable("cloudflared")
		setPlatformProvenance("remote_desktop", "remote-desktop provider classifier", SourceKindDerived, "sc.exe", "")
		setPlatformProvenance("init_system", "Windows service manager", SourceKindRuntime, "", "")
	default:
		snapshot.InitSystem = "unknown"
	}
	snapshot.ProbeStatuses["platform_facts"] = "ok"
}

func (c Collector) displayAttached() bool {
	var paths []string
	if globber, ok := c.Files.(interface{ Glob(string) []string }); ok {
		paths = globber.Glob("/sys/class/drm/*/status")
	}
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		if strings.EqualFold(strings.TrimSpace(c.readString(path)), "connected") {
			return true
		}
	}
	return false
}

func detectSystemd(c Collector) bool {
	if _, err := c.Commands.LookPath("systemctl"); err != nil {
		return false
	}
	return isDirectoryWith(c, "/run/systemd/system") || strings.TrimSpace(c.readString("/proc/1/comm")) == "systemd"
}

func (c Collector) readString(path string) string {
	data, err := c.Files.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func (c Collector) commandValue(ctx context.Context, name string, args ...string) string {
	if _, err := c.Commands.LookPath(name); err != nil {
		return ""
	}
	output, err := c.Commands.Run(ctx, name, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (c Collector) commandAvailable(name string) bool {
	_, err := c.Commands.LookPath(name)
	return err == nil
}

func (c Collector) envValue(name string) string {
	if c.Env == nil {
		return ""
	}
	return c.Env.Getenv(name)
}

func (c Collector) isWSL() bool {
	if snapshot := strings.ToLower(c.readString("/proc/version")); strings.Contains(snapshot, "microsoft") || strings.Contains(snapshot, "wsl") {
		return true
	}
	return strings.TrimSpace(c.envValue("WSL_DISTRO_NAME")) != ""
}

func (c Collector) displayManager(ctx context.Context) string {
	unit := c.commandValue(ctx, "systemctl", "show", "display-manager.service", "-p", "Id", "--value")
	unit = strings.TrimSuffix(filepath.Base(unit), ".service")
	for _, name := range DisplayManagerNames {
		if strings.EqualFold(unit, name) {
			return name
		}
	}
	return unit
}

type displayPolicy struct {
	Preference    string
	WaylandOff    bool
	NvidiaMarker  bool
	AutoLoginUser string
	Path          string
}

var displayPolicyPaths = []string{
	"/run/gdm3/custom.conf",
	"/run/gdm/custom.conf",
	"/etc/gdm3/custom.conf",
	"/etc/gdm/custom.conf",
	"/etc/sddm.conf",
}

func (c Collector) displayPolicy() displayPolicy {
	policy := displayPolicy{Preference: "unknown", Path: displayPolicyPaths[0]}
	if _, err := c.Files.ReadFile("/run/udev/gdm-machine-has-vendor-nvidia-driver"); err == nil {
		policy.NvidiaMarker = true
	}
	for _, path := range displayPolicyPaths {
		content := c.readString(path)
		if content == "" {
			continue
		}
		policy.Path = path
		lower := strings.ToLower(content)
		if strings.Contains(lower, "waylandenable=false") {
			policy.WaylandOff = true
		}
		if policy.Preference == "unknown" {
			if strings.Contains(lower, "preferreddisplayserver=xorg") {
				policy.Preference = "xorg"
			} else if strings.Contains(lower, "preferreddisplayserver=wayland") || strings.Contains(lower, "waylandenable=true") {
				policy.Preference = "wayland"
			}
		}
		if user := autoLoginUserFromConfig(content); user != "" && policy.AutoLoginUser == "" {
			policy.AutoLoginUser = user
		}
	}
	return policy
}

func autoLoginUserFromConfig(content string) string {
	inDaemonSection := false
	enabled := false
	user := ""
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "[") {
			inDaemonSection = strings.EqualFold(line, "[daemon]")
			continue
		}
		if !inDaemonSection || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "AutomaticLoginEnable":
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "true", "1", "yes":
				enabled = true
			}
		case "AutomaticLogin":
			user = strings.TrimSpace(value)
		}
	}
	if enabled {
		return user
	}
	return ""
}

func waylandCapability(policy displayPolicy) WaylandCapability {
	if policy.WaylandOff {
		return WaylandCapability{Reason: "display manager explicitly disables Wayland"}
	}
	if policy.Preference == "xorg" {
		return WaylandCapability{Attainable: true, Reason: "display manager prefers Xorg but does not explicitly disable Wayland"}
	}
	if policy.NvidiaMarker {
		return WaylandCapability{Attainable: true, Reason: "NVIDIA display-policy marker observed; it is diagnostic only"}
	}
	return WaylandCapability{Attainable: true, Reason: "no blocking display-manager policy was observed"}
}

func isDirectoryWith(c Collector, path string) bool {
	if reader, ok := c.Files.(interface{ IsDir(string) bool }); ok {
		return reader.IsDir(path)
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
