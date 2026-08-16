package androidadb

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	// Register JPEG decoder for image.Decode.
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func (commandRunner) Start(name string, args ...string) (Process, error) {
	return &commandProcess{cmd: exec.Command(name, args...)}, nil
}

type activeRecording struct {
	handle         strategy.RecordingHandle
	process        Process
	path           string
	priorStayAwake bool
	priorTimeout   string
	priorDoze      bool
	keepAliveStop  chan struct{}
	keepAliveDone  chan struct{}
	screenshotStop chan struct{}
	screenshotDone chan struct{}
	screenshots    *recordingScreenshots
}

const recordingScreenOffTimeout = "1800000"

type recordingScreenshots struct {
	mu     sync.Mutex
	frames [][]byte
}

type Adapter struct {
	runner         Runner
	serial         string
	identitySerial string
	endpoint       string
	transport      string
	recordingMu    sync.Mutex
	recordings     map[string]activeRecording
	rendererID     func(context.Context, string) (string, error)
	rendererTarget func(context.Context, string) (string, string, error)
}

func newAdapter(runner Runner, serial string) *Adapter {
	return &Adapter{runner: runner, serial: serial, identitySerial: serial, transport: transportForSerial(serial), recordings: map[string]activeRecording{}, rendererTarget: discoverWebViewRenderer}
}

func New() *Adapter {
	serial := strings.TrimSpace(os.Getenv("ANDROID_SERIAL"))
	return newAdapter(commandRunner{}, serial)
}

func NewWithRunner(r Runner, serial string) *Adapter {
	return newAdapter(r, serial)
}

func (a *Adapter) ForDevice(serial string) strategy.Strategy {
	serial = strings.TrimSpace(serial)
	return &Adapter{runner: a.runner, serial: serial, identitySerial: serial, transport: transportForSerial(serial), recordings: map[string]activeRecording{}, rendererID: a.rendererID, rendererTarget: a.rendererTarget}
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
	return &Adapter{runner: a.runner, serial: identity, identitySerial: identity, endpoint: strings.TrimSpace(endpoint), transport: "wireless", recordings: map[string]activeRecording{}, rendererID: a.rendererID, rendererTarget: a.rendererTarget}
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
	priorStayAwake, err := a.readStayAwake(context.Background())
	if err != nil {
		return strategy.RecordingHandle{}, fmt.Errorf("read display keep-awake state: %w", err)
	}
	priorTimeout, err := a.readScreenOffTimeout(context.Background())
	if err != nil {
		return strategy.RecordingHandle{}, fmt.Errorf("read screen-off timeout: %w", err)
	}
	priorDoze, err := a.readDozeEnabled(context.Background())
	if err != nil {
		return strategy.RecordingHandle{}, fmt.Errorf("read doze state: %w", err)
	}
	if err := a.setStayAwake(context.Background(), true); err != nil {
		return strategy.RecordingHandle{}, fmt.Errorf("keep display awake for recording: %w", err)
	}
	if err := a.setDisplayRetention(context.Background(), recordingScreenOffTimeout, false); err != nil {
		_ = a.setStayAwake(context.Background(), priorStayAwake)
		return strategy.RecordingHandle{}, fmt.Errorf("retain display for recording: %w", err)
	}
	// Each caller-owned recording is chapter-scoped, so the native encoder has
	// no artificial thirty-second ceiling. StopRecording sends the encoder its
	// normal trailer signal before reading the artifact.
	process, err := processRunner.Start("adb", a.args("shell", "screenrecord", "--bit-rate", "1000000", path)...)
	if err != nil {
		_ = a.setStayAwake(context.Background(), priorStayAwake)
		_ = a.setDisplayRetention(context.Background(), priorTimeout, priorDoze)
		return strategy.RecordingHandle{}, fmt.Errorf("start adb screenrecord: %w", err)
	}
	if err := process.Start(); err != nil {
		_ = a.setStayAwake(context.Background(), priorStayAwake)
		_ = a.setDisplayRetention(context.Background(), priorTimeout, priorDoze)
		return strategy.RecordingHandle{}, fmt.Errorf("start screenrecord process: %w", err)
	}
	handle := strategy.RecordingHandle{ID: id, ClaimClass: class, StartedAt: time.Now().UTC()}
	keepAliveStop := make(chan struct{})
	keepAliveDone := make(chan struct{})
	go a.keepDisplayAwake(keepAliveStop, keepAliveDone)
	screenshotStop := make(chan struct{})
	screenshotDone := make(chan struct{})
	screenshots := &recordingScreenshots{}
	go a.captureDisplaySamples(screenshotStop, screenshotDone, screenshots)
	a.recordingMu.Lock()
	a.recordings[id] = activeRecording{handle: handle, process: process, path: path, priorStayAwake: priorStayAwake, priorTimeout: priorTimeout, priorDoze: priorDoze, keepAliveStop: keepAliveStop, keepAliveDone: keepAliveDone, screenshotStop: screenshotStop, screenshotDone: screenshotDone, screenshots: screenshots}
	a.recordingMu.Unlock()
	return handle, nil
}

func (a *Adapter) StopRecording(ctx context.Context, handle strategy.RecordingHandle) (artifact strategy.RecordingArtifact, retErr error) {
	a.recordingMu.Lock()
	active, ok := a.recordings[handle.ID]
	if ok {
		delete(a.recordings, handle.ID)
	}
	a.recordingMu.Unlock()
	if !ok {
		return strategy.RecordingArtifact{}, fmt.Errorf("recording %q is not active", handle.ID)
	}
	defer func() {
		close(active.keepAliveStop)
		<-active.keepAliveDone
		close(active.screenshotStop)
		<-active.screenshotDone
		if err := a.setStayAwake(context.Background(), active.priorStayAwake); err != nil {
			restoreErr := fmt.Errorf("restore display keep-awake state: %w", err)
			if retErr == nil {
				retErr = restoreErr
			} else {
				retErr = errors.Join(retErr, restoreErr)
			}
		}
		if err := a.setDisplayRetention(context.Background(), active.priorTimeout, active.priorDoze); err != nil {
			restoreErr := fmt.Errorf("restore display retention: %w", err)
			if retErr == nil {
				retErr = restoreErr
			} else {
				retErr = errors.Join(retErr, restoreErr)
			}
		}
	}()
	// Ctrl-C sent to the local adb client is not consistently forwarded by
	// Samsung's wireless transport. Signal the remote encoder first, then
	// interrupt the host wrapper so the MP4 trailer is closed promptly.
	_, _ = a.runner.Run(ctx, "adb", a.args("shell", "pkill", "-INT", "screenrecord")...)
	if err := active.process.Interrupt(); err != nil {
		_ = active.process.Kill()
	}
	wait := make(chan error, 1)
	go func() { wait <- active.process.Wait() }()
	var waitErr error
	select {
	case waitErr = <-wait:
	case <-ctx.Done():
		_ = active.process.Kill()
		<-wait
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
	nativeVisible := nativeVideoVisible(video)
	method := "native"
	if !nativeVisible {
		if fallback, fallbackErr := a.encodeScreenshotFallback(active.screenshots); fallbackErr == nil {
			video = fallback
			method = "screenshot-fallback"
		}
	}
	duration := time.Since(handle.StartedAt)
	return strategy.RecordingArtifact{Bytes: video, Method: method, ClaimClass: handle.ClaimClass, Duration: duration, EffectiveFPS: 30}, nil
}

func (a *Adapter) readStayAwake(ctx context.Context) (bool, error) {
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "power")...)
	if err != nil {
		return false, err
	}
	match := regexp.MustCompile(`(?m)^\s*mStayOn=(true|false)\s*$`).FindStringSubmatch(string(out))
	if len(match) != 2 {
		return false, fmt.Errorf("dumpsys power did not report mStayOn")
	}
	return match[1] == "true", nil
}

func (a *Adapter) setStayAwake(ctx context.Context, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	_, err := a.runner.Run(ctx, "adb", a.args("shell", "svc", "power", "stayon", value)...)
	return err
}

func (a *Adapter) readScreenOffTimeout(ctx context.Context) (string, error) {
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "get", "system", "screen_off_timeout")...)
	if err != nil || strings.TrimSpace(string(out)) == "" {
		if err == nil {
			err = errors.New("setting was empty")
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (a *Adapter) readDozeEnabled(ctx context.Context) (bool, error) {
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "deviceidle")...)
	if err != nil {
		return false, err
	}
	text := string(out)
	for _, key := range []string{"mDeepEnabled", "mLightEnabled"} {
		if match := regexp.MustCompile(key + `=(true|false)`).FindStringSubmatch(text); len(match) == 2 {
			return match[1] == "true", nil
		}
	}
	return false, errors.New("dumpsys deviceidle did not report an enabled state")
}

func (a *Adapter) setDisplayRetention(ctx context.Context, timeout string, dozeEnabled bool) error {
	if strings.TrimSpace(timeout) == "" {
		return errors.New("screen-off timeout is empty")
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "put", "system", "screen_off_timeout", timeout)...); err != nil {
		return err
	}
	return a.setDoze(ctx, dozeEnabled)
}

func (a *Adapter) setDoze(ctx context.Context, enabled bool) error {
	command := "disable"
	if enabled {
		command = "enable"
	}
	_, err := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "deviceidle", command)...)
	return err
}

// keepDisplayAwake refreshes the non-input stay-awake setting while a native
// recording is active. Recording must never inject an unobserved input event.
func (a *Adapter) keepDisplayAwake(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	touch := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.setStayAwake(ctx, true)
	}
	touch()
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			touch()
		case <-stop:
			return
		}
	}
}

func (a *Adapter) captureDisplaySamples(stop <-chan struct{}, done chan<- struct{}, samples *recordingScreenshots) {
	defer close(done)
	capture := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		frame, err := a.runner.Run(ctx, "adb", a.args("exec-out", "screencap", "-p")...)
		if err != nil || !displayFrameVisible(frame) {
			return
		}
		samples.mu.Lock()
		samples.frames = append(samples.frames, append([]byte(nil), frame...))
		samples.mu.Unlock()
	}
	capture()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			capture()
		case <-stop:
			return
		}
	}
}

func displayFrameVisible(raw []byte) bool {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return false
	}
	bounds := img.Bounds()
	stepX := maxPositiveInt(1, bounds.Dx()/16)
	stepY := maxPositiveInt(1, bounds.Dy()/16)
	startY := bounds.Min.Y + bounds.Dy()/10
	endY := bounds.Min.Y + bounds.Dy()*9/10
	var total float64
	var maximum float64
	count := 0
	for y := startY; y < endY; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			luma := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 257
			total += luma
			if luma > maximum {
				maximum = luma
			}
			count++
		}
	}
	return count > 0 && total/float64(count) > 18 && maximum > 32
}

func maxPositiveInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var (
	androidVideoLumaPattern    = regexp.MustCompile(`lavfi\.signalstats\.YAVG=([0-9]+(?:\.[0-9]+)?)`)
	androidVideoMaxLumaPattern = regexp.MustCompile(`lavfi\.signalstats\.YMAX=([0-9]+(?:\.[0-9]+)?)`)
)

func nativeVideoVisible(raw []byte) bool {
	file, err := os.CreateTemp("", "device-control-native-content-*.mp4")
	if err != nil {
		return false
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return false
	}
	if err := file.Close(); err != nil {
		return false
	}
	out, err := exec.Command("ffmpeg", "-v", "error", "-i", path, "-vf", "crop=iw:ih*0.8:0:ih*0.1,signalstats,metadata=print:file=-", "-frames:v", "12", "-f", "null", "-").CombinedOutput()
	if err != nil {
		return false
	}
	averages := androidVideoLumaPattern.FindAllStringSubmatch(string(out), -1)
	maximums := androidVideoMaxLumaPattern.FindAllStringSubmatch(string(out), -1)
	if len(averages) == 0 || len(averages) != len(maximums) {
		return false
	}
	var average float64
	var maximum float64
	for i := range averages {
		value, parseErr := strconv.ParseFloat(averages[i][1], 64)
		if parseErr != nil {
			return false
		}
		maxValue, parseErr := strconv.ParseFloat(maximums[i][1], 64)
		if parseErr != nil {
			return false
		}
		average += value
		if maxValue > maximum {
			maximum = maxValue
		}
	}
	return average/float64(len(averages)) > 18 || maximum > 32
}

func (a *Adapter) encodeScreenshotFallback(samples *recordingScreenshots) ([]byte, error) {
	samples.mu.Lock()
	frames := append([][]byte(nil), samples.frames...)
	samples.mu.Unlock()
	if len(frames) == 0 {
		return nil, fmt.Errorf("no visible screenshot frames were captured")
	}
	dir, err := os.MkdirTemp("", "device-control-recording-fallback-")
	if err != nil {
		return nil, fmt.Errorf("create screenshot fallback workspace: %w", err)
	}
	defer os.RemoveAll(dir)
	for i, frame := range frames {
		path := filepath.Join(dir, fmt.Sprintf("frame-%04d.png", i))
		if err := os.WriteFile(path, frame, 0o600); err != nil {
			return nil, fmt.Errorf("write screenshot fallback frame: %w", err)
		}
	}
	outputPath := filepath.Join(dir, "recording.mp4")
	args := []string{"-y", "-loglevel", "error"}
	if len(frames) == 1 {
		args = append(args, "-loop", "1", "-i", filepath.Join(dir, "frame-0000.png"), "-t", "2")
	} else {
		args = append(args, "-framerate", "2", "-i", filepath.Join(dir, "frame-%04d.png"))
	}
	args = append(args, "-c:v", "libx264", "-r", "15", "-pix_fmt", "yuv420p", outputPath)
	if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("encode screenshot fallback: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	encoded, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read screenshot fallback: %w", err)
	}
	return encoded, nil
}

func (a *Adapter) WirelessEndpoint() string { return a.endpoint }

// ReconnectWireless re-establishes the persisted wireless endpoint and, when
// Android wireless debugging has rotated its TLS port, discovers a current
// endpoint through adb's mDNS browser. Every candidate is verified against the
// onboarded hardware serial before it can replace the durable endpoint.
func (a *Adapter) ReconnectWireless(ctx context.Context) error {
	if a.endpoint == "" {
		return fmt.Errorf("wireless reconnect requires a persisted endpoint")
	}
	candidates := []string{a.endpoint}
	if discovered, err := a.runner.Run(ctx, "adb", "mdns", "services"); err == nil {
		candidates = append(candidates, wirelessEndpoints(string(discovered))...)
	}
	var lastErr error
	seen := make(map[string]bool, len(candidates))
	for _, endpoint := range candidates {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" || seen[endpoint] {
			continue
		}
		seen[endpoint] = true
		if _, err := a.runner.Run(ctx, "adb", "connect", endpoint); err != nil {
			lastErr = fmt.Errorf("connect wireless endpoint: %w", err)
			continue
		}
		if err := a.waitForDevice(ctx, endpoint); err != nil {
			lastErr = err
			continue
		}
		if err := a.verifyWirelessEndpoint(ctx, endpoint); err != nil {
			lastErr = err
			continue
		}
		a.endpoint = endpoint
		a.transport = "wireless"
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no wireless ADB endpoint was discovered for the onboarded device")
}

func (a *Adapter) waitForDevice(ctx context.Context, endpoint string) error {
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := a.runner.Run(waitCtx, "adb", "-s", endpoint, "wait-for-device"); err != nil {
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("wireless_reconnect_timeout: wait-for-device timed out for %s", endpoint)
		}
		return fmt.Errorf("wireless_reconnect_wait_failed: wait-for-device for %s: %w", endpoint, err)
	}
	return nil
}

func (a *Adapter) verifyWirelessEndpoint(ctx context.Context, endpoint string) error {
	serialCheck, err := a.runner.Run(ctx, "adb", "-s", endpoint, "shell", "getprop", "ro.serialno")
	if err != nil {
		return fmt.Errorf("verify wireless endpoint: %w", err)
	}
	if strings.TrimSpace(string(serialCheck)) != a.identitySerial {
		return fmt.Errorf("wireless adb identity mismatch: expected onboarded device")
	}
	devices, err := a.runner.Run(ctx, "adb", "devices", "-l")
	if err != nil {
		return fmt.Errorf("verify wireless adb device state: %w", err)
	}
	for _, line := range strings.Split(string(devices), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[0] == endpoint && fields[1] == "device" {
			return nil
		}
	}
	return fmt.Errorf("wireless endpoint is not authorized")
}

func wirelessEndpoints(output string) []string {
	endpoints := make([]string, 0)
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		endpoint := ""
		if len(fields) >= 3 && fields[1] == "_adb-tls-connect._tcp" {
			endpoint = strings.TrimSpace(fields[2])
		} else if len(fields) >= 2 && strings.Contains(fields[0], "_adb-tls-connect._tcp") {
			endpoint = strings.TrimSpace(fields[1])
		}
		if strings.Contains(endpoint, ":") && !seen[endpoint] {
			seen[endpoint] = true
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

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

// AttachWebView discovers the debugging socket owned by packageName and asks
// adb to allocate a local TCP forward. This is intentionally implemented at
// the Android strategy boundary: callers above device-control never execute
// adb and never need to know the device's abstract socket naming convention.
func (a *Adapter) AttachWebView(ctx context.Context, packageName string) (strategy.WebViewEndpoint, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return strategy.WebViewEndpoint{}, fmt.Errorf("webview attach requires an application package")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.]*$`).MatchString(packageName) {
		return strategy.WebViewEndpoint{}, fmt.Errorf("webview attach rejected invalid package %q", packageName)
	}
	socket, err := a.webViewSocket(ctx, packageName)
	if err != nil {
		return strategy.WebViewEndpoint{}, err
	}
	forward := "localabstract:" + socket
	allocated, err := a.runner.Run(ctx, "adb", a.args("forward", "tcp:0", forward)...)
	if err != nil {
		return strategy.WebViewEndpoint{}, fmt.Errorf("forward WebView socket %q: %w", socket, err)
	}
	port, err := forwardedPort(allocated, socket)
	if err != nil {
		listing, listErr := a.runner.Run(ctx, "adb", a.args("forward", "--list")...)
		if listErr == nil {
			port, err = forwardedPort(listing, socket)
		}
	}
	if err != nil {
		return strategy.WebViewEndpoint{}, err
	}
	cdpEndpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	var rendererID, rendererURL string
	if a.rendererID != nil {
		rendererID, err = a.rendererID(ctx, cdpEndpoint)
	} else {
		readRenderer := a.rendererTarget
		if readRenderer == nil {
			readRenderer = discoverWebViewRenderer
		}
		rendererID, rendererURL, err = readRenderer(ctx, cdpEndpoint)
	}
	if err != nil {
		return strategy.WebViewEndpoint{}, fmt.Errorf("discover WebView renderer for %q: %w", packageName, err)
	}
	if strings.TrimSpace(rendererID) == "" || (a.rendererID == nil && strings.TrimSpace(rendererURL) == "") {
		return strategy.WebViewEndpoint{}, fmt.Errorf("discover WebView renderer for %q returned incomplete identity", packageName)
	}
	return strategy.WebViewEndpoint{Package: packageName, Socket: socket, CDPEndpoint: cdpEndpoint, RendererID: rendererID, RendererURL: rendererURL, Transport: "adb-forward"}, nil
}

func (a *Adapter) webViewSocket(ctx context.Context, packageName string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		pids, err := a.runner.Run(ctx, "adb", a.args("shell", "pidof", packageName)...)
		if err != nil {
			lastErr = fmt.Errorf("find WebView process for %q: %w", packageName, err)
		} else {
			pid := strings.Fields(string(pids))
			if len(pid) > 0 && regexp.MustCompile(`^\d+$`).MatchString(pid[0]) {
				sockets, socketErr := a.runner.Run(ctx, "adb", a.args("shell", "cat", "/proc/net/unix")...)
				if socketErr != nil {
					lastErr = fmt.Errorf("list Android WebView sockets: %w", socketErr)
				} else if socket, socketErr := webViewSocketForPID(sockets, pid[0]); socketErr == nil {
					return socket, nil
				} else {
					lastErr = fmt.Errorf("find WebView socket for %q: %w", packageName, socketErr)
				}
			} else {
				lastErr = fmt.Errorf("application %q has no running process", packageName)
			}
		}
		if attempt == 19 {
			break
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return "", lastErr
}

func discoverWebViewRendererID(ctx context.Context, endpoint string) (string, error) {
	rendererID, _, err := discoverWebViewRenderer(ctx, endpoint)
	return rendererID, err
}

func discoverWebViewRenderer(ctx context.Context, endpoint string) (string, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint, "/")+"/json/list", nil)
	if err != nil {
		return "", "", err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", fmt.Errorf("CDP target listing returned %s", response.Status)
	}
	var targets []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&targets); err != nil {
		return "", "", fmt.Errorf("decode CDP target listing: %w", err)
	}
	for _, target := range targets {
		if strings.EqualFold(target.Type, "page") && strings.TrimSpace(target.ID) != "" {
			if strings.TrimSpace(target.URL) == "" {
				return "", "", fmt.Errorf("CDP page renderer %q omitted its URL", target.ID)
			}
			return strings.TrimSpace(target.ID), strings.TrimSpace(target.URL), nil
		}
	}
	return "", "", fmt.Errorf("CDP target listing contained no page renderer")
}

func webViewSocketForPID(data []byte, pid string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "@")
		if strings.HasPrefix(name, "webview_devtools_remote_") && strings.Contains(name, "_"+pid) {
			return name, nil
		}
	}
	return "", fmt.Errorf("no webview_devtools_remote socket matched process %s", pid)
}

func forwardedPort(output []byte, socket string) (int, error) {
	remote := "localabstract:" + socket
	portPattern := regexp.MustCompile(`(?:^|\s)(?:tcp:)?([1-9][0-9]{1,4})(?:\s|$)`)
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, remote) {
			fields := strings.Fields(line)
			for _, field := range fields {
				if match := portPattern.FindStringSubmatch(" " + field + " "); len(match) == 2 {
					port, _ := strconv.Atoi(match[1])
					if port > 0 && port <= 65535 {
						return port, nil
					}
				}
			}
		}
	}
	if match := portPattern.FindStringSubmatch(" " + strings.TrimSpace(string(output)) + " "); len(match) == 2 {
		port, _ := strconv.Atoi(match[1])
		if port > 0 && port <= 65535 {
			return port, nil
		}
	}
	// Some adb versions do not print the allocated port for tcp:0. The list
	// command is the authoritative fallback and keeps the allocation owned by
	// this adapter.
	return 0, fmt.Errorf("adb did not report a local port for WebView socket %q", socket)
}

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

// shellQuote preserves spaces and shell metacharacters when adb serializes
// arguments after the `shell` boundary. Intent string extras are user content
// and must remain one remote argument rather than becoming an accidental
// package or component token.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (a *Adapter) connected(ctx context.Context) (bool, string) {
	if a.endpoint != "" || a.serial != "" {
		out, err := a.runner.Run(ctx, "adb", "devices", "-l")
		if err == nil {
			selector := a.serial
			if a.endpoint != "" {
				selector = a.endpoint
			}
			for _, line := range strings.Split(string(out), "\n") {
				fields := strings.Fields(strings.TrimSpace(line))
				if len(fields) >= 2 && fields[0] == selector && fields[1] == "device" {
					return true, ""
				}
			}
		}
		if a.endpoint != "" {
			return false, "Enable Android wireless debugging, authorize this host, and verify the saved wireless endpoint is reachable."
		}
		return false, "Enable USB debugging, authorize this computer, and verify the Android device is reachable."
	}
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
		devices = append(devices, strategy.Device{ID: "android-" + hex.EncodeToString(digest[:8]), Serial: identitySerial, Endpoint: endpoint, Model: model, OSVersion: osVersion, StrategyID: a.ID(), Transport: transportForSerial(serial), Health: health, HealthReason: reason, ObservedAt: now})
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
	sockets, socketErr := a.runner.Run(ctx, "adb", a.args("shell", "cat", "/proc/net/unix")...)
	hasWebViewSocket := socketErr == nil && strings.Contains(string(sockets), "webview_devtools_remote")
	caps[strategy.CapWebViewAttach] = strategy.ProbeCapability(strategy.CapWebViewAttach, hasWebViewSocket, "No running Android WebView exposes a debuggable socket", "Launch the target WebView with debugging enabled, then probe again", "adb shell cat /proc/net/unix")
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
	state.LockState = a.readLockState(ctx, &state)
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

// Unlock performs the narrow Android PIN/numeric-passcode procedure. It does
// not use `input text`: every digit is translated to a non-secret Android key
// code before it reaches adb. The only success proof is a fresh keyguard probe
// after submission.
func (a *Adapter) Unlock(ctx context.Context, request strategy.UnlockRequest) (result strategy.UnlockResult, err error) {
	defer func() {
		for i := range request.Secret {
			request.Secret[i] = 0
		}
	}()

	if request.Method != "pin" && request.Method != "numeric_passcode" {
		if request.Method == "biometric" || request.Method == "human_gated" {
			return strategy.UnlockResult{Outcome: "human_required", Detail: "this authentication method requires an operator"}, nil
		}
		return strategy.UnlockResult{Outcome: "unsupported_method", Detail: "android adb supports only numeric PIN methods"}, nil
	}
	if len(request.Secret) == 0 {
		return strategy.UnlockResult{Outcome: "credential_unconfigured", Detail: "the resolved credential was empty"}, nil
	}
	maxAttempts := request.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if maxAttempts > 1 {
		// The service may permit an explicit policy, but an adapter never loops
		// without a fresh live state check between attempts.
		maxAttempts = 1
	}
	if request.AttemptLimit <= 0 {
		request.AttemptLimit = 15 * time.Second
	}
	if request.Settle <= 0 {
		request.Settle = 750 * time.Millisecond
	}

	state, probeErr := a.lockState(ctx)
	if probeErr != nil {
		return strategy.UnlockResult{Outcome: unlockErrorOutcome(probeErr, "transport_error"), Detail: "unable to read Android keyguard state"}, probeErr
	}
	if state == "unlocked" {
		return strategy.UnlockResult{Outcome: "already_unlocked"}, nil
	}
	if state != "locked" {
		return strategy.UnlockResult{Outcome: "unknown_device_state", Detail: "Android keyguard state was not classifiable"}, nil
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, request.AttemptLimit)
		if _, runErr := a.runner.Run(attemptCtx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_WAKEUP")...); runErr != nil {
			cancel()
			return strategy.UnlockResult{Outcome: unlockErrorOutcome(runErr, "transport_error"), Attempts: attempt, Detail: "unable to wake the Android display"}, runErr
		}
		for _, digit := range request.Secret {
			keyCode, ok := digitKeyCode(digit)
			if !ok {
				cancel()
				return strategy.UnlockResult{Outcome: "invalid_credential", Attempts: attempt, Detail: "numeric PIN contains a non-digit"}, nil
			}
			if _, runErr := a.runner.Run(attemptCtx, "adb", a.args("shell", "input", "keyevent", keyCode)...); runErr != nil {
				cancel()
				return strategy.UnlockResult{Outcome: unlockErrorOutcome(runErr, "transport_error"), Attempts: attempt, Detail: "Android rejected a key event"}, runErr
			}
		}
		if _, runErr := a.runner.Run(attemptCtx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_ENTER")...); runErr != nil {
			cancel()
			return strategy.UnlockResult{Outcome: unlockErrorOutcome(runErr, "transport_error"), Attempts: attempt, Detail: "Android rejected PIN submission"}, runErr
		}
		if settleErr := waitForUnlockSettle(attemptCtx, request.Settle); settleErr != nil {
			cancel()
			return strategy.UnlockResult{Outcome: unlockErrorOutcome(settleErr, "timeout"), Attempts: attempt, Detail: "unlock settle window expired"}, nil
		}
		state, probeErr = a.lockState(attemptCtx)
		cancel()
		if probeErr != nil {
			return strategy.UnlockResult{Outcome: unlockErrorOutcome(probeErr, "transport_error"), Attempts: attempt, Detail: "unable to verify Android keyguard state"}, probeErr
		}
		if state == "unlocked" {
			return strategy.UnlockResult{Outcome: "unlocked", Attempts: attempt}, nil
		}
		if state != "locked" {
			return strategy.UnlockResult{Outcome: "unknown_device_state", Attempts: attempt, Detail: "Android keyguard state became unclassifiable"}, nil
		}
	}
	return strategy.UnlockResult{Outcome: "wrong_credential", Attempts: maxAttempts, Detail: "the postcondition remained locked; no automatic retry was attempted"}, nil
}

func (a *Adapter) lockState(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "unknown", err
	}
	state := strategy.DeviceState{Unavailable: map[string]string{}}
	lock := a.readLockState(ctx, &state)
	if lock == "" {
		if err := ctx.Err(); err != nil {
			return "unknown", err
		}
		return "unknown", errors.New("Android keyguard state unavailable")
	}
	return lock, nil
}

func unlockErrorOutcome(err error, fallback string) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	default:
		return fallback
	}
}

// readLockState prefers the broad window dump but falls back to the smaller
// policy dump. Samsung builds can truncate or reject the broad dump over a
// wireless ADB transport while still exposing the keyguard policy state.
func (a *Adapter) readLockState(ctx context.Context, state *strategy.DeviceState) string {
	for _, command := range [][]string{
		{"shell", "dumpsys", "window"},
		{"shell", "dumpsys", "window", "policy"},
		{"shell", "dumpsys", "keyguard"},
		{"shell", "dumpsys", "window", "displays"},
		{"shell", "cmd", "statusbar", "is-keyguard-showing"},
	} {
		out, err := a.runner.Run(ctx, "adb", a.args(command[:]...)...)
		if err == nil {
			if lock := lockState(string(out)); lock != "" {
				return lock
			}
			if lock := exactBooleanLockState(string(out)); lock != "" {
				return lock
			}
		}
	}
	if state != nil {
		state.Unavailable["lock_state"] = "Android keyguard probe failed"
	}
	return ""
}

func digitKeyCode(digit byte) (string, bool) {
	if digit < '0' || digit > '9' {
		return "", false
	}
	return "KEYCODE_" + string(digit), true
}

func waitForUnlockSettle(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
	if state.LockState == "locked" {
		current, err := a.lockState(ctx)
		if err != nil {
			return fmt.Errorf("verify keyguard before restore: %w", err)
		}
		if current == "unlocked" {
			if _, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_POWER")...); err != nil {
				return fmt.Errorf("restore locked keyguard: %w", err)
			}
			current, err = a.waitForLockState(ctx, "locked", 5*time.Second)
			if err != nil {
				return fmt.Errorf("verify restored keyguard: %w", err)
			}
		}
		if current != "locked" {
			return fmt.Errorf("restored keyguard state is %q, want locked", current)
		}
	}
	return nil
}

// waitForLockState accounts for the asynchronous transition after a power
// key. Android can report the old keyguard state for several probes while the
// display and policy services settle. The deadline keeps teardown fail-closed
// without turning a transient vendor delay into a false run failure.
func (a *Adapter) waitForLockState(ctx context.Context, want string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		return "unknown", errors.New("keyguard confirmation timeout must be positive")
	}
	deadline := time.Now().Add(timeout)
	var last string
	var lastErr error
	for {
		current, err := a.lockState(ctx)
		if err == nil {
			last = current
			lastErr = nil
			if current == want {
				return current, nil
			}
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			if lastErr != nil {
				return last, fmt.Errorf("keyguard remained unconfirmed: %w", lastErr)
			}
			return last, fmt.Errorf("restored keyguard state is %q, want %q", last, want)
		}
		timer := time.NewTimer(150 * time.Millisecond)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return last, ctx.Err()
		}
	}
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
	// Android's window/keyguard dump format is not stable across vendors and
	// releases. Samsung Android 13 commonly reports the delegate state as
	// `showing=true` or `mShowingState=SHOWING`, while older builds expose one
	// of the explicit m* fields. Keep this parser deliberately scoped to
	// keyguard/window terms so unrelated boolean fields cannot unlock a flow.
	if regexp.MustCompile(`(?i)(mShowingLockscreen|isStatusBarKeyguard|mKeyguardShowing|isKeyguardShowing)\s*[:=]\s*true`).MatchString(output) {
		return "locked"
	}
	if regexp.MustCompile(`(?i)(mShowingLockscreen|isStatusBarKeyguard|mKeyguardShowing|isKeyguardShowing)\s*[:=]\s*false`).MatchString(output) {
		return "unlocked"
	}
	if regexp.MustCompile(`(?i)\bmShowingState\s*[:=]\s*(?:SHOWING|LOCKED)\b`).MatchString(output) {
		return "locked"
	}
	if regexp.MustCompile(`(?i)\bmShowingState\s*[:=]\s*(?:NOT_SHOWING|UNLOCKED)\b`).MatchString(output) {
		return "unlocked"
	}
	if regexp.MustCompile(`(?i)\b(?:keyguard\s+)?showing\s*[:=]\s*true`).MatchString(output) {
		return "locked"
	}
	if regexp.MustCompile(`(?i)\b(?:keyguard\s+)?showing\s*[:=]\s*false`).MatchString(output) {
		return "unlocked"
	}
	return ""
}

func exactBooleanLockState(output string) string {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "true":
		return "locked"
	case "false":
		return "unlocked"
	default:
		return ""
	}
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
		if event.Pointer.Kind == "double-tap" {
			if _, err = a.runner.Run(ctx, "adb", a.args(args...)...); err != nil {
				return err
			}
			_, err = a.runner.Run(ctx, "adb", a.args(args...)...)
			return err
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
		case "share":
			if strings.TrimSpace(event.Value) == "" {
				return fmt.Errorf("share action requires text")
			}
			args = append(args, "shell", "am", "start", "-a", "android.intent.action.SEND", "-c", "android.intent.category.DEFAULT", "-t", "text/plain")
			if event.Package != "" {
				args = append(args, "-p", event.Package)
			}
			// Keep the package restriction before the string extra. Android's
			// ActivityTaskManager shell parser can otherwise consume the first
			// word of a multi-word extra as the package value.
			args = append(args, "--es", "android.intent.extra.TEXT", shellQuote(event.Value))
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
				args = append(args, "logcat", "-d", "-v", "epoch", "--pid", strings.TrimSpace(string(pid)))
			} else {
				args = append(args, "logcat", "-d", "-v", "epoch")
			}
		case "logcat-start":
			args = append(args, "logcat", "-c")
		case "logcat-stop":
			args = append(args, "logcat", "-d", "-v", "epoch")
		case "clock-sample":
			args = append(args, "shell", "date", "+%s.%N")
		case "screenshot":
			args = append(args, "exec-out", "screencap", "-p")
		case "clipboard-read":
			args = append(args, "shell", "cmd", "clipboard", "get")
		case "clipboard-write":
			if event.Value == "" {
				return fmt.Errorf("clipboard-write requires text")
			}
			args = append(args, "shell", "cmd", "clipboard", "set", event.Value)
		case "screenrecord":
			return fmt.Errorf("screenrecord is session-scoped; use recording-start and recording-stop")
		default:
			return fmt.Errorf("unsupported adb action %q", event.Action)
		}
		out, err := a.runner.Run(ctx, "adb", a.args(args...)...)
		if err != nil && len(out) > 0 {
			return fmt.Errorf("adb %s: %w: %s", event.Action, err, strings.TrimSpace(string(out)))
		}
		if err == nil && event.Output != nil && (event.Action == "device-logs" || event.Action == "logcat-stop" || event.Action == "clock-sample" || event.Action == "screenshot" || event.Action == "clipboard-read") {
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
