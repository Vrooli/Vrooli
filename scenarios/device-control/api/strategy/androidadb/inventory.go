package androidadb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"device-control/strategy"
)

func (a *Adapter) Enumerate(ctx context.Context) ([]strategy.Device, error) {
	out, err := a.runner.Run(ctx, "adb", "devices", "-l")
	if err != nil {
		return nil, fmt.Errorf("adb device enumeration: %w", err)
	}
	now := time.Now().UTC()
	devices := make([]strategy.Device, 0)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || fields[0] == "List" {
			continue
		}
		serial, state := fields[0], fields[1]
		if a.endpoint != "" && serial != a.endpoint {
			continue
		}
		identitySerial := serial
		if a.transport == "wireless" && a.endpoint != "" && serial == a.endpoint && a.identitySerial != "" {
			identitySerial = a.identitySerial
		}
		model := ""
		for _, field := range fields[2:] {
			if strings.HasPrefix(field, "model:") {
				model = strings.TrimPrefix(field, "model:")
			}
		}
		health, reason := strategy.StatusAvailable, ""
		switch state {
		case "unauthorized":
			health, reason = "unauthorized", "device is present but unauthorized; accept the Allow USB debugging prompt on the device screen (choose Always allow)"
		case "offline":
			health, reason = strategy.HealthUnreachable, "device is present but offline; verify Android wireless debugging and network reachability"
			if a.transport != "wireless" {
				reason = "device is present but offline; set USB mode to File Transfer and replug"
			}
		case "no permissions":
			health, reason = strategy.HealthUnreachable, "insufficient permissions; verify the host authorization and Android debugging state"
			if a.transport != "wireless" {
				reason = "insufficient permissions; install the udev rule and replug"
			}
		case "device":
		default:
			health, reason = strategy.HealthUnreachable, "device state is "+state
		}
		endpoint := serial
		if state == "device" && strings.Contains(serial, ":") && !strings.HasPrefix(strings.ToLower(serial), "emulator-") {
			selector := serial
			if a.endpoint != "" && serial == a.endpoint {
				selector = a.endpoint
			}
			resolved, resolveErr := a.runner.Run(ctx, "adb", "-s", selector, "shell", "getprop", "ro.serialno")
			if resolveErr != nil || strings.TrimSpace(string(resolved)) == "" {
				if a.identitySerial == "" || selector != a.endpoint {
					// Do not mint a durable identity from a transport endpoint.
					continue
				}
				identitySerial = a.identitySerial
			} else {
				identitySerial = strings.TrimSpace(string(resolved))
			}
		}
		if a.endpoint != "" && serial == a.endpoint && a.identitySerial != "" {
			identitySerial = a.identitySerial
		}
		digest := sha256.Sum256([]byte(identitySerial))
		osVersion := ""
		if state == "device" {
			selector := serial
			if a.endpoint != "" && serial == a.endpoint {
				selector = a.endpoint
			}
			if version, versionErr := a.runner.Run(ctx, "adb", "-s", selector, "shell", "getprop", "ro.build.version.release"); versionErr == nil {
				osVersion = strings.TrimSpace(string(version))
			}
		}
		devices = append(devices, strategy.Device{ID: "android-" + hex.EncodeToString(digest[:8]), Serial: identitySerial, IdentityKey: identitySerial, IdentityKind: "adb-serial", Endpoint: endpoint, Model: model, OSVersion: osVersion, StrategyID: a.ID(), Transport: transportForSerial(serial), Health: health, HealthReason: reason, ObservedAt: now})
	}
	return devices, nil
}

func (a *Adapter) DescribePerformance(ctx context.Context) (strategy.Declaration, error) {
	return a.DescribeCapture(ctx, "performance")
}

func (a *Adapter) DescribeCapture(ctx context.Context, kind string) (strategy.Declaration, error) {
	if unsupported, ok := strategy.ResolveHostSupport(a.ID(), "Android phones and emulators through ADB", []string{"linux", "darwin", "windows"}); ok {
		return unsupported, nil
	}
	if ok, next := a.connected(ctx); !ok {
		return strategy.WithSupportedHostOS(unavailable(next), "linux", "darwin", "windows"), nil
	}
	caps := map[string]strategy.Capability{}
	switch kind {
	case "performance":
		_, err := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "SurfaceFlinger", "--list")...)
		caps[strategy.CapAppLifecycle] = strategy.ProbeCapability(strategy.CapAppLifecycle, err == nil, "Android performance diagnostics are unavailable", "Check device diagnostics access", "adb shell dumpsys SurfaceFlinger --list")
	case "screenshot":
		available, next := a.probeScreenshot(ctx)
		caps[strategy.CapScreenshot] = strategy.ProbeCapability(strategy.CapScreenshot, available, next, next, "adb exec-out screencap -p")
	case "recording":
		_, err := a.runner.Run(ctx, "adb", a.args("shell", "screenrecord", "--help")...)
		caps[strategy.CapNativeRecording] = strategy.ProbeCapability(strategy.CapNativeRecording, err == nil, "adb screenrecord is unavailable", "Install a device image with screenrecord support", "adb shell screenrecord --help")
		caps[strategy.CapScreenRecording] = caps[strategy.CapNativeRecording]
		caps[strategy.CapRecordingAudio] = a.probeAudioCapture(ctx)
	case "logs":
		_, err := a.runner.Run(ctx, "adb", a.args("logcat", "-d", "-t", "1")...)
		caps[strategy.CapDeviceLogs] = strategy.ProbeCapability(strategy.CapDeviceLogs, err == nil, "logcat is unavailable", "Check device log access", "adb logcat -d -t 1")
	default:
		return strategy.Declaration{}, fmt.Errorf("unsupported capture modality %q", kind)
	}
	return strategy.WithSupportedHostOS(strategy.Declaration{StrategyID: a.ID(), Status: strategy.StatusAvailable, Capabilities: caps}, "linux", "darwin", "windows"), nil
}

func (a *Adapter) Describe(ctx context.Context) (strategy.Declaration, error) {
	if unsupported, ok := strategy.ResolveHostSupport(a.ID(), "Android phones and emulators through ADB", []string{"linux", "darwin", "windows"}); ok {
		return unsupported, nil
	}
	ok, next := a.connected(ctx)
	if !ok {
		return strategy.WithSupportedHostOS(unavailable(next), "linux", "darwin", "windows"), nil
	}
	caps := map[string]strategy.Capability{}
	probes := []struct {
		name string
		args []string
		next string
	}{
		{strategy.CapDeviceRestart, []string{"shell", "getprop", "sys.boot_completed"}, "restart requires an authorized, boot-complete Android device"},
		{strategy.CapInput, []string{"shell", "input", "--help"}, "adb shell input is unavailable; verify the device is authorized"},
		{strategy.CapScreenshot, []string{"exec-out", "screencap", "-p"}, "adb screencap is unavailable; verify the device is authorized"},
		{strategy.CapSemanticTree, []string{"shell", "uiautomator", "help"}, "uiautomator is unavailable on this device image"},
		{strategy.CapAppLifecycle, []string{"shell", "pm", "list", "packages"}, "package manager access is unavailable on this device image"},
		{strategy.CapPermissions, []string{"shell", "pm", "list", "packages"}, "package permission access is unavailable on this device image"},
		{strategy.CapOrientation, []string{"shell", "wm", "size"}, "window manager access is unavailable on this device image"},
		{strategy.CapNetworkControl, []string{"shell", "svc", "wifi"}, "network control is unavailable on this device image"},
		{strategy.CapClipboard, []string{"shell", "cmd", "clipboard", "help"}, "clipboard service is unavailable on this device image"},
		{strategy.CapDeviceLogs, []string{"logcat", "-d", "-t", "1"}, "logcat is unavailable; grant device log access"},
		{strategy.CapReverseTCP, []string{"reverse", "--list"}, "ADB reverse TCP forwarding is unavailable on this selected transport"},
	}
	for _, probe := range probes {
		if probe.name == strategy.CapDeviceRestart {
			out, err := a.runner.Run(ctx, "adb", a.args(probe.args...)...)
			caps[probe.name] = strategy.ProbeCapability(probe.name, err == nil && strings.TrimSpace(string(out)) == "1", probe.next, probe.next, "adb shell getprop sys.boot_completed; restart acceptance still requires execution")
			continue
		}
		if probe.name == strategy.CapNetworkControl {
			available, evidence := a.probeNetworkControl(ctx)
			caps[probe.name] = strategy.ProbeCapability(probe.name, available, probe.next, probe.next, evidence)
			continue
		}
		if probe.name == strategy.CapScreenshot {
			// A non-zero exit is not the only screenshot failure: some firmware
			// images stream a valid container prefixed with log noise, and
			// others return non-image bytes. Require an actually decodable frame
			// so the declaration cannot claim screenshot support the observer
			// path would then reject with "image: unknown format".
			available, next := a.probeScreenshot(ctx)
			caps[probe.name] = strategy.ProbeCapability(probe.name, available, next, next, "adb exec-out screencap -p")
			continue
		}
		_, err := a.runner.Run(ctx, "adb", a.args(probe.args...)...)
		caps[probe.name] = strategy.ProbeCapability(probe.name, err == nil, probe.next, probe.next, "adb "+strings.Join(probe.args, " "))
	}
	screenCap := caps[strategy.CapInput]
	screenCap.Name = strategy.CapScreenControl
	caps[strategy.CapScreenControl] = screenCap
	debugState := a.readWirelessDebuggingState(ctx)
	nativeAvailable := a.transport == "usb" && debugState != nil && debugState.NativeSupported != nil && *debugState.NativeSupported
	caps[strategy.CapWirelessDebugging] = strategy.ProbeCapability(strategy.CapWirelessDebugging, nativeAvailable, "native wireless debugging requires a supported Android version and an authorized USB connection", "Connect the device over USB and authorize debugging; use legacy wireless promotion on older Android versions", "Android SDK level and device class; execution must verify the setting")
	_, recordingErr := a.runner.Run(ctx, "adb", a.args("shell", "screenrecord", "--help")...)
	caps[strategy.CapNativeRecording] = strategy.ProbeCapability(strategy.CapNativeRecording, recordingErr == nil, "adb screenrecord is unavailable on this device image", "Install a device image with screenrecord support", "adb shell screenrecord --help")
	caps[strategy.CapScreenRecording] = caps[strategy.CapNativeRecording]
	caps[strategy.CapRecordingAudio] = a.probeAudioCapture(ctx)
	sockets, socketErr := a.runner.Run(ctx, "adb", a.args("shell", "cat", "/proc/net/unix")...)
	hasWebViewSocket := socketErr == nil && strings.Contains(string(sockets), "webview_devtools_remote")
	caps[strategy.CapWebViewAttach] = strategy.ProbeCapability(strategy.CapWebViewAttach, hasWebViewSocket, "No running Android WebView exposes a debuggable socket", "Launch the target WebView with debugging enabled, then probe again", "adb shell cat /proc/net/unix")
	d := strategy.Declaration{StrategyID: a.ID(), Description: "Android phones and emulators through ADB", Status: strategy.StatusAvailable, Capabilities: caps, Promotable: true, EvidenceClass: "release-grade", MinimumUsefulFPS: 5}
	d = strategy.WithSupportedHostOS(d, "linux", "darwin", "windows")
	d.Tiers = strategy.Tiers(d)
	return d, nil
}

// probeNetworkControl mirrors the fallback used by Actuate. Some Android
// images return a non-zero usage status for `svc wifi` without an action even
// though the connectivity service exposes the supported airplane-mode
// control. The fallback is a read-only capability probe; it does not change
// network state.
func (a *Adapter) probeNetworkControl(ctx context.Context) (bool, string) {
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "svc", "wifi")...); err == nil {
		return true, "adb shell svc wifi"
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "connectivity", "airplane-mode")...); err == nil {
		return true, "adb shell cmd connectivity airplane-mode"
	}
	return false, "adb shell svc wifi; adb shell cmd connectivity airplane-mode"
}

func unavailable(next string) strategy.Declaration {
	return strategy.UnavailableDeclaration("android-adb", "Android ADB strategy", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next)
}

func (a *Adapter) RestartDevice(ctx context.Context) error {
	if strings.TrimSpace(a.serial) == "" && strings.TrimSpace(a.endpoint) == "" {
		return fmt.Errorf("restart requires one exact bound Android device")
	}
	_, err := a.runner.Run(ctx, "adb", a.args("reboot")...)
	return err
}
