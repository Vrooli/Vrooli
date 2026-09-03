//go:build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func nativeSystemctl(options NativeServiceOptions, action string) ([]byte, error) {
	args := []string{}
	if options.User {
		args = append(args, "--user")
	}
	args = append(args, action)
	if action != "daemon-reload" {
		args = append(args, options.Name)
	}
	return exec.Command("systemctl", args...).CombinedOutput()
}

func installNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	if options.Path == "" || options.Name == "" {
		return NativeServiceResult{}, fmt.Errorf("platform: service name and path are required")
	}
	if err := os.MkdirAll(filepath.Dir(options.Path), 0o755); err != nil {
		return NativeServiceResult{}, err
	}
	if err := os.WriteFile(options.Path, []byte(options.Content), 0o644); err != nil {
		return NativeServiceResult{}, err
	}
	if output, err := nativeSystemctl(options, "daemon-reload"); err != nil {
		return NativeServiceResult{}, fmt.Errorf("platform: daemon-reload: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if output, err := nativeSystemctl(options, "enable"); err != nil {
		return NativeServiceResult{}, fmt.Errorf("platform: enable: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if output, err := nativeSystemctl(options, "start"); err != nil {
		return NativeServiceResult{}, fmt.Errorf("platform: start: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return NativeServiceResult{Name: options.Name, Path: options.Path, Scope: map[bool]string{true: "user", false: "system"}[options.User], Running: true, Enabled: true, State: ServiceStateRunning, Evidence: ServiceEvidence{Source: "systemctl", RawState: "active"}}, nil
}

func uninstallNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	_, _ = nativeSystemctl(options, "stop")
	_, _ = nativeSystemctl(options, "disable")
	if options.Path != "" {
		if err := os.Remove(options.Path); err != nil && !os.IsNotExist(err) {
			return NativeServiceResult{}, err
		}
	}
	_, _ = nativeSystemctl(options, "daemon-reload")
	return NativeServiceResult{Name: options.Name, Path: options.Path, Scope: map[bool]string{true: "user", false: "system"}[options.User]}, nil
}

func startNativeService(options NativeServiceOptions) error {
	_, err := nativeSystemctl(options, "start")
	return err
}

func stopNativeService(options NativeServiceOptions) error {
	_, err := nativeSystemctl(options, "stop")
	return err
}

func restartNativeService(options NativeServiceOptions) error {
	_, err := nativeSystemctl(options, "restart")
	return err
}

func nativeServiceStatus(options NativeServiceOptions) (NativeServiceResult, error) {
	r := NativeServiceResult{Name: options.Name, Path: options.Path, Scope: map[bool]string{true: "user", false: "system"}[options.User], State: ServiceStateUnknown, Evidence: ServiceEvidence{Source: "systemctl"}}
	if output, err := nativeSystemctl(options, "is-active"); err == nil {
		r.Evidence.RawState = strings.TrimSpace(string(output))
		r.State = parseSystemdState(r.Evidence.RawState)
	} else if len(output) > 0 {
		r.Evidence.RawState = strings.TrimSpace(string(output))
		r.State = parseSystemdState(r.Evidence.RawState)
	}
	r.Running = r.State == ServiceStateRunning
	if output, err := nativeSystemctl(options, "is-enabled"); err == nil && strings.TrimSpace(string(output)) == "enabled" {
		r.Enabled = true
	}
	return r, nil
}

func parseSystemdState(raw string) ServiceState {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "active":
		return ServiceStateRunning
	case "inactive", "deactivating", "dead":
		return ServiceStateStopped
	case "failed":
		return ServiceStateFailed
	default:
		return ServiceStateUnknown
	}
}

func nativeServiceLogs(options NativeServiceOptions, tail int) ([]byte, error) {
	args := []string{"-u", options.Name, "--no-pager"}
	if tail > 0 {
		args = append(args, "-n", strconv.Itoa(tail))
	}
	return exec.Command("journalctl", args...).CombinedOutput()
}

func readHostLogs(options HostLogOptions) (HostLogResult, error) {
	args := append([]string(nil), options.Arguments...)
	if len(args) == 0 {
		args = []string{"--no-pager"}
	}
	if options.Unit != "" {
		args = append(args, "-u", options.Unit)
	}
	if options.Since != "" {
		args = append(args, "--since", options.Since)
	}
	if options.Tail > 0 {
		args = append(args, "-n", strconv.Itoa(options.Tail))
	}
	out, err := exec.Command("journalctl", args...).CombinedOutput()
	return HostLogResult{Source: "journalctl", Raw: out, Entries: ParseJournalJSON(out), Evidence: ServiceEvidence{Source: "journalctl", Detail: strings.TrimSpace(string(out))}}, err
}

func installService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	if !options.User {
		return ServiceInstallResult{}, fmt.Errorf("platform: system service install requires explicit broker support")
	}
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	executable := options.Executable
	if executable == "" {
		executable, err = os.Executable()
		if err != nil {
			return ServiceInstallResult{}, fmt.Errorf("platform: resolve executable: %w", err)
		}
	}
	path := RuntimeSupervisorUnitPath("linux", home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: create systemd unit dir: %w", err)
	}
	if logPath := strings.TrimSpace(options.LogPath); logPath != "" {
		// systemd creates the log file but not its directory; a missing
		// directory fails the unit at start with a message that does not
		// mention logging at all.
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return ServiceInstallResult{}, fmt.Errorf("platform: create supervisor log dir: %w", err)
		}
	}
	definition, err := RuntimeSupervisorDefinition("linux", RuntimeSupervisorOptions{Home: home, Executable: executable, SourceRoot: options.SourceRoot, LogPath: options.LogPath})
	if err != nil {
		return ServiceInstallResult{}, err
	}
	artifact, err := RenderSystemd(definition)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	result := ServiceInstallResult{UnitName: runtimeSupervisorUnit, UnitPath: path, Scope: "user"}
	// A unit systemd will not load is not installed, whatever `enable` says.
	// Ask systemd before touching the unit directory so a bad render never
	// replaces a working unit.
	result.Verdict = ValidateSystemd(artifact, ScopeUser)
	if result.Verdict.Rejected() {
		return result, fmt.Errorf("platform: systemd rejected the rendered %s: %s", runtimeSupervisorUnit, result.Verdict.Output)
	}
	if err := os.WriteFile(path, []byte(artifact.Primary().Content), 0o644); err != nil {
		return result, fmt.Errorf("platform: write systemd unit: %w", err)
	}
	if output, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return result, fmt.Errorf("platform: systemctl daemon-reload: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if output, err := exec.Command("systemctl", "--user", "enable", "--now", runtimeSupervisorUnit).CombinedOutput(); err != nil {
		return result, fmt.Errorf("platform: systemctl enable: %w: %s\n%s", err, strings.TrimSpace(string(output)), systemdUnitDiagnostics())
	}
	// `enable --now` alone does not prove the unit runs: a bad-setting unit
	// can leave enablement recorded while the service never starts, which is
	// how the supervisor stayed dead for days while every surface reported
	// the install had succeeded.
	if state := awaitSystemdActive(runtimeSupervisorUnit); state != ServiceStateRunning {
		return result, fmt.Errorf("platform: installed %s but it is %s, not running:\n%s", runtimeSupervisorUnit, state, systemdUnitDiagnostics())
	}
	result.Active = true
	return result, nil
}

const runtimeSupervisorUnit = "vrooli-runtime-supervisor.service"

// serviceStartHint names the operator command for the installed unit.
func serviceStartHint() string { return "systemctl --user start " + runtimeSupervisorUnit }

// awaitSystemdActive polls is-active until the unit settles. Type=simple units
// report active as soon as they fork, so a unit that starts and immediately
// dies can read running on the first sample; polling to the deadline and
// reporting the LAST observation catches the crash-on-start case that a single
// sample would call success.
func awaitSystemdActive(unit string) ServiceState {
	state := ServiceStateUnknown
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// is-active exits non-zero for anything but active; the state we want
		// is on stdout either way, so the exit code is not the signal here.
		output, err := exec.Command("systemctl", "--user", "is-active", unit).CombinedOutput()
		state = ParseNativeServiceState("linux", string(output), err != nil)
		if state == ServiceStateFailed {
			return state
		}
	}
	return state
}

// systemdUnitDiagnostics captures what an operator would run by hand, so a
// failed install explains itself instead of pointing at another command.
func systemdUnitDiagnostics() string {
	status, _ := exec.Command("systemctl", "--user", "status", "--no-pager", "--lines=10", runtimeSupervisorUnit).CombinedOutput()
	verify, _ := exec.Command("systemd-analyze", "--user", "verify", runtimeSupervisorUnit).CombinedOutput()
	parts := []string{strings.TrimSpace(string(status))}
	if trimmed := strings.TrimSpace(string(verify)); trimmed != "" {
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, "\n")
}

func uninstallService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	path := RuntimeSupervisorUnitPath("linux", home)
	_, _ = exec.Command("systemctl", "--user", "disable", "--now", runtimeSupervisorUnit).CombinedOutput()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return ServiceInstallResult{}, fmt.Errorf("platform: remove systemd unit: %w", err)
	}
	return ServiceInstallResult{UnitName: runtimeSupervisorUnit, UnitPath: path, Scope: "user", Active: false}, nil
}

func startInstalledService(options ServiceInstallOptions) (bool, error) {
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return false, err
	}
	path := RuntimeSupervisorUnitPath("linux", home)
	if _, err := os.Stat(path); err != nil {
		// No unit installed: the caller falls back to launching it directly.
		return false, nil
	}
	if output, err := exec.Command("systemctl", "--user", "start", runtimeSupervisorUnit).CombinedOutput(); err != nil {
		return false, fmt.Errorf("platform: systemctl start %s: %w: %s", runtimeSupervisorUnit, err, strings.TrimSpace(string(output)))
	}
	if state := awaitSystemdActive(runtimeSupervisorUnit); state != ServiceStateRunning {
		return false, fmt.Errorf("platform: started %s but it is %s, not running:\n%s", runtimeSupervisorUnit, state, systemdUnitDiagnostics())
	}
	return true, nil
}

func supportsService(user bool) bool {
	return user && exec.Command("systemctl", "--user", "--version").Run() == nil
}
