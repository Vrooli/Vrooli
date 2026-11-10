package androidadb

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestRecordingFallbackSamplesStayBoundedAndKeepTimeline(t *testing.T) {
	var samples recordingScreenshots
	start := time.Now()
	for i := 0; i < 5; i++ {
		samples.appendSample([]byte{byte(i), 1, 2, 3}, start.Add(time.Duration(i)*time.Second), 12)
	}
	require.LessOrEqual(t, samples.bytes, 12)
	require.Len(t, samples.frames, len(samples.times))
	require.Equal(t, start, samples.times[0])
	require.Equal(t, start.Add(4*time.Second), samples.times[len(samples.times)-1])
	require.Greater(t, samples.interval, 500*time.Millisecond)
	before := len(samples.frames)
	samples.appendSample(make([]byte, 13), start.Add(5*time.Second), 12)
	require.Len(t, samples.frames, before, "a frame larger than the entire budget cannot be retained")
}

type cancelDuringRecordingSetup struct {
	*processRunner
	cancel             context.CancelFunc
	restoreHasDeadline bool
}

func (r *cancelDuringRecordingSetup) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.Contains(strings.Join(args, " "), "settings put global stay_on_while_plugged_in 2") {
		_, r.restoreHasDeadline = ctx.Deadline()
	}
	out, err := r.processRunner.Run(ctx, name, args...)
	if strings.Contains(strings.Join(args, " "), "svc power stayon true") {
		r.cancel()
		return out, ctx.Err()
	}
	return out, err
}

func TestRecordingStartHonorsCancellationAndRestoresSettings(t *testing.T) {
	base := &processRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{
		"adb -s serial shell settings get global stay_on_while_plugged_in": []byte("2\n"),
		"adb -s serial shell settings get system screen_off_timeout":       []byte("60000\n"),
		"adb -s serial shell dumpsys deviceidle":                           []byte("mDeepEnabled=true\nmLightEnabled=false\n"),
	}}, process: &scriptedProcess{}}
	ctx, cancel := context.WithCancel(context.Background())
	runner := &cancelDuringRecordingSetup{processRunner: base, cancel: cancel}
	handle, err := NewWithRunner(runner, "serial").StartRecording(ctx, strategy.ClaimTransition)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, handle.ID)
	calls := strings.Join(base.calls, "\n")
	require.Contains(t, calls, "settings put global stay_on_while_plugged_in 2")
	require.Contains(t, calls, "settings put system screen_off_timeout 60000")
	require.NotContains(t, calls, "screenrecord")
	require.True(t, runner.restoreHasDeadline, "cleanup must have a bounded independent deadline")
}

func TestRecordingStartAlreadyCanceledHasNoDeviceEffects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &processRunner{scriptedRunner: &scriptedRunner{}, process: &scriptedProcess{}}
	_, err := NewWithRunner(runner, "serial").StartRecording(ctx, strategy.ClaimTransition)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, runner.calls)
}

func TestRecordingStopReportsArtifactAndRestoreSeparately(t *testing.T) {
	for _, tc := range []struct {
		name, failingCommand string
		artifactAvailable    bool
		want                 string
	}{
		{name: "artifact read failed", failingCommand: "read", want: "read screenrecord artifact"},
		{name: "restore failed after artifact", failingCommand: "restore", artifactAvailable: true, want: "restore display keep-awake state"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &processRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{
				"adb -s serial-1 shell settings get global stay_on_while_plugged_in": []byte("2\n"),
				"adb -s serial-1 shell settings get system screen_off_timeout":       []byte("60000\n"),
				"adb -s serial-1 shell dumpsys deviceidle":                           []byte("mDeepEnabled=true\nmLightEnabled=false\n"),
			}, errors: map[string]error{}}, process: &scriptedProcess{}}
			adapter := NewWithRunner(runner, "serial-1")
			handle, err := adapter.StartRecording(context.Background(), strategy.ClaimTransition)
			require.NoError(t, err)
			read := "adb -s serial-1 exec-out cat /data/local/tmp/" + handle.ID + ".mp4"
			runner.mu.Lock()
			runner.responses["adb -s serial-1 shell ps -A -o PID,ARGS"] = []byte("123 screenrecord /data/local/tmp/" + handle.ID + ".mp4\n")
			runner.responses[read] = []byte("ftyp-video")
			if tc.failingCommand == "read" {
				runner.errors[read] = errors.New("injected read failure")
			} else {
				runner.errors["adb -s serial-1 shell settings put global stay_on_while_plugged_in 2"] = errors.New("injected restore failure")
			}
			runner.mu.Unlock()
			artifact, err := adapter.StopRecording(context.Background(), handle)
			require.ErrorContains(t, err, tc.want)
			require.True(t, runner.process.signaled, "host wrapper was signaled; this alone does not prove remote encoder termination")
			if tc.artifactAvailable {
				require.Equal(t, []byte("ftyp-video"), artifact.Bytes)
			} else {
				require.Empty(t, artifact.Bytes)
			}
			runner.mu.Lock()
			calls := strings.Join(runner.calls, "\n")
			runner.mu.Unlock()
			require.Contains(t, calls, "shell settings put system screen_off_timeout 60000", "restoration is attempted even after an artifact error")
			require.Contains(t, adapter.recordings, handle.ID, "failed stop must retain the path and owner state for inspection")
			_, err = adapter.StopRecording(context.Background(), handle)
			require.ErrorContains(t, err, "already has a stop attempt", "native stop is one-shot")
			runner.mu.Lock()
			callsAfterRetry := strings.Join(runner.calls, "\n")
			runner.mu.Unlock()
			require.Equal(t, calls, callsAfterRetry, "refused retry must not send another device command")
		})
	}
}

func TestInspectRecordingReadsProcessAndArtifactWithoutMutation(t *testing.T) {
	const ps = "adb -s serial shell ps -A -o PID,ARGS"
	const artifact = "/data/local/tmp/android-recording-inspect.mp4"
	runner := &scriptedRunner{responses: map[string][]byte{
		ps:                                   []byte("USER PID ARGS\n123 screenrecord " + artifact + "\n"),
		"adb -s serial shell ls " + artifact: []byte(artifact + "\n"),
	}}
	adapter := NewWithRunner(runner, "serial")
	handle := strategy.RecordingHandle{ID: "android-recording-inspect"}
	adapter.recordings[handle.ID] = activeRecording{handle: handle, path: artifact, stopAttempted: true}

	inspection, err := adapter.InspectRecording(context.Background(), handle)
	require.NoError(t, err)
	require.Equal(t, "process_active", inspection.State)
	require.False(t, inspection.ArtifactAvailable)
	require.Equal(t, []string{ps}, runner.calls, "active encoder inspection must not query artifacts or signal the process")
	require.Contains(t, adapter.recordings, handle.ID)
	require.NotContains(t, strings.Join(runner.calls, "\n"), "shell kill")

	runner.mu.Lock()
	runner.responses[ps] = []byte("USER PID ARGS\n")
	runner.calls = nil
	runner.mu.Unlock()
	inspection, err = adapter.InspectRecording(context.Background(), handle)
	require.NoError(t, err)
	require.Equal(t, "process_stopped_artifact_available", inspection.State)
	require.True(t, inspection.ArtifactAvailable)
	require.Equal(t, []string{ps, "adb -s serial shell ls " + artifact}, runner.calls)
	require.Contains(t, adapter.recordings, handle.ID, "readback must retain owner state")
}

func TestStopOwnedScreenrecordRejectsMissingAndAmbiguousProcesses(t *testing.T) {
	const ps = "adb -s serial shell ps -A -o PID,ARGS"
	const path = "/sdcard/owned.mp4"
	for _, tc := range []struct{ name, output string }{
		{"missing", "123 screenrecord /sdcard/other.mp4\n"},
		{"ambiguous", "123 screenrecord /sdcard/owned.mp4\n124 screenrecord /sdcard/owned.mp4\n"},
		{"different executable", "123 sh /sdcard/owned.mp4\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &scriptedRunner{responses: map[string][]byte{ps: []byte(tc.output)}}
			err := NewWithRunner(runner, "serial").stopOwnedScreenrecord(context.Background(), path)
			require.Error(t, err)
			require.NotContains(t, strings.Join(runner.calls, "\n"), "shell kill")
		})
	}
}

func TestRecordingProcessReportsBoundedEncoderFailure(t *testing.T) {
	process, err := (commandRunner{}).Start("sh", "-c", "printf 'encoder unavailable' >&2; exit 2")
	require.NoError(t, err)
	require.NoError(t, process.Start())
	require.ErrorContains(t, process.Wait(), "encoder unavailable")
}

func TestRecordingRetentionRestoresIndependentDozeAndUnsetSettings(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial shell dumpsys deviceidle":                           []byte("mLightEnabled=false mDeepEnabled=true\n"),
		"adb -s serial shell settings get global stay_on_while_plugged_in": []byte("null\n"),
		"adb -s serial shell settings get system screen_off_timeout":       []byte("null\n"),
	}}
	adapter := NewWithRunner(runner, "serial")
	state, err := adapter.readDozeEnabled(context.Background())
	require.NoError(t, err)
	require.True(t, state.deep)
	require.False(t, state.light)
	stay, err := adapter.readStayAwake(context.Background())
	require.NoError(t, err)
	timeout, err := adapter.readScreenOffTimeout(context.Background())
	require.NoError(t, err)
	require.NoError(t, adapter.restoreStayAwake(context.Background(), stay))
	require.NoError(t, adapter.setDisplayRetention(context.Background(), timeout, state))
	calls := strings.Join(runner.calls, "\n")
	require.Contains(t, calls, "settings delete global stay_on_while_plugged_in")
	require.Contains(t, calls, "settings delete system screen_off_timeout")
	require.Contains(t, calls, "dumpsys deviceidle enable deep")
	require.Contains(t, calls, "dumpsys deviceidle disable light")
}
