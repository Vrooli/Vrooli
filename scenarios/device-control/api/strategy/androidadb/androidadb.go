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

	"device-control/internal/androidartifact"
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
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stderr cappedBuffer
}

func (p *commandProcess) Start() error {
	p.stderr.limit = 2048
	p.cmd.Stderr = &p.stderr
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
func (p *commandProcess) Wait() error {
	err := p.cmd.Wait()
	if err != nil {
		if detail := strings.TrimSpace(p.stderr.buffer.String()); detail != "" {
			return fmt.Errorf("%w: %s", err, detail)
		}
	}
	return err
}

type ProcessRunner interface {
	Start(string, ...string) (Process, error)
}
type commandRunner struct{ requireDefaultADBServer bool }

type boundedRunner interface {
	RunBounded(context.Context, string, int, ...string) ([]byte, error)
}

const wirelessADBPort = "5555"

func (commandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil && ctx.Err() != nil {
		return output, fmt.Errorf("command failed (%v) after context ended: %w", err, ctx.Err())
	}
	return output, err
}

func (commandRunner) RunBounded(ctx context.Context, name string, maxBytes int, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr cappedBuffer
	stdout.limit, stderr.limit = maxBytes, 8*1024
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if stdout.exceeded || stderr.exceeded {
		return nil, fmt.Errorf("command output exceeded configured limit")
	}
	if err != nil && ctx.Err() != nil {
		return stdout.buffer.Bytes(), fmt.Errorf("command failed (%v) after context ended: %w", err, ctx.Err())
	}
	return stdout.buffer.Bytes(), err
}

func (commandRunner) Start(name string, args ...string) (Process, error) {
	return &commandProcess{cmd: exec.Command(name, args...)}, nil
}

type activeRecording struct {
	handle          strategy.RecordingHandle
	process         Process
	path            string
	stopping        bool
	stopAttempted   bool
	audioSessionID  string
	videoStartNanos int64
	priorStayAwake  string
	priorTimeout    string
	priorDoze       dozeState
	keepAliveStop   chan struct{}
	keepAliveDone   chan struct{}
	screenshotStop  chan struct{}
	screenshotDone  chan struct{}
	screenshots     *recordingScreenshots
}

const recordingScreenOffTimeout = "1800000"

type recordingScreenshots struct {
	mu       sync.Mutex
	frames   [][]byte
	times    []time.Time
	bytes    int
	interval time.Duration
}

const maxRecordingScreenshotBytes = 64 * 1024 * 1024

// appendSample bounds retained fallback data and progressively thins older
// samples rather than ending coverage at the point the budget is reached.
// Timestamps remain attached to their original frames.
func (s *recordingScreenshots) appendSample(frame []byte, captured time.Time, budget int) {
	if len(frame) == 0 || len(frame) > budget {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.interval == 0 {
		s.interval = 500 * time.Millisecond
	}
	s.frames = append(s.frames, append([]byte(nil), frame...))
	s.times = append(s.times, captured)
	s.bytes += len(frame)
	for s.bytes > budget && len(s.frames) > 1 {
		frames := make([][]byte, 0, (len(s.frames)+2)/2)
		times := make([]time.Time, 0, cap(frames))
		s.bytes = 0
		for i := range s.frames {
			if i%2 == 0 || i == len(s.frames)-1 {
				frames = append(frames, s.frames[i])
				times = append(times, s.times[i])
				s.bytes += len(s.frames[i])
			}
		}
		// Two unusually large frames may exceed the budget even after thinning.
		if len(frames) == len(s.frames) {
			s.bytes -= len(frames[0])
			frames, times = frames[1:], times[1:]
		}
		s.frames, s.times = frames, times
		if s.interval < 30*time.Second {
			s.interval *= 2
		}
	}
}

type Adapter struct {
	runner           Runner
	serial           string
	identitySerial   string
	endpoint         string
	transport        string
	recordingMu      sync.Mutex
	recordings       map[string]activeRecording
	rendererID       func(context.Context, string) (string, error)
	rendererTarget   func(context.Context, string) (string, string, error)
	signingInspector androidartifact.APKSigningInspector
}

func newAdapter(runner Runner, serial string) *Adapter {
	return &Adapter{runner: runner, serial: serial, identitySerial: serial, transport: transportForSerial(serial), recordings: map[string]activeRecording{}, rendererTarget: discoverWebViewRenderer, signingInspector: androidartifact.AndroidSDKSigningInspector{ADBPath: "adb"}}
}

type Config struct {
	Serial               string
	RequireDefaultServer bool
}

// New constructs an unbound ADB strategy. Production configuration is passed
// by registry.DefaultWithEnvironment.
func New() *Adapter { return NewWithConfig(Config{}) }

func NewWithConfig(config Config) *Adapter {
	return newAdapter(commandRunner{requireDefaultADBServer: config.RequireDefaultServer}, strings.TrimSpace(config.Serial))
}

func NewWithRunner(r Runner, serial string) *Adapter {
	return newAdapter(r, serial)
}

func (a *Adapter) ForDevice(serial string) strategy.Strategy {
	serial = strings.TrimSpace(serial)
	return &Adapter{runner: a.runner, serial: serial, identitySerial: serial, transport: transportForSerial(serial), recordings: map[string]activeRecording{}, rendererID: a.rendererID, rendererTarget: a.rendererTarget, signingInspector: a.signingInspector}
}

// NewNetworkTransport returns a fresh, unbound adapter that shares this
// adapter's runner. The control service calls ConnectNetwork on the returned
// adapter to onboard a directly-addressable endpoint without mutating the shared
// registry instance.
func (a *Adapter) NewNetworkTransport() strategy.Strategy {
	return &Adapter{runner: a.runner, recordings: map[string]activeRecording{}, rendererID: a.rendererID, rendererTarget: a.rendererTarget}
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

func (a *Adapter) StartRecording(ctx context.Context, class strategy.ClaimClass) (strategy.RecordingHandle, error) {
	return a.startRecording(ctx, class, "")
}

// StartRecordingWithAudio starts the Android consent companion first, then
// starts the existing device video recorder. The companion owns the per-session
// MediaProjection token; a fresh system consent is required for each call.
func (a *Adapter) StartRecordingWithAudio(ctx context.Context, class strategy.ClaimClass) (strategy.RecordingHandle, error) {
	id := fmt.Sprintf("android-recording-%d", time.Now().UnixNano())
	if err := a.startAudioCapture(ctx, id); err != nil {
		return strategy.RecordingHandle{}, err
	}
	handle, err := a.startRecording(ctx, class, id)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		_, _, _ = a.stopAudioCapture(cleanupCtx, id)
		return strategy.RecordingHandle{}, err
	}
	return handle, nil
}

func (a *Adapter) startRecording(ctx context.Context, class strategy.ClaimClass, sessionID string) (strategy.RecordingHandle, error) {
	if err := ctx.Err(); err != nil {
		return strategy.RecordingHandle{}, err
	}
	processRunner, ok := a.runner.(ProcessRunner)
	if !ok {
		return strategy.RecordingHandle{}, fmt.Errorf("android recording requires a process-capable runner")
	}
	if class == "" {
		class = strategy.ClaimTransition
	}
	id := sessionID
	if id == "" {
		id = fmt.Sprintf("android-recording-%d", time.Now().UnixNano())
	}
	path := "/data/local/tmp/" + id + ".mp4"
	priorStayAwake, err := a.readStayAwake(ctx)
	if err != nil {
		return strategy.RecordingHandle{}, fmt.Errorf("read display keep-awake state: %w", err)
	}
	priorTimeout, err := a.readScreenOffTimeout(ctx)
	if err != nil {
		return strategy.RecordingHandle{}, fmt.Errorf("read screen-off timeout: %w", err)
	}
	priorDoze, err := a.readDozeEnabled(ctx)
	if err != nil {
		return strategy.RecordingHandle{}, fmt.Errorf("read doze state: %w", err)
	}
	restore := func() error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		return errors.Join(a.restoreStayAwake(cleanupCtx, priorStayAwake), a.setDisplayRetention(cleanupCtx, priorTimeout, priorDoze))
	}
	if err := a.setStayAwake(ctx, true); err != nil {
		return strategy.RecordingHandle{}, errors.Join(fmt.Errorf("keep display awake for recording: %w", err), restore())
	}
	if err := a.setDisplayRetention(ctx, recordingScreenOffTimeout, dozeState{}); err != nil {
		return strategy.RecordingHandle{}, errors.Join(fmt.Errorf("retain display for recording: %w", err), restore())
	}
	// Each caller-owned recording is chapter-scoped, so the native encoder has
	// no artificial thirty-second ceiling. StopRecording sends the encoder its
	// normal trailer signal before reading the artifact.
	// The bitrate is a bounded bandwidth setting; actual quality and frame
	// rate remain device and content dependent.
	if err := ctx.Err(); err != nil {
		return strategy.RecordingHandle{}, errors.Join(err, restore())
	}
	videoStartNanos := int64(0)
	if sessionID != "" {
		before, clockErr := a.readDeviceUptime(ctx)
		if clockErr != nil {
			return strategy.RecordingHandle{}, errors.Join(fmt.Errorf("read device clock before recording: %w", clockErr), restore())
		}
		process, err := processRunner.Start("adb", a.args("shell", "screenrecord", "--bit-rate", "1000000", path)...)
		if err != nil {
			return strategy.RecordingHandle{}, errors.Join(fmt.Errorf("start adb screenrecord: %w", err), restore())
		}
		if err := process.Start(); err != nil {
			return strategy.RecordingHandle{}, errors.Join(fmt.Errorf("start screenrecord process: %w", err), restore())
		}
		after, clockErr := a.readDeviceUptime(ctx)
		if clockErr != nil {
			_ = process.Interrupt()
			_ = process.Kill()
			_ = process.Wait()
			return strategy.RecordingHandle{}, errors.Join(fmt.Errorf("read device clock after recording start: %w", clockErr), restore())
		}
		videoStartNanos = before + (after-before)/2
		activeProcess := process
		return a.finishRecordingStart(ctx, class, id, path, priorStayAwake, priorTimeout, priorDoze, activeProcess, videoStartNanos)
	}
	process, err := processRunner.Start("adb", a.args("shell", "screenrecord", "--bit-rate", "1000000", path)...)
	if err != nil {
		return strategy.RecordingHandle{}, errors.Join(fmt.Errorf("start adb screenrecord: %w", err), restore())
	}
	if err := process.Start(); err != nil {
		return strategy.RecordingHandle{}, errors.Join(fmt.Errorf("start screenrecord process: %w", err), restore())
	}
	return a.finishRecordingStart(ctx, class, id, path, priorStayAwake, priorTimeout, priorDoze, process, videoStartNanos)
}

func (a *Adapter) finishRecordingStart(ctx context.Context, class strategy.ClaimClass, id, path string, priorStayAwake, priorTimeout string, priorDoze dozeState, process Process, videoStartNanos int64) (strategy.RecordingHandle, error) {
	handle := strategy.RecordingHandle{ID: id, ClaimClass: class, StartedAt: time.Now().UTC()}
	keepAliveStop := make(chan struct{})
	keepAliveDone := make(chan struct{})
	go a.keepDisplayAwake(keepAliveStop, keepAliveDone)
	screenshotStop := make(chan struct{})
	screenshotDone := make(chan struct{})
	screenshots := &recordingScreenshots{}
	go a.captureDisplaySamples(screenshotStop, screenshotDone, screenshots)
	a.recordingMu.Lock()
	active := activeRecording{handle: handle, process: process, path: path, priorStayAwake: priorStayAwake, priorTimeout: priorTimeout, priorDoze: priorDoze, keepAliveStop: keepAliveStop, keepAliveDone: keepAliveDone, screenshotStop: screenshotStop, screenshotDone: screenshotDone, screenshots: screenshots, videoStartNanos: videoStartNanos}
	if strings.HasPrefix(id, "android-recording-") && videoStartNanos > 0 {
		active.audioSessionID = id
	}
	a.recordings[id] = active
	a.recordingMu.Unlock()
	return handle, nil
}

func (a *Adapter) StopRecording(ctx context.Context, handle strategy.RecordingHandle) (artifact strategy.RecordingArtifact, retErr error) {
	captureElapsed := time.Since(handle.StartedAt)
	a.recordingMu.Lock()
	active, ok := a.recordings[handle.ID]
	if !ok {
		a.recordingMu.Unlock()
		return strategy.RecordingArtifact{}, fmt.Errorf("recording %q is not active", handle.ID)
	}
	if active.stopping || active.stopAttempted {
		a.recordingMu.Unlock()
		return strategy.RecordingArtifact{}, fmt.Errorf("recording %q already has a stop attempt; inspect its status before any recovery", handle.ID)
	}
	active.stopping, active.stopAttempted = true, true
	a.recordings[handle.ID] = active
	a.recordingMu.Unlock()
	defer func() {
		a.recordingMu.Lock()
		defer a.recordingMu.Unlock()
		if retErr == nil {
			delete(a.recordings, handle.ID)
			return
		}
		active.stopping = false
		a.recordings[handle.ID] = active
	}()
	audioTerminal := false
	var stopBackgroundOnce sync.Once
	stopBackground := func() {
		stopBackgroundOnce.Do(func() {
			close(active.keepAliveStop)
			<-active.keepAliveDone
			close(active.screenshotStop)
			<-active.screenshotDone
		})
	}
	defer func() {
		stopBackground()
		if active.audioSessionID != "" && !audioTerminal {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			if err := a.cleanupAudioCapture(cleanupCtx, active.audioSessionID); err != nil {
				cleanupErr := fmt.Errorf("clean up Android playback capture: %w", err)
				if retErr == nil {
					retErr = cleanupErr
				} else {
					retErr = errors.Join(retErr, cleanupErr)
				}
			}
			cancel()
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := a.restoreStayAwake(cleanupCtx, active.priorStayAwake); err != nil {
			restoreErr := fmt.Errorf("restore display keep-awake state: %w", err)
			if retErr == nil {
				retErr = restoreErr
			} else {
				retErr = errors.Join(retErr, restoreErr)
			}
		}
		if err := a.setDisplayRetention(cleanupCtx, active.priorTimeout, active.priorDoze); err != nil {
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
	if err := a.stopOwnedScreenrecord(ctx, active.path); err != nil {
		retErr = fmt.Errorf("stop owned screenrecord: %w", err)
	}
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
	if active.audioSessionID != "" {
		wave, status, audioErr := a.stopAudioCapture(ctx, active.audioSessionID)
		switch status.State {
		case "stopped", "failed", "duration_limit", "projection_stopped", "audio_permission_denied", "projection_consent_denied", "unavailable":
			audioTerminal = true
		}
		if audioErr != nil {
			return strategy.RecordingArtifact{}, fmt.Errorf("finalize required Android playback audio (state=%s, rms=%.1f dBFS): %w", status.State, status.RMSDBFS, audioErr)
		}
		offset, offsetErr := audioTrimOffset(status.FirstSampleElapsedNanos, active.videoStartNanos)
		if offsetErr != nil {
			return strategy.RecordingArtifact{}, fmt.Errorf("align Android playback audio: %w", offsetErr)
		}
		video, err = muxPlaybackAudio(ctx, video, wave, offset)
		if err != nil {
			return strategy.RecordingArtifact{}, err
		}
	}
	stopBackground()
	nativeVisible := nativeVideoVisible(video)
	method := "native"
	sampleFPS := 0.0
	if !nativeVisible {
		if fallback, count, fallbackErr := a.encodeScreenshotFallback(active.screenshots, handle.StartedAt, captureElapsed); fallbackErr == nil {
			video = fallback
			method = "screenshot-fallback"
			if elapsed := captureElapsed.Seconds(); elapsed > 0 {
				sampleFPS = float64(count) / elapsed
			}
		}
	}
	return strategy.RecordingArtifact{Bytes: video, Method: method, ClaimClass: handle.ClaimClass, Duration: captureElapsed, EffectiveFPS: sampleFPS}, retErr
}

// InspectRecording reports whether the exact owned encoder is still present
// and whether its producer-owned output path remains available. It never
// signals the encoder, reads the artifact bytes, or clears retained state.
func (a *Adapter) InspectRecording(ctx context.Context, handle strategy.RecordingHandle) (strategy.RecordingInspection, error) {
	a.recordingMu.Lock()
	active, ok := a.recordings[handle.ID]
	a.recordingMu.Unlock()
	inspection := strategy.RecordingInspection{State: "unknown", ObservedAt: time.Now().UTC()}
	if !ok || active.path == "" {
		inspection.Reason = "recording handle has no retained adapter state"
		return inspection, nil
	}
	if active.stopping {
		inspection.State = "stopping"
		inspection.Reason = "the original stop attempt is still in progress"
		return inspection, nil
	}
	processes, err := a.ownedScreenrecordProcesses(ctx, active.path)
	if err != nil {
		inspection.Reason = "could not inspect the exact owned encoder process"
		return inspection, err
	}
	if len(processes) > 1 {
		inspection.Reason = "multiple encoder processes match the recording path"
		return inspection, nil
	}
	if len(processes) == 1 {
		inspection.State = "process_active"
		inspection.Reason = "the exact owned encoder is still running; no stop was sent"
		return inspection, nil
	}
	inspection.State = "process_stopped_artifact_missing"
	inspection.Reason = "the exact owned encoder is absent; artifact availability was not confirmed"
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "ls", active.path)...); err == nil {
		inspection.State = "process_stopped_artifact_available"
		inspection.ArtifactAvailable = true
		inspection.Reason = "the exact owned encoder is absent and its output path exists; no artifact bytes were read"
	}
	return inspection, nil
}

// stopOwnedScreenrecord signals only the encoder whose complete argument list
// ends in this recording's unique path. Ambiguity is not authority to stop
// another session's encoder.
func (a *Adapter) stopOwnedScreenrecord(ctx context.Context, path string) error {
	processes, err := a.ownedScreenrecordProcesses(ctx, path)
	if err != nil {
		return err
	}
	if len(processes) == 0 {
		return errors.New("owned encoder process not found")
	}
	if len(processes) != 1 {
		return errors.New("ambiguous encoder process identity")
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "kill", "-INT", processes[0])...); err != nil {
		return fmt.Errorf("signal owned encoder: %w", err)
	}
	return nil
}

func (a *Adapter) ownedScreenrecordProcesses(ctx context.Context, path string) ([]string, error) {
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "ps", "-A", "-o", "PID,ARGS")...)
	if err != nil {
		return nil, fmt.Errorf("inspect encoder processes: %w", err)
	}
	processes := make([]string, 0, 1)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[len(fields)-1] != path || (fields[1] != "screenrecord" && fields[1] != "/system/bin/screenrecord") {
			continue
		}
		n, parseErr := strconv.ParseUint(fields[0], 10, 31)
		if parseErr != nil || n == 0 {
			return nil, errors.New("owned encoder process has an invalid PID")
		}
		processes = append(processes, fields[0])
	}
	return processes, nil
}

func (a *Adapter) readStayAwake(ctx context.Context) (string, error) {
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "get", "global", "stay_on_while_plugged_in")...)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(out))
	if value != "null" {
		if _, err := strconv.ParseUint(value, 10, 31); err != nil {
			return "", errors.New("invalid stay-awake setting")
		}
	}
	return value, nil
}

func (a *Adapter) restoreStayAwake(ctx context.Context, value string) error {
	operation := "put"
	args := []string{"shell", "settings", operation, "global", "stay_on_while_plugged_in", value}
	if value == "null" {
		args = []string{"shell", "settings", "delete", "global", "stay_on_while_plugged_in"}
	}
	_, err := a.runner.Run(ctx, "adb", a.args(args...)...)
	return err
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
	value := strings.TrimSpace(string(out))
	if value != "null" {
		if _, err := strconv.ParseUint(value, 10, 31); err != nil {
			return "", errors.New("invalid screen-off timeout setting")
		}
	}
	return value, nil
}

type dozeState struct{ deep, light bool }

func (a *Adapter) readDozeEnabled(ctx context.Context) (dozeState, error) {
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "deviceidle")...)
	if err != nil {
		return dozeState{}, err
	}
	var state dozeState
	for _, field := range []struct {
		key string
		dst *bool
	}{{"mDeepEnabled", &state.deep}, {"mLightEnabled", &state.light}} {
		match := regexp.MustCompile(field.key + `=(true|false)`).FindStringSubmatch(string(out))
		if len(match) != 2 {
			return dozeState{}, fmt.Errorf("device idle did not report %s", field.key)
		}
		*field.dst = match[1] == "true"
	}
	return state, nil
}

func (a *Adapter) setDisplayRetention(ctx context.Context, timeout string, doze dozeState) error {
	if strings.TrimSpace(timeout) == "" {
		return errors.New("screen-off timeout is empty")
	}
	var result error
	args := []string{"shell", "settings", "put", "system", "screen_off_timeout", timeout}
	if timeout == "null" {
		args = []string{"shell", "settings", "delete", "system", "screen_off_timeout"}
	}
	if _, err := a.runner.Run(ctx, "adb", a.args(args...)...); err != nil {
		result = errors.Join(result, err)
	}
	for _, mode := range []struct {
		name    string
		enabled bool
	}{{"deep", doze.deep}, {"light", doze.light}} {
		action := "disable"
		if mode.enabled {
			action = "enable"
		}
		if _, err := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "deviceidle", action, mode.name)...); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
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
	var lastCapture time.Time
	capture := func() {
		samples.mu.Lock()
		interval := samples.interval
		samples.mu.Unlock()
		if interval > 0 && !lastCapture.IsZero() && time.Since(lastCapture) < interval {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		frame, err := a.runner.Run(ctx, "adb", a.args("exec-out", "screencap", "-p")...)
		if err != nil {
			return
		}
		frame = sanitizeCapturedImage(frame)
		if !displayFrameVisible(frame) {
			return
		}
		lastCapture = time.Now()
		samples.appendSample(frame, lastCapture, maxRecordingScreenshotBytes)
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

// screencapImageSignatures are the encoded-frame prefixes device-control
// accepts from `adb screencap -p`. Some Android and Android-TV firmware images
// print startup log lines to stdout before the encoded frame — observed on a
// SmartTV 4K where 65 bytes ("Init wrapper sys mutex successful...\nopen mma
// dev failed\n") precede the PNG signature. Left unstripped, that preamble
// makes an otherwise valid frame decode as "image: unknown format".
var screencapImageSignatures = [][]byte{
	{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, // PNG
	{0xff, 0xd8, 0xff},                            // JPEG
}

// sanitizeCapturedImage returns the byte slice starting at the first decodable
// image signature. It fast-paths a frame that already decodes at offset 0 and,
// when a firmware log preamble is present, trims to the earliest signature that
// yields a valid image header. Genuinely corrupt output is returned unchanged
// so the caller's decode error still surfaces the real failure.
func sanitizeCapturedImage(raw []byte) []byte {
	if _, _, err := image.DecodeConfig(bytes.NewReader(raw)); err == nil {
		return raw
	}
	for _, sig := range screencapImageSignatures {
		if idx := bytes.Index(raw, sig); idx > 0 {
			if _, _, err := image.DecodeConfig(bytes.NewReader(raw[idx:])); err == nil {
				return raw[idx:]
			}
		}
	}
	return raw
}

// validateCapturedImage checks all compressed image data, not merely the header.
// Bound decoded size before allocation so a malformed device response cannot
// exhaust the service while advertising a plausible image header.
func validateCapturedImage(raw []byte) ([]byte, image.Rectangle, string, error) {
	data := sanitizeCapturedImage(raw)
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, image.Rectangle{}, "", err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width) > (64*1024*1024)/int64(config.Height) {
		return nil, image.Rectangle{}, "", fmt.Errorf("screenshot dimensions exceed supported limit")
	}
	decoded, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, image.Rectangle{}, "", err
	}
	return data, decoded.Bounds(), format, nil
}

func displayFrameVisible(raw []byte) bool {
	img, _, err := image.Decode(bytes.NewReader(sanitizeCapturedImage(raw)))
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
	if len(averages) < 2 || len(averages) != len(maximums) {
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

func (a *Adapter) encodeScreenshotFallback(samples *recordingScreenshots, started time.Time, elapsed time.Duration) ([]byte, int, error) {
	samples.mu.Lock()
	frames := append([][]byte(nil), samples.frames...)
	times := append([]time.Time(nil), samples.times...)
	samples.mu.Unlock()
	if len(frames) != len(times) || elapsed <= 0 {
		return nil, 0, errors.New("screenshot timing is unavailable")
	}
	end := started.Add(elapsed)
	for len(times) > 0 && times[len(times)-1].After(end) {
		times, frames = times[:len(times)-1], frames[:len(frames)-1]
	}
	if len(frames) < 2 {
		return nil, 0, fmt.Errorf("fewer than two visible screenshot frames were captured")
	}
	dir, err := os.MkdirTemp("", "device-control-recording-fallback-")
	if err != nil {
		return nil, 0, fmt.Errorf("create screenshot fallback workspace: %w", err)
	}
	defer os.RemoveAll(dir)
	var manifest strings.Builder
	manifest.WriteString("ffconcat version 1.0\n")
	for i, frame := range frames {
		path := filepath.Join(dir, fmt.Sprintf("frame-%04d.png", i))
		if err := os.WriteFile(path, frame, 0o600); err != nil {
			return nil, 0, fmt.Errorf("write screenshot fallback frame: %w", err)
		}
		begin := times[i]
		if i == 0 {
			begin = started
		}
		finish := end
		if i+1 < len(times) {
			finish = times[i+1]
		}
		if !finish.After(begin) {
			return nil, 0, errors.New("screenshot timestamps are not ordered")
		}
		fmt.Fprintf(&manifest, "file 'frame-%04d.png'\nduration %.6f\n", i, finish.Sub(begin).Seconds())
	}
	fmt.Fprintf(&manifest, "file 'frame-%04d.png'\n", len(frames)-1)
	manifestPath := filepath.Join(dir, "frames.ffconcat")
	if err := os.WriteFile(manifestPath, []byte(manifest.String()), 0o600); err != nil {
		return nil, 0, fmt.Errorf("write screenshot timing: %w", err)
	}
	outputPath := filepath.Join(dir, "recording.mp4")
	args := []string{"-y", "-loglevel", "error", "-f", "concat", "-safe", "1", "-i", manifestPath, "-t", strconv.FormatFloat(elapsed.Seconds(), 'f', 6, 64), "-c:v", "libx264", "-r", "15", "-pix_fmt", "yuv420p", outputPath}
	if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		return nil, 0, fmt.Errorf("encode screenshot fallback: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	encoded, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, 0, fmt.Errorf("read screenshot fallback: %w", err)
	}
	return encoded, len(frames), nil
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
		out, err := a.runner.Run(ctx, "adb", "connect", endpoint)
		if err != nil {
			lastErr = fmt.Errorf("connect wireless endpoint: %w", err)
			continue
		}
		if adbConnectFailed(out) {
			lastErr = fmt.Errorf("connect wireless endpoint %s: %s", endpoint, boundedDiagnostic(out))
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
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
	endpoint := address + ":" + wirelessADBPort
	// adbd restarts asynchronously; retry connection failures until the new
	// listener exists. A successful USB transport is never proof that this
	// particular network endpoint is connected to the same hardware.
	var connectErr error
	for attempt := 0; attempt < 20; attempt++ {
		var output []byte
		output, connectErr = a.runner.Run(ctx, "adb", "connect", endpoint)
		if connectErr == nil && !adbConnectFailed(output) {
			break
		}
		if connectErr == nil {
			connectErr = fmt.Errorf("%s", boundedDiagnostic(output))
		}
		if attempt < 19 {
			timer := time.NewTimer(250 * time.Millisecond)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
	}
	if connectErr != nil {
		return fmt.Errorf("connect wireless adb: %w", connectErr)
	}
	if err := a.waitForDevice(ctx, endpoint); err != nil {
		return err
	}
	if err := a.verifyWirelessEndpoint(ctx, endpoint); err != nil {
		return err
	}
	a.endpoint = endpoint
	a.transport = "wireless"
	return nil
}

// validateADBEndpoint bounds the caller-supplied network address to a host:port
// pair with a numeric port. It rejects shell metacharacters so the value can
// never widen `adb connect` into an arbitrary command; only a transport address
// is ever accepted from the caller.
func validateADBEndpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("network adb endpoint is required (host:port)")
	}
	if len(endpoint) > 255 {
		return fmt.Errorf("network adb endpoint exceeds bounded length")
	}
	if strings.ContainsAny(endpoint, " \t\r\n;&|$`\"'\\<>()") {
		return fmt.Errorf("network adb endpoint contains unsupported characters")
	}
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("network adb endpoint must be host:port: %w", err)
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("network adb endpoint requires a host")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("network adb endpoint requires a valid TCP port")
	}
	return nil
}

// endpointTransportState reports whether the endpoint currently shows a "device"
// state in `adb devices -l`, returning a typed, owner-actionable reason when it
// is unauthorized, offline, or absent.
func (a *Adapter) endpointTransportState(ctx context.Context, endpoint string) (string, error) {
	out, err := a.runner.Run(ctx, "adb", "devices", "-l")
	if err != nil {
		return "", fmt.Errorf("verify adb endpoint transport: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[0] == endpoint {
			return fields[1], nil
		}
	}
	return "", nil
}

// ConnectNetwork onboards a directly-addressable Android ADB endpoint — classic
// TCP ADB (port 5555, e.g. an Android TV or TV box) or Wireless Debugging/TLS —
// without a prior USB attachment. It runs `adb connect`, waits for the
// transport, confirms authorization, and reads the hardware serial over the
// verified endpoint so the durable identity is the device, never the network
// address. The receiver becomes endpoint-bound and the verified Device is
// returned for the caller to persist as the device's transport strategy.
func adbConnectFailed(out []byte) bool {
	message := strings.ToLower(string(out))
	for _, failure := range []string{"cannot connect", "failed to connect", "unable to connect", "failed to authenticate"} {
		if strings.Contains(message, failure) {
			return true
		}
	}
	return false
}

func (a *Adapter) ConnectNetwork(ctx context.Context, endpoint string) (strategy.Device, error) {
	endpoint = strings.TrimSpace(endpoint)
	if err := validateADBEndpoint(endpoint); err != nil {
		return strategy.Device{}, err
	}
	if out, err := a.runner.Run(ctx, "adb", "connect", endpoint); err != nil {
		return strategy.Device{}, fmt.Errorf("connect adb endpoint %q: %w: %s", endpoint, err, strings.TrimSpace(string(out)))
	} else if adbConnectFailed(out) {
		return strategy.Device{}, fmt.Errorf("connect adb endpoint %q: %s", endpoint, strings.TrimSpace(string(out)))
	}
	if _, err := a.runner.Run(ctx, "adb", "-s", endpoint, "wait-for-device"); err != nil {
		return strategy.Device{}, fmt.Errorf("wait for adb endpoint %q: %w", endpoint, err)
	}
	state, err := a.endpointTransportState(ctx, endpoint)
	if err != nil {
		return strategy.Device{}, err
	}
	switch state {
	case "device":
	case "unauthorized":
		return strategy.Device{}, fmt.Errorf("adb endpoint %q is unauthorized; accept the debugging prompt on the device and retry", endpoint)
	case "":
		return strategy.Device{}, fmt.Errorf("adb endpoint %q did not attach; verify the address and that ADB debugging is enabled", endpoint)
	default:
		return strategy.Device{}, fmt.Errorf("adb endpoint %q is in state %q; expected an authorized device", endpoint, state)
	}
	// The hardware serial (never the transport address) is the durable identity.
	serial := ""
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		out, propErr := a.runner.Run(ctx, "adb", "-s", endpoint, "shell", "getprop", "ro.serialno")
		if propErr == nil {
			if candidate := strings.TrimSpace(string(out)); candidate != "" {
				serial = candidate
				break
			}
		}
		lastErr = propErr
		if attempt < 4 {
			time.Sleep(250 * time.Millisecond)
		}
	}
	if serial == "" {
		if lastErr == nil {
			lastErr = fmt.Errorf("device did not report ro.serialno")
		}
		return strategy.Device{}, fmt.Errorf("verify adb endpoint identity: %w", lastErr)
	}
	a.endpoint = endpoint
	a.serial = serial
	a.identitySerial = serial
	a.transport = "wireless"
	devices, err := a.Enumerate(ctx)
	if err != nil {
		return strategy.Device{}, fmt.Errorf("enumerate onboarded endpoint %q: %w", endpoint, err)
	}
	for _, device := range devices {
		if device.Endpoint == endpoint || device.Serial == serial {
			return device, nil
		}
	}
	return strategy.Device{}, fmt.Errorf("onboarded endpoint %q was not present after connect", endpoint)
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
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageRequestValidation, Err: fmt.Errorf("webview attach requires an application package")}
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.]*$`).MatchString(packageName) {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageRequestValidation, Err: fmt.Errorf("webview attach rejected invalid package %q", packageName)}
	}
	socket, err := a.webViewSocket(ctx, packageName)
	if err != nil {
		return strategy.WebViewEndpoint{}, err
	}
	forward := "localabstract:" + socket
	allocated, err := a.runner.Run(ctx, "adb", a.args("forward", "tcp:0", forward)...)
	if err != nil {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStagePortForward, Err: fmt.Errorf("forward WebView socket %q: %w", socket, err)}
	}
	port, err := forwardedPort(allocated, socket)
	if err != nil {
		listing, listErr := a.runner.Run(ctx, "adb", a.args("forward", "--list")...)
		if listErr == nil {
			port, err = forwardedPort(listing, socket)
		}
	}
	if err != nil {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageForwardReadback, Err: err}
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
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageRendererDiscovery, Err: fmt.Errorf("discover WebView renderer for %q: %w", packageName, err)}
	}
	if strings.TrimSpace(rendererID) == "" || (a.rendererID == nil && strings.TrimSpace(rendererURL) == "") {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageRendererIdentity, Err: fmt.Errorf("discover WebView renderer for %q returned incomplete identity", packageName)}
	}
	return strategy.WebViewEndpoint{Package: packageName, Socket: socket, CDPEndpoint: cdpEndpoint, RendererID: rendererID, RendererURL: rendererURL, Transport: "adb-forward"}, nil
}

func (a *Adapter) webViewSocket(ctx context.Context, packageName string) (string, error) {
	var lastErr error
	lastStage := strategy.WebViewStageProcessLookup
	for attempt := 0; attempt < 20; attempt++ {
		pids, err := a.runner.Run(ctx, "adb", a.args("shell", "pidof", packageName)...)
		if err != nil {
			lastStage = strategy.WebViewStageProcessLookup
			lastErr = fmt.Errorf("find WebView process for %q: %w", packageName, err)
		} else {
			pid := strings.Fields(string(pids))
			if len(pid) > 0 && regexp.MustCompile(`^\d+$`).MatchString(pid[0]) {
				sockets, socketErr := a.runner.Run(ctx, "adb", a.args("shell", "cat", "/proc/net/unix")...)
				if socketErr != nil {
					lastStage = strategy.WebViewStageSocketTableRead
					lastErr = fmt.Errorf("list Android WebView sockets: %w", socketErr)
				} else if socket, socketErr := webViewSocketForPID(sockets, pid[0]); socketErr == nil {
					return socket, nil
				} else {
					lastStage = strategy.WebViewStageSocketDiscovery
					lastErr = fmt.Errorf("find WebView socket for %q: %w", packageName, socketErr)
				}
			} else {
				lastStage = strategy.WebViewStageProcessLookup
				lastErr = fmt.Errorf("application %q has no running process", packageName)
			}
		}
		if attempt == 19 {
			break
		}
		select {
		case <-ctx.Done():
			return "", &strategy.WebViewAttachFailure{Stage: lastStage, Err: ctx.Err()}
		case <-time.After(250 * time.Millisecond):
		}
	}
	return "", &strategy.WebViewAttachFailure{Stage: lastStage, Err: lastErr}
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

// probeScreenshot verifies that the device can produce a frame the observer
// path will actually decode. It applies the same log-preamble sanitization used
// at capture time so an Android-TV firmware that prefixes screencap output is
// reported as available, while a device that returns non-image bytes is
// reported unavailable with a decode-specific next action.
func (a *Adapter) probeScreenshot(ctx context.Context) (bool, string) {
	out, err := a.runner.Run(ctx, "adb", a.args("exec-out", "screencap", "-p")...)
	if err != nil {
		return false, "adb screencap is unavailable; verify the device is authorized"
	}
	if _, _, _, decodeErr := validateCapturedImage(out); decodeErr != nil {
		return false, "adb screencap returned bytes device-control cannot decode as an image; the device may lack a display surface (use Cast/Remote for screenless control)"
	}
	return true, "adb screencap is unavailable; verify the device is authorized"
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

func (a *Adapter) ReadState(ctx context.Context) (strategy.DeviceState, error) {
	// A stale ADB transport can remain listed as connected while a shell
	// command blocks indefinitely. Bound the whole observation so another
	// transport can be tried and caller cancellation remains effective.
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	state := strategy.DeviceState{Unavailable: map[string]string{}}
	probe := func(field string, args ...string) string {
		out, err := a.runner.Run(ctx, "adb", a.args(args...)...)
		if err != nil {
			state.Unavailable[field] = "adb " + strings.Join(a.args(args...), " ")
			return ""
		}
		return string(out)
	}
	state.WirelessDebugging = a.readWirelessDebuggingState(ctx)
	if state.WirelessDebugging == nil {
		state.Unavailable["wireless_debugging"] = "Android version and wireless debugging state could not be determined"
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
	rotation := strings.TrimSpace(probe("rotation_setting", "shell", "settings", "get", "system", "user_rotation"))
	state.RotationSettings = map[string]string{}
	if rotation == "null" || rotation == "0" || rotation == "1" || rotation == "2" || rotation == "3" {
		state.RotationSettings["user_rotation"] = rotation
	}
	if state.Orientation == "" {
		state.Orientation = orientationValue(rotation)
		if state.Orientation != "" {
			delete(state.Unavailable, "orientation")
		}
	}
	autoRotate := probe("auto_rotate", "shell", "settings", "get", "system", "accelerometer_rotation")
	if value := strings.TrimSpace(autoRotate); value == "0" || value == "1" || value == "null" {
		state.RotationSettings["accelerometer_rotation"] = value
	}
	if strings.TrimSpace(autoRotate) == "0" || strings.TrimSpace(autoRotate) == "1" {
		state.AutoRotate = strings.TrimSpace(autoRotate) == "1"
		state.AutoRotateKnown = true
	}
	battery := probe("battery", "shell", "dumpsys", "battery")
	state.BatteryLevel, state.Charging = batteryState(battery)
	if state.BatteryLevel < 0 {
		if battery != "" {
			state.Unavailable["battery"] = "adb " + strings.Join(a.args("shell", "dumpsys", "battery"), " ")
		}
	} else {
		state.ChargingKnown = true
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
	// Keep the bounded, additive dreams probe last so it cannot consume the
	// caller's remaining budget before any existing state fields are read.
	dreamStateValue, dreamComponent, unavailableDreamState := probeDreamState(ctx, a.runner, a.args("shell", "dumpsys", "dreams"), dreamStateProbeTimeout)
	state.DreamState, state.DreamComponent = dreamStateValue, dreamComponent
	if unavailableDreamState != "" {
		state.Unavailable["dream_state"] = unavailableDreamState
	}
	if len(state.Unavailable) == 0 {
		state.Unavailable = nil
	}
	if err := ctx.Err(); err != nil {
		return state, fmt.Errorf("android device state observation: %w", err)
	}
	return state, nil
}

func (a *Adapter) readWirelessDebuggingState(ctx context.Context) *strategy.WirelessDebuggingState {
	state := &strategy.WirelessDebuggingState{}
	if raw, err := a.runner.Run(ctx, "adb", a.args("shell", "getprop", "ro.build.version.sdk")...); err == nil {
		if sdk, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && sdk > 0 {
			supported := sdk >= 30
			known := true
			// Android's documented minimum for TV and Wear is API 33, while
			// phones support paired wireless debugging from API 30.
			if sdk >= 30 && sdk < 33 {
				characteristics, err := a.runner.Run(ctx, "adb", a.args("shell", "getprop", "ro.build.characteristics")...)
				value := strings.ToLower(strings.TrimSpace(string(characteristics)))
				if err != nil || value == "" {
					known = false
				} else if strings.Contains(value, "tv") || strings.Contains(value, "watch") {
					supported = false
				}
			}
			if known {
				state.NativeSupported = &supported
			}
			if known && supported {
				if enabledRaw, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "get", "global", "adb_wifi_enabled")...); err == nil {
					switch strings.TrimSpace(string(enabledRaw)) {
					case "0", "1":
						enabled := strings.TrimSpace(string(enabledRaw)) == "1"
						state.NativeEnabled = &enabled
					}
				}
			}
		}
	}
	if raw, err := a.runner.Run(ctx, "adb", a.args("shell", "getprop", "service.adb.tcp.port")...); err == nil {
		if port, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && port >= -1 && port <= 65535 {
			enabled := port > 0
			state.LegacyTCPEnabled = &enabled
			if enabled {
				state.LegacyTCPPort = port
			}
		}
	}
	if state.NativeSupported == nil && state.LegacyTCPEnabled == nil {
		return nil
	}
	return state
}

func (a *Adapter) SetWirelessDebugging(ctx context.Context, enabled bool) error {
	if a.transport != "usb" || strings.TrimSpace(a.serial) == "" {
		return fmt.Errorf("native wireless debugging changes require an authorized serial-bound USB transport")
	}
	before := a.readWirelessDebuggingState(ctx)
	if before == nil || before.NativeSupported == nil || !*before.NativeSupported {
		return &strategy.UnsupportedCapabilityError{Capability: strategy.CapWirelessDebugging, Operation: "set"}
	}
	if before.NativeEnabled != nil && *before.NativeEnabled == enabled {
		return nil
	}
	value := "0"
	if enabled {
		value = "1"
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "put", "global", "adb_wifi_enabled", value)...); err != nil {
		return fmt.Errorf("change native wireless debugging: %w", err)
	}
	after := a.readWirelessDebuggingState(ctx)
	if after == nil || after.NativeEnabled == nil || *after.NativeEnabled != enabled {
		return fmt.Errorf("native wireless debugging change was not verified; the OS may require network authorization")
	}
	return nil
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
			if state != nil {
				if focused := focusedPackage(string(out)); focused != "" && state.ForegroundPackage == "" {
					state.ForegroundPackage = focused
				}
			}
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
	if state.Orientation != "" {
		if len(state.RotationSettings) != 2 {
			return fmt.Errorf("cannot restore rotation without exact raw settings")
		}
		for _, name := range []string{"accelerometer_rotation", "user_rotation"} {
			value, known := state.RotationSettings[name]
			if !known || (value != "null" && value != "0" && value != "1" && value != "2" && value != "3") {
				return fmt.Errorf("cannot restore unknown rotation setting %s", name)
			}
			args := []string{"shell", "settings", "put", "system", name, value}
			if value == "null" {
				args = []string{"shell", "settings", "delete", "system", name}
			}
			if _, err := a.runner.Run(ctx, "adb", a.args(args...)...); err != nil {
				return fmt.Errorf("restore %s: %w", name, err)
			}
		}
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
	if top := topResumedComponent(output); top != "" {
		return strings.SplitN(top, "/", 2)[0]
	}
	matches := regexp.MustCompile(`(?i)(?:mResumedActivity|ResumedActivity):.*?\s(?:u\d+\s+)?([A-Za-z0-9._]+)/(?:[A-Za-z0-9.$_]+)`).FindAllStringSubmatch(output, -1)
	pkg := ""
	for _, match := range matches {
		if pkg != "" && pkg != match[1] {
			return ""
		}
		pkg = match[1]
	}
	return pkg
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
	data, bounds, format, err := validateCapturedImage(data)
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("decode adb screenshot: %w", err)
	}
	return strategy.Frame{Width: bounds.Dx(), Height: bounds.Dy(), Scale: 1, Timestamp: time.Now().UTC(), MediaType: "image/" + format, Bytes: data}, nil
}

func (a *Adapter) ResolveSemantic(ctx context.Context, target string) (strategy.SemanticResult, error) {
	tree, err := a.runner.Run(ctx, "adb", a.args("exec-out", "uiautomator", "dump", "/dev/stdout")...)
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

func (a *Adapter) GoHome(ctx context.Context) error {
	_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_HOME")...)
	return err
}

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

func (a *Adapter) DismissKeyguard(ctx context.Context) error {
	_, err := a.runner.Run(ctx, "adb", a.args("shell", "wm", "dismiss-keyguard")...)
	return err
}

func (a *Adapter) SetScreenState(ctx context.Context, state string) error {
	var key string
	switch state {
	case "on":
		key = "KEYCODE_WAKEUP"
	case "off":
		key = "KEYCODE_SLEEP"
	default:
		return fmt.Errorf("screen state must be on or off")
	}
	_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", key)...)
	return err
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
			output, err := a.launchPackage(ctx, event.Package)
			if event.Output != nil {
				*event.Output = output
			}
			return err
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
			// Refuse mutation if exact recovery cannot be captured. Normalized
			// orientation alone cannot distinguish an absent setting from zero.
			baseline, err := a.ReadState(ctx)
			if err != nil || len(baseline.RotationSettings) != 2 {
				return fmt.Errorf("rotation requires exact raw settings for recovery")
			}
			rotations := map[string]string{"portrait": "0", "landscape": "1", "180": "2", "reverse-portrait": "2", "270": "3", "reverse-landscape": "3"}
			rotation, ok := rotations[strings.ToLower(strings.TrimSpace(event.Value))]
			if !ok {
				return fmt.Errorf("rotation must be portrait, landscape, 180, or 270")
			}
			args = append(args, "shell", "settings", "put", "system", "accelerometer_rotation", "0")
			_, err = a.runner.Run(ctx, "adb", a.args(args...)...)
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
				return a.SetScreenState(ctx, "on")
			case "sleep":
				return a.SetScreenState(ctx, "off")
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
			tree, err := a.runner.Run(ctx, "adb", a.args("exec-out", "uiautomator", "dump", "/dev/stdout")...)
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
			if event.Value == "crash" {
				if !applicationIdentityPattern.MatchString(event.Package) {
					return fmt.Errorf("crash logs require one exact package identity")
				}
				out, err := a.runner.Run(ctx, "adb", a.args("logcat", "-b", "crash", "-d", "-v", "brief", "-t", "500")...)
				if err != nil || len(out) > 1024*1024 {
					return fmt.Errorf("bounded crash buffer is unavailable")
				}
				filtered := packageJavaCrashes(string(out), event.Package)
				if len(filtered) == 0 {
					return fmt.Errorf("no matching Java crash in the bounded recent buffer; other crash types and older exits remain unknown")
				}
				if event.Output != nil {
					*event.Output = filtered
				}
				return nil
			}
			if event.Package != "" {
				pid, pidErr := a.runner.Run(ctx, "adb", a.args("shell", "pidof", event.Package)...)
				if pidErr != nil || strings.TrimSpace(string(pid)) == "" {
					return fmt.Errorf("device logs: package %q is not running", event.Package)
				}
				args = append(args, "logcat", "-d", "-v", "epoch", "--pid", strings.TrimSpace(string(pid)))
			} else {
				args = append(args, "logcat", "-d", "-v", "epoch")
			}
		case "framework-logs":
			if !validExactAndroidComponent(event.Component) {
				return fmt.Errorf("framework-logs requires an exact package/class component")
			}
			if event.LogTag != "ActivityManager" && event.LogTag != "AccessibilityManagerService" {
				return fmt.Errorf("framework-logs supports only ActivityManager or AccessibilityManagerService tags")
			}
			priority := event.LogPriority
			if priority == "" {
				priority = "W"
			}
			if priority != "W" && priority != "I" {
				return fmt.Errorf("framework-logs priority must be W or I")
			}
			probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			bounded, ok := a.runner.(boundedRunner)
			if !ok {
				return fmt.Errorf("bounded framework log output is unsupported by this runner")
			}
			// Android logcat treats the tag priority as a minimum: I includes
			// Info and higher. W remains the default and the only non-opt-in mode.
			out, err := bounded.RunBounded(probeCtx, "adb", 256*1024, a.args("logcat", "-d", "-v", "epoch", "-t", "250", "-s", event.LogTag+":"+priority)...)
			if err != nil {
				return fmt.Errorf("bounded framework log read failed: %w", err)
			}
			matched := filterExactComponentLogLines(string(out), event.Component)
			if event.Output != nil {
				*event.Output = summarizeFrameworkLogMatches(event.Component, event.LogTag, priority, matched)
			}
			return nil
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
		if err == nil && event.Action == "screenshot" {
			var decodeErr error
			out, _, _, decodeErr = validateCapturedImage(out)
			if decodeErr != nil {
				return fmt.Errorf("decode adb screenshot: %w", decodeErr)
			}
		}
		if err == nil && event.Output != nil && (event.Action == "device-logs" || event.Action == "logcat-stop" || event.Action == "clock-sample" || event.Action == "screenshot" || event.Action == "clipboard-read") {
			*event.Output = out
		}
		return err
	}
	return fmt.Errorf("actuation must contain pointer or key event")
}

// AppLifecycle exposes the bounded subset of Android lifecycle commands that
// this adapter can execute without a shell. The control plane validates the
// package identity and confirmation policy before reaching this method.
func (a *Adapter) AppLifecycle(ctx context.Context, request strategy.AppLifecycleRequest) (strategy.AppLifecycleResult, error) {
	operation := strings.ToLower(strings.TrimSpace(request.Operation))
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
	supported := map[string]bool{"launch": true, "focus": true, "close": true, "minimize": true, "restore": true, "stop": true, "uninstall": true, "clear-data": true, "grant-permission": true, "revoke-permission": true, "package-state": true, "package-signing-state": true}
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

const maxSemanticSnapshotNodes = 200
const maxSemanticLabelRunes = 160

// parseSemanticSnapshot returns a compact, bounded view without retaining raw
// accessibility XML. Editable values and password descriptions are omitted.
func parseSemanticSnapshot(tree []byte) (strategy.SemanticSnapshot, error) {
	result := strategy.SemanticSnapshot{Elements: []strategy.SemanticElement{}}
	label := func(value string) string {
		runes := []rune(strings.TrimSpace(value))
		if len(runes) > maxSemanticLabelRunes {
			result.Truncated = true
			runes = runes[:maxSemanticLabelRunes]
		}
		return string(runes)
	}
	optionalBool := func(value string) *bool {
		switch value {
		case "true":
			v := true
			return &v
		case "false":
			v := false
			return &v
		default:
			return nil
		}
	}
	decoder := xml.NewDecoder(bytes.NewReader(tree))
	depth, root := 0, false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return result, fmt.Errorf("parse accessibility hierarchy: %w", err)
		}
		switch element := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				if root || element.Name.Local != "hierarchy" {
					return result, errors.New("accessibility hierarchy root unavailable")
				}
				root = true
			}
			depth++
			if element.Name.Local != "node" {
				continue
			}
			attrs := make(map[string]string, len(element.Attr))
			for _, attr := range element.Attr {
				attrs[attr.Name.Local] = attr.Value
			}
			x0, y0, x1, y1, err := semanticBoundsRect(attrs, "node")
			if err != nil || x1 <= x0 || y1 <= y0 {
				continue
			}
			editable := attrs["password"] == "true" || attrs["editable"] == "true" || strings.HasSuffix(attrs["class"], ".EditText")
			node := strategy.SemanticElement{ResourceID: label(attrs["resource-id"]), Class: label(attrs["class"]), Bounds: [4]int{x0, y0, x1, y1}, Clickable: attrs["clickable"] == "true", Focusable: attrs["focusable"] == "true", Focused: attrs["focused"] == "true", Scrollable: attrs["scrollable"] == "true", Enabled: attrs["enabled"] == "true", Editable: editable}
			node.Selected = optionalBool(attrs["selected"])
			if attrs["checkable"] == "true" {
				node.Checked = optionalBool(attrs["checked"])
			}
			if editable {
				node.TextRedacted = attrs["text"] != "" || attrs["content-desc"] != ""
			} else {
				node.Text, node.Description = label(attrs["text"]), label(attrs["content-desc"])
			}
			if node.ResourceID == "" && node.Text == "" && node.Description == "" && !node.Clickable && !node.Focusable && !node.Scrollable && !node.Editable {
				continue
			}
			if len(result.Elements) >= maxSemanticSnapshotNodes {
				result.Truncated = true
				continue
			}
			result.Elements = append(result.Elements, node)
		case xml.EndElement:
			depth--
		}
	}
	if !root || depth != 0 {
		return result, errors.New("accessibility hierarchy is unavailable or incomplete")
	}
	result.ObservedAt = time.Now().UTC()
	return result, nil
}

func (a *Adapter) ObserveSemantics(ctx context.Context) (strategy.SemanticSnapshot, error) {
	bounded, ok := a.runner.(boundedRunner)
	if !ok {
		return strategy.SemanticSnapshot{}, errors.New("bounded accessibility observation is unavailable")
	}
	data, err := bounded.RunBounded(ctx, "adb", 2*1024*1024, a.args("exec-out", "uiautomator", "dump", "/dev/stdout")...)
	if err != nil {
		return strategy.SemanticSnapshot{}, fmt.Errorf("capture accessibility hierarchy: %w", err)
	}
	return parseSemanticSnapshot(data)
}

// findSemanticNode resolves a stable Android accessibility identifier. The XML
// is device-owned input; only matched attributes and numeric bounds are used.
func findSemanticNode(tree []byte, target string) (map[string]string, error) {
	if strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("accessibility target is required")
	}
	decoder := xml.NewDecoder(bytes.NewReader(tree))
	var match map[string]string
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
		if match != nil {
			return nil, fmt.Errorf("accessibility target %q is ambiguous in the view hierarchy", target)
		}
		match = values
	}
	if match != nil {
		return match, nil
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
