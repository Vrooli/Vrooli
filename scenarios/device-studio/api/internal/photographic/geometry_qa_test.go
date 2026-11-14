package photographic

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image/color"
	"image/png"
	"io"
	"testing"

	preparationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
)

func emptyPendingFixture(t *testing.T) []byte {
	t.Helper()
	files, manifest := unpackFixture(t, photographicFixture(t))
	manifest.Route = preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_EMPTY_SCENE
	manifest.Background.CleanPlate = manifest.Source.Oriented
	manifest.Background.DeviceMask = nil
	manifest.Background.RemovalMask = nil
	manifest.Background.RepairProvenance = nil
	manifest.Review = &preparationv1.VisualReview{Decision: "pending"}
	delete(files, "clean.png")
	delete(files, "device.png")
	delete(files, "removal.png")
	delete(files, "repair.json")
	return repackFixture(t, files, manifest)
}

func TestInspectPendingReportsLegacyP90StoredAsOwnerP95WithoutAllowingConversion(t *testing.T) {
	files, manifest := unpackFixture(t, emptyPendingFixture(t))
	geometryZip, err := zip.NewReader(bytes.NewReader(files["geometry.zip"]), int64(len(files["geometry.zip"])))
	if err != nil {
		t.Fatal(err)
	}
	geometryFiles := map[string][]byte{}
	for _, member := range geometryZip.File {
		reader, err := member.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		geometryFiles[member.Name] = data
	}
	var geometryManifest map[string]any
	if err := json.Unmarshal(geometryFiles["manifest.json"], &geometryManifest); err != nil {
		t.Fatal(err)
	}
	plane := geometryManifest["support_plane"].(map[string]any)
	plane["inlier_absolute_residual_m_estimate_quantiles"] = []any{0.001, 0.002, 0.003, 0.004}
	geometryFiles["manifest.json"], _ = json.Marshal(geometryManifest)
	files["geometry.zip"] = zipBytes(t, geometryFiles)
	manifest.Geometry.Archive.Sha256 = digest(files["geometry.zip"])
	manifest.Geometry.Archive.SizeBytes = int64(len(files["geometry.zip"]))
	manifest.Geometry.SupportPlane.ResidualP50 = 0.001
	manifest.Geometry.SupportPlane.ResidualP95 = 0.002 // historical owner recorded P90

	qa, _, err := InspectPending(repackFixture(t, files, manifest))
	if err != nil {
		t.Fatal(err)
	}
	if !qa.Plane.OwnerP95Mismatch || qa.Plane.OwnerManifestP95M != 0.002 || qa.Plane.ResidualP95EstimateM != 0.003 {
		t.Fatalf("QA failed to report owner/inner quantile discrepancy: %+v", qa.Plane)
	}
	if _, err := Convert(repackFixture(t, files, manifest)); err == nil {
		t.Fatal("ordinary conversion must remain strict for inconsistent owner residual metadata")
	}
}

func TestInspectPendingProducesStrictBoundedDiagnosticOnly(t *testing.T) {
	qa, preview, err := InspectPending(emptyPendingFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if !qa.InspectionOnly || qa.Accepted || qa.ReviewDecision != "pending" || qa.SourceBundleSHA256 == "" || qa.SourceRasterSHA256 == "" || qa.GeometryArchiveSHA256 == "" {
		t.Fatalf("invalid QA trust state/provenance: %+v", qa)
	}
	if qa.Projection.CheckedSamples < 100 || qa.Projection.MaxEuclideanPx > .01 || qa.Plane.ObservedInlierPixels != qa.Plane.InlierPoints || len(qa.ValidatedArrays) != 6 {
		t.Fatalf("strict geometry checks missing: projection=%+v plane=%+v arrays=%d", qa.Projection, qa.Plane, len(qa.ValidatedArrays))
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(preview))
	if err != nil || cfg.Width != qa.GeometryRaster.Width*3 || cfg.Height != qa.GeometryRaster.Height {
		t.Fatalf("bad three-panel preview dimensions=%+v err=%v", cfg, err)
	}
	bundle, err := GeometryQAZIP(qa, preview)
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil || len(r.File) != 2 {
		t.Fatalf("invalid diagnostic archive: members=%d err=%v", len(r.File), err)
	}
	var report map[string]any
	for _, f := range r.File {
		if f.Name == "geometry-qa.json" {
			body, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(body)
			_ = body.Close()
			if err != nil || json.Unmarshal(data, &report) != nil {
				t.Fatal("invalid QA report JSON")
			}
		}
	}
	if report["accepted"] != false || report["inspection_only"] != true || report["review_decision"] != "pending" {
		t.Fatalf("diagnostic archive changed trust state: %#v", report)
	}
}

func TestInspectPendingRejectsAcceptedAndChangedCleanPlate(t *testing.T) {
	raw := emptyPendingFixture(t)
	files, manifest := unpackFixture(t, raw)
	manifest.Review = &preparationv1.VisualReview{Decision: "accepted", ReviewedBy: "operator", Rationale: "ok", EvidenceRefs: []string{"evidence"}, ReviewedAtUnixMs: 1}
	if _, _, err := InspectPending(repackFixture(t, files, manifest)); err == nil {
		t.Fatal("accepted source should not pass pending-only QA")
	}
	manifest.Review = &preparationv1.VisualReview{Decision: "pending"}
	files["clean.png"] = encodePNG(t, 20, 20, func(x, y int) color.NRGBA { return color.NRGBA{R: uint8(x), A: 255} })
	manifest.Background.CleanPlate = &preparationv1.Artifact{Path: "clean.png", Sha256: digest(files["clean.png"]), SizeBytes: int64(len(files["clean.png"])), MediaType: "image/png"}
	if _, _, err := InspectPending(repackFixture(t, files, manifest)); err == nil {
		t.Fatal("changed clean plate must not qualify as original EMPTY_SCENE source")
	}
}

func TestROIPixelBoundsUseHalfOpenPixelCenterCoordinates(t *testing.T) {
	bounds := roiPixelCentersBounds(.03, .52, .94, .22, 1024, 576)
	if bounds != (roiBounds{X0: 31, X1: 993, Y0: 300, Y1: 426}) {
		t.Fatalf("ROI pixel-center bounds=%+v, want producer's exclusive bounds", bounds)
	}
}
