package androidadb

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

func (a *Adapter) ReadApplicationLogs(ctx context.Context, pkg string) ([]byte, error) {
	if len(pkg) > 256 || !applicationIdentityPattern.MatchString(pkg) {
		return nil, fmt.Errorf("application logs require one exact package identity")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pid, err := a.runner.Run(probeCtx, "adb", a.args("shell", "pidof", pkg)...)
	if err != nil {
		return nil, &strategy.EvidenceUnavailableError{Reason: "application process could not be located; no conclusion about its logs"}
	}
	fields := strings.Fields(string(pid))
	if len(fields) != 1 {
		return nil, &strategy.EvidenceUnavailableError{Reason: "application process identity is unavailable or ambiguous"}
	}
	if _, err := strconv.ParseUint(fields[0], 10, 32); err != nil {
		return nil, fmt.Errorf("application process identity is invalid")
	}
	bounded, ok := a.runner.(boundedRunner)
	if !ok {
		return nil, fmt.Errorf("bounded application log reading is unavailable")
	}
	return bounded.RunBounded(probeCtx, "adb", 256*1024, a.args("logcat", "-d", "-v", "epoch", "-t", "500", "--pid", fields[0])...)
}

// AppLifecycle exposes the bounded subset of Android lifecycle commands that
// this adapter can execute without a shell. The control plane validates the
// package identity and confirmation policy before reaching this method.
func (a *Adapter) AppLifecycle(ctx context.Context, request strategy.AppLifecycleRequest) (strategy.AppLifecycleResult, error) {
	operation := strings.ToLower(strings.TrimSpace(request.Operation))
	if operation == "network-state" {
		if request.Package != "" || len(request.Arguments) != 0 || request.Executable != "" {
			return strategy.AppLifecycleResult{}, fmt.Errorf("android network-state does not accept a package identity or command")
		}
		state, err := a.readNetworkState(ctx)
		if err != nil {
			return strategy.AppLifecycleResult{Operation: operation, Status: strategy.StatusUnavailable, Reason: err.Error()}, err
		}
		return strategy.AppLifecycleResult{Operation: operation, Status: strategy.StatusAvailable, NetworkState: state, Reason: "Android Wi-Fi, mobile-data, and airplane-mode settings were read"}, nil
	}
	if !applicationIdentityPattern.MatchString(request.Package) || len(request.Package) > 256 || len(request.Arguments) != 0 || request.Executable != "" {
		return strategy.AppLifecycleResult{}, fmt.Errorf("android lifecycle requires one bounded package identity")
	}
	if operation == "home-set" {
		return a.setHome(ctx, request)
	}
	if operation == "home-state" {
		return a.homeState(ctx, request.Package)
	}
	if operation == "accessibility-state" {
		return a.accessibilityState(ctx, request)
	}
	if operation == "foreground-state" {
		return a.foregroundState(ctx, request.Package)
	}
	if operation == "process-state" {
		return a.processState(ctx, request.Package)
	}
	if operation == "app-status" {
		return a.applicationStatus(ctx, request.Package)
	}
	if operation == "performance-state" {
		return a.performanceState(ctx, request.Package)
	}
	if operation == "package-enabled-set" {
		return a.setPackageEnabled(ctx, request)
	}
	supported := map[string]bool{"launch": true, "focus": true, "close": true, "minimize": true, "restore": true, "stop": true, "uninstall": true, "clear-data": true, "grant-permission": true, "revoke-permission": true, "package-state": true, "package-signing-state": true, "network-state": true}
	if !supported[operation] {
		return strategy.AppLifecycleResult{}, &strategy.UnsupportedCapabilityError{Capability: strategy.CapAppLifecycle, Operation: operation}
	}
	if operation == "package-state" {
		installed, version, versionCode, err := a.packageState(ctx, request.Package)
		if err != nil {
			return strategy.AppLifecycleResult{Operation: operation, Package: request.Package, Status: strategy.StatusUnavailable, Reason: err.Error()}, err
		}
		result := strategy.AppLifecycleResult{Operation: operation, Package: request.Package, Status: strategy.StatusAvailable, Installed: installed, VersionCode: versionCode}
		if installed {
			result.Version = version
			result.Reason = "package is installed"
		} else {
			result.Reason = "package is not installed"
		}
		return result, nil
	}
	if operation == "package-signing-state" {
		installed, digest, err := a.packageSigningState(ctx, request.Package)
		if err != nil {
			return strategy.AppLifecycleResult{Operation: operation, Package: request.Package, Status: strategy.StatusUnavailable, Reason: err.Error()}, err
		}
		result := strategy.AppLifecycleResult{Operation: operation, Package: request.Package, Status: strategy.StatusAvailable, Installed: installed, SigningCertificateSHA256: digest}
		if installed {
			result.Reason = "installed package signing certificate was read back"
		} else {
			result.Reason = "package is not installed"
		}
		return result, nil
	}
	event := strategy.Actuation{Action: operation, Package: request.Package, Permission: request.Permission, Value: request.Value}
	switch operation {
	case "focus", "restore":
		event.Action = "launch"
	case "close":
		event.Action = "stop"
	case "minimize":
		event.Action = ""
		event.Key = &strategy.KeyEvent{Kind: "press", Key: "HOME"}
	}
	var output []byte
	event.Output = &output
	if err := a.Actuate(ctx, event); err != nil {
		return strategy.AppLifecycleResult{Operation: operation, Package: request.Package, Status: strategy.StatusUnavailable, Reason: err.Error()}, err
	}
	result := strategy.AppLifecycleResult{Operation: operation, Package: request.Package, Status: strategy.StatusAvailable, Outcome: "accepted", Diagnostic: boundedDiagnostic(output)}
	if event.Action == "launch" {
		// A successful `am start` can leave Ambient or another task visible.
		// Retain that distinction rather than claiming the app is ready.
		observeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		for {
			probeCtx, cancelProbe := context.WithTimeout(observeCtx, 2*time.Second)
			activity, _ := a.runner.Run(probeCtx, "adb", a.args("shell", "dumpsys", "activity", "activities")...)
			cancelProbe()
			foreground := foregroundPackage(string(activity))
			if foreground == "" {
				window, windowErr := a.runner.Run(observeCtx, "adb", a.args("shell", "dumpsys", "window")...)
				if windowErr == nil {
					foreground = focusedPackage(string(window))
				}
			}
			result.ForegroundPackage = foreground
			if foreground != "" {
				result.ObservedAt = time.Now().UTC()
			}
			if foreground == request.Package {
				result.Outcome = "foreground_observed"
				break
			}
			if err := waitForUnlockSettle(observeCtx, 200*time.Millisecond); err != nil {
				result.Reason = "launch accepted; requested foreground was not observed before the deadline"
				break
			}
		}
	}
	return result, nil
}

func boundedDiagnostic(output []byte) string {
	if len(output) > 4096 {
		output = output[:4096]
	}
	return strings.TrimSpace(string(output))
}

var launchComponentPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.$]*/[A-Za-z0-9_.$]+$`)

func (a *Adapter) launchPackage(ctx context.Context, pkg string) ([]byte, error) {
	if len(pkg) > 256 || !applicationIdentityPattern.MatchString(pkg) {
		return nil, fmt.Errorf("launch requires a bounded application identity")
	}
	for _, category := range []string{"android.intent.category.LEANBACK_LAUNCHER", "android.intent.category.LAUNCHER"} {
		resolved, err := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "package", "resolve-activity", "--brief", "-a", "android.intent.action.MAIN", "-c", category, "-p", pkg)...)
		if err != nil {
			return resolved, fmt.Errorf("resolve launch activity: %w: %s", err, boundedDiagnostic(resolved))
		}
		for _, line := range strings.Split(string(resolved), "\n") {
			component := strings.TrimSpace(line)
			if !strings.HasPrefix(component, pkg+"/") || !launchComponentPattern.MatchString(component) {
				continue
			}
			output, err := a.runner.Run(ctx, "adb", a.args("shell", "am", "start", "-W", "-n", component)...)
			if err != nil {
				return output, fmt.Errorf("start launch activity: %w: %s", err, boundedDiagnostic(output))
			}
			diagnostic := strings.ToLower(string(output))
			if strings.Contains(diagnostic, "error:") || strings.Contains(diagnostic, "exception") || strings.Contains(diagnostic, "status: timeout") {
				return output, fmt.Errorf("Android did not start the launch activity: %s", boundedDiagnostic(output))
			}
			return output, nil
		}
	}
	return nil, fmt.Errorf("package %q has no resolvable TV or launcher activity", pkg)
}

func focusedPackage(output string) string {
	match := regexp.MustCompile(`mCurrentFocus=Window\{[^\n]*?\s([A-Za-z0-9._]+)/[A-Za-z0-9.$_]+`).FindStringSubmatch(output)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

var packageVersionPattern = regexp.MustCompile(`versionName=(\S+)`)

var packageVersionCodePattern = regexp.MustCompile(`versionCode=([0-9]+)`)

// packageState reports whether a package is installed on the target and, when
// present, its versionName. It matches the exact package line (not a prefix) so
// `com.foo` is never reported present because `com.foobar` is installed.
func (a *Adapter) packageState(ctx context.Context, pkg string) (bool, string, int64, error) {
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "pm", "list", "packages", pkg)...)
	if err != nil {
		return false, "", 0, fmt.Errorf("query package state: %w", err)
	}
	installed := false
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == "package:"+pkg {
			installed = true
			break
		}
	}
	if !installed {
		return false, "", 0, nil
	}
	version := ""
	var versionCode int64
	if info, infoErr := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "package", pkg)...); infoErr == nil {
		if match := packageVersionPattern.FindStringSubmatch(string(info)); len(match) == 2 {
			version = match[1]
		}
		if match := packageVersionCodePattern.FindStringSubmatch(string(info)); len(match) == 2 {
			versionCode, _ = strconv.ParseInt(match[1], 10, 64)
		}
	}
	return true, version, versionCode, nil
}

var installedAPKPathPattern = regexp.MustCompile(`^/data/app/[A-Za-z0-9_./=~-]+/base\.apk$`)

// packageSigningState reads the exact installed base APK through the bound
// ADB strategy, then inspects its signer with the Android SDK build-tools.
func (a *Adapter) packageSigningState(ctx context.Context, pkg string) (bool, string, error) {
	installed, _, _, err := a.packageState(ctx, pkg)
	if err != nil || !installed {
		return installed, "", err
	}
	paths, err := a.runner.Run(ctx, "adb", a.args("shell", "pm", "path", pkg)...)
	if err != nil {
		return false, "", fmt.Errorf("query installed APK path: %w", err)
	}
	var baseAPK string
	for _, line := range strings.Split(string(paths), "\n") {
		path := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "package:"))
		if !strings.HasSuffix(path, "/base.apk") {
			continue
		}
		if !installedAPKPathPattern.MatchString(path) || strings.Contains(path, "..") {
			return false, "", fmt.Errorf("installed APK path is not a safe package-manager path")
		}
		if baseAPK != "" {
			return false, "", fmt.Errorf("package manager returned multiple base APK paths")
		}
		baseAPK = path
	}
	if baseAPK == "" {
		return false, "", fmt.Errorf("package manager did not return an installed base APK path")
	}
	localAPK, err := os.CreateTemp("", "device-control-installed-*.apk")
	if err != nil {
		return false, "", fmt.Errorf("create installed APK signer inspection file: %w", err)
	}
	localPath := localAPK.Name()
	if err := localAPK.Close(); err != nil {
		_ = os.Remove(localPath)
		return false, "", fmt.Errorf("close installed APK signer inspection file: %w", err)
	}
	defer os.Remove(localPath)
	if _, err := a.runner.Run(ctx, "adb", a.args("pull", baseAPK, localPath)...); err != nil {
		return false, "", fmt.Errorf("read installed APK for signer inspection: %w", err)
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return false, "", fmt.Errorf("stat installed APK for signer inspection: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 1<<30 {
		return false, "", fmt.Errorf("installed APK size or file type is outside signer inspection limits")
	}
	if a.signingInspector == nil {
		return false, "", fmt.Errorf("Android APK signing inspector is unavailable")
	}
	digest, err := a.signingInspector.InspectSigningCertificate(ctx, localPath)
	if err != nil {
		return false, "", fmt.Errorf("inspect installed APK signer: %w", err)
	}
	return true, strings.ToLower(strings.TrimSpace(digest)), nil
}
