//go:build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	return os.ReadFile(filepath.Join(filepath.Dir(options.Path), "..", "Logs", options.Name+".log"))
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
	path := filepath.Join(home, "Library", "LaunchAgents", "com.vrooli.runtime-supervisor.plist")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ServiceInstallResult{}, err
	}
	content := launchAgentContent(executable, home, options.SourceRoot)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return ServiceInstallResult{}, err
	}
	domain := launchdDomain(os.Getuid())
	target := domain + "/com.vrooli.runtime-supervisor"
	_ = exec.Command("launchctl", "bootout", target).Run()
	if output, err := exec.Command("launchctl", "bootstrap", domain, path).CombinedOutput(); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: launchctl bootstrap: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if output, err := exec.Command("launchctl", "enable", target).CombinedOutput(); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: launchctl enable: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return ServiceInstallResult{UnitName: "com.vrooli.runtime-supervisor", UnitPath: path, Scope: "user", Active: true}, nil
}

func uninstallService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	path := filepath.Join(home, "Library", "LaunchAgents", "com.vrooli.runtime-supervisor.plist")
	target := fmt.Sprintf("gui/%d/com.vrooli.runtime-supervisor", os.Getuid())
	_ = exec.Command("launchctl", "bootout", target).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return ServiceInstallResult{}, err
	}
	return ServiceInstallResult{UnitName: "com.vrooli.runtime-supervisor", UnitPath: path, Scope: "user", Active: false}, nil
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
	return fmt.Sprintf("launchctl kickstart gui/%d/com.vrooli.runtime-supervisor", os.Getuid())
}

func launchAgentContent(executable, home, sourceRoot string) string {
	sourceRoot = strings.TrimSpace(sourceRoot)
	logPath := filepath.Join(home, "Library", "Logs", "com.vrooli.runtime-supervisor.log")
	escape := func(value string) string {
		return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;").Replace(value)
	}
	args := ""
	for _, arg := range []string{executable, "--no-stale-check", "runtime", "supervisor", "run"} {
		args += fmt.Sprintf("    <string>%s</string>\n", escape(arg))
	}
	extra := ""
	if sourceRoot != "" {
		extra = fmt.Sprintf("    <key>VROOLI_SOURCE_ROOT</key>\n    <string>%s</string>\n", escape(sourceRoot))
	}
	working := ""
	if sourceRoot != "" {
		working = fmt.Sprintf("  <key>WorkingDirectory</key>\n  <string>%s</string>\n", escape(sourceRoot))
	}
	return fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\"><dict>\n  <key>Label</key><string>com.vrooli.runtime-supervisor</string>\n  <key>ProgramArguments</key><array>\n%s  </array>\n  <key>EnvironmentVariables</key><dict>\n    <key>HOME</key><string>%s</string>\n    <key>VROOLI_RUNTIME_SUPERVISOR</key><string>on</string>\n%s  </dict>\n%s  <key>RunAtLoad</key><true/>\n  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>\n  <key>ThrottleInterval</key><integer>5</integer>\n  <key>StandardOutPath</key><string>%s</string>\n  <key>StandardErrorPath</key><string>%s</string>\n</dict></plist>\n", args, escape(home), extra, working, escape(logPath), escape(logPath))
}

func resolvedHome(home string) (string, error) {
	if strings.TrimSpace(home) != "" {
		return home, nil
	}
	return HomeDir()
}
