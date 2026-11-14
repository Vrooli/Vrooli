package androidadb

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

func TestAudioCaptureStatusRequiresState(t *testing.T) {
	status, err := decodeAudioCaptureStatus([]byte(`{"state":"recording","first_sample_elapsed_nanos":123}`))
	require.NoError(t, err)
	require.Equal(t, "recording", status.State)
	require.EqualValues(t, 123, status.FirstSampleElapsedNanos)
	_, err = decodeAudioCaptureStatus([]byte(`{"detail":"missing state"}`))
	require.ErrorContains(t, err, "no state")
	_, err = decodeAudioCaptureStatus([]byte(`{`))
	require.Error(t, err)
}

func TestAudioCaptureConsentTimeoutExplainsTheDeviceAction(t *testing.T) {
	err := audioCaptureConsentTimeoutError()
	require.Contains(t, err.Error(), "approve the on-device audio and screen-capture prompts")
	require.Contains(t, err.NextAction, "Keep the phone awake and unlocked")
	require.Contains(t, err.NextAction, "per-session screen-capture prompt")
}

func TestWaitForAudioCaptureConsentTimeoutReturnsActionableAvailabilityError(t *testing.T) {
	adapter := NewWithRunner(&scriptedRunner{}, "phone")
	err := adapter.waitForAudioCaptureReady(context.Background(), "session-1", 10*time.Millisecond)

	var unavailable *strategy.AvailabilityError
	require.ErrorAs(t, err, &unavailable)
	require.Contains(t, unavailable.Reason, "did not become ready")
	require.Contains(t, unavailable.NextAction, "Approve the companion's audio permission prompt")
}

func TestAudioCaptureCapabilityRequiresAndroidTen(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{"adb -s phone shell getprop ro.build.version.sdk": []byte("28\n")}}
	capability := NewWithRunner(runner, "phone").probeAudioCapture(context.Background())
	require.Equal(t, strategy.StatusUnsupported, capability.Status)
	require.Contains(t, capability.Reason, "Android 10")
	require.Equal(t, []string{"adb -s phone shell getprop ro.build.version.sdk"}, runner.calls)
}

func TestMuxPlaybackAudioProducesVerifiableTrack(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg unavailable")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe unavailable")
	}
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "video.mp4")
	wavePath := filepath.Join(dir, "audio.wav")
	makeFixture := func(args ...string) {
		cmd := exec.Command(ffmpeg, append([]string{"-y", "-v", "error"}, args...)...)
		output, runErr := cmd.CombinedOutput()
		require.NoError(t, runErr, string(output))
	}
	makeFixture("-f", "lavfi", "-i", "color=c=black:s=320x240:r=10:d=1", "-an", "-c:v", "mpeg4", videoPath)
	makeFixture("-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=1", "-ac", "2", "-c:a", "pcm_s16le", wavePath)
	video, err := os.ReadFile(videoPath)
	require.NoError(t, err)
	wave, err := os.ReadFile(wavePath)
	require.NoError(t, err)
	result, err := muxPlaybackAudio(context.Background(), video, wave, 100*time.Millisecond)
	require.NoError(t, err)
	outputPath := filepath.Join(dir, "muxed.mp4")
	require.NoError(t, os.WriteFile(outputPath, result, 0o600))
	probe := exec.Command(ffprobe, "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=codec_name,sample_rate,channels", "-of", "default=noprint_wrappers=1", outputPath)
	output, err := probe.CombinedOutput()
	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "codec_name=aac")
	require.Contains(t, string(output), "sample_rate=48000")
	require.Contains(t, string(output), "channels=2")
}

func TestAudioVideoAlignmentRejectsUnsafeOffsets(t *testing.T) {
	const base = int64(50_000_000_000)
	offset, err := audioTrimOffset(base, base+250_000_000)
	require.NoError(t, err)
	require.Equal(t, 250*time.Millisecond, offset)
	_, err = audioTrimOffset(base, base-1)
	require.ErrorContains(t, err, "video started before audio")
	_, err = audioTrimOffset(base, base+41_000_000_000)
	require.ErrorContains(t, err, "capture bound")
	_, err = audioTrimOffset(0, base)
	require.ErrorContains(t, err, "timestamps are missing")
}

func TestAudioHelpersRejectInvalidInputs(t *testing.T) {
	for _, id := range []string{"", "../escape", "has space", string(make([]byte, 81))} {
		require.False(t, safeAudioSessionID(id), "session ID %q should be rejected", id)
	}
	require.True(t, safeAudioSessionID("android-recording-123"))
	got, err := parseDeviceUptime([]byte("123.500 45.0\n"))
	require.NoError(t, err)
	require.Equal(t, int64(123500*time.Millisecond), got)
	for _, value := range []string{"", "0 1", "NaN 1", "bad"} {
		_, err := parseDeviceUptime([]byte(value))
		require.Error(t, err)
	}
	_, err = parseDeviceUptime([]byte(json.Number("+Inf").String()))
	require.Error(t, err)
	_, err = muxPlaybackAudio(context.Background(), []byte("not video"), make([]byte, 44), 0)
	require.Error(t, err)
	_, err = muxPlaybackAudio(context.Background(), []byte("video"), []byte("not audio"), time.Duration(math.MaxInt64))
	require.Error(t, err)
}
