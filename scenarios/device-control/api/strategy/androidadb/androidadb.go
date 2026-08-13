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
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"device-control/strategy"
)

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type Process interface {
	Start() error
	Interrupt() error
	Kill() error
	Wait() error
}

type commandProcess struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func (p *commandProcess) Start() error {
	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		return err
	}
	p.stdin = stdin
	return p.cmd.Start()
}
func (p *commandProcess) Interrupt() error {
	if p.stdin == nil {
		return fmt.Errorf("process has not started")
	}
	_, err := p.stdin.Write([]byte{3})
	return err
}
func (p *commandProcess) Kill() error {
	if p.cmd.Process == nil {
		return fmt.Errorf("process has not started")
	}
	return p.cmd.Process.Kill()
}
func (p *commandProcess) Wait() error { return p.cmd.Wait() }

type ProcessRunner interface {
	Start(string, ...string) (Process, error)
}
type commandRunner struct{}

const wirelessADBPort = "5555"

func (commandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

func (commandRunner) Start(name string, args ...string) (Process, error) {
	return &commandProcess{cmd: exec.Command(name, args...)}, nil
}

type activeRecording struct {
	handle  strategy.RecordingHandle
	process Process
	path    string
}

type Adapter struct {
	runner         Runner
	serial         string
	identitySerial string
	endpoint       string
	transport      string
	recordingMu    sync.Mutex
	recordings     map[string]activeRecording
}

func New() *Adapter {
	serial := strings.TrimSpace(os.Getenv("ANDROID_SERIAL"))
	return &Adapter{runner: commandRunner{}, serial: serial, identitySerial: serial, transport: transportForSerial(serial), recordings: map[string]activeRecording{}}
}
func NewWithRunner(r Runner, serial string) *Adapter {
	return &Adapter{runner: r, serial: serial, identitySerial: serial, transport: transportForSerial(serial), recordings: map[string]activeRecording{}}
}
func (a *Adapter) ForDevice(serial string) strategy.Strategy {
	serial = strings.TrimSpace(serial)
	return &Adapter{runner: a.runner, serial: serial, identitySerial: serial, transport: transportForSerial(serial), recordings: map[string]activeRecording{}}
}

// RestoreWireless rebuilds an endpoint-bound adapter after the control
// service is restarted. The endpoint was previously verified against the
// hardware serial during promotion and is restored only from durable local
// state, never from operator-supplied input.
func (a *Adapter) RestoreWireless(endpoint string) strategy.Strategy {
	identity := a.identitySerial
	if identity == "" {
		identity = a.serial
	}
	return &Adapter{runner: a.runner, serial: identity, identitySerial: identity, endpoint: strings.TrimSpace(endpoint), transport: "wireless", recordings: map[string]activeRecording{}}
}

func (a *Adapter) StartRecording(_ context.Context, class strategy.ClaimClass) (strategy.RecordingHandle, error) {
	processRunner, ok := a.runner.(ProcessRunner)
	if !ok {
		return strategy.RecordingHandle{}, fmt.Errorf("android recording requires a process-capable runner")
	}
	if class == "" {
		class = strategy.ClaimTransition
	}
	id := fmt.Sprintf("android-recording-%d", time.Now().UnixNano())
	path := "/sdcard/" + id + ".mp4"
	// Do not depend on an interactive PTY to deliver Ctrl-C. Some Android 13
	// images accept the PTY command but leave a zero-byte file when the local
	// adb wrapper is interrupted. A bounded native segment is allowed to close
	// naturally; StopRecording waits for that trailer instead of publishing an
	// incomplete file. The segment is short enough to keep stop bounded while
	// remaining well below Android's three-minute screenrecord ceiling.
	process, err := processRunner.Start("adb", append(a.args("shell", "screenrecord", "--time-limit", "30", "--bit-rate", "1000000", path))...)
	if err != nil {
		return strategy.RecordingHandle{}, fmt.Errorf("start adb screenrecord: %w", err)
	}
	if err := process.Start(); err != nil {
		return strategy.RecordingHandle{}, fmt.Errorf("start screenrecord process: %w", err)
	}
	handle := strategy.RecordingHandle{ID: id, ClaimClass: class, StartedAt: time.Now().UTC()}
	a.recordingMu.Lock()
	a.recordings[id] = activeRecording{handle: handle, process: process, path: path}
	a.recordingMu.Unlock()
	return handle, nil
}

func (a *Adapter) StopRecording(ctx context.Context, handle strategy.RecordingHandle) (strategy.RecordingArtifact, error) {
	a.recordingMu.Lock()
	active, ok := a.recordings[handle.ID]
	if ok {
		delete(a.recordings, handle.ID)
	}
	a.recordingMu.Unlock()
	if !ok {
		return strategy.RecordingArtifact{}, fmt.Errorf("recording %q is not active", handle.ID)
	}
	// The recording was started with a bounded native segment. Let screenrecord
	// close its MP4 trailer naturally; interrupting the host adb transport can
	// leave the remote encoder with a zero-byte or otherwise unverifiable file.
	wait := make(chan error, 1)
	go func() { wait <- active.process.Wait() }()
	var waitErr error
	select {
	case waitErr = <-wait:
	case <-ctx.Done():
		_ = active.process.Kill()
		waitErr = <-wait
		return strategy.RecordingArtifact{}, fmt.Errorf("stop screenrecord %q exceeded deadline: %w", handle.ID, ctx.Err())
	}
	if waitErr != nil {
		// adb exits non-zero when interrupted after producing a valid recording;
		// verify the artifact before treating it as a failure.
		if _, probeErr := a.runner.Run(ctx, "adb", a.args("shell", "ls", active.path)...); probeErr != nil {
			return strategy.RecordingArtifact{}, fmt.Errorf("screenrecord process: %w", waitErr)
		}
	}
	video, err := a.runner.Run(ctx, "adb", a.args("exec-out", "cat", active.path)...)
	if err != nil {
		return strategy.RecordingArtifact{}, fmt.Errorf("read screenrecord artifact %q: %w", active.path, err)
	}
	if len(video) == 0 {
		return strategy.RecordingArtifact{}, fmt.Errorf("read screenrecord artifact %q: adb returned zero bytes", active.path)
	}
	_, _ = a.runner.Run(ctx, "adb", a.args("shell", "rm", "-f", active.path)...)
	duration := time.Since(handle.StartedAt)
	return strategy.RecordingArtifact{Bytes: video, Method: "native", ClaimClass: handle.ClaimClass, Duration: duration, EffectiveFPS: 30}, nil
}

func (a *Adapter) WirelessEndpoint() string { return a.endpoint }

func transportForSerial(serial string) string {
	if strings.Contains(serial, ":") {
		return "wireless"
	}
	return "usb"
}

// WirelessPromoter is the narrow service seam for promoting an already
// onboarded USB device. It keeps transport changes out of the inventory store
// until the returned ADB identity has been verified.
type WirelessPromoter interface {
	PromoteWireless(context.Context) error
}

func (a *Adapter) PromoteWireless(ctx context.Context) error {
	if strings.TrimSpace(a.serial) == "" {
		return fmt.Errorf("wireless promotion requires a USB-onboarded device serial")
	}
	if a.transport != "usb" {
		return fmt.Errorf("device is already using %s transport", a.transport)
	}
	// Read the address while the USB transport is still usable. `adb tcpip`
	// restarts adbd and commonly drops the USB-selected endpoint immediately.
	route, err := a.runner.Run(ctx, "adb", a.args("shell", "ip", "route")...)
	if err != nil {
		return fmt.Errorf("read device wireless address: %w", err)
	}
	address := wirelessAddress(string(route))
	if address == "" {
		return fmt.Errorf("device did not report a wireless address")
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("tcpip", wirelessADBPort)...); err != nil {
		return fmt.Errorf("enable adb tcpip mode: %w", err)
	}
	if _, err := a.runner.Run(ctx, "adb", "connect", address+":"+wirelessADBPort); err != nil {
		return fmt.Errorf("connect wireless adb: %w", err)
	}
	endpoint := address + ":" + wirelessADBPort
	// `adb get-serialno` reports the network endpoint for a wireless target,
	// not the hardware identity. Read the device property over the verified
	// endpoint so the stable inventory identity remains tied to the handset.
	var serialCheck []byte
	for attempt := 0; attempt < 5; attempt++ {
		serialCheck, err = a.runner.Run(ctx, "adb", "-s", endpoint, "shell", "getprop", "ro.serialno")
		if err == nil && strings.TrimSpace(string(serialCheck)) == a.identitySerial {
			break
		}
		if attempt < 4 {
			time.Sleep(250 * time.Millisecond)
		}
	}
	if err != nil || strings.TrimSpace(string(serialCheck)) != a.identitySerial {
		return fmt.Errorf("wireless adb identity mismatch: expected serial %q, got %q", a.identitySerial, strings.TrimSpace(string(serialCheck)))
	}
	devices, err := a.runner.Run(ctx, "adb", "devices", "-l")
	if err != nil {
		return fmt.Errorf("verify wireless adb identity: %w", err)
	}
	matched := false
	for _, line := range strings.Split(string(devices), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "device" && (fields[0] == endpoint || fields[0] == a.identitySerial) {
			matched = true
			break
		}
	}
	if !matched {
		return fmt.Errorf("wireless adb identity mismatch: expected serial %q", a.serial)
	}
	a.endpoint = endpoint
	a.transport = "wireless"
	return nil
}

func wirelessAddress(route string) string {
	fields := strings.Fields(route)
	for i, field := range fields {
		if field == "src" && i+1 < len(fields) && isIPv4(fields[i+1]) {
			return fields[i+1]
		}
	}
	for _, field := range fields {
		if isIPv4(field) && !strings.HasPrefix(field, "0.") {
			return field
		}
	}
	return ""
}

func isIPv4(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && ip.To4() != nil
}
func (a *Adapter) ID() string { return "android-adb" }
func (a *Adapter) args(args ...string) []string {
	selector := a.serial
	if a.endpoint != "" {
		selector = a.endpoint
	}
	if selector != "" {
		return append([]string{"-s", selector}, args...)
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
			health, reason = "unauthorized", "device is present but unauthorized; accept the RSA prompt on the phone"
		case "offline":
			health, reason = strategy.HealthUnreachable, "device is present but offline; set USB mode to File Transfer and replug"
		case "no permissions":
			health, reason = strategy.HealthUnreachable, "insufficient permissions; install the udev rule and replug"
		case "device":
		default:
			health, reason = strategy.HealthUnreachable, "device state is "+state
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
		devices = append(devices, strategy.Device{ID: "android-" + hex.EncodeToString(digest[:8]), Serial: identitySerial, Model: model, OSVersion: osVersion, StrategyID: a.ID(), Transport: transportForSerial(serial), Health: health, HealthReason: reason, ObservedAt: now})
	}
	return devices, nil
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
	d := strategy.Declaration{StrategyID: a.ID(), Description: "Android phones and emulators through ADB", Status: strategy.StatusAvailable, Capabilities: caps, Promotable: true, EvidenceClass: "release-grade", MinimumUsefulFPS: 5}
	d = strategy.WithSupportedHostOS(d, "linux", "darwin", "windows")
	d.Tiers = strategy.Tiers(d)
	return d, nil
}

func unavailable(next string) strategy.Declaration {
	return strategy.UnavailableDeclaration("android-adb", "Android ADB strategy", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next)
}

func (a *Adapter) ReadState(ctx context.Context) (strategy.DeviceState, error) {
	state := strategy.DeviceState{Unavailable: map[string]string{}}
	probe := func(field string, args ...string) string {
		out, err := a.runner.Run(ctx, "adb", a.args(args...)...)
		if err != nil {
			state.Unavailable[field] = "adb " + strings.Join(a.args(args...), " ")
			return ""
		}
		return string(out)
	}
	activity := probe("foreground_package", "shell", "dumpsys", "activity", "activities")
	state.ForegroundPackage = foregroundPackage(activity)
	if state.ForegroundPackage == "" && activity != "" {
		state.Unavailable["foreground_package"] = "adb " + strings.Join(a.args("shell", "dumpsys", "activity", "activities"), " ")
	}
	power := probe("screen_state", "shell", "dumpsys", "power")
	state.ScreenState = screenState(power)
	if state.ScreenState == "" && power != "" {
		state.Unavailable["screen_state"] = "adb " + strings.Join(a.args("shell", "dumpsys", "power"), " ")
	}
	window := probe("lock_state", "shell", "dumpsys", "window")
	state.LockState = lockState(window)
	if state.LockState == "" && window != "" {
		state.Unavailable["lock_state"] = "adb " + strings.Join(a.args("shell", "dumpsys", "window"), " ")
	}
	input := probe("orientation", "shell", "dumpsys", "input")
	state.Orientation = orientation(input)
	if state.Orientation == "" && input != "" {
		state.Unavailable["orientation"] = "adb " + strings.Join(a.args("shell", "dumpsys", "input"), " ")
	}
	if state.Orientation == "" {
		rotation := probe("orientation", "shell", "settings", "get", "system", "user_rotation")
		state.Orientation = orientationValue(rotation)
		if state.Orientation != "" {
			delete(state.Unavailable, "orientation")
		}
	}
	autoRotate := probe("auto_rotate", "shell", "settings", "get", "system", "accelerometer_rotation")
	state.AutoRotate = strings.TrimSpace(autoRotate) == "1"
	battery := probe("battery", "shell", "dumpsys", "battery")
	state.BatteryLevel, state.Charging = batteryState(battery)
	if state.BatteryLevel < 0 && battery != "" {
		state.Unavailable["battery"] = "adb " + strings.Join(a.args("shell", "dumpsys", "battery"), " ")
	}
	thermal := probe("thermal_status", "shell", "dumpsys", "thermalservice")
	state.ThermalStatus = thermalState(thermal)
	if state.ThermalStatus == "" && thermal != "" {
		state.Unavailable["thermal_status"] = "adb " + strings.Join(a.args("shell", "dumpsys", "thermalservice"), " ")
	}
	size := probe("display_metrics", "shell", "wm", "size")
	state.DisplayWidth, state.DisplayHeight = displaySize(size)
	density := probe("display_density", "shell", "wm", "density")
	state.DisplayDensity = displayDensity(density)
	if state.DisplayWidth <= 0 || state.DisplayHeight <= 0 {
		state.Unavailable["display_metrics"] = "adb " + strings.Join(a.args("shell", "wm", "size"), " ")
	}
	if state.DisplayDensity <= 0 {
		state.Unavailable["display_density"] = "adb " + strings.Join(a.args("shell", "wm", "density"), " ")
	}
	if len(state.Unavailable) == 0 {
		state.Unavailable = nil
	}
	return state, nil
}

func (a *Adapter) RestoreState(ctx context.Context, state strategy.DeviceState) error {
	rotation := map[string]string{"portrait": "0", "landscape": "1", "reverse-portrait": "2", "reverse-landscape": "3"}[state.Orientation]
	if rotation == "" {
		return fmt.Errorf("cannot restore unknown orientation %q", state.Orientation)
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "put", "system", "accelerometer_rotation", boolSetting(state.AutoRotate))...); err != nil {
		return fmt.Errorf("restore auto-rotate: %w", err)
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "put", "system", "user_rotation", rotation)...); err != nil {
		return fmt.Errorf("restore orientation: %w", err)
	}
	return nil
}

func boolSetting(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func foregroundPackage(output string) string {
	match := regexp.MustCompile(`(?i)(?:mResumedActivity|ResumedActivity):.*?\s(?:u\d+\s+)?([A-Za-z0-9._]+)/(?:[A-Za-z0-9.$_]+)`).FindStringSubmatch(output)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

func screenState(output string) string {
	if regexp.MustCompile(`(?i)(Display Power|mWakefulness).*?(ON|Awake)`).MatchString(output) {
		return "on"
	}
	if regexp.MustCompile(`(?i)(Display Power|mWakefulness).*?(OFF|Asleep|Dozing)`).MatchString(output) {
		return "off"
	}
	return ""
}

func lockState(output string) string {
	if regexp.MustCompile(`(?i)(mShowingLockscreen|isStatusBarKeyguard|mKeyguardShowing)=true`).MatchString(output) {
		return "locked"
	}
	if regexp.MustCompile(`(?i)(mShowingLockscreen|isStatusBarKeyguard|mKeyguardShowing)=false`).MatchString(output) {
		return "unlocked"
	}
	return ""
}

func orientation(output string) string {
	match := regexp.MustCompile(`(?i)(?:mCurrentRotation|SurfaceOrientation)\s*[:=]\s*(\d)`).FindStringSubmatch(output)
	if len(match) != 2 {
		return ""
	}
	values := map[string]string{"0": "portrait", "1": "landscape", "2": "reverse-portrait", "3": "reverse-landscape"}
	return values[match[1]]
}

func orientationValue(output string) string {
	value := strings.TrimSpace(output)
	if match := regexp.MustCompile(`\d`).FindString(value); match != "" {
		return map[string]string{"0": "portrait", "1": "landscape", "2": "reverse-portrait", "3": "reverse-landscape"}[match]
	}
	return ""
}

func batteryState(output string) (int, bool) {
	level := -1
	if match := regexp.MustCompile(`(?m)^\s*level:\s*(\d+)`).FindStringSubmatch(output); len(match) == 2 {
		level, _ = strconv.Atoi(match[1])
	}
	charging := regexp.MustCompile(`(?mi)^\s*(?:AC powered|USB powered|Wireless powered):\s*true`).MatchString(output)
	return level, charging
}

func thermalState(output string) string {
	match := regexp.MustCompile(`(?i)(?:status|thermal status)\s*[:=]\s*([A-Za-z0-9_-]+)`).FindStringSubmatch(output)
	if len(match) == 2 {
		return strings.ToLower(match[1])
	}
	return ""
}

func displaySize(output string) (int, int) {
	match := regexp.MustCompile(`(\d+)x(\d+)`).FindStringSubmatch(output)
	if len(match) != 3 {
		return 0, 0
	}
	width, _ := strconv.Atoi(match[1])
	height, _ := strconv.Atoi(match[2])
	return width, height
}

func displayDensity(output string) int {
	match := regexp.MustCompile(`(\d+)`).FindStringSubmatch(output)
	if len(match) != 2 {
		return 0
	}
	density, _ := strconv.Atoi(match[1])
	return density
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

func (a *Adapter) ResolveSemantic(ctx context.Context, target string) (strategy.SemanticResult, error) {
	tree, err := a.runner.Run(ctx, "adb", a.args("exec-out", "uiautomator", "dump", "/dev/tty")...)
	if err != nil {
		return strategy.SemanticResult{}, err
	}
	values, err := findSemanticNode(tree, target)
	if err != nil {
		return strategy.SemanticResult{}, err
	}
	left, top, right, bottom, err := semanticBoundsRect(values, target)
	if err != nil {
		return strategy.SemanticResult{}, err
	}
	frame, err := a.Observe(ctx)
	if err != nil || frame.Width <= 0 || frame.Height <= 0 {
		if err == nil {
			err = fmt.Errorf("semantic resolution requires positive display dimensions")
		}
		return strategy.SemanticResult{}, err
	}
	return strategy.SemanticResult{
		Bounds:     []float64{float64(left) / float64(frame.Width), float64(top) / float64(frame.Height), float64(right) / float64(frame.Width), float64(bottom) / float64(frame.Height)},
		Confidence: 1,
	}, nil
}

func (a *Adapter) Actuate(ctx context.Context, event strategy.Actuation) error {
	if event.Pointer != nil {
		x, y, err := a.pointerPixels(ctx, event.Pointer)
		if err != nil {
			return err
		}
		verb := "tap"
		args := []string{"shell", "input", verb, strconv.Itoa(x), strconv.Itoa(y)}
		switch event.Pointer.Kind {
		case "long-press":
			args = []string{"shell", "input", "swipe", strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(maxInt(event.Pointer.DurationMS, 800))}
		case "drag", "fling", "swipe":
			endX, endY, parseErr := parsePointerEnd(event.Pointer.Button)
			if parseErr != nil {
				return parseErr
			}
			finalEndX, finalEndY := int(endX), int(endY)
			if event.Pointer.Normalized {
				finalEndX, finalEndY, parseErr = a.normalizedPixels(ctx, endX, endY)
				if parseErr != nil {
					return parseErr
				}
			}
			args = []string{"shell", "input", "swipe", strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(finalEndX), strconv.Itoa(finalEndY), strconv.Itoa(maxInt(event.Pointer.DurationMS, 300))}
		}
		_, err = a.runner.Run(ctx, "adb", a.args(args...)...)
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
			parts := strings.Split(event.Value, ",")
			if len(parts) < 4 || len(parts) > 5 {
				return fmt.Errorf("swipe requires start x,y, end x,y, and optional duration")
			}
			args = append(args, parts...)
		case "scroll-to":
			startX, startY, err := a.normalizedPixels(ctx, .5, .8)
			if err != nil {
				return err
			}
			endX, endY, err := a.normalizedPixels(ctx, .5, .2)
			if err != nil {
				return err
			}
			args = append(args, "shell", "input", "swipe", strconv.Itoa(startX), strconv.Itoa(startY), strconv.Itoa(endX), strconv.Itoa(endY), "400")
		case "pinch":
			return &strategy.AvailabilityError{Reason: "multi-touch is unavailable on this ADB adapter", NextAction: "Use a strategy that declares multi-touch."}
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
			rotations := map[string]string{"portrait": "0", "landscape": "1", "180": "2", "reverse-portrait": "2", "270": "3", "reverse-landscape": "3"}
			rotation, ok := rotations[strings.ToLower(strings.TrimSpace(event.Value))]
			if !ok {
				return fmt.Errorf("rotation must be portrait, landscape, 180, or 270")
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
				// Some Android 13 images reject svc wifi while still exposing the
				// connectivity shell command. Preserve the exact primary failure
				// only if the named fallback also fails.
				_, fallbackErr := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "connectivity", "airplane-mode", enabled)...)
				if fallbackErr != nil {
					return fmt.Errorf("network control unavailable: adb svc wifi: %v; adb cmd connectivity airplane-mode: %w", err, fallbackErr)
				}
				return nil
			}
			_, err = a.runner.Run(ctx, "adb", a.args("shell", "svc", "data", enabled)...)
			if err != nil {
				_, fallbackErr := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "connectivity", "airplane-mode", enabled)...)
				if fallbackErr != nil {
					return fmt.Errorf("network data control unavailable: adb svc data: %v; adb cmd connectivity airplane-mode: %w", err, fallbackErr)
				}
				err = nil
			}
			return err
		case "bluetooth":
			state := strings.ToLower(strings.TrimSpace(event.Value))
			if state != "online" && state != "offline" && state != "enable" && state != "disable" {
				return fmt.Errorf("bluetooth action requires online, offline, enable, or disable")
			}
			enabled := "enable"
			if state == "offline" || state == "disable" {
				enabled = "disable"
			}
			_, err := a.runner.Run(ctx, "adb", a.args("shell", "svc", "bluetooth", enabled)...)
			if err != nil {
				_, fallbackErr := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "bluetooth_manager", enabled)...)
				if fallbackErr != nil {
					return fmt.Errorf("bluetooth unavailable: adb svc bluetooth: %v; adb cmd bluetooth_manager: %w", err, fallbackErr)
				}
			}
			return nil
		case "airplane-mode":
			state := strings.ToLower(strings.TrimSpace(event.Value))
			if state != "online" && state != "offline" && state != "enable" && state != "disable" {
				return fmt.Errorf("airplane-mode action requires online, offline, enable, or disable")
			}
			value := "1"
			if state == "online" || state == "disable" {
				value = "0"
			}
			if _, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "put", "global", "airplane_mode_on", value)...); err != nil {
				return err
			}
			broadcastState := "false"
			if value == "1" {
				broadcastState = "true"
			}
			_, err := a.runner.Run(ctx, "adb", a.args("shell", "am", "broadcast", "-a", "android.intent.action.AIRPLANE_MODE", "--ez", "state", broadcastState)...)
			return err
		case "screen":
			switch strings.ToLower(strings.TrimSpace(event.Value)) {
			case "wake":
				_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_WAKEUP")...)
				return err
			case "sleep":
				_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_SLEEP")...)
				return err
			case "unlock":
				_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_WAKEUP")...)
				if err != nil {
					return err
				}
				_, err = a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_MENU")...)
				return err
			default:
				return fmt.Errorf("screen action requires wake, sleep, or unlock")
			}
		case "deep-link":
			if strings.TrimSpace(event.Value) == "" {
				return fmt.Errorf("deep-link action requires a URI")
			}
			args = append(args, "shell", "am", "start", "-a", "android.intent.action.VIEW", "-d", event.Value)
			if event.Package != "" {
				args = append(args, "-p", event.Package)
			}
		case "semantic-target":
			return fmt.Errorf("semantic-target is resolver-scoped; execute it through the flow resolver")
		case "semantic-assert":
			tree, err := a.runner.Run(ctx, "adb", a.args("exec-out", "uiautomator", "dump", "/dev/tty")...)
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
			return fmt.Errorf("screenrecord is session-scoped; use recording-start and recording-stop")
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

func maxInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func parsePointerEnd(value string) (float64, float64, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("pointer gesture requires end coordinates")
	}
	x, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, err
	}
	y, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func (a *Adapter) pointerPixels(ctx context.Context, event *strategy.PointerEvent) (int, int, error) {
	if !event.Normalized {
		return int(event.X), int(event.Y), nil
	}
	return a.normalizedPixels(ctx, event.X, event.Y)
}

func (a *Adapter) normalizedPixels(ctx context.Context, x, y float64) (int, int, error) {
	if x < 0 || x > 1 || y < 0 || y > 1 {
		return 0, 0, fmt.Errorf("normalized pointer coordinates must be between 0 and 1")
	}
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "wm", "size")...)
	if err != nil {
		return 0, 0, fmt.Errorf("read display metrics: %w", err)
	}
	match := regexp.MustCompile(`(\d+)x(\d+)`).FindStringSubmatch(string(out))
	if len(match) != 3 {
		return 0, 0, fmt.Errorf("display metrics did not contain a size")
	}
	width, _ := strconv.Atoi(match[1])
	height, _ := strconv.Atoi(match[2])
	return int(x * float64(width)), int(y * float64(height)), nil
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
	x0, y0, x1, y1, err := semanticBoundsRect(values, target)
	if err != nil {
		return 0, 0, err
	}
	return (x0 + x1) / 2, (y0 + y1) / 2, nil
}

func semanticBoundsRect(values map[string]string, target string) (int, int, int, int, error) {
	match := boundsPattern.FindStringSubmatch(values["bounds"])
	if len(match) != 5 {
		return 0, 0, 0, 0, fmt.Errorf("accessibility target %q has invalid bounds", target)
	}
	x0, _ := strconv.Atoi(match[1])
	y0, _ := strconv.Atoi(match[2])
	x1, _ := strconv.Atoi(match[3])
	y1, _ := strconv.Atoi(match[4])
	return x0, y0, x1, y1, nil
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
