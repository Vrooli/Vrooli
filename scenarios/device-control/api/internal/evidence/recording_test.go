package evidence

import (
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInspectAudioTrackDistinguishesSilentVideoAndAudioStream(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe unavailable")
	}
	for _, withAudio := range []bool{false, true} {
		name := "video-only"
		if withAudio {
			name = "video-with-audio"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.mp4")
			args := []string{"-v", "error", "-f", "lavfi", "-i", "color=c=red:s=16x16:r=10:d=1"}
			if withAudio {
				args = append(args, "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-c:a", "aac")
			}
			args = append(args, "-c:v", "libx264", "-t", "1", "-y", path)
			output, err := exec.Command("ffmpeg", args...).CombinedOutput()
			require.NoError(t, err, string(output))
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			present, err := InspectAudioTrack(raw)
			require.NoError(t, err)
			require.Equal(t, withAudio, present)
		})
	}
	_, err := InspectAudioTrack([]byte("invalid container"))
	require.Error(t, err)
}

func TestEncodeFramesProducesSynthesizedEvidence(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 4, 4))
	video, err := EncodeFrames([]image.Image{frame, frame}, 5)
	require.NoError(t, err)
	require.Equal(t, "synthesized", video.RecordingMethod)
	require.Equal(t, 2, video.FrameCount)
	require.NotEmpty(t, video.Bytes)
}

func TestClaimClassAssessmentReportsDegradedRates(t *testing.T) { // [REQ:DVC-P0-012]
	tests := []struct {
		name  string
		class ClaimClass
		fps   float64
		want  Disposition
	}{
		{"static pass", ClaimStatic, 1, DispositionPassed},
		{"static degraded", ClaimStatic, .5, DispositionDegraded},
		{"transition pass", ClaimTransition, 5, DispositionPassed},
		{"transition degraded", ClaimTransition, 4.99, DispositionDegraded},
		{"animation pass", ClaimAnimation, 15, DispositionPassed},
		{"animation degraded", ClaimAnimation, 14.99, DispositionDegraded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := NativeMetadataForClaim(tt.fps, tt.class)
			require.NoError(t, err)
			require.Equal(t, "native", meta.Method)
			require.Equal(t, tt.want, meta.Assessment.Disposition)
		})
	}
}

func TestAssessVideoContentRejectsUniformBlackBody(t *testing.T) {
	assessment := AssessVideoContent(VideoContentAssessment{SampledFrames: 12, AverageLuma: 16, MaximumLuma: 16})
	require.False(t, assessment.Verified)
	require.Contains(t, assessment.Reason, "uniformly near black")
}

func TestAssessVideoContentAcceptsVisibleBody(t *testing.T) {
	assessment := AssessVideoContent(VideoContentAssessment{SampledFrames: 12, AverageLuma: 42, MaximumLuma: 180})
	require.True(t, assessment.Verified)
	require.Empty(t, assessment.Reason)
}

func TestDarkRecordingIsRetainedAsUnverifiedEvidence(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	path := filepath.Join(t.TempDir(), "dark.mp4")
	output, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=black:s=64x64:r=10:d=1", "-c:v", "libx264", "-y", path).CombinedOutput()
	require.NoError(t, err, string(output))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	capture, err := CaptureWithPolicy(raw, "video/mp4", DefaultPolicy, false, false, "operator", nil)
	require.NoError(t, err)
	ref, err := NewClaimedVideoReference("dark", raw, capture, "native", 10, ClaimTransition)
	require.NoError(t, err, "dark footage must remain available when it might reflect the actual scene")
	require.False(t, ref.ContentVerified)
	require.Equal(t, DispositionDegraded, ref.Disposition)
	require.Contains(t, ref.DispositionReason, "near black")
}
