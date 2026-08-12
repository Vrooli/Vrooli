package compose

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"backdrop-studio/internal/catalog"
	"github.com/stretchr/testify/require"
)

func baseStyle() catalog.Style {
	return catalog.Style{ID: "style", Strategy: "procedural", Treatments: []string{"duotone", "halftone"}, Placements: []string{"full_bleed"}}
}

func TestResolveIsInspectableAndDoesNotExecute(t *testing.T) {
	plan, err := Resolve(baseStyle(), Brief{Placement: "full_bleed"}, nil, "", true)
	require.NoError(t, err)
	require.Equal(t, "procedural", plan.ExpectedExecutionPath)
	require.Equal(t, []Operation{{Name: "duotone"}, {Name: "halftone"}}, plan.Operations)
}

func TestResolveRefusesPaletteAndLicenceBeforeExecution(t *testing.T) {
	style := baseStyle()
	style.Treatments = []string{"$brand.primary"}
	_, err := Resolve(style, Brief{}, MapPalette{}, "", true)
	require.ErrorContains(t, err, "$brand.primary")
	_, err = Resolve(baseStyle(), Brief{}, nil, "restricted-adapter", false)
	require.ErrorContains(t, err, "restricted-adapter")
}

func TestComposeDeviceFrameSupportsEveryArrangementAndReservesFootprint(t *testing.T) {
	backdrop := pngBytes(t, image.NewRGBA(image.Rect(0, 0, 100, 80)))
	screenshot := pngBytes(t, image.NewRGBA(image.Rect(0, 0, 20, 30)))
	for _, arrangement := range []DeviceArrangement{DeviceCenter, CaptionAbove, CaptionBelow, CaptionOnly} {
		out, region, err := ComposeDeviceFrame(backdrop, screenshot, arrangement, "Scenario headline")
		require.NoError(t, err)
		decoded, err := png.Decode(bytes.NewReader(out))
		require.NoError(t, err)
		require.Equal(t, image.Pt(100, 80), decoded.Bounds().Size())
		require.Greater(t, region.Width, 0.0)
		require.Greater(t, region.Height, 0.0)
		require.Contains(t, []string{"occlusion", "overlay"}, region.Kind)
		if dir := os.Getenv("EVIDENCE_DIR"); dir != "" && arrangement == DeviceCenter {
			require.NoError(t, os.MkdirAll(dir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "store-device-center.png"), out, 0o644))
		}
	}
}

func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}
