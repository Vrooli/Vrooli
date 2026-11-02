package control

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCaptureUsesProbedObserverAndPreservesFullFrame(t *testing.T) {
	adapter := fakes.New("capture-fixture", strategy.StatusAvailable, strategy.CapScreenshot)
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Capture(context.Background(), CaptureRequest{Device: "target", Kind: "screenshot", Actor: "test"})
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, result.Evidence, 1)
	require.Equal(t, "image", result.Evidence[0].Kind)
	require.False(t, result.Evidence[0].RedactionVerified)
	require.Contains(t, result.Evidence[0].AppliedRules, "redaction_not_requested")
	require.Empty(t, svc.ListLiveSessions())
}

func TestCaptureReportsMissingRecordingCapabilityWithoutActuation(t *testing.T) {
	adapter := fakes.New("capture-fixture", strategy.StatusAvailable, strategy.CapScreenshot)
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Capture(context.Background(), CaptureRequest{Device: "target", Kind: "recording", Seconds: 1})
	require.NoError(t, err)
	require.Equal(t, "capability_gap", result.Disposition)
	require.Equal(t, "target", result.Binding.DeviceID)
	require.Empty(t, adapter.Calls())
	require.Empty(t, svc.ListLiveSessions())
}

func TestCaptureNetworkAndConsoleDoNotClaimAbsentEvents(t *testing.T) {
	adapter := fakes.New("capture-fixture", strategy.StatusAvailable, strategy.CapScreenshot)
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	for _, kind := range []string{"network", "console"} {
		result, err := svc.Capture(context.Background(), CaptureRequest{Device: "target", Kind: kind})
		require.NoError(t, err)
		require.Equal(t, "no_evidence", result.Disposition)
		require.Contains(t, result.Chapters[0].Message, "No conclusion")
	}
	require.Empty(t, adapter.Calls())
}

type appLogFixture struct {
	*fakes.Strategy
	output      []byte
	packageName string
	err         error
}

func (f *appLogFixture) ReadApplicationLogs(_ context.Context, pkg string) ([]byte, error) {
	f.packageName = pkg
	return f.output, f.err
}

func TestCaptureUnavailableApplicationLogsAreNotNegativeEvidence(t *testing.T) {
	adapter := &appLogFixture{Strategy: fakes.New("log-fixture", strategy.StatusAvailable, strategy.CapDeviceLogs), err: &strategy.EvidenceUnavailableError{Reason: "application process is unavailable"}}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Capture(context.Background(), CaptureRequest{Device: "target", Kind: "logs", Package: "com.example.app"})
	require.NoError(t, err)
	require.Equal(t, "no_evidence", result.Disposition)
	require.Contains(t, result.Chapters[0].Message, "process is unavailable")
	require.Empty(t, result.Evidence)
}

func TestCaptureApplicationLogsKeepsEmptyOutputAsNoEvidence(t *testing.T) {
	adapter := &appLogFixture{Strategy: fakes.New("log-fixture", strategy.StatusAvailable, strategy.CapDeviceLogs)}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Capture(context.Background(), CaptureRequest{Device: "target", Kind: "logs", Package: "com.example.app"})
	require.NoError(t, err)
	require.Equal(t, "no_evidence", result.Disposition)
	require.Equal(t, "no_evidence", svc.runs[result.RunID].Disposition)
	require.Equal(t, "com.example.app", adapter.packageName)
	require.Empty(t, result.Evidence)
	require.Empty(t, svc.ListLiveSessions())
}

func TestDarkRecordingIsRetainedButDoesNotProveVisibleContent(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	path := filepath.Join(t.TempDir(), "dark.mp4")
	out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=black:s=32x32:r=10:d=1", "-c:v", "libx264", "-y", path).CombinedOutput()
	require.NoError(t, err, string(out))
	video, err := os.ReadFile(path)
	require.NoError(t, err)
	fixture := &flowRecordingFixture{Strategy: fakes.New("recorder", strategy.StatusAvailable, strategy.CapScreenRecording), artifact: strategy.RecordingArtifact{Bytes: video, Method: "native", ClaimClass: strategy.ClaimTransition}}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Capture(context.Background(), CaptureRequest{Device: "target", Kind: "recording", Seconds: 1})
	require.NoError(t, err)
	require.Equal(t, "no_evidence", result.Disposition)
	require.Len(t, result.Evidence, 1)
	require.False(t, result.Evidence[0].ContentVerified)
	require.Equal(t, "degraded", string(result.Evidence[0].Disposition))
	require.Contains(t, result.Evidence[0].DispositionReason, "visible content cannot be verified")
	retained, _, err := svc.Artifact(result.Evidence[0].ID)
	require.NoError(t, err)
	require.Equal(t, video, retained)
}

type flowRecordingFixture struct {
	*fakes.Strategy
	starts     int
	stops      int
	stopErr    error
	cancel     context.CancelFunc
	cleanupErr error
	artifact   strategy.RecordingArtifact
}

type audioRecordingFixture struct {
	*flowRecordingFixture
	audioStarts int
}

func (f *audioRecordingFixture) StartRecordingWithAudio(ctx context.Context, class strategy.ClaimClass) (strategy.RecordingHandle, error) {
	f.audioStarts++
	return f.StartRecording(ctx, class)
}

func TestCaptureRequiredAudioIsNotSilentlyDropped(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for video evidence")
	}
	for _, includeTrack := range []bool{false, true} {
		t.Run(fmt.Sprintf("track-%t", includeTrack), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "recording.mp4")
			args := []string{"-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=10:d=1"}
			if includeTrack {
				args = append(args, "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-c:a", "aac")
			}
			args = append(args, "-c:v", "libx264", "-t", "1", path)
			out, err := exec.Command("ffmpeg", args...).CombinedOutput()
			require.NoError(t, err, string(out))
			video, err := os.ReadFile(path)
			require.NoError(t, err)
			fixture := &audioRecordingFixture{flowRecordingFixture: &flowRecordingFixture{Strategy: fakes.New("recorder", strategy.StatusAvailable, strategy.CapScreenRecording, strategy.CapRecordingAudio), artifact: strategy.RecordingArtifact{Bytes: video, Method: "native", ClaimClass: strategy.ClaimTransition}}}
			svc := New(strategyregistry.New(fixture))
			svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
			result, err := svc.Capture(context.Background(), CaptureRequest{Device: "target", Kind: "recording", Seconds: 1, RequireAudio: true})
			require.NoError(t, err)
			require.Equal(t, 1, fixture.audioStarts)
			if includeTrack {
				require.Equal(t, "passed", result.Disposition)
			} else {
				require.Equal(t, "no_evidence", result.Disposition)
				require.Contains(t, result.Evidence[0].DispositionReason, "audio track")
			}
		})
	}
}

func TestCaptureRequiredAudioRejectsVideoOnlyAdapterBeforeRecording(t *testing.T) {
	fixture := &flowRecordingFixture{Strategy: fakes.New("recorder", strategy.StatusAvailable, strategy.CapScreenRecording)}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Capture(context.Background(), CaptureRequest{Device: "target", Kind: "recording", Seconds: 1, RequireAudio: true})
	require.NoError(t, err)
	require.Equal(t, "capability_gap", result.Disposition)
	require.Equal(t, "target", result.Binding.DeviceID)
	require.Equal(t, 0, fixture.stops)
	require.Equal(t, 0, fixture.starts)
}

func TestFlowRequiredAudioNeedsExecutableAudioRecorder(t *testing.T) {
	fixture := &flowRecordingFixture{Strategy: fakes.New("recorder", strategy.StatusAvailable, strategy.CapScreenRecording, strategy.CapRecordingAudio)}
	flow := Flow{Steps: []Step{{ID: "record", Kind: "recording-start", Arguments: map[string]any{"require_audio": true}}}}
	report := validateAgainstDeclaration(context.Background(), flow, fixture)
	require.False(t, report.Runnable)
	require.Contains(t, strings.Join(report.Gaps, " "), "requires recording with audio")
	flow.Steps[0].Arguments["require_audio"] = "true"
	report = validateAgainstDeclaration(context.Background(), flow, fixture)
	require.False(t, report.Runnable)
	require.Contains(t, strings.Join(report.Gaps, " "), "boolean require_audio")
}

func (f *flowRecordingFixture) StartRecording(context.Context, strategy.ClaimClass) (strategy.RecordingHandle, error) {
	f.starts++
	if f.cancel != nil {
		f.cancel()
	}
	return strategy.RecordingHandle{ID: "recording-fixture"}, nil
}
func (f *flowRecordingFixture) StopRecording(ctx context.Context, _ strategy.RecordingHandle) (strategy.RecordingArtifact, error) {
	f.stops++
	f.cleanupErr = ctx.Err()
	return f.artifact, f.stopErr
}

func TestCaptureRejectsOneFrameAsRecordingEvidence(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for video evidence")
	}
	path := filepath.Join(t.TempDir(), "one.mp4")
	require.NoError(t, exec.Command("ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=25", "-frames:v", "1", "-c:v", "libx264", "-pix_fmt", "yuv420p", path).Run())
	video, err := os.ReadFile(path)
	require.NoError(t, err)
	fixture := &flowRecordingFixture{Strategy: fakes.New("recorder", strategy.StatusAvailable, strategy.CapScreenRecording), artifact: strategy.RecordingArtifact{Bytes: video, Method: "native", ClaimClass: strategy.ClaimTransition}}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Capture(context.Background(), CaptureRequest{Device: "target", Kind: "recording", Seconds: 1})
	require.NoError(t, err)
	require.Equal(t, "no_evidence", result.Disposition)
	require.Len(t, result.Evidence, 1)
	require.Equal(t, "degraded", string(result.Evidence[0].Disposition))
	require.Equal(t, 1, fixture.stops)
}

func TestFlowRecordingCleanupSurvivesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fixture := &flowRecordingFixture{Strategy: fakes.New("recorder", strategy.StatusAvailable, strategy.CapScreenRecording), cancel: cancel}
	svc := New(strategyregistry.New(fixture))
	result, err := svc.Run(ctx, Flow{Steps: []Step{{ID: "start", Kind: "recording-start"}, {ID: "wait", Kind: "wait"}}}, fixture.ID(), "test")
	require.NoError(t, err)
	require.Equal(t, "cancelled", result.Disposition)
	require.Equal(t, 1, fixture.stops)
	require.NoError(t, fixture.cleanupErr)
	require.True(t, result.Incomplete)
}

func TestFlowStopsRecordingWhenLaterStepFails(t *testing.T) {
	fixture := &flowRecordingFixture{Strategy: fakes.New("recorder", strategy.StatusAvailable, strategy.CapScreenRecording)}
	svc := New(strategyregistry.New(fixture))
	result, err := svc.Run(context.Background(), Flow{Steps: []Step{{ID: "start", Kind: "recording-start"}, {ID: "fail", Kind: "wait", Preconditions: []Condition{{Kind: "device-unlocked"}}}}}, fixture.ID(), "test")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Equal(t, 1, fixture.stops)
}

func TestFlowDoesNotRepeatUncertainRecordingStop(t *testing.T) {
	fixture := &flowRecordingFixture{Strategy: fakes.New("recorder", strategy.StatusAvailable, strategy.CapScreenRecording), stopErr: errors.New("stop uncertain")}
	svc := New(strategyregistry.New(fixture))
	result, err := svc.Run(context.Background(), Flow{Steps: []Step{{ID: "start", Kind: "recording-start"}, {ID: "stop", Kind: "recording-stop", Target: "start"}}}, fixture.ID(), "test")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Equal(t, 1, fixture.stops)
	require.True(t, svc.externalRecordings["recording-fixture"].Uncertain)
	require.True(t, result.Incomplete)
}

func TestFinalizeReviewRecordingRetainsProducerOwnedReviewCopy(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for the recording producer test")
	}
	svc, _ := testService(t)
	svc.SetEvidenceDir(t.TempDir())
	session, err := svc.Acquire("fake", "operator", 5*time.Minute)
	require.NoError(t, err)
	chapterPath := filepath.Join(t.TempDir(), "chapter.mp4")
	require.NoError(t, exec.Command("ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=5:d=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", chapterPath).Run())
	chapter, err := os.ReadFile(chapterPath)
	require.NoError(t, err)
	require.NoError(t, svc.persistRecording(context.Background(), "chapter-1", chapter, "fake", session.ID))
	require.NoError(t, svc.persistRecording(context.Background(), "chapter-2", chapter, "fake", session.ID))
	// A restarted service can find retained bytes on disk but has no in-memory
	// kind or path index for them.
	delete(svc.artifacts, "chapter-2")
	delete(svc.artifactKinds, "chapter-2")

	result, err := svc.FinalizeReviewRecording(context.Background(), "fake", "operator", session.LeaseToken, []string{"chapter-1", "chapter-2"})
	require.NoError(t, err)
	require.NotEmpty(t, result.Reference.ID)
	require.False(t, result.Reference.RedactionVerified)
	require.Contains(t, result.Reference.AppliedRules, "redaction_not_requested")
	require.True(t, result.Reference.ContentVerified)
	require.Equal(t, "chapter-concat", result.Reference.RecordingMethod)
	require.Equal(t, "no_chapters", result.Reference.AudioTrackCoverage)
	require.NotNil(t, result.Reference.AudioTrackPresent)
	require.False(t, *result.Reference.AudioTrackPresent)
	require.True(t, filepath.IsAbs(result.Path))
	_, err = os.Stat(result.Path)
	require.NoError(t, err)
	retained, kind, err := svc.Artifact(result.Reference.ID)
	require.NoError(t, err)
	require.Equal(t, "video", kind)
	require.NotEmpty(t, retained)
}

func TestFinalizeReviewRecordingPreservesAudioAndLabelsMissingChapters(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	for _, secondHasAudio := range []bool{true, false} {
		t.Run(fmt.Sprintf("second-audio-%t", secondHasAudio), func(t *testing.T) {
			svc, _ := testService(t)
			svc.SetEvidenceDir(t.TempDir())
			session, err := svc.Acquire("fake", "operator", 5*time.Minute)
			require.NoError(t, err)
			for index, withAudio := range []bool{true, secondHasAudio} {
				path := filepath.Join(t.TempDir(), "chapter.mp4")
				args := []string{"-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=10:d=1"}
				if withAudio {
					args = append(args, "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-c:a", "aac")
				}
				args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-t", "1", path)
				out, err := exec.Command("ffmpeg", args...).CombinedOutput()
				require.NoError(t, err, string(out))
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				require.NoError(t, svc.persistRecording(context.Background(), fmt.Sprintf("chapter-%d", index), data, "fake", session.ID))
			}
			result, err := svc.FinalizeReviewRecording(context.Background(), "fake", "operator", session.LeaseToken, []string{"chapter-0", "chapter-1"})
			require.NoError(t, err)
			require.NotNil(t, result.Reference.AudioTrackPresent)
			require.True(t, *result.Reference.AudioTrackPresent)
			coverage := "some_chapters"
			if secondHasAudio {
				coverage = "all_chapters"
			}
			require.Equal(t, coverage, result.Reference.AudioTrackCoverage)
			pcm, err := exec.Command("ffmpeg", "-v", "error", "-i", result.Path, "-map", "0:a:0", "-f", "s16le", "-ac", "1", "-ar", "16000", "-").Output()
			require.NoError(t, err)
			energy := func(start, end int) int64 {
				var sum int64
				for i := start * 2; i+1 < end*2 && i+1 < len(pcm); i += 2 {
					sample := int64(int16(binary.LittleEndian.Uint16(pcm[i : i+2])))
					sum += sample * sample
				}
				return sum
			}
			require.Greater(t, energy(3200, 12800), int64(1000000), "first chapter retains sound")
			if secondHasAudio {
				require.Greater(t, energy(19200, 28800), int64(1000000), "second chapter retains sound")
			} else {
				require.Less(t, energy(19200, 28800), int64(1000000), "missing audio is represented as silence")
			}
		})
	}
}

func TestReviewRecordingPreservesDelayedAudioStart(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	svc, _ := testService(t)
	svc.SetEvidenceDir(t.TempDir())
	session, err := svc.Acquire("fake", "operator", 5*time.Minute)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "delayed.mp4")
	out, err := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=10:d=1", "-itsoffset", "0.4", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-t", "1", path).CombinedOutput()
	require.NoError(t, err, string(out))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, svc.persistRecording(context.Background(), "delayed", data, "fake", session.ID))
	result, err := svc.FinalizeReviewRecording(context.Background(), "fake", "operator", session.LeaseToken, []string{"delayed"})
	require.NoError(t, err)
	pcm, err := exec.Command("ffmpeg", "-v", "error", "-i", result.Path, "-map", "0:a:0", "-f", "s16le", "-ac", "1", "-ar", "16000", "-").Output()
	require.NoError(t, err)
	energy := func(start, end int) int64 {
		var sum int64
		for i := start * 2; i+1 < end*2 && i+1 < len(pcm); i += 2 {
			sample := int64(int16(binary.LittleEndian.Uint16(pcm[i : i+2])))
			sum += sample * sample
		}
		return sum
	}
	require.Less(t, energy(1600, 4800), int64(1000000), "audio must not start before its original timestamp")
	require.Greater(t, energy(9600, 12800), int64(1000000), "audio should be present after its delayed start")
}

func TestFinalizeReviewRecordingRejectsEmptyChapterSet(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", 5*time.Minute)
	require.NoError(t, err)
	_, err = svc.FinalizeReviewRecording(context.Background(), "fake", "operator", session.LeaseToken, nil)
	require.ErrorContains(t, err, "at least one chapter")
}

func TestFinalizeReviewRecordingRejectsUnverifiedProvenance(t *testing.T) {
	for _, tc := range []struct {
		name        string
		ownerDevice string
		otherLease  bool
		legacy      bool
		tamper      bool
	}{
		{name: "other device", ownerDevice: "other"},
		{name: "other lease", ownerDevice: "fake", otherLease: true},
		{name: "legacy recording", legacy: true},
		{name: "modified recording", ownerDevice: "fake", tamper: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := testService(t)
			svc.SetEvidenceDir(t.TempDir())
			session, err := svc.Acquire("fake", "operator", time.Minute)
			require.NoError(t, err)
			if tc.legacy {
				require.NoError(t, svc.persistArtifact(context.Background(), "chapter", []byte("video"), "video"))
			} else {
				leaseID := session.ID
				if tc.otherLease {
					leaseID = "other-lease"
				}
				require.NoError(t, svc.persistRecording(context.Background(), "chapter", []byte("video"), tc.ownerDevice, leaseID))
				if tc.tamper {
					require.NoError(t, os.WriteFile(svc.artifacts["chapter"], []byte("changed"), 0o600))
				}
			}
			_, err = svc.FinalizeReviewRecording(context.Background(), "fake", "operator", session.LeaseToken, []string{"chapter"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "recording \"chapter\"")
		})
	}
}

func TestStopRecordingAllowsSuccessorLeaseToCleanUp(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	recorder := &failingExternalRecorder{}
	svc.externalRecordings["handle"] = externalRecording{DeviceID: "fake", LeaseID: "previous-lease", Recorder: recorder, Handle: strategy.RecordingHandle{ID: "handle"}}
	_, err = svc.StopExternalRecording(context.Background(), "fake", "operator", session.LeaseToken, "handle")
	require.ErrorContains(t, err, "destination stop outcome unknown")
	require.EqualValues(t, 1, recorder.stops.Load())
	require.Contains(t, svc.externalRecordings, "handle")
}

func TestStopRecordingWrongDevicePreservesOwnerHandle(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	svc.externalRecordings["other-handle"] = externalRecording{DeviceID: "other"}
	_, err = svc.StopExternalRecording(context.Background(), "fake", "operator", session.LeaseToken, "other-handle")
	require.ErrorContains(t, err, "not active for device")
	require.Contains(t, svc.externalRecordings, "other-handle", "a mismatched caller must not erase another device's recording")
}

type failingExternalRecorder struct{ stops atomic.Int32 }

func (*failingExternalRecorder) StartRecording(context.Context, strategy.ClaimClass) (strategy.RecordingHandle, error) {
	return strategy.RecordingHandle{}, nil
}
func (r *failingExternalRecorder) StopRecording(context.Context, strategy.RecordingHandle) (strategy.RecordingArtifact, error) {
	r.stops.Add(1)
	return strategy.RecordingArtifact{}, errors.New("destination stop outcome unknown")
}

func TestStopRecordingFailurePreservesUncertainOutcomeWithoutReplay(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	recorder := &failingExternalRecorder{}
	svc.externalRecordings["handle"] = externalRecording{DeviceID: "fake", LeaseID: session.ID, Recorder: recorder, Handle: strategy.RecordingHandle{ID: "handle"}}
	_, err = svc.StopExternalRecording(context.Background(), "fake", "operator", session.LeaseToken, "handle")
	require.ErrorContains(t, err, "destination stop outcome unknown")
	require.Contains(t, svc.externalRecordings, "handle")
	_, err = svc.StopExternalRecording(context.Background(), "fake", "operator", session.LeaseToken, "handle")
	require.ErrorContains(t, err, "outcome uncertain")
	require.EqualValues(t, 1, recorder.stops.Load(), "the owner must not replay a possibly consumed stop")
}

type partialExternalRecorder struct {
	data  []byte
	stops atomic.Int32
}

func (*partialExternalRecorder) StartRecording(context.Context, strategy.ClaimClass) (strategy.RecordingHandle, error) {
	return strategy.RecordingHandle{}, nil
}
func (r *partialExternalRecorder) StopRecording(context.Context, strategy.RecordingHandle) (strategy.RecordingArtifact, error) {
	r.stops.Add(1)
	return strategy.RecordingArtifact{Bytes: r.data, Method: "native", ClaimClass: strategy.ClaimTransition}, errors.New("restore failed")
}

func TestStopRecordingPreservesRedactedArtifactWhenRestorationFails(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	svc, _ := testService(t)
	svc.SetEvidenceDir(t.TempDir())
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "partial.mp4")
	require.NoError(t, exec.Command("ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=5:d=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", path).Run())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	recorder := &partialExternalRecorder{data: data}
	svc.externalRecordings["handle"] = externalRecording{DeviceID: "fake", LeaseID: session.ID, Actor: "operator", Recorder: recorder, Handle: strategy.RecordingHandle{ID: "handle", StartedAt: time.Now()}}
	recovered, err := svc.StopExternalRecording(context.Background(), "fake", "operator", session.LeaseToken, "handle")
	require.ErrorContains(t, err, "restore failed")
	require.NotEmpty(t, recovered.Reference.ID)
	require.False(t, recovered.Reference.RedactionVerified)
	require.Contains(t, recovered.Reference.AppliedRules, "redaction_not_requested")
	bytes, kind, err := svc.Artifact(recovered.Reference.ID)
	require.NoError(t, err)
	require.NotEmpty(t, bytes)
	require.Equal(t, "video", kind)
	require.True(t, svc.externalRecordings["handle"].Uncertain)
	repeated, err := svc.StopExternalRecording(context.Background(), "fake", "operator", session.LeaseToken, "handle")
	require.ErrorContains(t, err, "outcome uncertain")
	require.Equal(t, recovered.Reference.ID, repeated.Reference.ID)
	require.EqualValues(t, 1, recorder.stops.Load())
}

func TestSuccessorLeaseRetainsOriginalRecordingProvenance(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	svc, _ := testService(t)
	svc.SetEvidenceDir(t.TempDir())
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "partial.mp4")
	require.NoError(t, exec.Command("ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=5:d=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", path).Run())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	recorder := &partialExternalRecorder{data: data}
	svc.externalRecordings["handle"] = externalRecording{DeviceID: "fake", LeaseID: "original-lease", Recorder: recorder, Handle: strategy.RecordingHandle{ID: "handle", StartedAt: time.Now()}}
	recovered, err := svc.StopExternalRecording(context.Background(), "fake", "operator", session.LeaseToken, "handle")
	require.ErrorContains(t, err, "restore failed")
	retained, _, err := svc.Artifact(recovered.Reference.ID)
	require.NoError(t, err)
	require.NoError(t, svc.verifyRecordingProvenance(context.Background(), recovered.Reference.ID, retained, "fake", "original-lease"))
	require.Error(t, svc.verifyRecordingProvenance(context.Background(), recovered.Reference.ID, retained, "fake", session.ID))
}

func TestStopRecordingPreservesRestorationErrorWhenArtifactProcessingFails(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	recorder := &partialExternalRecorder{data: []byte("invalid video")}
	svc.externalRecordings["handle"] = externalRecording{DeviceID: "fake", LeaseID: session.ID, Recorder: recorder, Handle: strategy.RecordingHandle{ID: "handle"}}
	result, err := svc.StopExternalRecording(context.Background(), "fake", "operator", session.LeaseToken, "handle")
	require.ErrorContains(t, err, "restore failed")
	require.ErrorContains(t, err, "external recording")
	require.Empty(t, result.Reference.ID)
	require.True(t, svc.externalRecordings["handle"].Uncertain)
	require.EqualValues(t, 1, recorder.stops.Load())
}
