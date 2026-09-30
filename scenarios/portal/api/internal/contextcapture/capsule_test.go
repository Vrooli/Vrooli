package contextcapture

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
)

func captureFixture(t *testing.T) (Import, time.Time) {
	t.Helper()
	now := time.Now().UTC()
	raster := image.NewRGBA(image.Rect(0, 0, 4, 2))
	raster.SetRGBA(1, 1, color.RGBA{R: 123, A: 255})
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, raster))
	return Import{Source: Source{Surface: targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}, CaptureID: uuid.NewString(), DisplayID: "d", GeometryRevision: "g", CapturedAt: now.Add(-time.Second), Bounds: Bounds{X: -1920, Y: -100, Width: 4, Height: 2}}, Region: Region{X: 1, Y: 0, Width: 2, Height: 2}, Strokes: [][]Point{{{X: 1, Y: 1}, {X: 2.5, Y: 1.5}}}, PNG: encoded.Bytes()}, now
}

func TestPreparePreservesOriginalAndSourceCoordinates(t *testing.T) {
	input, now := captureFixture(t)
	original := append([]byte(nil), input.PNG...)
	input.PNG = append(input.PNG, []byte("untrusted trailing payload")...)
	doc, pixels, err := Prepare("operator", input, now, time.Hour)
	require.NoError(t, err)
	require.Equal(t, input.Source, doc.Source)
	require.Equal(t, input.Region, doc.Region)
	require.Equal(t, now.Add(time.Hour), doc.ExpiresAt)
	require.NotEmpty(t, doc.OriginalSHA256)
	require.NotContains(t, string(pixels), "untrusted trailing payload")
	decoded, err := png.Decode(bytes.NewReader(pixels))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 4, 2), decoded.Bounds(), "region must not replace the original image")
	r, _, _, a := decoded.At(1, 1).RGBA()
	require.EqualValues(t, 123*257, r)
	require.EqualValues(t, 65535, a)
	input.Strokes[0][0].X = 4
	input.PNG[0] = 0
	require.EqualValues(t, 1, doc.Strokes[0][0].X)
	_, err = png.Decode(bytes.NewReader(pixels))
	require.NoError(t, err)
	clean, _ := captureFixture(t)
	clean.PNG = original
	second, _, err := Prepare("operator", clean, now, time.Hour)
	require.NoError(t, err)
	require.Equal(t, doc.OriginalSHA256, second.OriginalSHA256)
	require.NotEqual(t, doc.ID, second.ID)
}

func TestPrepareRefusesMalformedOrUnboundedContext(t *testing.T) {
	for _, fault := range []string{"owner", "surface", "capture-id", "future", "stale", "bounds", "region-overflow", "empty-region", "nan", "outside", "too-many-strokes", "too-many-points", "wrong-png-size", "truncated-png", "retention"} {
		t.Run(fault, func(t *testing.T) {
			input, now := captureFixture(t)
			owner := "operator"
			retention := time.Hour
			switch fault {
			case "owner":
				owner = ""
			case "surface":
				input.Source.Surface.SurfaceID = ""
			case "capture-id":
				input.Source.CaptureID = uuid.Nil.String()
			case "future":
				input.Source.CapturedAt = now.Add(time.Second)
			case "stale":
				input.Source.CapturedAt = now.Add(-31 * time.Second)
			case "bounds":
				input.Source.Bounds.Width = 65535
				input.Source.Bounds.Height = 65535
			case "region-overflow":
				input.Region.X = math.MaxUint32
			case "empty-region":
				input.Region.Width = 0
			case "nan":
				input.Strokes[0][0].X = math.NaN()
			case "outside":
				input.Strokes[0][0].Y = 3
			case "too-many-strokes":
				input.Strokes = make([][]Point, 33)
			case "too-many-points":
				input.Strokes[0] = make([]Point, 257)
			case "wrong-png-size":
				input.Source.Bounds.Width = 5
			case "truncated-png":
				input.PNG = input.PNG[:len(input.PNG)/2]
			case "retention":
				retention = 25 * time.Hour
			}
			doc, pixels, err := Prepare(owner, input, now, retention)
			require.ErrorIs(t, err, ErrInvalid)
			require.Empty(t, doc.ID)
			require.Nil(t, pixels)
		})
	}
}
