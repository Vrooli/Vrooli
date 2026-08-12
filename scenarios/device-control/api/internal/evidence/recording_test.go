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

func TestNativeMetadataEnforcesUsefulFPS(t *testing.T) { // [REQ:DVC-P0-012]
	_, err := NativeMetadata(1)
	require.Error(t, err)
	meta, err := NativeMetadata(5)
	require.NoError(t, err)
	require.Equal(t, "native", meta.Method)
}
