package androidadb

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
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
func (a *Adapter) ID() string                        { return "android-adb" }
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
func (a *Adapter) Describe(ctx context.Context) (strategy.Declaration, error) {
	ok, next := a.connected(ctx)
	if !ok {
		return unavailable(next), nil
	}
	caps := map[string]strategy.Capability{}
	for _, n := range []string{strategy.CapInput, strategy.CapScreenshot, strategy.CapSemanticTree, strategy.CapAppLifecycle, strategy.CapPermissions, strategy.CapOrientation, strategy.CapClipboard, strategy.CapDeviceLogs} {
		caps[n] = strategy.ProbeCapability(n, true, "", "adb probe succeeded", "adb shell probe")
	}
	caps[strategy.CapNativeRecording] = strategy.ProbeCapability(strategy.CapNativeRecording, true, "", "", "adb screenrecord probe")
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
	return fmt.Errorf("actuation must contain pointer or key event")
}
