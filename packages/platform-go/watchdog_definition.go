package platform

import (
	"path/filepath"
	"time"
)

// WatchdogDefinitionOptions contains the platform-neutral inputs for the
// autoheal loop's boot-supervisor unit. Rendering belongs here with the
// native lifecycle backends; scenario packages do not carry unit templates.
type WatchdogDefinitionOptions struct {
	Root         string
	Home         string
	LoopBinary   string
	VrooliBinary string
	// Username is the principal the unit runs as; required for windows.
	Username string
	// SystemService installs into the system manager (as root) instead of
	// the invoking user's manager.
	SystemService bool
}

// WatchdogDefinition builds the autoheal loop's ServiceDefinition for a
// target. The loop is what restarts everything else, so it carries the
// strongest supervision: restart on failure with a bounded burst, the
// emergency watchdog as its OnFailure= escalation, and the pressure
// protections learned on 2026-08-19 when this host reached a load average of
// 110 on 32 CPUs and autoheal's own health ticks timed out precisely when its
// reports mattered most.
func WatchdogDefinition(target string, options WatchdogDefinitionOptions) (ServiceDefinition, error) {
	target, err := NormalizeTarget(target)
	if err != nil {
		return ServiceDefinition{}, err
	}
	unit, _ := CoreUnitByID(CoreUnitAutohealLoop)
	emergency, _ := CoreUnitByID(CoreUnitEmergencyWatchdog)
	scope := ScopeUser
	username := options.Username
	if options.SystemService {
		scope = ScopeSystem
		if username == "" {
			username = "root"
		}
	}
	d := ServiceDefinition{
		Name:             "vrooli-autoheal",
		Label:            unit.Launchd,
		Description:      "Vrooli Autoheal - Self-healing infrastructure supervisor",
		DocumentationURL: DocumentationURL(unit.OwnerPath),
		Executable:       options.LoopBinary,
		Env: map[string]string{
			"VROOLI_LIFECYCLE_MANAGED": "true",
			"HOME":                     options.Home,
			"VROOLI_ROOT":              options.Root,
			"VROOLI_BIN":               options.VrooliBinary,
			"PATH":                     DefaultPath(target, options.Home),
		},
		WorkingDirectory: targetJoin(target, options.Root, "scenarios", "vrooli-autoheal"),
		Kind:             KindDaemon,
		Restart:          RestartPolicy{Mode: RestartOnFailure, Delay: 15 * time.Second, BurstLimit: 5, BurstWindow: 300 * time.Second},
		OnFailureUnit:    emergency.Systemd,
		Scope:            scope,
		Protections:      Protections{Containment: Containment{CPUWeight: 400}, MemoryMin: "128M", OOMScoreAdjust: -500},
		StopTimeout:      30 * time.Second,
		Username:         username,
	}
	if target == "darwin" {
		logPath := LaunchdLogPath(LaunchAgentPath(options.Home, unit.Launchd), unit.Launchd)
		d.Logs = LogPaths{Stdout: logPath, Stderr: logPath}
	}
	return d, nil
}

// RenderWatchdogArtifact renders the autoheal loop unit for a target.
func RenderWatchdogArtifact(target string, options WatchdogDefinitionOptions) (RenderedArtifact, error) {
	d, err := WatchdogDefinition(target, options)
	if err != nil {
		return RenderedArtifact{}, err
	}
	return RenderDefinition(d, target)
}

// RenderWatchdogDefinition renders the autoheal loop unit body for a target.
// The target argument is explicit so injected platform tests can exercise
// every renderer on one host.
func RenderWatchdogDefinition(target string, options WatchdogDefinitionOptions) (string, error) {
	artifact, err := RenderWatchdogArtifact(target, options)
	if err != nil {
		return "", err
	}
	return artifact.Primary().Content, nil
}

// RenderDefinition renders a definition with the target's renderer.
func RenderDefinition(d ServiceDefinition, target string) (RenderedArtifact, error) {
	target, err := NormalizeTarget(target)
	if err != nil {
		return RenderedArtifact{}, err
	}
	switch target {
	case "darwin":
		return RenderLaunchd(d)
	case "windows":
		return RenderWindowsTaskXML(d)
	default:
		return RenderSystemd(d)
	}
}

// LaunchAgentPath is where a per-user LaunchAgent plist for label lives.
func LaunchAgentPath(home, label string) string {
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist")
}

// SystemdUserUnitPath is where a per-user systemd unit file lives.
func SystemdUserUnitPath(home, unitName string) string {
	return filepath.Join(home, ".config", "systemd", "user", unitName)
}
