package evidence

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

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
