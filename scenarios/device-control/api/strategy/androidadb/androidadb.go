package androidadb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"image"
	// Register JPEG decoder for image.Decode.
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}
type commandRunner struct{}

func (commandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

type Adapter struct {
	runner Runner
	serial string
}

func New() *Adapter {
	return &Adapter{runner: commandRunner{}, serial: strings.TrimSpace(os.Getenv("ANDROID_SERIAL"))}
}
func NewWithRunner(r Runner, serial string) *Adapter { return &Adapter{runner: r, serial: serial} }
func (a *Adapter) ForDevice(serial string) strategy.Strategy {
	return &Adapter{runner: a.runner, serial: strings.TrimSpace(serial)}
}
func (a *Adapter) ID() string { return "android-adb" }
func (a *Adapter) args(args ...string) []string {
	if a.serial != "" {
		return append([]string{"-s", a.serial}, args...)
	}
	return args
}

func (a *Adapter) connected(ctx context.Context) (bool, string) {
	out, err := a.runner.Run(ctx, "adb", a.args("devices")...)
	if err != nil {
		return false, "Install the android-sdk resource (adb/platform-tools) and put adb on PATH."
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "device" {
			return true, ""
		}
	}
	return false, "Enable USB debugging, authorize this computer, and attach an Android device or boot the emulator."
}

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
		model := ""
		for _, field := range fields[2:] {
			if strings.HasPrefix(field, "model:") {
				model = strings.TrimPrefix(field, "model:")
			}
		}
		health, reason := strategy.StatusAvailable, ""
		switch state {
		case "unauthorized":
			health, reason = "unauthorized", "device is present but unauthorized; accept the RSA prompt on the phone"
		case "offline":
			health, reason = strategy.HealthUnreachable, "device is present but offline; set USB mode to File Transfer and replug"
		case "no permissions":
			health, reason = strategy.HealthUnreachable, "insufficient permissions; install the udev rule and replug"
		case "device":
		default:
			health, reason = strategy.HealthUnreachable, "device state is "+state
		}
		digest := sha256.Sum256([]byte(serial))
		osVersion := ""
		if state == "device" {
			if version, versionErr := a.runner.Run(ctx, "adb", "-s", serial, "shell", "getprop", "ro.build.version.release"); versionErr == nil {
				osVersion = strings.TrimSpace(string(version))
			}
		}
		devices = append(devices, strategy.Device{ID: "android-" + hex.EncodeToString(digest[:8]), Serial: serial, Model: model, OSVersion: osVersion, StrategyID: a.ID(), Transport: "usb", Health: health, HealthReason: reason, ObservedAt: now})
	}
	return devices, nil
}

func (a *Adapter) Describe(ctx context.Context) (strategy.Declaration, error) {
	ok, next := a.connected(ctx)
	if !ok {
		return unavailable(next), nil
	}
	caps := map[string]strategy.Capability{}
	probes := []struct {
		name string
		args []string
		next string
	}{
		{strategy.CapInput, []string{"shell", "input", "--help"}, "adb shell input is unavailable; verify the device is authorized"},
		{strategy.CapScreenshot, []string{"exec-out", "screencap", "-p"}, "adb screencap is unavailable; verify the device is authorized"},
		{strategy.CapSemanticTree, []string{"shell", "uiautomator", "help"}, "uiautomator is unavailable on this device image"},
		{strategy.CapAppLifecycle, []string{"shell", "pm", "list", "packages"}, "package manager access is unavailable on this device image"},
		{strategy.CapPermissions, []string{"shell", "pm", "list", "packages"}, "package permission access is unavailable on this device image"},
		{strategy.CapOrientation, []string{"shell", "wm", "size"}, "window manager access is unavailable on this device image"},
		{strategy.CapNetworkControl, []string{"shell", "svc", "wifi"}, "network control is unavailable on this device image"},
		{strategy.CapClipboard, []string{"shell", "cmd", "clipboard", "help"}, "clipboard service is unavailable on this device image"},
		{strategy.CapDeviceLogs, []string{"logcat", "-d", "-t", "1"}, "logcat is unavailable; grant device log access"},
	}
	for _, probe := range probes {
		_, err := a.runner.Run(ctx, "adb", a.args(probe.args...)...)
		caps[probe.name] = strategy.ProbeCapability(probe.name, err == nil, probe.next, probe.next, "adb "+strings.Join(probe.args, " "))
	}
	_, recordingErr := a.runner.Run(ctx, "adb", a.args("shell", "screenrecord", "--help")...)
	caps[strategy.CapNativeRecording] = strategy.ProbeCapability(strategy.CapNativeRecording, recordingErr == nil, "adb screenrecord is unavailable on this device image", "Install a device image with screenrecord support", "adb shell screenrecord --help")
	caps[strategy.CapScreenRecording] = caps[strategy.CapNativeRecording]
	d := strategy.Declaration{StrategyID: a.ID(), Description: "Android phones and emulators through ADB", Status: strategy.StatusAvailable, Capabilities: caps, Promotable: true}
	d.Tiers = strategy.Tiers(d)
	return d, nil
}

func unavailable(next string) strategy.Declaration {
	return strategy.UnavailableDeclaration("android-adb", "Android ADB strategy", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next)
}

func (a *Adapter) Observe(ctx context.Context) (strategy.Frame, error) {
	data, err := a.runner.Run(ctx, "adb", a.args("exec-out", "screencap", "-p")...)
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("adb screencap: %w", err)
	}
	config, _, err := image.DecodeConfig(strings.NewReader(string(data)))
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("decode adb screenshot: %w", err)
	}
	return strategy.Frame{Width: config.Width, Height: config.Height, Scale: 1, Timestamp: time.Now().UTC(), MediaType: "image/png", Bytes: data}, nil
}

func (a *Adapter) Actuate(ctx context.Context, event strategy.Actuation) error {
	if event.Pointer != nil {
		_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "tap", fmt.Sprintf("%d", int(event.Pointer.X)), fmt.Sprintf("%d", int(event.Pointer.Y)))...)
		return err
	}
	if event.Key != nil {
		_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", event.Key.Key)...)
		return err
	}
	if event.Text != "" {
		// adb input text treats %s as a space and interprets shell metacharacters;
		// quote them into the input-text vocabulary before passing one argument.
		text := strings.ReplaceAll(event.Text, "%", "%%")
		text = strings.ReplaceAll(text, " ", "%s")
		_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "text", text)...)
		return err
	}
	if event.Action != "" {
		var args []string
		switch event.Action {
		case "swipe":
			args = append(args, "shell", "input", "swipe")
			args = append(args, strings.Fields(event.Value)...)
		case "install":
			// Reinstall in place so update_migration preserves application data and
			// the same verb is safe when a prior chapter already installed the APK.
			args = append(args, "install", "-r", event.Value)
		case "launch":
			args = append(args, "shell", "monkey", "-p", event.Package, "1")
		case "stop":
			args = append(args, "shell", "am", "force-stop", event.Package)
		case "uninstall":
			args = append(args, "shell", "pm", "uninstall", event.Package)
		case "clear-data":
			args = append(args, "shell", "pm", "clear", event.Package)
		case "grant-permission":
			args = append(args, "shell", "pm", "grant", event.Package, event.Permission)
		case "revoke-permission":
			args = append(args, "shell", "pm", "revoke", event.Package, event.Permission)
		case "rotate":
			rotation := "0"
			if strings.EqualFold(event.Value, "landscape") {
				rotation = "1"
			}
			args = append(args, "shell", "settings", "put", "system", "accelerometer_rotation", "0")
			_, err := a.runner.Run(ctx, "adb", a.args(args...)...)
			if err != nil {
				return err
			}
			_, err = a.runner.Run(ctx, "adb", a.args("shell", "settings", "put", "system", "user_rotation", rotation)...)
			return err
		case "network":
			state := strings.ToLower(strings.TrimSpace(event.Value))
			if state != "offline" && state != "online" {
				return fmt.Errorf("network action requires offline or online")
			}
			enabled := "enable"
			if state == "offline" {
				enabled = "disable"
			}
			_, err := a.runner.Run(ctx, "adb", a.args("shell", "svc", "wifi", enabled)...)
			if err != nil {
				return err
			}
			_, err = a.runner.Run(ctx, "adb", a.args("shell", "svc", "data", enabled)...)
			return err
		case "deep-link":
			if strings.TrimSpace(event.Value) == "" {
				return fmt.Errorf("deep-link action requires a URI")
			}
			args = append(args, "shell", "am", "start", "-a", "android.intent.action.VIEW", "-d", event.Value)
			if event.Package != "" {
				args = append(args, "-p", event.Package)
			}
		case "semantic-target", "semantic-assert":
			_, err := a.runner.Run(ctx, "adb", a.args("shell", "uiautomator", "dump", "/sdcard/window.xml")...)
			if err != nil {
				return err
			}
			tree, err := a.runner.Run(ctx, "adb", a.args("shell", "cat", "/sdcard/window.xml")...)
			if err != nil {
				return err
			}
			values, resolveErr := findSemanticNode(tree, event.Value)
			if resolveErr != nil {
				return resolveErr
			}
			if event.Action == "semantic-assert" {
				if strings.TrimSpace(event.Expected) == "" {
					return fmt.Errorf("semantic assertion for %q requires expected text", event.Value)
				}
				for _, key := range []string{"text", "content-desc", "hint-text"} {
					if values[key] == event.Expected {
						return nil
					}
				}
				return fmt.Errorf("accessibility target %q did not contain expected text %q", event.Value, event.Expected)
			}
			x, y, boundsErr := semanticBounds(values, event.Value)
			if boundsErr != nil {
				return boundsErr
			}
			_, err = a.runner.Run(ctx, "adb", a.args("shell", "input", "tap", strconv.Itoa(x), strconv.Itoa(y))...)
			return err
		case "package-state":
			if strings.TrimSpace(event.Package) == "" {
				return fmt.Errorf("package-state action requires a package")
			}
			state, err := a.runner.Run(ctx, "adb", a.args("shell", "pm", "list", "packages", event.Package)...)
			if err != nil {
				return err
			}
			present := strings.Contains(string(state), "package:"+event.Package)
			expected := strings.ToLower(strings.TrimSpace(event.Value))
			switch expected {
			case "present":
				if !present {
					return fmt.Errorf("package %q is not installed", event.Package)
				}
			case "absent":
				if present {
					return fmt.Errorf("package %q is still installed", event.Package)
				}
			default:
				return fmt.Errorf("package-state action requires expected present or absent")
			}
			return nil
		case "device-logs":
			if event.Package != "" {
				pid, pidErr := a.runner.Run(ctx, "adb", a.args("shell", "pidof", event.Package)...)
				if pidErr != nil || strings.TrimSpace(string(pid)) == "" {
					return fmt.Errorf("device logs: package %q is not running", event.Package)
				}
				args = append(args, "logcat", "-d", "-v", "brief", "--pid", strings.TrimSpace(string(pid)))
			} else {
				args = append(args, "logcat", "-d", "-v", "brief")
			}
		case "screenrecord":
			path := event.Value
			if path == "" {
				path = "/sdcard/device-control-proof.mp4"
			}
			args = append(args, "shell", "screenrecord", "--time-limit", "2", "--bit-rate", "1000000", path)
			_, err := a.runner.Run(ctx, "adb", a.args(args...)...)
			if err != nil {
				return err
			}
			if event.Output == nil {
				return fmt.Errorf("screenrecord output sink is required")
			}
			video, readErr := a.runner.Run(ctx, "adb", a.args("exec-out", "cat", path)...)
			if readErr != nil || len(video) == 0 {
				return fmt.Errorf("screenrecord produced no video")
			}
			*event.Output = video
			return nil
		default:
			return fmt.Errorf("unsupported adb action %q", event.Action)
		}
		out, err := a.runner.Run(ctx, "adb", a.args(args...)...)
		if err == nil && event.Output != nil && event.Action == "device-logs" {
			*event.Output = out
		}
		return err
	}
	return fmt.Errorf("actuation must contain pointer or key event")
}

var boundsPattern = regexp.MustCompile(`^\[(\d+),(\d+)\]\[(\d+),(\d+)\]$`)

// findSemanticNode resolves a stable Android accessibility identifier. The XML
// is device-owned input; only matched attributes and numeric bounds are used.
func findSemanticNode(tree []byte, target string) (map[string]string, error) {
	if strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("accessibility target is required")
	}
	decoder := xml.NewDecoder(strings.NewReader(string(tree)))
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse accessibility hierarchy: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "node" {
			continue
		}
		values := make(map[string]string, len(start.Attr))
		for _, attr := range start.Attr {
			values[attr.Name.Local] = attr.Value
		}
		if !semanticValueMatches(values, target) {
			continue
		}
		return values, nil
	}
	return nil, fmt.Errorf("accessibility target %q was not found in the view hierarchy", target)
}

// resolveSemanticTarget resolves a stable Android accessibility identifier and
// returns the center of its bounds. The XML is device-owned input; only the
// four numeric bounds are carried into the bounded input verb.
func resolveSemanticTarget(tree []byte, target string) (int, int, error) {
	values, err := findSemanticNode(tree, target)
	if err != nil {
		return 0, 0, err
	}
	return semanticBounds(values, target)
}

func semanticBounds(values map[string]string, target string) (int, int, error) {
	match := boundsPattern.FindStringSubmatch(values["bounds"])
	if len(match) != 5 {
		return 0, 0, fmt.Errorf("accessibility target %q has invalid bounds", target)
	}
	x0, _ := strconv.Atoi(match[1])
	y0, _ := strconv.Atoi(match[2])
	x1, _ := strconv.Atoi(match[3])
	y1, _ := strconv.Atoi(match[4])
	return (x0 + x1) / 2, (y0 + y1) / 2, nil
}

func semanticValueMatches(values map[string]string, target string) bool {
	for _, key := range []string{"resource-id", "text", "content-desc", "hint-text"} {
		value := values[key]
		if value == target || strings.HasSuffix(value, "/"+target) || strings.HasSuffix(value, ":"+target) {
			return true
		}
	}
	return false
}
