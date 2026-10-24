package photographic

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"strings"
	"testing"

	"device-studio/internal/rendering"
	preparationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	"google.golang.org/protobuf/encoding/protojson"
)

func photographicFixture(t *testing.T) []byte {
	t.Helper()
	const w, h = 20, 20
	source := encodePNG(t, w, h, func(x, y int) color.NRGBA { return color.NRGBA{R: uint8(x * 7), G: uint8(y * 9), B: 90, A: 255} })
	mask := encodePNG(t, w, h, func(x, y int) color.NRGBA {
		if x >= 8 && x < 12 && y >= 8 && y < 12 {
			return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		}
		return color.NRGBA{A: 255}
	})
	clean := encodePNG(t, w, h, func(x, y int) color.NRGBA { return color.NRGBA{R: uint8(20 + x), G: uint8(30 + y), B: 110, A: 255} })
	normal := []float32{0, -.8, -.6}
	k := []float32{1, 0, .5, 0, 1, .5, 0, 0, 1}
	points, depth := make([]float32, w*h*3), make([]float32, w*h)
	validity, normals, rejected := make([]byte, w*h), make([]float32, w*h*3), make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			rx := (float64(x)+.5)/w - .5
			ry := (float64(y)+.5)/h - .5
			denom := -.8*ry - .6
			z := -1 / denom
			points[3*i], points[3*i+1], points[3*i+2] = float32(rx*z), float32(ry*z), float32(z)
			depth[i] = float32(z)
			validity[i] = 1
			normals[3*i], normals[3*i+1], normals[3*i+2] = normal[0], normal[1], normal[2]
		}
	}
	arrays := map[string][]byte{"points.npy": npy(t, "float32", []int{h, w, 3}, points), "depth.npy": npy(t, "float32", []int{h, w}, depth), "normals.npy": npy(t, "float32", []int{h, w, 3}, normals), "validity.npy": npy(t, "uint8", []int{h, w}, validity), "intrinsics.npy": npy(t, "float32", []int{3, 3}, k), "support_plane_rejected.npy": npy(t, "uint8", []int{h, w}, rejected)}
	geometryManifest := map[string]any{"format": geometryFormat, "input": map[string]any{"sha256": digest(source), "width": w, "height": h}, "raster": map[string]any{"width": w, "height": h, "pixel_centers": true}, "coordinates": map[string]any{"frame": "OpenCV camera", "intrinsics": "normalized"}, "camera_recovery": map[string]any{"postprocess": "MoGe-2 infer focal/shift recovery and pinhole reprojection"}, "model": map[string]any{"id": "moge-test", "checkpoint_sha256": stringsOf('a', 64), "checkpoint_revision": "test-revision"}, "units": map[string]any{"metric_scale": 1.3, "scale_kind": "estimated_metric", "uncertainty": "monocular prediction; not ground truth"}, "support_plane": map[string]any{"normal_opencv_camera": []float32{0, -.8, -.6}, "offset_m_estimate": 1, "inlier_points": uint32(w * h), "selected_valid_points": uint32(w * h), "rejected_roi_pixels": uint32(0), "distance_threshold_m_estimate": .01, "roi_normalized": map[string]any{"x": 0, "y": 0, "width": 1, "height": 1}, "inlier_absolute_residual_m_estimate_quantiles": []float64{0, 0, 0, 0}, "metric_status": "estimated_monocular_scale_not_measured"}, "arrays": map[string]any{"points.npy": map[string]any{"dtype": "float32", "shape": []int{h, w, 3}}, "depth.npy": map[string]any{"dtype": "float32", "shape": []int{h, w}}, "normals.npy": map[string]any{"dtype": "float32", "shape": []int{h, w, 3}}, "validity.npy": map[string]any{"dtype": "uint8", "shape": []int{h, w}}, "intrinsics.npy": map[string]any{"dtype": "float32", "shape": []int{3, 3}}, "support_plane_rejected.npy": map[string]any{"dtype": "uint8", "shape": []int{h, w}}}}
	geoJSON, _ := json.Marshal(geometryManifest)
	geometryZip := zipBytes(t, map[string][]byte{"manifest.json": geoJSON, "points.npy": arrays["points.npy"], "depth.npy": arrays["depth.npy"], "normals.npy": arrays["normals.npy"], "validity.npy": arrays["validity.npy"], "intrinsics.npy": arrays["intrinsics.npy"], "support_plane_rejected.npy": arrays["support_plane_rejected.npy"]})
	repair, _ := json.Marshal(repairDoc{Schema: "vrooli.image-tools.inpaint/v1", SourceSHA256: digest(source), MaskSHA256: digest(mask), ModelID: "approved-repair", ModelRevision: "revision-1"})
	art := func(name, mime string, raw []byte) *preparationv1.Artifact {
		return &preparationv1.Artifact{Path: name, Sha256: digest(raw), SizeBytes: int64(len(raw)), MediaType: mime}
	}
	manifest := &preparationv1.PreparedPhotograph{ContractVersion: "hybrid-device-scene/v1", Route: preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL,
		Source:     &preparationv1.SourceRaster{Original: art("original.png", "image/png", source), Oriented: art("oriented.png", "image/png", source), OriginalWidth: w, OriginalHeight: h, Width: w, Height: h, OriginalExifOrientation: 1, OriginalToOriented: []float64{1, 0, 0, 0, 1, 0, 0, 0, 1}},
		Geometry:   &preparationv1.Geometry{Archive: art("geometry.zip", "application/zip", geometryZip), Camera: &preparationv1.Camera{Basis: cameraBasis, RasterWidth: w, RasterHeight: h, NormalizedIntrinsics: []float64{1, 0, .5, 0, 1, .5, 0, 0, 1}, WorldBasis: "device-studio-right-handed-meters-x-right-y-up-z-front-v1", CameraToWorld: []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}}, SupportPlane: &preparationv1.SupportPlane{Normal: []float64{0, -.8, -.6}, Offset: 1, ValidPixelCount: w * h, InlierCount: w * h, ResidualP50: 0, ResidualP95: 0}},
		Background: &preparationv1.Background{CleanPlate: art("clean.png", "image/png", clean), DeviceMask: art("device.png", "image/png", mask), RemovalMask: art("removal.png", "image/png", mask), RepairProvenance: art("repair.json", "application/json", repair)}, ScaleStatus: "estimated_monocular_scale_not_measured", CameraEnvelope: &preparationv1.CameraEnvelope{}, Review: &preparationv1.VisualReview{Decision: "accepted", ReviewedBy: "operator-reviewer", Rationale: "Reviewed source camera clean plate", EvidenceRefs: []string{"evidence:sha256:" + stringsOf('b', 64)}, ReviewedAtUnixMs: 1}}
	manifestJSON, err := protojson.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return zipBytes(t, map[string][]byte{"manifest.json": manifestJSON, "original.png": source, "oriented.png": source, "geometry.zip": geometryZip, "clean.png": clean, "device.png": mask, "removal.png": mask, "repair.json": repair})
}

func TestConvertBuildsCameraAlignedBackgroundAndFillsDeviceFromSupportPlane(t *testing.T) {
	result, err := Convert(photographicFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Report.DeviceMaskExcluded || result.Report.RemovedPixelsReplaced == 0 || result.Report.TriangleCount < 300 {
		t.Fatalf("missing removal reconstruction: %+v", result.Report)
	}
	if result.Report.ScaleStatus != "estimated_monocular_scale_not_measured" || result.Report.EstimatedScaleMeters != 1.3 {
		t.Fatalf("scale claim lost: %+v", result.Report)
	}
	if result.Report.SourceBundleSHA256 == "" || result.Report.GeometryArchiveSHA256 == "" || !bytes.HasPrefix(result.GeometryGLB, []byte("glTF")) {
		t.Fatal("source provenance or valid GLB header missing")
	}
	if !bytes.HasPrefix(result.Texture, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatal("clean plate texture not retained")
	}
	jsonLen := int(binary.LittleEndian.Uint32(result.GeometryGLB[12:16]))
	var gltf struct {
		Accessors []struct {
			Count      int `json:"count"`
			BufferView int `json:"bufferView"`
		} `json:"accessors"`
		BufferViews []struct {
			ByteOffset int `json:"byteOffset"`
			ByteLength int `json:"byteLength"`
		} `json:"bufferViews"`
	}
	if err := json.Unmarshal(result.GeometryGLB[20:20+jsonLen], &gltf); err != nil {
		t.Fatal(err)
	}
	if len(gltf.Accessors) != 4 || gltf.Accessors[0].Count != int(result.Report.VertexCount) || gltf.Accessors[3].Count != int(result.Report.TriangleCount*3) {
		t.Fatalf("GLB counts differ from report: %+v %+v", gltf.Accessors, result.Report)
	}
	if result.Report.UVSource != "gltf_top_origin_original_source_camera_pixel_centers_v1" {
		t.Fatalf("UV convention is not explicit in conversion provenance: %q", result.Report.UVSource)
	}
	uvAccessor := gltf.Accessors[1]
	if uvAccessor.BufferView < 0 || uvAccessor.BufferView >= len(gltf.BufferViews) {
		t.Fatal("TEXCOORD_0 accessor references an invalid buffer view")
	}
	uvOffset := 20 + jsonLen + 8 + gltf.BufferViews[uvAccessor.BufferView].ByteOffset
	firstU := math.Float32frombits(binary.LittleEndian.Uint32(result.GeometryGLB[uvOffset:]))
	firstV := math.Float32frombits(binary.LittleEndian.Uint32(result.GeometryGLB[uvOffset+4:]))
	positionOffset := 20 + jsonLen + 8 + gltf.BufferViews[gltf.Accessors[0].BufferView].ByteOffset
	positionX := math.Float32frombits(binary.LittleEndian.Uint32(result.GeometryGLB[positionOffset:]))
	positionY := math.Float32frombits(binary.LittleEndian.Uint32(result.GeometryGLB[positionOffset+4:]))
	positionZ := math.Float32frombits(binary.LittleEndian.Uint32(result.GeometryGLB[positionOffset+8:]))
	// Recover OpenCV camera coordinates from canonical GLB coordinates. glTF
	// stores top-origin V; Blender's official importer performs the one flip
	// into its bottom-origin UV convention.
	expectedU := float64(positionX)/float64(-positionZ) + .5
	expectedV := float64(positionY)/float64(positionZ) + .5
	if math.Abs(float64(firstU)-expectedU) > 1e-6 || math.Abs(float64(firstV)-expectedV) > 1e-6 {
		t.Fatalf("glTF UV differs from source-camera projection: got (%.8f, %.8f), want (%.8f, %.8f)", firstU, firstV, expectedU, expectedV)
	}
	if math.Abs(expectedU-expectedV) < 0.01 {
		t.Fatalf("fixture lacks nonsymmetric camera coordinates to catch diagonal UV mapping: u=%.8f v=%.8f", expectedU, expectedV)
	}
	inventory, err := rendering.InspectPhysicalGLB(result.GeometryGLB)
	if err != nil {
		t.Fatalf("generated photo geometry is not accepted by the renderer's bounded GLB inspector: %v", err)
	}
	if len(inventory.Materials) != 1 || len(inventory.Meshes) != 1 || len(inventory.Meshes[0].Primitives) != 1 || inventory.Meshes[0].Primitives[0].Material != 0 {
		t.Fatalf("generated photo GLB must expose material slot 0 for the candidate's explicit emission binding: %+v", inventory)
	}
}

func TestConvertAcceptsReviewedUnmodifiedEmptySceneWithoutRemovalClaims(t *testing.T) {
	raw := photographicFixture(t)
	files, manifest := unpackFixture(t, raw)
	manifest.Route = preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_EMPTY_SCENE
	manifest.Background.DeviceMask = nil
	manifest.Background.RemovalMask = nil
	manifest.Background.RepairProvenance = nil
	delete(files, "device.png")
	delete(files, "removal.png")
	delete(files, "repair.json")
	files["clean.png"] = append([]byte(nil), files["oriented.png"]...)
	manifest.Background.CleanPlate.Sha256 = digest(files["clean.png"])
	manifest.Background.CleanPlate.SizeBytes = int64(len(files["clean.png"]))
	result, err := Convert(repackFixture(t, files, manifest))
	if err != nil {
		t.Fatal(err)
	}
	if result.Report.Route != "PHOTOGRAPH_ROUTE_EMPTY_SCENE" || result.Report.DeviceMaskExcluded || result.Report.RemovedPixelsReplaced != 0 {
		t.Fatalf("empty scene was reported as repaired/removal geometry: %+v", result.Report)
	}
}

func TestConvertRejectsSourceMaskReviewAndGeometryInconsistency(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"tampered-member", func(raw []byte) []byte {
			files, manifest := unpackFixture(t, raw)
			files["clean.png"][50] ^= 1
			return repackFixture(t, files, manifest)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Convert(tc.mutate(photographicFixture(t))); err == nil {
				t.Fatal("tampered photo bundle accepted")
			}
		})
	}
	// Refusal for unreviewed input is checked before any conversion or publication.
	raw := photographicFixture(t)
	files, manifest := unpackFixture(t, raw)
	manifest.Review.Decision = "pending"
	if _, err := Convert(repackFixture(t, files, manifest)); err == nil {
		t.Fatal("pending photograph was converted")
	}
	raw = photographicFixture(t)
	files, manifest = unpackFixture(t, raw)
	mask := encodePNG(t, 20, 20, func(x, y int) color.NRGBA {
		if x == 9 && y == 9 {
			return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		}
		return color.NRGBA{A: 255}
	})
	files["removal.png"] = mask
	manifest.Background.RemovalMask.Sha256 = digest(mask)
	manifest.Background.RemovalMask.SizeBytes = int64(len(mask))
	// Rebind repair provenance to this mask to prove the failure is device-mask coverage.
	var repair repairDoc
	_ = json.Unmarshal(files["repair.json"], &repair)
	repair.MaskSHA256 = digest(mask)
	files["repair.json"], _ = json.Marshal(repair)
	manifest.Background.RepairProvenance.Sha256 = digest(files["repair.json"])
	manifest.Background.RepairProvenance.SizeBytes = int64(len(files["repair.json"]))
	if _, err := Convert(repackFixture(t, files, manifest)); err == nil {
		t.Fatal("removal mask omitted device pixels but was accepted")
	}
}

func encodePNG(t *testing.T, w, h int, pixel func(int, int) color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, pixel(x, y))
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func npy(t *testing.T, dtype string, shape []int, values any) []byte {
	t.Helper()
	var descr string
	var data []byte
	switch v := values.(type) {
	case []float32:
		descr = "'<f4'"
		data = make([]byte, 4*len(v))
		for i, x := range v {
			binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(x))
		}
	case []byte:
		descr = "'|u1'"
		data = append([]byte(nil), v...)
	default:
		t.Fatalf("unsupported npy data %T", values)
	}
	_ = dtype
	tuple := ""
	for i, n := range shape {
		if i > 0 {
			tuple += ", "
		}
		tuple += fmt.Sprint(n)
	}
	if len(shape) == 1 {
		tuple += ","
	}
	header := fmt.Sprintf("{'descr': %s, 'fortran_order': False, 'shape': (%s), }", descr, tuple)
	padding := 16 - (10+len(header)+1)%16
	if padding == 16 {
		padding = 0
	}
	header += strings.Repeat(" ", padding) + "\n"
	out := []byte("\x93NUMPY\x01\x00")
	length := make([]byte, 2)
	binary.LittleEndian.PutUint16(length, uint16(len(header)))
	out = append(out, length...)
	out = append(out, header...)
	out = append(out, data...)
	return out
}
func zipBytes(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for name, data := range files {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func unpackFixture(t *testing.T, raw []byte) (map[string][]byte, *preparationv1.PreparedPhotograph) {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range z.File {
		r, _ := f.Open()
		data, _ := io.ReadAll(r)
		_ = r.Close()
		files[f.Name] = data
	}
	m := &preparationv1.PreparedPhotograph{}
	if err := protojson.Unmarshal(files["manifest.json"], m); err != nil {
		t.Fatal(err)
	}
	delete(files, "manifest.json")
	return files, m
}
func repackFixture(t *testing.T, files map[string][]byte, m *preparationv1.PreparedPhotograph) []byte {
	t.Helper()
	raw, err := protojson.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	copyFiles := map[string][]byte{}
	for n, b := range files {
		copyFiles[n] = b
	}
	copyFiles["manifest.json"] = raw
	return zipBytes(t, copyFiles)
}
func stringsOf(ch byte, n int) string { return string(bytes.Repeat([]byte{ch}, n)) }
func TestDigestFixtureStable(t *testing.T) {
	sum := sha256.Sum256([]byte("x"))
	if hex.EncodeToString(sum[:]) != digest([]byte("x")) {
		t.Fatal("digest helper mismatch")
	}
}
