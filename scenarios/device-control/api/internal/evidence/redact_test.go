package evidence

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	"testing"
)

func TestBinaryImageRequiresPixelMasking(t *testing.T) {
	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	result := RedactFrame(raw.Bytes(), DefaultPolicy)
	require.False(t, result.Verified)
	masked := RedactImage(raw.Bytes(), "image/png", []Region{{0, 0, 2, 2}}, DefaultPolicy)
	require.True(t, masked.Verified)
	require.NotEqual(t, raw.Bytes(), masked.Bytes)
}

func TestTextRedactionCountsAppliedRegions(t *testing.T) {
	result := RedactFrame([]byte("password=secret otp 123456"), DefaultPolicy)
	require.True(t, result.Verified)
	require.Equal(t, 2, result.Regions)
}
