//go:build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func nativeLaunchctl(options NativeServiceOptions, action string) ([]byte, error) {
	domain := launchdDomain(os.Getuid())
	target := domain + "/" + options.Name
	args := []string{action}
	switch action {
	case "bootstrap":
		args = append(args, domain, options.Path)
	case "bootout":
		args = append(args, target)
	case "print":
		args = append(args, target)
	default:
		args = append(args, options.Name)
	}
	return exec.Command("launchctl", args...).CombinedOutput()
}

func installNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	if options.Name == "" || options.Path == "" {
		return NativeServiceResult{}, fmt.Errorf("platform: service name and path are required")
	}
	if err := os.MkdirAll(filepath.Dir(options.Path), 0o755); err != nil {
		return NativeServiceResult{}, err
	}
	if err := os.WriteFile(options.Path, []byte(options.Content), 0o644); err != nil {
		return NativeServiceResult{}, err
	}
	if output, err := nativeLaunchctl(options, "bootstrap"); err != nil && !strings.Contains(strings.ToLower(string(output)), "already bootstrapped") {
		return NativeServiceResult{}, fmt.Errorf("platform: launchctl bootstrap: %w: %s", err, output)
	}
	return NativeServiceResult{Name: options.Name, Path: options.Path, Scope: "user", Running: true, Enabled: true, State: ServiceStateRunning, Evidence: ServiceEvidence{Source: "launchctl", RawState: "running"}}, nil
}

func uninstallNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	_, _ = nativeLaunchctl(options, "bootout")
	if options.Path != "" {
		if err := os.Remove(options.Path); err != nil && !os.IsNotExist(err) {
			return NativeServiceResult{}, err
		}
	}
	return NativeServiceResult{Name: options.Name, Path: options.Path, Scope: "user", State: ServiceStateStopped, Evidence: ServiceEvidence{Source: "launchctl", RawState: "stopped"}}, nil
}

func startNativeService(options NativeServiceOptions) error {
	_, err := nativeLaunchctl(options, "start")
	return err
}

func stopNativeService(options NativeServiceOptions) error {
	_, err := nativeLaunchctl(options, "stop")
	return err
}

func restartNativeService(options NativeServiceOptions) error {
	_ = stopNativeService(options)
	return startNativeService(options)
}

func nativeServiceStatus(options NativeServiceOptions) (NativeServiceResult, error) {
	output, err := nativeLaunchctl(options, "print")
	state := parseLaunchctlState(string(output), err)
	return NativeServiceResult{Name: options.Name, Path: options.Path, Scope: "user", Running: state == ServiceStateRunning, Enabled: options.Path != "", State: state, Evidence: ServiceEvidence{Source: "launchctl", RawState: strings.TrimSpace(string(output))}}, nil
}

func parseLaunchctlState(raw string, commandErr error) ServiceState {
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "state = running") || strings.Contains(lower, "state = active") {
		return ServiceStateRunning
	}
	if strings.Contains(lower, "state = exited") || strings.Contains(lower, "state = stopped") || strings.Contains(lower, "could not find service") {
		return ServiceStateStopped
	}
	if commandErr != nil {
		return ServiceStateUnknown
	}
	return ServiceStateUnknown
}

func nativeServiceLogs(options NativeServiceOptions, tail int) ([]byte, error) {
	return os.ReadFile(LaunchdLogPath(options.Path, options.Name))
}

func readHostLogs(options HostLogOptions) (HostLogResult, error) {
	args := []string{"show", "--style", "ndjson"}
	if options.Since != "" {
		args = append(args, "--last", options.Since)
	}
	if options.Unit != "" {
		args = append(args, "--predicate", fmt.Sprintf("process == '%s'", options.Unit))
	}
	if options.Tail > 0 {
		args = append(args, "--last", fmt.Sprintf("%d", options.Tail))
	}
	out, err := exec.Command("log", args...).CombinedOutput()
	return HostLogResult{Source: "log show", Raw: out, Entries: ParseMacOSNDJSON(out), Evidence: ServiceEvidence{Source: "log show", Detail: strings.TrimSpace(string(out))}}, err
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
	path := RuntimeSupervisorUnitPath("darwin", home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ServiceInstallResult{}, err
	}
	definition, err := RuntimeSupervisorDefinition("darwin", RuntimeSupervisorOptions{Home: home, Executable: executable, SourceRoot: options.SourceRoot, LogPath: options.LogPath})
	if err != nil {
		return ServiceInstallResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(definition.Logs.Stdout), 0o755); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: create supervisor log dir: %w", err)
	}
	artifact, err := RenderLaunchd(definition)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	result := ServiceInstallResult{UnitName: runtimeSupervisorLabel, UnitPath: path, Scope: "user"}
	result.Verdict = ValidateLaunchd(artifact)
	if result.Verdict.Rejected() {
		return result, fmt.Errorf("platform: plutil rejected the rendered %s: %s", runtimeSupervisorLabel, result.Verdict.Output)
	}
	if err := os.WriteFile(path, []byte(artifact.Primary().Content), 0o644); err != nil {
		return result, err
	}
	domain := launchdDomain(os.Getuid())
	target := domain + "/" + runtimeSupervisorLabel
	_ = exec.Command("launchctl", "bootout", target).Run()
	if output, err := exec.Command("launchctl", "bootstrap", domain, path).CombinedOutput(); err != nil {
		return result, fmt.Errorf("platform: launchctl bootstrap: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if output, err := exec.Command("launchctl", "enable", target).CombinedOutput(); err != nil {
		return result, fmt.Errorf("platform: launchctl enable: %w: %s", err, strings.TrimSpace(string(output)))
	}
	// Prove the agent is actually running rather than merely bootstrapped:
	// a plist launchd accepts but cannot execute leaves the install looking
	// successful while nothing supervises the fleet.
	if state := awaitLaunchdRunning(target); state != ServiceStateRunning {
		output, _ := exec.Command("launchctl", "print", target).CombinedOutput()
		return result, fmt.Errorf("platform: installed %s but it is %s, not running:\n%s", runtimeSupervisorLabel, state, strings.TrimSpace(string(output)))
	}
	result.Active = true
	return result, nil
}

const runtimeSupervisorLabel = "com.vrooli.runtime-supervisor"

// awaitLaunchdRunning polls launchctl print until the agent settles, reporting
// the last observation so a start-then-crash is not mistaken for a start.
func awaitLaunchdRunning(target string) ServiceState {
	state := ServiceStateUnknown
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			time.Sleep(250 * time.Millisecond)
		}
		output, err := exec.Command("launchctl", "print", target).CombinedOutput()
		state = ParseNativeServiceState("macos", string(output), err != nil)
	}
	return state
}

func uninstallService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	path := RuntimeSupervisorUnitPath("darwin", home)
	target := launchdDomain(os.Getuid()) + "/" + runtimeSupervisorLabel
	_ = exec.Command("launchctl", "bootout", target).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return ServiceInstallResult{}, err
	}
	return ServiceInstallResult{UnitName: runtimeSupervisorLabel, UnitPath: path, Scope: "user", Active: false}, nil
}

func startInstalledService(options ServiceInstallOptions) (bool, error) {
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return false, err
	}
	path := RuntimeSupervisorUnitPath("darwin", home)
	if _, err := os.Stat(path); err != nil {
		return false, nil
	}
	target := launchdDomain(os.Getuid()) + "/" + runtimeSupervisorLabel
	if output, err := exec.Command("launchctl", "kickstart", target).CombinedOutput(); err != nil {
		return false, fmt.Errorf("platform: launchctl kickstart: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if state := awaitLaunchdRunning(target); state != ServiceStateRunning {
		return false, fmt.Errorf("platform: started %s but it is %s, not running", runtimeSupervisorLabel, state)
	}
	return true, nil
}

func supportsService(user bool) bool {
	return user && exec.Command("launchctl", "version").Run() == nil
}

// launchdDomain selects the namespace that is actually available to the
// current login. SSH-only/headless macOS sessions do not have a gui/<uid>
// bootstrap, but they do expose user/<uid>; using the latter keeps user-level
// agents installable without requiring a graphical login.
func launchdDomain(uid int) string {
	gui := fmt.Sprintf("gui/%d", uid)
	if exec.Command("launchctl", "print", gui).Run() == nil {
		return gui
	}
	return fmt.Sprintf("user/%d", uid)
}

func serviceStartHint() string {
	return fmt.Sprintf("launchctl kickstart gui/%d/%s", os.Getuid(), runtimeSupervisorLabel)
}
