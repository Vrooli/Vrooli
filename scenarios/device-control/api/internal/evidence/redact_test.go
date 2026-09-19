package evidence

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
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

func TestCaptureRedactionDoesNotBlankPortraitContent(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 120, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 120; x++ {
			frame.Set(x, y, color.RGBA{R: 220, G: 38, B: 38, A: 255})
		}
	}
	var raw bytes.Buffer
	require.NoError(t, png.Encode(&raw, frame))

	result, err := RedactCapture(raw.Bytes(), "image/png", DefaultPolicy, false, "operator")
	require.NoError(t, err)
	decoded, _, err := image.Decode(bytes.NewReader(result.Bytes))
	require.NoError(t, err)

	// The status bar is protected, but the content immediately below it must
	// remain visible. A quarter-screen mask regresses this contract.
	top := decoded.At(60, 0)
	content := decoded.At(60, 30)
	require.Equal(t, color.RGBA{0, 0, 0, 255}, color.RGBAModel.Convert(top))
	require.Equal(t, color.RGBA{220, 38, 38, 255}, color.RGBAModel.Convert(content))
}

func TestVideoRedactionDoesNotBlankPortraitContent(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for native-video redaction coverage")
	}
	tmp := t.TempDir()
	inputPath := filepath.Join(tmp, "input.mp4")
	outputPath := filepath.Join(tmp, "redacted.mp4")
	framePath := filepath.Join(tmp, "frame.png")

	cmd := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=red:s=120x120:d=0.25", "-c:v", "libx264", "-pix_fmt", "yuv420p", inputPath)
	require.NoError(t, cmd.Run())
	raw, err := os.ReadFile(inputPath)
	require.NoError(t, err)

	result, err := RedactCapture(raw, "video/mp4", DefaultPolicy, false, "operator")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(outputPath, result.Bytes, 0o600))
	require.NoError(t, exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", outputPath, "-frames:v", "1", framePath).Run())
	frame, err := os.ReadFile(framePath)
	require.NoError(t, err)
	decoded, _, err := image.Decode(bytes.NewReader(frame))
	require.NoError(t, err)

	top := decoded.At(60, 0)
	content := decoded.At(60, 30)
	require.Equal(t, color.RGBA{0, 0, 0, 255}, color.RGBAModel.Convert(top))
	contentColor := color.RGBAModel.Convert(content).(color.RGBA)
	require.GreaterOrEqual(t, int(contentColor.R), 240)
	require.LessOrEqual(t, int(contentColor.G), 8)
	require.LessOrEqual(t, int(contentColor.B), 8)
}

func TestTextRedactionCountsAppliedRegions(t *testing.T) {
	result := RedactFrame([]byte("password=secret otp 123456"), DefaultPolicy)
	require.True(t, result.Verified)
	require.Equal(t, 2, result.Regions)
}
