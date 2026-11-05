package androidadb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

const audioCapturePackage = "com.vrooli.devicecontrol.capture"

type audioCaptureStatus struct {
	State                    string  `json:"state"`
	Detail                   string  `json:"detail"`
	CaptureStartElapsedNanos int64   `json:"capture_start_elapsed_nanos"`
	FirstSampleElapsedNanos  int64   `json:"first_sample_elapsed_nanos"`
	Peak                     int     `json:"peak"`
	RMSDBFS                  float64 `json:"rms_dbfs"`
}

func audioCaptureDirectory(sessionID string) string {
	return "/sdcard/Android/data/" + audioCapturePackage + "/files/Music/DeviceControlAudio/" + sessionID
}

func decodeAudioCaptureStatus(raw []byte) (audioCaptureStatus, error) {
	var status audioCaptureStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return status, fmt.Errorf("decode Android playback capture status: %w", err)
	}
	if status.State == "" {
		return status, errors.New("Android playback capture status has no state")
	}
	return status, nil
}

func audioTrimOffset(firstSampleElapsedNanos, videoStartElapsedNanos int64) (time.Duration, error) {
	if firstSampleElapsedNanos <= 0 || videoStartElapsedNanos <= 0 {
		return 0, errors.New("Android capture timestamps are missing")
	}
	if videoStartElapsedNanos < firstSampleElapsedNanos {
		return 0, errors.New("video started before audio capture; synchronized mux is unsafe")
	}
	offset := time.Duration(videoStartElapsedNanos-firstSampleElapsedNanos) * time.Nanosecond
	if offset > 40*time.Second {
		return 0, errors.New("audio/video start offset exceeds the capture bound")
	}
	return offset, nil
}

func parseDeviceUptime(raw []byte) (int64, error) {
	fields := strings.Fields(string(raw))
	if len(fields) == 0 {
		return 0, errors.New("device uptime is empty")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return 0, errors.New("device uptime is invalid")
	}
	return int64(seconds * float64(time.Second)), nil
}

func (a *Adapter) readDeviceUptime(ctx context.Context) (int64, error) {
	output, err := a.runner.Run(ctx, "adb", a.args("shell", "cat", "/proc/uptime")...)
	if err != nil {
		return 0, err
	}
	return parseDeviceUptime(output)
}

func (a *Adapter) probeAudioCapture(ctx context.Context) strategy.Capability {
	next := "Install the Device Control Capture companion from the governed Android build artifact, then retry."
	sdkOutput, sdkErr := a.runner.Run(ctx, "adb", a.args("shell", "getprop", "ro.build.version.sdk")...)
	sdk, parseErr := strconv.Atoi(strings.TrimSpace(string(sdkOutput)))
	if sdkErr != nil || parseErr != nil {
		return strategy.Capability{Name: strategy.CapRecordingAudio, Status: strategy.StatusUnavailable, Reason: "could not verify the Android API level required for playback capture", NextAction: "Verify the device is connected and reports its Android SDK version.", ProbeEvidence: "adb shell getprop ro.build.version.sdk"}
	}
	if sdk < 29 {
		return strategy.Capability{Name: strategy.CapRecordingAudio, Status: strategy.StatusUnsupported, Reason: "Android playback capture requires Android 10 (API 29) or newer", ProbeEvidence: "Android SDK " + strconv.Itoa(sdk)}
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return strategy.Capability{Name: strategy.CapRecordingAudio, Status: strategy.StatusUnavailable, Reason: "host ffmpeg is unavailable for muxing captured playback audio", NextAction: "Provision the governed ffmpeg host capability, then retry."}
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return strategy.Capability{Name: strategy.CapRecordingAudio, Status: strategy.StatusUnavailable, Reason: "host ffprobe is unavailable for verifying the muxed audio track", NextAction: "Provision the governed ffprobe host capability, then retry."}
	}
	output, err := a.runner.Run(ctx, "adb", a.args("shell", "pm", "path", audioCapturePackage)...)
	if err != nil || !strings.Contains(string(output), "package:") {
		return strategy.Capability{Name: strategy.CapRecordingAudio, Status: strategy.StatusUnavailable, Reason: "Device Control Capture companion is not installed on this Android device", NextAction: next, ProbeEvidence: "adb shell pm path " + audioCapturePackage}
	}
	return strategy.Capability{Name: strategy.CapRecordingAudio, Status: strategy.StatusAvailable, ProbeEvidence: "installed MediaProjection playback-audio companion; each session still requires Android permission and user consent", NextAction: "Approve Android audio and screen-capture prompts for each requested session."}
}

func (a *Adapter) startAudioCapture(ctx context.Context, sessionID string) error {
	if !safeAudioSessionID(sessionID) {
		return errors.New("invalid Android audio capture session ID")
	}
	capability := a.probeAudioCapture(ctx)
	if capability.Status != strategy.StatusAvailable {
		return &strategy.AvailabilityError{Reason: capability.Reason, NextAction: capability.NextAction}
	}
	_, _ = a.runner.Run(ctx, "adb", a.args("shell", "rm", "-rf", audioCaptureDirectory(sessionID))...)
	component := audioCapturePackage + "/.CaptureActivity"
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "am", "start", "-W", "-n", component, "--es", "session_id", sessionID)...); err != nil {
		return fmt.Errorf("open Android playback-capture consent activity: %w", err)
	}
	if err := a.waitForAudioCaptureReady(ctx, sessionID, 60*time.Second); err != nil {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cleanupCancel()
		_, _, _ = a.stopAudioCapture(cleanupCtx, sessionID)
		return err
	}
	return nil
}

func (a *Adapter) waitForAudioCaptureReady(ctx context.Context, sessionID string, timeout time.Duration) error {
	consentCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, err := a.readAudioCaptureStatus(consentCtx, sessionID)
		if err == nil {
			switch status.State {
			case "recording":
				return nil
			case "audio_permission_denied", "projection_consent_denied", "failed", "unavailable":
				return &strategy.AvailabilityError{Reason: status.Detail, NextAction: "Grant RECORD_AUDIO and approve Android's projection prompt, then retry."}
			}
		}
		select {
		case <-consentCtx.Done():
			if err := ctx.Err(); err != nil {
				return err
			}
			return audioCaptureConsentTimeoutError()
		case <-ticker.C:
		}
	}
}

func audioCaptureConsentTimeoutError() *strategy.AvailabilityError {
	return &strategy.AvailabilityError{
		Reason:     "Android playback capture did not become ready within 60 seconds; approve the on-device audio and screen-capture prompts, then retry.",
		NextAction: "Keep the phone awake and unlocked. Approve the companion's audio permission prompt and Android's per-session screen-capture prompt, then retry. If both were approved, inspect the companion status and device logs.",
	}
}

func (a *Adapter) stopAudioCapture(ctx context.Context, sessionID string) ([]byte, audioCaptureStatus, error) {
	if !safeAudioSessionID(sessionID) {
		return nil, audioCaptureStatus{}, errors.New("invalid Android audio capture session ID")
	}
	component := audioCapturePackage + "/.CaptureActivity"
	_, stopErr := a.runner.Run(ctx, "adb", a.args("shell", "am", "start", "-W", "-n", component, "-a", "com.vrooli.devicecontrol.capture.STOP", "--es", "session_id", sessionID)...)
	deadline, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	var status audioCaptureStatus
	for {
		status, _ = a.readAudioCaptureStatus(deadline, sessionID)
		if status.State == "stopped" || status.State == "failed" || status.State == "duration_limit" || status.State == "projection_stopped" {
			break
		}
		select {
		case <-deadline.Done():
			if stopErr != nil {
				return nil, status, errors.Join(fmt.Errorf("stop Android playback capture: %w", stopErr), deadline.Err())
			}
			return nil, status, fmt.Errorf("Android playback capture did not finalize: %w", deadline.Err())
		case <-ticker.C:
		}
	}
	if status.State != "stopped" {
		_, _ = a.runner.Run(ctx, "adb", a.args("shell", "rm", "-rf", audioCaptureDirectory(sessionID))...)
		return nil, status, fmt.Errorf("Android playback capture ended as %s: %s", status.State, status.Detail)
	}
	if stopErr != nil {
		_, _ = a.runner.Run(ctx, "adb", a.args("shell", "rm", "-rf", audioCaptureDirectory(sessionID))...)
		return nil, status, fmt.Errorf("stop Android playback capture: %w", stopErr)
	}
	wavePath := audioCaptureDirectory(sessionID) + "/playback.wav"
	wave, err := a.runner.Run(ctx, "adb", a.args("exec-out", "cat", wavePath)...)
	if err != nil {
		_, _ = a.runner.Run(ctx, "adb", a.args("shell", "rm", "-rf", audioCaptureDirectory(sessionID))...)
		return nil, status, fmt.Errorf("read Android playback audio: %w", err)
	}
	if len(wave) < 44 || string(wave[:4]) != "RIFF" || string(wave[8:12]) != "WAVE" {
		_, _ = a.runner.Run(ctx, "adb", a.args("shell", "rm", "-rf", audioCaptureDirectory(sessionID))...)
		return nil, status, errors.New("Android playback capture did not produce a valid WAV stream")
	}
	_, _ = a.runner.Run(ctx, "adb", a.args("shell", "rm", "-rf", audioCaptureDirectory(sessionID))...)
	return wave, status, nil
}

// cleanupAudioCapture is used when video recording fails before its ordinary
// stop path can collect the WAV. It stops the matching capture session, waits
// for its terminal status, and removes private session files.
func (a *Adapter) cleanupAudioCapture(ctx context.Context, sessionID string) error {
	if !safeAudioSessionID(sessionID) {
		return errors.New("invalid Android audio capture session ID")
	}
	status, statusErr := a.readAudioCaptureStatus(ctx, sessionID)
	switch status.State {
	case "stopped", "failed", "duration_limit", "projection_stopped", "audio_permission_denied", "projection_consent_denied", "unavailable":
		_, cleanupErr := a.runner.Run(ctx, "adb", a.args("shell", "rm", "-rf", audioCaptureDirectory(sessionID))...)
		return cleanupErr
	}
	if statusErr == nil {
		component := audioCapturePackage + "/.CaptureActivity"
		_, _ = a.runner.Run(ctx, "adb", a.args("shell", "am", "start", "-W", "-n", component, "-a", "com.vrooli.devicecontrol.capture.STOP", "--es", "session_id", sessionID)...)
	}
	deadline, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, statusErr = a.readAudioCaptureStatus(deadline, sessionID)
		if statusErr == nil {
			switch status.State {
			case "stopped", "failed", "duration_limit", "projection_stopped", "audio_permission_denied", "projection_consent_denied", "unavailable":
				_, cleanupErr := a.runner.Run(deadline, "adb", a.args("shell", "rm", "-rf", audioCaptureDirectory(sessionID))...)
				return cleanupErr
			}
		}
		select {
		case <-deadline.Done():
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cleanupCancel()
			_, cleanupErr := a.runner.Run(cleanupCtx, "adb", a.args("shell", "rm", "-rf", audioCaptureDirectory(sessionID))...)
			return errors.Join(fmt.Errorf("Android audio capture cleanup did not reach terminal status: %w", deadline.Err()), cleanupErr)
		case <-ticker.C:
		}
	}
}

func (a *Adapter) readAudioCaptureStatus(ctx context.Context, sessionID string) (audioCaptureStatus, error) {
	path := audioCaptureDirectory(sessionID) + "/status.json"
	raw, err := a.runner.Run(ctx, "adb", a.args("exec-out", "cat", path)...)
	if err != nil {
		return audioCaptureStatus{}, err
	}
	return decodeAudioCaptureStatus(raw)
}

func safeAudioSessionID(id string) bool {
	if len(id) == 0 || len(id) > 80 {
		return false
	}
	for _, char := range id {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' {
			continue
		}
		return false
	}
	return true
}

func muxPlaybackAudio(ctx context.Context, video, wave []byte, trim time.Duration) ([]byte, error) {
	if len(video) == 0 || len(wave) < 44 || string(wave[:4]) != "RIFF" || string(wave[8:12]) != "WAVE" {
		return nil, errors.New("Android recording is missing a valid video or WAV audio stream")
	}
	if trim < 0 || trim > 40*time.Second {
		return nil, errors.New("Android audio alignment offset is outside the capture bound")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg is required to mux Android playback audio: %w", err)
	}
	dir, err := os.MkdirTemp("", "device-control-audio-mux-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	videoPath := filepath.Join(dir, "video.mp4")
	audioPath := filepath.Join(dir, "audio.wav")
	outputPath := filepath.Join(dir, "recording.mp4")
	if err := os.WriteFile(videoPath, video, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(audioPath, wave, 0o600); err != nil {
		return nil, err
	}
	args := []string{"-y", "-v", "error", "-i", videoPath}
	if trim > 0 {
		args = append(args, "-ss", strconv.FormatFloat(trim.Seconds(), 'f', 6, 64))
	}
	args = append(args, "-i", audioPath, "-map", "0:v:0", "-map", "1:a:0", "-c:v", "copy", "-c:a", "aac", "-b:a", "128k", "-shortest", "-movflags", "+faststart", outputPath)
	command := exec.CommandContext(ctx, "ffmpeg", args...)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("mux Android playback audio: %w: %s", err, strings.TrimSpace(string(output)))
	}
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() <= 0 || info.Size() > 64<<20 {
		return nil, errors.New("muxed Android audio/video artifact is missing or exceeds 64 MiB")
	}
	result, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}
	return result, nil
}
