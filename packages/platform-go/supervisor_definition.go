package platform

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/repo-contract-go/cliinvoke"
)

// RuntimeSupervisorOptions are the inputs for the runtime supervisor unit.
type RuntimeSupervisorOptions struct {
	Home       string
	Executable string
	SourceRoot string
	// LogPath is the absolute file the supervisor writes stdout and stderr
	// to. The caller resolves it (platform-go does not know the Vrooli home
	// contract). Empty means the manager's default: the journal on Linux,
	// LaunchdLogPath on macOS.
	LogPath string
}

// RuntimeSupervisorDefinition builds the runtime supervisor's definition for
// a target. The argv comes from the cliinvoke catalog with no leading global
// flags, so a CLI that retires a flag cannot strand the unit the way
// Older supervisor definitions may carry obsolete freshness options; current
// definitions leave freshness decisions to the lifecycle engine.
func RuntimeSupervisorDefinition(target string, options RuntimeSupervisorOptions) (ServiceDefinition, error) {
	target, err := NormalizeTarget(target)
	if err != nil {
		return ServiceDefinition{}, err
	}
	unit, _ := CoreUnitByID(CoreUnitRuntimeSupervisor)
	sourceRoot := strings.TrimSpace(options.SourceRoot)
	env := map[string]string{
		"HOME":                      options.Home,
		"VROOLI_RUNTIME_SUPERVISOR": "on",
		"PATH":                      DefaultPath(target, options.Home),
	}
	if sourceRoot != "" {
		env["VROOLI_SOURCE_ROOT"] = sourceRoot
	}
	logPath := strings.TrimSpace(options.LogPath)
	if logPath == "" && target == "darwin" {
		logPath = LaunchdLogPath(LaunchAgentPath(options.Home, unit.Launchd), unit.Launchd)
	}
	d := ServiceDefinition{
		Name:             "vrooli-runtime-supervisor",
		Label:            unit.Launchd,
		Description:      "Vrooli runtime supervisor",
		DocumentationURL: DocumentationURL(unit.OwnerPath),
		Executable:       options.Executable,
		Args:             cliinvoke.RuntimeSupervisorRun(),
		Env:              env,
		WorkingDirectory: sourceRoot,
		Kind:             KindDaemon,
		Restart:          RestartPolicy{Mode: RestartAlways, Delay: 5 * time.Second, BurstLimit: 5, BurstWindow: 300 * time.Second},
		Scope:            ScopeUser,
		Protections:      Protections{Containment: Containment{CPUWeight: 400}, MemoryMin: "128M", OOMScoreAdjust: -500},
		StopTimeout:      30 * time.Second,
		Logs:             LogPaths{Stdout: logPath, Stderr: logPath},
	}
	return d, nil
}

// RuntimeSupervisorUnitPath is where the supervisor's user unit lives on a
// target.
func RuntimeSupervisorUnitPath(target, home string) string {
	unit, _ := CoreUnitByID(CoreUnitRuntimeSupervisor)
	switch target {
	case "darwin":
		return LaunchAgentPath(home, unit.Launchd)
	default:
		return filepath.Join(home, ".config", "systemd", "user", unit.Systemd)
	}
}

// resolvedHome returns the explicit home when given, else the platform's.
func resolvedHome(home string) (string, error) {
	if strings.TrimSpace(home) != "" {
		return home, nil
	}
	return HomeDir()
}
