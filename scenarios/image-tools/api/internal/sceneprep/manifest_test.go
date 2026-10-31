package sceneprep

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	preparationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func raster(w, h int) []byte {
	image := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			image.Set(x, y, color.NRGBA{R: uint8(x * 23), G: uint8(y * 37), A: 255})
		}
	}
	var result bytes.Buffer
	if err := png.Encode(&result, image); err != nil {
		panic(err)
	}
	return result.Bytes()
}
func artifact(name, mime string, raw []byte) *preparationv1.Artifact {
	digest := sha256.Sum256(raw)
	return &preparationv1.Artifact{Path: name, Sha256: hex.EncodeToString(digest[:]), SizeBytes: int64(len(raw)), MediaType: mime}
}
func fixture() (*preparationv1.PreparedPhotograph, map[string][]byte) {
	// The outer boundary deliberately treats geometry as opaque owner output.
	// Its array semantics are independently verified by the geometry converter.
	files := map[string][]byte{"original.png": raster(3, 2), "oriented.png": raster(3, 2), "clean.png": raster(3, 2), "geometry.zip": []byte("opaque retained geometry-owner artifact")}
	m := &preparationv1.PreparedPhotograph{
		ContractVersion: ContractVersion, Route: preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_EMPTY_SCENE,
		Source:     &preparationv1.SourceRaster{Original: artifact("original.png", "image/png", files["original.png"]), Oriented: artifact("oriented.png", "image/png", files["oriented.png"]), OriginalWidth: 3, OriginalHeight: 2, Width: 3, Height: 2, OriginalExifOrientation: 1, OriginalToOriented: []float64{1, 0, 0, 0, 1, 0, 0, 0, 1}},
		Geometry:   &preparationv1.Geometry{Archive: artifact("geometry.zip", "application/zip", files["geometry.zip"]), Camera: &preparationv1.Camera{Basis: CameraBasis, WorldBasis: WorldBasis, RasterWidth: 3, RasterHeight: 2, NormalizedIntrinsics: []float64{.9, 0, .5, 0, 1.35, .5, 0, 0, 1}, CameraToWorld: []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}}, SupportPlane: &preparationv1.SupportPlane{Normal: []float64{0, -1, 0}, Offset: .4, ValidPixelCount: 6, InlierCount: 5, ResidualP50: .001, ResidualP95: .003}},
		Background: &preparationv1.Background{CleanPlate: artifact("clean.png", "image/png", files["clean.png"])}, ScaleStatus: "estimated_monocular_scale_not_measured", CameraEnvelope: &preparationv1.CameraEnvelope{}, Review: &preparationv1.VisualReview{Decision: "pending"},
	}
	return m, files
}
func bundleBytes(t *testing.T, m *preparationv1.PreparedPhotograph, files map[string][]byte, duplicate bool) []byte {
	t.Helper()
	var result bytes.Buffer
	writer := zip.NewWriter(&result)
	raw, err := protojson.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	all := map[string][]byte{"manifest.json": raw}
	for name, data := range files {
		all[name] = data
	}
	for name, data := range all {
		member, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = member.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if duplicate {
		member, err := writer.Create("manifest.json")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = member.Write(raw); err != nil {
			t.Fatal(err)
		}
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	return result.Bytes()
}
func TestOuterBundlePinsActualMembersAndRasters(t *testing.T) {
	m, files := fixture()
	raw := bundleBytes(t, m, files, false)
	opened, err := Open(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.Source.Width != 3 {
		t.Fatal("source raster lost")
	}
	for _, tc := range []struct {
		name      string
		change    func(*preparationv1.PreparedPhotograph, map[string][]byte)
		duplicate bool
	}{
		{"changed-bytes", func(_ *preparationv1.PreparedPhotograph, f map[string][]byte) { f["geometry.zip"][0] = 'X' }, false},
		{"undeclared-member", func(_ *preparationv1.PreparedPhotograph, f map[string][]byte) { f["extra"] = []byte("unretained") }, false},
		{"missing-member", func(_ *preparationv1.PreparedPhotograph, f map[string][]byte) { delete(f, "clean.png") }, false},
		{"raster-size-mismatch", func(m *preparationv1.PreparedPhotograph, f map[string][]byte) {
			f["clean.png"] = raster(2, 3)
			m.Background.CleanPlate = artifact("clean.png", "image/png", f["clean.png"])
		}, false},
		{"decoded-media-type", func(m *preparationv1.PreparedPhotograph, _ map[string][]byte) {
			m.Background.CleanPlate.MediaType = "image/jpeg"
		}, false},
		{"duplicate-manifest", func(_ *preparationv1.PreparedPhotograph, _ map[string][]byte) {}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, files := fixture()
			tc.change(m, files)
			raw := bundleBytes(t, m, files, tc.duplicate)
			if _, err := Open(bytes.NewReader(raw), int64(len(raw))); err == nil {
				t.Fatal("invalid bundle accepted")
			}
		})
	}
}
func TestSceneCoordinatesRejectAmbiguityAndInvalidGeometry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*preparationv1.PreparedPhotograph)
	}{
		{"unknown-version", func(m *preparationv1.PreparedPhotograph) { m.ContractVersion = "v2" }},
		{"missing-route", func(m *preparationv1.PreparedPhotograph) { m.Route = 0 }},
		{"missing-exif-map", func(m *preparationv1.PreparedPhotograph) { m.Source.OriginalToOriented = nil }},
		{"wrong-orientation", func(m *preparationv1.PreparedPhotograph) { m.Source.OriginalExifOrientation = 6 }},
		{"unsupported-camera-basis", func(m *preparationv1.PreparedPhotograph) { m.Geometry.Camera.Basis = "world" }},
		{"negative-focal", func(m *preparationv1.PreparedPhotograph) { m.Geometry.Camera.NormalizedIntrinsics[0] = -1 }},
		{"nan-focal", func(m *preparationv1.PreparedPhotograph) { m.Geometry.Camera.NormalizedIntrinsics[0] = math.NaN() }},
		{"reflection-not-rigid-rotation", func(m *preparationv1.PreparedPhotograph) { m.Geometry.Camera.CameraToWorld[0] = -1 }},
		{"scale-not-rigid", func(m *preparationv1.PreparedPhotograph) { m.Geometry.Camera.CameraToWorld[0] = 2 }},
		{"transposed-translation", func(m *preparationv1.PreparedPhotograph) { m.Geometry.Camera.CameraToWorld[3] = .3 }},
		{"invented-measured-scale", func(m *preparationv1.PreparedPhotograph) { m.ScaleStatus = "measured" }},
		{"nonunit-plane", func(m *preparationv1.PreparedPhotograph) { m.Geometry.SupportPlane.Normal[1] = -2 }},
		{"impossible-inliers", func(m *preparationv1.PreparedPhotograph) { m.Geometry.SupportPlane.InlierCount = 7 }},
		{"unordered-residuals", func(m *preparationv1.PreparedPhotograph) { m.Geometry.SupportPlane.ResidualP50 = .4 }},
		{"unsupported-movement", func(m *preparationv1.PreparedPhotograph) { m.CameraEnvelope.MaxRotationDegrees = 45 }},
		{"missing-removal-provenance", func(m *preparationv1.PreparedPhotograph) {
			m.Route = preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL
		}},
		{"traversal", func(m *preparationv1.PreparedPhotograph) { m.Geometry.Archive.Path = "../geometry.zip" }},
		{"windows-path", func(m *preparationv1.PreparedPhotograph) { m.Geometry.Archive.Path = "C:\\geometry.zip" }},
		{"hash-missing", func(m *preparationv1.PreparedPhotograph) { m.Geometry.Archive.Sha256 = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := fixture()
			tc.change(m)
			if _, err := Validate(m); err == nil {
				t.Fatal("ambiguous scene accepted")
			}
		})
	}
}
func TestAllEXIFTransformsMapPixelCentersWithinOrientedRaster(t *testing.T) {
	for orientation := uint32(1); orientation <= 8; orientation++ {
		transform, w, h := orientationMap(3, 2, orientation)
		seen := map[[2]int]bool{}
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				px, py := float64(x)+.5, float64(y)+.5
				ox, oy := transform[0]*px+transform[1]*py+transform[2], transform[3]*px+transform[4]*py+transform[5]
				if ox < 0 || oy < 0 || ox >= float64(w) || oy >= float64(h) {
					t.Fatalf("orientation %d moved a center outside raster", orientation)
				}
				seen[[2]int{int(ox), int(oy)}] = true
			}
		}
		if len(seen) != 6 {
			t.Fatalf("orientation %d merged pixels", orientation)
		}
	}
}
func TestVisualPromotionRequiresIndependentReviewAndMovementCoverage(t *testing.T) {
	m, _ := fixture()
	if err := ValidateForRender(m); err == nil {
		t.Fatal("pending inference promoted")
	}
	m.Review = &preparationv1.VisualReview{Decision: "accepted", ReviewedBy: "interactive-root", Rationale: "Inspected actual output at playback size", EvidenceRefs: []string{"owner/sample/sha256"}, ReviewedAtUnixMs: 1}
	if err := ValidateForRender(m); err != nil {
		t.Fatal(err)
	}
	m.CameraEnvelope.MaxTranslationM = .02
	if err := ValidateForRender(m); err == nil {
		t.Fatal("unreviewed disocclusion promoted")
	}
	m.CameraEnvelope.OverscanReviewed = true
	if err := ValidateForRender(m); err != nil {
		t.Fatal(err)
	}
	clone := proto.Clone(m).(*preparationv1.PreparedPhotograph)
	clone.Review.Decision = "rejected"
	if err := ValidateForRender(clone); err == nil {
		t.Fatal("rejected scene promoted")
	}
}
