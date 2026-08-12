package evidence

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinaryImageRequiresPixelMasking(t *testing.T) { // [REQ:DVC-P0-008]
	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	result := RedactFrame(raw.Bytes(), DefaultPolicy)
	require.False(t, result.Verified)
	masked := RedactImage(raw.Bytes(), "image/png", []Region{{0, 0, 2, 2}}, DefaultPolicy)
	require.True(t, masked.Verified)
	require.NotEqual(t, raw.Bytes(), masked.Bytes)
}

func TestCaptureNamesNotificationAndFlowSensitiveRules(t *testing.T) {
	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 40, 40))))
	result, err := RedactCaptureWithRegions(raw.Bytes(), "image/png", DefaultPolicy, false, "operator", []Region{{10, 10, 20, 20}})
	require.NoError(t, err)
	require.True(t, result.Verified)
	require.Contains(t, result.Rules, "notification_content")
	require.Contains(t, result.Rules, "flow_sensitive_regions")
}

func TestTextRedactionCountsAppliedRegions(t *testing.T) {
	result := RedactFrame([]byte("password=secret otp 123456"), DefaultPolicy)
	require.True(t, result.Verified)
	require.Equal(t, 2, result.Regions)
}
