package evidence

import (
	"bytes"
	"image"
	"image/color"
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

func TestCaptureNamesAppliedImageRules(t *testing.T) {
	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 40, 40))))
	result, err := RedactCaptureWithRegions(raw.Bytes(), "image/png", DefaultPolicy, false, "operator", []Region{{10, 10, 20, 20}})
	require.NoError(t, err)
	require.True(t, result.Verified)
	require.NotContains(t, result.Rules, "status_bar_identifiers")
	require.NotContains(t, result.Rules, "notification_content")
	require.Contains(t, result.Rules, "flow_sensitive_regions")
	require.Contains(t, result.Rules, "content_classification_not_performed")
}

func TestCaptureDefaultsToUnredactedWhenNotRequested(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 120, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 120; x++ {
			frame.Set(x, y, color.RGBA{R: 220, G: 38, B: 38, A: 255})
		}
	}
	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, frame))

	result, err := CaptureWithPolicy(raw.Bytes(), "image/png", DefaultPolicy, false, false, "operator", nil)
	require.NoError(t, err)
	require.False(t, result.Verified)
	require.Equal(t, raw.Bytes(), result.Bytes)
	require.Contains(t, result.Rules, "redaction_not_requested")
	require.Contains(t, result.Rules, "content_classification_not_performed")
}

func TestExplicitCaptureRedactionAppliesConfiguredRegionsOnly(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 120, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 120; x++ {
			frame.Set(x, y, color.RGBA{R: 220, G: 38, B: 38, A: 255})
		}
	}
	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, frame))

	result, err := CaptureWithPolicy(raw.Bytes(), "image/png", DefaultPolicy, true, false, "operator", []Region{{X0: 40, Y0: 40, X1: 60, Y1: 60}})
	require.NoError(t, err)
	require.True(t, result.Verified)
	decoded, _, err := image.Decode(bytes.NewReader(result.Bytes))
	require.NoError(t, err)
	require.Equal(t, color.RGBA{0, 0, 0, 255}, color.RGBAModel.Convert(decoded.At(50, 50)))
	require.Equal(t, color.RGBA{220, 38, 38, 255}, color.RGBAModel.Convert(decoded.At(5, 5)))
	require.Equal(t, color.RGBA{220, 38, 38, 255}, color.RGBAModel.Convert(decoded.At(80, 80)))
	require.Contains(t, result.Rules, "flow_sensitive_regions")
	require.Contains(t, result.Rules, "content_classification_not_performed")
}

func TestVisualRedactionDoesNotPretendToDetectSensitiveContent(t *testing.T) {
	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 40, 40))))
	_, err := RedactCapture(raw.Bytes(), "image/png", DefaultPolicy, false, "operator")
	require.ErrorContains(t, err, "explicit sensitive regions")
	_, err = RedactCapture([]byte("1234ftyp"), "video/mp4", DefaultPolicy, false, "operator")
	require.ErrorContains(t, err, "video redaction is unavailable")
	invalid := RedactImage(raw.Bytes(), "image/png", []Region{{100, 100, 110, 110}}, DefaultPolicy)
	require.False(t, invalid.Verified)
}

func TestTextRedactionCountsAppliedRegions(t *testing.T) {
	result, err := RedactCapture([]byte("password=secret otp 123456"), "text/plain", DefaultPolicy, false, "operator")
	require.NoError(t, err)
	require.True(t, result.Verified)
	require.Equal(t, 2, result.Regions)
	require.Contains(t, result.Rules, "content_classification_not_performed")
	require.Contains(t, result.Rules, "credential_patterns")
	require.Contains(t, result.Rules, "one_time_codes")
}
