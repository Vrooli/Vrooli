package shots

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"device-studio/internal/projects"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

type physicalPresentationMediaStub struct {
	projects.SourceMediaService
	media projects.SourceMedia
	data  []byte
}

func (s physicalPresentationMediaStub) Get(_ context.Context, projectID, mediaID string) (projects.SourceMedia, error) {
	if projectID != s.media.ProjectID || mediaID != s.media.ID {
		return projects.SourceMedia{}, fmt.Errorf("source not found")
	}
	return s.media, nil
}

func (s physicalPresentationMediaStub) Open(_ context.Context, projectID, mediaID string) (projects.SourceMedia, io.ReadCloser, error) {
	media, err := s.Get(context.Background(), projectID, mediaID)
	if err != nil {
		return projects.SourceMedia{}, nil, err
	}
	return media, io.NopCloser(bytes.NewReader(s.data)), nil
}

func encodeTestRaster(t *testing.T, raster *image.NRGBA) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, raster); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestPhysicalPlainAlreadyComposedPreservesExactRasterPixels(t *testing.T) {
	if version, err := resolvePresentationVersion(&shotsv1.ScreenPresentation{CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN}, "", true); err != nil || version != presentationSemanticVersion {
		t.Fatalf("legacy identity path version = %q, %v", version, err)
	}
	input := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	copy(input.Pix, []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 20, 30, 40, 255, 50, 60, 70, 255, 80, 90, 100, 255})
	data := encodeTestRaster(t, input)
	sum := sha256.Sum256(data)
	media := projects.SourceMedia{ID: "screen", ProjectID: "project", SHA256: fmt.Sprintf("%x", sum), SizeBytes: int64(len(data)), MediaType: "image/png", Kind: projects.MediaKindImage, Readiness: projects.ReadinessReady, Width: 3, Height: 2, RasterKnown: true}
	resolver := &ownedPhysicalResolver{media: physicalPresentationMediaStub{media: media, data: data}}
	saved := &shotsv1.ScreenPresentation{SourceMediaId: media.ID, CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN}
	encoded, _, _, _, err := resolver.composePhysicalScreen(context.Background(), media.ProjectID, media, saved, "device-studio-presentation-v1", 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := png.Decode(bytes.NewReader(encoded))
	if err != nil || got.Bounds().Dx() != 3 || got.Bounds().Dy() != 2 {
		t.Fatalf("composed raster dimensions = %v, %v", got.Bounds(), err)
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			if color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA) != input.NRGBAAt(x, y) {
				t.Fatalf("already-composed pixel (%d,%d) changed: got=%v want=%v", x, y, got.At(x, y), input.At(x, y))
			}
		}
	}
}

func TestPhysicalPlainAppOnlyFitsAndInvalidInputIsTyped(t *testing.T) {
	if version, err := resolvePresentationVersion(&shotsv1.ScreenPresentation{CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN}, "", true); err != nil || version != presentationSemanticV2 {
		t.Fatalf("new plain app-only path version = %q, %v", version, err)
	}
	input := image.NewNRGBA(image.Rect(0, 0, 8, 4))
	for i := 0; i < len(input.Pix); i += 4 {
		input.Pix[i], input.Pix[i+1], input.Pix[i+2], input.Pix[i+3] = 240, 30, 20, 255
	}
	input.SetNRGBA(7, 3, color.NRGBA{})
	data := encodeTestRaster(t, input)
	sum := sha256.Sum256(data)
	media := projects.SourceMedia{ID: "app", ProjectID: "project", SHA256: fmt.Sprintf("%x", sum), SizeBytes: int64(len(data)), MediaType: "image/png", Kind: projects.MediaKindImage, Readiness: projects.ReadinessReady, Width: 8, Height: 4, RasterKnown: true}
	resolver := &ownedPhysicalResolver{media: physicalPresentationMediaStub{media: media, data: data}}
	saved := &shotsv1.ScreenPresentation{SourceMediaId: media.ID, CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN, FitMode: shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN, Matte: &shotsv1.PresentationRGB{Red: 4, Green: 9, Blue: 14}}
	if err := validatePhysicalPresentation(saved, &shotsv1.PhysicalShotBinding{ScreenWidthPixels: 4, ScreenHeightPixels: 4}, presentationSemanticV2); err != nil {
		t.Fatal(err)
	}
	encoded, _, _, _, err := resolver.composePhysicalScreen(context.Background(), media.ProjectID, media, saved, presentationSemanticV2, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	got, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA) != (color.NRGBA{R: 4, G: 9, B: 14, A: 255}) || color.NRGBAModel.Convert(got.At(0, 1)).(color.NRGBA) != (color.NRGBA{R: 240, G: 30, B: 20, A: 255}) {
		t.Fatalf("plain app fit did not matte letterbox uniformly: top=%v content=%v", got.At(0, 0), got.At(0, 1))
	}
	saved.Matte = nil
	_, _, _, _, err = resolver.composePhysicalScreen(context.Background(), media.ProjectID, media, saved, presentationSemanticV2, 4, 4)
	var invalid ErrInvalidPhysicalBinding
	if !errors.As(err, &invalid) || invalid.Error() == "" {
		t.Fatalf("invalid presentation should remain an actionable typed physical error: %v", err)
	}
}
