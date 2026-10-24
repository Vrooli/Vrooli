package platform

import (
	"time"
)

// EmergencyWatchdogOptions are the inputs for the emergency watchdog timer:
// the portable host-pressure binary that runs every few minutes as the last
// line of defense when everything Vrooli builds is broken.
type EmergencyWatchdogOptions struct {
	Home string
	// Binary is the installed vrooli-watchdog executable.
	Binary string
	// SetpointPath is the reliability setpoint the watchdog reads; empty
	// leaves the binary's built-in thresholds.
	SetpointPath string
	// Interval is the cadence; the caller passes the tuned default.
	Interval time.Duration
	// Username is the principal the task runs as; required for windows.
	Username string
}

// EmergencyWatchdogDefinition builds the emergency watchdog's definition for
// a target. It is one timer-kind definition everywhere: systemd renders a
// oneshot service plus a timer, launchd a StartInterval agent, and Windows a
// repeating task, all running the same `--report-only --request-pressure`
// argv. Before this seam the three platforms disagreed on the arguments.
func EmergencyWatchdogDefinition(target string, options EmergencyWatchdogOptions) (ServiceDefinition, error) {
	target, err := NormalizeTarget(target)
	if err != nil {
		return ServiceDefinition{}, err
	}
	unit, _ := CoreUnitByID(CoreUnitEmergencyWatchdog)
	env := map[string]string{
		"HOME": options.Home,
		"PATH": DefaultPath(target, options.Home),
	}
	if options.SetpointPath != "" {
		env["VROOLI_SETPOINT_PATH"] = options.SetpointPath
	}
	d := ServiceDefinition{
		Name:             "vrooli-emergency-watchdog",
		Label:            unit.Launchd,
		Description:      "Vrooli emergency watchdog (portable host-pressure binary)",
		DocumentationURL: DocumentationURL(unit.OwnerPath),
		Executable:       options.Binary,
		Args:             []string{"--report-only", "--request-pressure"},
		Env:              env,
		Kind:             KindTimer,
		Schedule:         &Schedule{OnBoot: 2 * time.Minute, Every: options.Interval, Persistent: true},
		Restart:          RestartPolicy{Mode: RestartNever},
		Scope:            ScopeUser,
		Username:         options.Username,
	}
	if target == "darwin" {
		logPath := LaunchdLogPath(LaunchAgentPath(options.Home, unit.Launchd), unit.Launchd)
		d.Logs = LogPaths{Stdout: logPath, Stderr: logPath}
	}
	return d, nil
}
