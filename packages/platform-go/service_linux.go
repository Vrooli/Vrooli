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
	path := filepath.Join(home, ".config", "systemd", "user", "vrooli-runtime-supervisor.service")
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
	content := systemdUnitContent(executable, home, options.SourceRoot, options.LogPath)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: write systemd unit: %w", err)
	}
	if output, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: systemctl daemon-reload: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if output, err := exec.Command("systemctl", "--user", "enable", "--now", runtimeSupervisorUnit).CombinedOutput(); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: systemctl enable: %w: %s\n%s", err, strings.TrimSpace(string(output)), systemdUnitDiagnostics())
	}
	// An install that wrote a unit the manager refuses to load is a failed
	// install, not a successful one. `enable --now` alone does not prove it:
	// a bad-setting unit can leave enablement recorded while the service never
	// starts, which is how the supervisor stayed dead for days while every
	// surface reported the install had succeeded.
	if state := awaitSystemdActive(runtimeSupervisorUnit); state != ServiceStateRunning {
		return ServiceInstallResult{UnitName: runtimeSupervisorUnit, UnitPath: path, Scope: "user", Active: false},
			fmt.Errorf("platform: installed %s but it is %s, not running:\n%s", runtimeSupervisorUnit, state, systemdUnitDiagnostics())
	}
	return ServiceInstallResult{UnitName: runtimeSupervisorUnit, UnitPath: path, Scope: "user", Active: true}, nil
}

const runtimeSupervisorUnit = "vrooli-runtime-supervisor.service"

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
	path := filepath.Join(home, ".config", "systemd", "user", "vrooli-runtime-supervisor.service")
	_, _ = exec.Command("systemctl", "--user", "disable", "--now", "vrooli-runtime-supervisor.service").CombinedOutput()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return ServiceInstallResult{}, fmt.Errorf("platform: remove systemd unit: %w", err)
	}
	return ServiceInstallResult{UnitName: "vrooli-runtime-supervisor.service", UnitPath: path, Scope: "user", Active: false}, nil
}

func startInstalledService(options ServiceInstallOptions) (bool, error) {
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return false, err
	}
	path := filepath.Join(home, ".config", "systemd", "user", runtimeSupervisorUnit)
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

func serviceStartHint() string { return "systemctl --user start vrooli-runtime-supervisor.service" }

// systemdUnitContent renders the user unit.
//
// Quoting is per-directive, not uniform, and getting it wrong is fatal rather
// than cosmetic:
//
//   - Environment= and ExecStart= go through systemd's quote-aware parser, so a
//     value containing spaces MUST be quoted.
//   - WorkingDirectory= takes the rest of the line verbatim. A quoted value is
//     read with the quotes as part of the path, which systemd then rejects as
//     "path is not absolute" and the whole unit refuses to load. A path with
//     spaces is fine here unquoted.
//
// See TestSystemdUnitContentLoadsUnderSystemd, which asserts this against real
// systemd rather than against our belief about it.
func systemdUnitContent(executable, home, sourceRoot, logPath string) string {
	sourceRoot = strings.TrimSpace(sourceRoot)
	sourceEnv := ""
	workingDir := ""
	if sourceRoot != "" {
		sourceEnv = fmt.Sprintf("Environment=VROOLI_SOURCE_ROOT=%s\n", strconv.Quote(sourceRoot))
		workingDir = fmt.Sprintf("WorkingDirectory=%s\n", sourceRoot)
	}
	// A daemon whose failures are only visible in a journal nobody reads is a
	// daemon whose failures are invisible. Point it at the same log tree every
	// scenario already writes to, so both launch paths (systemd here, the
	// lifecycle fallback spawn) land in one file.
	logging := ""
	if logPath = strings.TrimSpace(logPath); logPath != "" {
		logging = fmt.Sprintf("StandardOutput=append:%s\nStandardError=append:%s\n", logPath, logPath)
	}
	return fmt.Sprintf("[Unit]\nDescription=Vrooli runtime supervisor\nAfter=default.target\n\n[Service]\nType=simple\nEnvironment=HOME=%s\nEnvironment=VROOLI_RUNTIME_SUPERVISOR=on\n%s%s%sExecStart=%s --no-stale-check runtime supervisor run\nRestart=always\nRestartSec=5s\n\n[Install]\nWantedBy=default.target\n", strconv.Quote(home), sourceEnv, workingDir, logging, strconv.Quote(executable))
}

func resolvedHome(home string) (string, error) {
	if strings.TrimSpace(home) != "" {
		return home, nil
	}
	return HomeDir()
}
