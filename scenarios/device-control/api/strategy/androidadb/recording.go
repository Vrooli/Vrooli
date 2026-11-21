package androidadb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"device-control/strategy"
)

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
