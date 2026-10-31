package control

import (
	"context"
	"errors"
	"sync/atomic"

	"device-control/strategy"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFinalizeReviewRecordingRetainsProducerOwnedReviewCopy(t *testing.T) {
	t.Setenv("DEVICE_CONTROL_EVIDENCE_DIR", t.TempDir())
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for the recording producer test")
	}
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", 5*time.Minute)
	require.NoError(t, err)
	chapterPath := filepath.Join(t.TempDir(), "chapter.mp4")
	require.NoError(t, exec.Command("ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=5:d=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", chapterPath).Run())
	chapter, err := os.ReadFile(chapterPath)
	require.NoError(t, err)
	require.NoError(t, svc.persistArtifact(context.Background(), "chapter-1", chapter, "video"))
	require.NoError(t, svc.persistArtifact(context.Background(), "chapter-2", chapter, "video"))

	result, err := svc.FinalizeReviewRecording(context.Background(), "fake", "operator", session.LeaseToken, []string{"chapter-1", "chapter-2"})
	require.NoError(t, err)
	require.NotEmpty(t, result.Reference.ID)
	require.False(t, result.Reference.RedactionVerified)
	require.Contains(t, result.Reference.AppliedRules, "redaction_not_requested")
	require.True(t, result.Reference.ContentVerified)
	require.Equal(t, "chapter-concat", result.Reference.RecordingMethod)
	require.True(t, filepath.IsAbs(result.Path))
	_, err = os.Stat(result.Path)
	require.NoError(t, err)
	retained, kind, err := svc.Artifact(result.Reference.ID)
	require.NoError(t, err)
	require.Equal(t, "video", kind)
	require.NotEmpty(t, retained)
}

func TestFinalizeReviewRecordingRejectsEmptyChapterSet(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", 5*time.Minute)
	require.NoError(t, err)
	_, err = svc.FinalizeReviewRecording(context.Background(), "fake", "operator", session.LeaseToken, nil)
	require.ErrorContains(t, err, "at least one chapter")
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
	svc.externalRecordings["handle"] = externalRecording{DeviceID: "fake", Recorder: recorder, Handle: strategy.RecordingHandle{ID: "handle"}}
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
	t.Setenv("DEVICE_CONTROL_EVIDENCE_DIR", t.TempDir())
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "partial.mp4")
	require.NoError(t, exec.Command("ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:r=5:d=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", path).Run())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	recorder := &partialExternalRecorder{data: data}
	svc.externalRecordings["handle"] = externalRecording{DeviceID: "fake", Actor: "operator", Recorder: recorder, Handle: strategy.RecordingHandle{ID: "handle", StartedAt: time.Now()}}
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

func TestStopRecordingPreservesRestorationErrorWhenArtifactProcessingFails(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	recorder := &partialExternalRecorder{data: []byte("invalid video")}
	svc.externalRecordings["handle"] = externalRecording{DeviceID: "fake", Recorder: recorder, Handle: strategy.RecordingHandle{ID: "handle"}}
	result, err := svc.StopExternalRecording(context.Background(), "fake", "operator", session.LeaseToken, "handle")
	require.ErrorContains(t, err, "restore failed")
	require.ErrorContains(t, err, "external recording")
	require.Empty(t, result.Reference.ID)
	require.True(t, svc.externalRecordings["handle"].Uncertain)
	require.EqualValues(t, 1, recorder.stops.Load())
}
