// Package photographic converts a retained Image Tools photographic exchange
// into source-camera geometry. It has no renderer or inference dependencies.
package photographic

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"path"
	"strings"

	preparationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	maxBundleBytes   = 256 << 20
	maxExpandedBytes = 192 << 20
	maxRasterPixels  = 16_000_000
	maxGeometryCells = 1_500_000
	maxGLBBytes      = 128 << 20
	geometryFormat   = "vrooli.image-tools.scene-geometry.v1"
	cameraBasis      = "opencv-camera-x-right-y-down-z-forward-v1"
)

// Background is a deterministic, source-camera-aligned triangle mesh plus its
// exact clean-plate raster. Scale remains an estimate inherited from MoGe.
type Background struct {
	GeometryGLB []byte
	Texture     []byte
	Report      Report
}

type Report struct {
	ContractVersion        string     `json:"contract_version"`
	Route                  string     `json:"photograph_route"`
	SourceBundleSHA256     string     `json:"source_bundle_sha256"`
	SourceRasterSHA256     string     `json:"source_raster_sha256"`
	GeometryArchiveSHA256  string     `json:"geometry_archive_sha256"`
	CleanPlateSHA256       string     `json:"clean_plate_sha256"`
	DeviceMaskSHA256       string     `json:"device_mask_sha256,omitempty"`
	RemovalMaskSHA256      string     `json:"removal_mask_sha256,omitempty"`
	RepairProvenanceSHA256 string     `json:"repair_provenance_sha256"`
	SourceReviewSHA256     string     `json:"source_visual_review_sha256"`
	SourceReviewer         string     `json:"source_visual_reviewer"`
	SourceReviewRationale  string     `json:"source_visual_review_rationale"`
	SourceReviewEvidence   []string   `json:"source_visual_review_evidence"`
	SourceUseClaimSHA256   string     `json:"source_use_claim_sha256,omitempty"`
	LocalRenderAuthorized  bool       `json:"local_render_authorized"`
	SourceUseRationale     string     `json:"source_use_rationale,omitempty"`
	GeometryModel          string     `json:"geometry_model"`
	CheckpointSHA256       string     `json:"checkpoint_sha256"`
	Postprocess            string     `json:"postprocess"`
	ScaleStatus            string     `json:"scale_status"`
	EstimatedScaleMeters   float64    `json:"estimated_scale_meters"`
	Intrinsics             [9]float64 `json:"normalized_intrinsics"`
	SupportPlaneNormal     [3]float64 `json:"support_plane_normal_opencv"`
	SupportPlaneOffsetM    float64    `json:"support_plane_offset_estimate_m"`
	SupportPlaneInliers    uint32     `json:"support_plane_inliers"`
	SupportPlaneResidual95 float64    `json:"support_plane_residual_p95_estimate_m"`
	RasterWidth            uint32     `json:"raster_width"`
	RasterHeight           uint32     `json:"raster_height"`
	VertexCount            uint32     `json:"vertex_count"`
	TriangleCount          uint32     `json:"triangle_count"`
	DeviceMaskExcluded     bool       `json:"device_mask_excluded"`
	RemovedPixelsReplaced  uint32     `json:"removed_pixels_replaced_by_support_plane"`
	UVSource               string     `json:"uv_source"`
}

type artifactDoc struct {
	Path string `json:"path"`
	SHA  string `json:"sha256"`
	Size int64  `json:"size_bytes"`
	MIME string `json:"media_type"`
}

type geometryDoc struct {
	Format string `json:"format"`
	Input  struct {
		SHA    string `json:"sha256"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"input"`
	Raster struct {
		Width   int  `json:"width"`
		Height  int  `json:"height"`
		Centers bool `json:"pixel_centers"`
	} `json:"raster"`
	Coordinates struct {
		Frame      string `json:"frame"`
		Intrinsics string `json:"intrinsics"`
	} `json:"coordinates"`
	Model struct {
		ID                 string `json:"id"`
		CheckpointSHA256   string `json:"checkpoint_sha256"`
		CheckpointRevision string `json:"checkpoint_revision"`
	} `json:"model"`
	Postprocess struct {
		Method string `json:"postprocess"`
	} `json:"camera_recovery"`
	Units struct {
		MetricScale float64 `json:"metric_scale"`
		ScaleKind   string  `json:"scale_kind"`
		Status      string  `json:"uncertainty"`
	} `json:"units"`
	SupportPlane struct {
		Normal              []float64 `json:"normal_opencv_camera"`
		Offset              float64   `json:"offset_m_estimate"`
		Inliers             uint32    `json:"inlier_points"`
		SelectedValidPoints uint32    `json:"selected_valid_points"`
		RejectedROIPixels   uint32    `json:"rejected_roi_pixels"`
		DistanceThreshold   float64   `json:"distance_threshold_m_estimate"`
		ROI                 struct {
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"roi_normalized"`
		Residuals                []float64 `json:"inlier_absolute_residual_m_estimate_quantiles"`
		MetricStatus             string    `json:"metric_status"`
		OuterResidualP95Mismatch bool      `json:"-"`
	} `json:"support_plane"`
	Arrays map[string]arrayDoc `json:"arrays"`
}

type arrayDoc struct {
	DType string `json:"dtype"`
	Shape []int  `json:"shape"`
}

type repairDoc struct {
	Schema        string `json:"schema"`
	SourceSHA256  string `json:"source_sha256"`
	MaskSHA256    string `json:"mask_sha256"`
	ModelID       string `json:"model_id"`
	ModelRevision string `json:"model_revision"`
	// Authored repairs retain caller-supplied tool/prompt evidence separately
	// from local inference. A full-frame edit does not promise mask-exact pixels.
	OutputSHA256        string `json:"output_sha256"`
	Tool                string `json:"tool"`
	ToolOutputReference string `json:"tool_output_reference"`
	Prompt              string `json:"prompt"`
	PromptSHA256        string `json:"prompt_sha256"`
	PixelScope          string `json:"pixel_scope"`
}

type numpyArray struct {
	dtype string
	shape []int
	data  []byte
}

type bundle struct {
	manifest *preparationv1.PreparedPhotograph
	files    map[string][]byte
	sha      string
}

// Convert verifies the exact retained ZIP and its inner MoGe arrays before
// constructing a bounded camera-projected mesh. It never claims measured scale.
func Convert(raw []byte) (Background, error) {
	if len(raw) == 0 || len(raw) > maxBundleBytes {
		return Background{}, errors.New("prepared photograph bundle is empty or oversized")
	}
	b, err := openBundle(raw)
	if err != nil {
		return Background{}, err
	}
	if err := validatePrepared(b.manifest); err != nil {
		return Background{}, err
	}
	geoBytes := b.files[b.manifest.Geometry.Archive.Path]
	geo, arrays, err := openGeometry(geoBytes, b.manifest, false)
	if err != nil {
		return Background{}, err
	}
	plate := b.files[b.manifest.Background.CleanPlate.Path]
	if b.manifest.Route == preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL {
		if err := validateRepairProvenance(b.manifest, b.files); err != nil {
			return Background{}, err
		}
	} else if !bytes.Equal(plate, b.files[b.manifest.Source.Oriented.Path]) {
		return Background{}, errors.New("empty-scene route requires the unmodified oriented source as its clean plate")
	}
	if err := validateRemovalMasks(b.manifest, b.files); err != nil {
		return Background{}, err
	}
	glb, report, err := makeBackgroundGLB(b, geo, arrays, plate)
	if err != nil {
		return Background{}, err
	}
	return Background{GeometryGLB: glb, Texture: append([]byte(nil), plate...), Report: report}, nil
}

func openBundle(raw []byte) (*bundle, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("prepared photograph ZIP: %w", err)
	}
	files := map[string][]byte{}
	var expanded uint64
	for _, f := range zr.File {
		if len(files) >= 17 || !portablePath(f.Name) || f.FileInfo().IsDir() || !f.Mode().IsRegular() {
			return nil, fmt.Errorf("invalid prepared photograph member %q", f.Name)
		}
		if _, exists := files[f.Name]; exists {
			return nil, fmt.Errorf("duplicate prepared photograph member %q", f.Name)
		}
		if f.UncompressedSize64 > maxExpandedBytes-expanded {
			return nil, errors.New("prepared photograph exceeds expanded byte limit")
		}
		expanded += f.UncompressedSize64
		if f.UncompressedSize64 > uint64(maxBundleBytes) {
			return nil, errors.New("prepared photograph member exceeds byte limit")
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(r, int64(f.UncompressedSize64)+1))
		closeErr := r.Close()
		if readErr != nil || closeErr != nil || uint64(len(data)) != f.UncompressedSize64 {
			return nil, errors.Join(readErr, closeErr, errors.New("prepared photograph member size mismatch"))
		}
		files[f.Name] = data
	}
	manifestBytes, ok := files["manifest.json"]
	if !ok || len(manifestBytes) > 64<<10 {
		return nil, errors.New("prepared photograph manifest missing or oversized")
	}
	manifest := &preparationv1.PreparedPhotograph{}
	if err := protojson.Unmarshal(manifestBytes, manifest); err != nil {
		return nil, fmt.Errorf("decode prepared photograph manifest: %w", err)
	}
	declared, err := declaredArtifacts(manifest)
	if err != nil {
		return nil, err
	}
	if len(files) != len(declared)+1 {
		return nil, errors.New("prepared photograph contains undeclared or missing files")
	}
	for name, spec := range declared {
		data, exists := files[name]
		if !exists || int64(len(data)) != spec.Size || digest(data) != spec.SHA {
			return nil, fmt.Errorf("prepared photograph artifact %q size or digest mismatch", name)
		}
	}
	return &bundle{manifest: manifest, files: files, sha: digest(raw)}, nil
}

func declaredArtifacts(m *preparationv1.PreparedPhotograph) (map[string]artifactDoc, error) {
	if m == nil || m.Source == nil || m.Geometry == nil || m.Background == nil {
		return nil, errors.New("prepared photograph source, geometry and background are required")
	}
	artifacts := map[string]artifactDoc{}
	add := func(a *preparationv1.Artifact) error {
		if a == nil {
			return nil
		}
		if !portablePath(a.Path) || a.Path == "manifest.json" || a.SizeBytes <= 0 || len(a.Sha256) != 64 || a.MediaType == "" {
			return fmt.Errorf("invalid prepared artifact identity %q", a.Path)
		}
		if prior, exists := artifacts[a.Path]; exists && (prior.SHA != a.Sha256 || prior.Size != a.SizeBytes || prior.MIME != a.MediaType) {
			return fmt.Errorf("prepared artifact path %q has conflicting declarations", a.Path)
		}
		artifacts[a.Path] = artifactDoc{Path: a.Path, SHA: a.Sha256, Size: a.SizeBytes, MIME: a.MediaType}
		return nil
	}
	for _, a := range []*preparationv1.Artifact{m.Source.Original, m.Source.Oriented, m.Geometry.Archive, m.Background.CleanPlate, m.Background.DeviceMask, m.Background.RemovalMask, m.Background.RepairProvenance} {
		if err := add(a); err != nil {
			return nil, err
		}
	}
	for _, a := range m.Background.OccluderMasks {
		if err := add(a); err != nil {
			return nil, err
		}
	}
	return artifacts, nil
}

func validatePrepared(m *preparationv1.PreparedPhotograph) error {
	return validatePreparedWithPending(m, false)
}

func validatePreparedWithPending(m *preparationv1.PreparedPhotograph, allowPending bool) error {
	if m.ContractVersion != "hybrid-device-scene/v1" || m.ScaleStatus != "estimated_monocular_scale_not_measured" {
		return errors.New("unsupported photographic contract or metric-scale claim")
	}
	if m.Route != preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL && m.Route != preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_EMPTY_SCENE {
		return errors.New("photographic background conversion requires an empty-scene or device-removal preparation")
	}
	if m.Review == nil {
		return errors.New("photographic input requires explicit review state")
	}
	if allowPending {
		if m.Review.Decision != "pending" || m.Review.ReviewedAtUnixMs != 0 {
			return errors.New("geometry QA inspection is restricted to pending preparations")
		}
	} else if m.Review.Decision != "accepted" || m.Review.ReviewedAtUnixMs <= 0 || strings.TrimSpace(m.Review.ReviewedBy) == "" || strings.TrimSpace(m.Review.Rationale) == "" || len(m.Review.EvidenceRefs) == 0 {
		return errors.New("photographic input requires an accepted, attributed visual source review")
	}
	s, g, bg := m.Source, m.Geometry, m.Background
	if s.Original == nil || s.Oriented == nil || g.Archive == nil || bg.CleanPlate == nil || g.Camera == nil || g.SupportPlane == nil {
		return errors.New("photographic input requires source, complete geometry, clean plate, camera and support plane")
	}
	if m.Route == preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL && (bg.DeviceMask == nil || bg.RemovalMask == nil || bg.RepairProvenance == nil) {
		return errors.New("device removal requires masks and repair provenance")
	}
	if m.Route == preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_EMPTY_SCENE && (bg.DeviceMask != nil || bg.RemovalMask != nil || bg.RepairProvenance != nil || len(bg.OccluderMasks) > 0) {
		return errors.New("empty-scene route cannot contain device-removal or occluder artifacts")
	}
	if g.Camera.Basis != cameraBasis || g.Camera.RasterWidth == 0 || g.Camera.RasterHeight == 0 || len(g.Camera.NormalizedIntrinsics) != 9 || !finite(g.Camera.NormalizedIntrinsics) || g.Camera.NormalizedIntrinsics[0] <= 0 || g.Camera.NormalizedIntrinsics[4] <= 0 {
		return errors.New("camera intrinsics or coordinate basis are invalid")
	}
	if len(g.SupportPlane.Normal) != 3 || !finite(g.SupportPlane.Normal) || !finite([]float64{g.SupportPlane.Offset, g.SupportPlane.ResidualP50, g.SupportPlane.ResidualP95}) || math.Abs(norm3(g.SupportPlane.Normal)-1) > 1e-5 || g.SupportPlane.InlierCount < 3 || g.SupportPlane.ResidualP95 < g.SupportPlane.ResidualP50 || g.SupportPlane.ResidualP50 < 0 {
		return errors.New("support plane is malformed or has insufficient stable inliers")
	}
	if s.Width == 0 || s.Height == 0 || uint64(s.Width)*uint64(s.Height) > maxRasterPixels {
		return errors.New("source raster dimensions are invalid")
	}
	if s.OriginalWidth == 0 || s.OriginalHeight == 0 || uint64(s.OriginalWidth)*uint64(s.OriginalHeight) > maxRasterPixels || s.OriginalExifOrientation < 1 || s.OriginalExifOrientation > 8 {
		return errors.New("original source dimensions or EXIF orientation are invalid")
	}
	if g.Camera.WorldBasis != "device-studio-right-handed-meters-x-right-y-up-z-front-v1" || !rigid(g.Camera.CameraToWorld) {
		return errors.New("camera requires an explicit rigid Device Studio camera-to-world transform")
	}
	if math.Abs(g.Camera.NormalizedIntrinsics[1])+math.Abs(g.Camera.NormalizedIntrinsics[3])+math.Abs(g.Camera.NormalizedIntrinsics[6])+math.Abs(g.Camera.NormalizedIntrinsics[7])+math.Abs(g.Camera.NormalizedIntrinsics[8]-1) > 1e-8 {
		return errors.New("camera K must be a normalized pinhole matrix")
	}
	return nil
}

func validateRepairProvenance(m *preparationv1.PreparedPhotograph, files map[string][]byte) error {
	data := files[m.Background.RepairProvenance.Path]
	if len(data) == 0 || len(data) > 64<<10 {
		return errors.New("repair provenance must be present and bounded")
	}
	var p repairDoc
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("decode repair provenance: %w", err)
	}
	if p.Schema == "vrooli.image-tools.authored-clean-plate/v1" {
		if p.SourceSHA256 != m.Source.Oriented.Sha256 || p.MaskSHA256 != m.Background.RemovalMask.Sha256 || p.OutputSHA256 != m.Background.CleanPlate.Sha256 {
			return errors.New("authored clean-plate provenance must bind the oriented source, removal mask and exact output")
		}
		if !boundedEvidenceText(p.Tool, 128) || !boundedEvidenceText(p.ToolOutputReference, 1024) || !boundedEvidenceText(p.Prompt, 16<<10) || digest([]byte(p.Prompt)) != p.PromptSHA256 {
			return errors.New("authored clean-plate provenance requires a bounded tool reference and exact prompt/hash")
		}
		if p.PixelScope != "full_frame_generated" || p.ModelID != "" || p.ModelRevision != "" {
			return errors.New("authored clean plate must declare full-frame generated pixels separately from local model provenance")
		}
		return nil
	}
	if p.Schema != "vrooli.image-tools.inpaint/v1" || p.SourceSHA256 != m.Source.Original.Sha256 || p.MaskSHA256 != m.Background.RemovalMask.Sha256 || strings.TrimSpace(p.ModelID) == "" || strings.TrimSpace(p.ModelRevision) == "" {
		return errors.New("repair provenance source, effective removal mask, or pinned model identity mismatch")
	}
	return nil
}

func boundedEvidenceText(value string, limit int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= limit && !strings.ContainsAny(value, "\x00\r")
}

func validateRemovalMasks(m *preparationv1.PreparedPhotograph, files map[string][]byte) error {
	if m.Route == preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_EMPTY_SCENE {
		return nil
	}
	device, err := decodeMask(files[m.Background.DeviceMask.Path], int(m.Source.Width), int(m.Source.Height))
	if err != nil {
		return fmt.Errorf("device mask: %w", err)
	}
	removal, err := decodeMask(files[m.Background.RemovalMask.Path], int(m.Source.Width), int(m.Source.Height))
	if err != nil {
		return fmt.Errorf("removal mask: %w", err)
	}
	for i, selected := range device {
		if selected && !removal[i] {
			return errors.New("removal mask does not exclude every original device pixel")
		}
	}
	if len(m.Background.OccluderMasks) > 8 {
		return errors.New("too many foreground occluder masks")
	}
	for _, a := range m.Background.OccluderMasks {
		if _, err := decodeMask(files[a.Path], int(m.Source.Width), int(m.Source.Height)); err != nil {
			return fmt.Errorf("occluder mask %q: %w", a.Path, err)
		}
	}
	return nil
}

func decodeMask(data []byte, width, height int) ([]bool, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil || img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		return nil, errors.New("mask must be a decodable source-sized raster")
	}
	out := make([]bool, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a > 0 && (r+g+b)/3 >= 0x8000 {
				out[y*width+x] = true
			}
		}
	}
	return out, nil
}

func openGeometry(raw []byte, m *preparationv1.PreparedPhotograph, inspectionOnly bool) (geometryDoc, map[string]numpyArray, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil || len(zr.File) > 12 {
		return geometryDoc{}, nil, errors.New("invalid or oversized geometry archive")
	}
	files := map[string][]byte{}
	var expanded uint64
	for _, f := range zr.File {
		if !portablePath(f.Name) || f.FileInfo().IsDir() || f.UncompressedSize64 > maxExpandedBytes-expanded {
			return geometryDoc{}, nil, errors.New("unsafe geometry archive member or expanded size")
		}
		if _, exists := files[f.Name]; exists {
			return geometryDoc{}, nil, errors.New("duplicate geometry archive member")
		}
		expanded += f.UncompressedSize64
		r, err := f.Open()
		if err != nil {
			return geometryDoc{}, nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(r, int64(f.UncompressedSize64)+1))
		closeErr := r.Close()
		if readErr != nil || closeErr != nil || uint64(len(data)) != f.UncompressedSize64 {
			return geometryDoc{}, nil, errors.Join(readErr, closeErr, errors.New("geometry member size mismatch"))
		}
		files[f.Name] = data
	}
	var doc geometryDoc
	if err := json.Unmarshal(files["manifest.json"], &doc); err != nil || len(files["manifest.json"]) > 64<<10 {
		return geometryDoc{}, nil, errors.New("geometry archive manifest missing or invalid")
	}
	if doc.Format != geometryFormat || doc.Coordinates.Frame != "OpenCV camera" || !doc.Raster.Centers || doc.Units.ScaleKind != "estimated_metric" || doc.SupportPlane.MetricStatus != "estimated_monocular_scale_not_measured" {
		return geometryDoc{}, nil, errors.New("geometry archive coordinate, pixel-center, or estimated-scale contract mismatch")
	}
	if doc.Input.SHA != m.Source.Original.Sha256 || doc.Input.Width != int(m.Source.OriginalWidth) || doc.Input.Height != int(m.Source.OriginalHeight) || doc.Raster.Width != int(m.Geometry.Camera.RasterWidth) || doc.Raster.Height != int(m.Geometry.Camera.RasterHeight) {
		return geometryDoc{}, nil, errors.New("geometry archive source or raster dimensions disagree with photographic manifest")
	}
	if doc.Model.ID == "" || !validSHA(doc.Model.CheckpointSHA256) || doc.Postprocess.Method == "" || !finite([]float64{doc.Units.MetricScale, doc.SupportPlane.Offset, doc.SupportPlane.DistanceThreshold, doc.SupportPlane.ROI.X, doc.SupportPlane.ROI.Y, doc.SupportPlane.ROI.Width, doc.SupportPlane.ROI.Height}) || doc.Units.MetricScale <= 0 || doc.SupportPlane.DistanceThreshold <= 0 || doc.SupportPlane.ROI.X < 0 || doc.SupportPlane.ROI.Y < 0 || doc.SupportPlane.ROI.Width <= 0 || doc.SupportPlane.ROI.Height <= 0 || doc.SupportPlane.ROI.X+doc.SupportPlane.ROI.Width > 1 || doc.SupportPlane.ROI.Y+doc.SupportPlane.ROI.Height > 1 || doc.SupportPlane.SelectedValidPoints < doc.SupportPlane.Inliers || len(doc.SupportPlane.Normal) != 3 || !finite(doc.SupportPlane.Normal) {
		return geometryDoc{}, nil, errors.New("geometry archive model, postprocess, scale or plane provenance is incomplete")
	}
	for i, value := range doc.SupportPlane.Normal {
		if math.Abs(value-m.Geometry.SupportPlane.Normal[i]) > 1e-5 {
			return geometryDoc{}, nil, errors.New("support-plane normal disagrees with geometry archive")
		}
	}
	if math.Abs(doc.SupportPlane.Offset-m.Geometry.SupportPlane.Offset) > 1e-5 {
		return geometryDoc{}, nil, errors.New("support-plane offset disagrees with geometry archive")
	}
	// The inner producer emits [P50, P90, P95, P100]. Keep ordinary conversion
	// strict. Pending-only QA may inspect an older owner record that stored P90
	// in its P95 field, but must report that discrepancy explicitly.
	if doc.SupportPlane.Inliers != uint32(m.Geometry.SupportPlane.InlierCount) || len(doc.SupportPlane.Residuals) != 4 || !finite(doc.SupportPlane.Residuals) || math.Abs(doc.SupportPlane.Residuals[0]-m.Geometry.SupportPlane.ResidualP50) > 1e-5 {
		return geometryDoc{}, nil, errors.New("support-plane fit counts or residual quantiles disagree with photographic manifest")
	}
	if math.Abs(doc.SupportPlane.Residuals[2]-m.Geometry.SupportPlane.ResidualP95) > 1e-5 {
		if !inspectionOnly || math.Abs(doc.SupportPlane.Residuals[1]-m.Geometry.SupportPlane.ResidualP95) > 1e-5 {
			return geometryDoc{}, nil, errors.New("support-plane fit counts or residual quantiles disagree with photographic manifest")
		}
		doc.SupportPlane.OuterResidualP95Mismatch = true
	}
	shape2 := []int{doc.Raster.Height, doc.Raster.Width}
	shape3 := []int{doc.Raster.Height, doc.Raster.Width, 3}
	expected := map[string]arrayDoc{
		"points.npy": {DType: "float32", Shape: shape3}, "depth.npy": {DType: "float32", Shape: shape2},
		"normals.npy": {DType: "float32", Shape: shape3}, "validity.npy": {DType: "uint8", Shape: shape2},
		"intrinsics.npy": {DType: "float32", Shape: []int{3, 3}}, "support_plane_rejected.npy": {DType: "uint8", Shape: shape2},
	}
	if len(files) != len(expected)+1 || len(doc.Arrays) != len(expected) {
		return geometryDoc{}, nil, errors.New("geometry archive must contain the exact v1 array set")
	}
	arrays := map[string]numpyArray{}
	for filename, spec := range expected {
		declared, ok := doc.Arrays[filename]
		if !ok || declared.DType != spec.DType || !sameInts(declared.Shape, spec.Shape) {
			return geometryDoc{}, nil, fmt.Errorf("geometry array %q declaration differs from contract", filename)
		}
		data, ok := files[filename]
		if !ok {
			return geometryDoc{}, nil, fmt.Errorf("geometry array %q missing", filename)
		}
		arr, err := parseNPY(data, spec.DType, spec.Shape)
		if err != nil {
			return geometryDoc{}, nil, fmt.Errorf("geometry array %q: %w", filename, err)
		}
		arrays[filename] = arr
	}
	compareK, err := parseNPY(files["intrinsics.npy"], "float32", []int{3, 3})
	if err != nil {
		return geometryDoc{}, nil, err
	}
	for i, got := range float32Values(compareK.data) {
		if !finite([]float64{float64(got)}) || math.Abs(float64(got)-m.Geometry.Camera.NormalizedIntrinsics[i]) > 1e-5 {
			return geometryDoc{}, nil, errors.New("geometry array intrinsics disagree with PreparedPhotograph K")
		}
	}
	if err := validateProjection(arrays, doc, m.Geometry.Camera); err != nil {
		return geometryDoc{}, nil, err
	}
	return doc, arrays, nil
}

func parseNPY(data []byte, wantDType string, wantShape []int) (numpyArray, error) {
	if len(data) < 10 || !bytes.Equal(data[:6], []byte("\x93NUMPY")) || data[6] != 1 && data[6] != 2 {
		return numpyArray{}, errors.New("unsupported NPY magic or version")
	}
	headerLen := int(binary.LittleEndian.Uint16(data[8:10]))
	headerStart := 10
	if data[6] == 2 {
		if len(data) < 12 {
			return numpyArray{}, errors.New("truncated NPY v2 header")
		}
		headerLen = int(binary.LittleEndian.Uint32(data[8:12]))
		headerStart = 12
	}
	if headerLen <= 0 || headerLen > 4096 || headerStart+headerLen > len(data) {
		return numpyArray{}, errors.New("NPY header is oversized or truncated")
	}
	header := string(data[headerStart : headerStart+headerLen])
	dtype := ""
	for _, marker := range []string{"'descr':", "\"descr\":"} {
		if at := strings.Index(header, marker); at >= 0 {
			tail := strings.TrimSpace(header[at+len(marker):])
			if len(tail) >= 3 && (tail[0] == '\'' || tail[0] == '"') {
				end := strings.IndexByte(tail[1:], tail[0])
				if end > 0 {
					rawType := tail[1 : end+1]
					switch rawType {
					case "<f4", "=f4":
						dtype = "float32"
					case "|u1", "u1":
						dtype = "uint8"
					}
				}
			}
			break
		}
	}
	if dtype != wantDType || strings.Contains(header, "fortran_order': True") || strings.Contains(header, "fortran_order\": True") {
		return numpyArray{}, errors.New("NPY dtype or memory order is unsupported")
	}
	shapeStart := strings.Index(header, "'shape':")
	if shapeStart < 0 {
		shapeStart = strings.Index(header, "\"shape\":")
	}
	if shapeStart < 0 {
		return numpyArray{}, errors.New("NPY shape header missing")
	}
	left := strings.Index(header[shapeStart:], "(")
	right := strings.Index(header[shapeStart:], ")")
	if left < 0 || right <= left {
		return numpyArray{}, errors.New("NPY shape header malformed")
	}
	parts := strings.Split(header[shapeStart+left+1:shapeStart+right], ",")
	shape := make([]int, 0, len(parts))
	count := uint64(1)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var dim uint64
		if _, err := fmt.Sscanf(part, "%d", &dim); err != nil || dim == 0 || dim > maxGeometryCells*3 {
			return numpyArray{}, errors.New("NPY dimension is invalid")
		}
		if count > uint64(maxGeometryCells*9)/dim {
			return numpyArray{}, errors.New("NPY element count exceeds bound")
		}
		count *= dim
		shape = append(shape, int(dim))
	}
	if !sameInts(shape, wantShape) {
		return numpyArray{}, errors.New("NPY dimensions disagree with manifest")
	}
	itemSize := uint64(4)
	if wantDType == "uint8" {
		itemSize = 1
	}
	expectedBytes := count * itemSize
	start := headerStart + headerLen
	if expectedBytes > uint64(len(data)-start) || uint64(len(data)-start) != expectedBytes {
		return numpyArray{}, errors.New("NPY data byte count differs from declared shape")
	}
	return numpyArray{dtype: dtype, shape: shape, data: data[start:]}, nil
}

func validateProjection(a map[string]numpyArray, doc geometryDoc, camera *preparationv1.Camera) error {
	points, depth, validity := a["points.npy"].data, a["depth.npy"].data, a["validity.npy"].data
	w, h := doc.Raster.Width, doc.Raster.Height
	k := camera.NormalizedIntrinsics
	checked := 0
	for i := 0; i < w*h; i += maxInt(1, w*h/4096) {
		if validity[i] == 0 {
			continue
		}
		z := float64(math.Float32frombits(binary.LittleEndian.Uint32(depth[4*i:])))
		x := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[12*i:])))
		y := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[12*i+4:])))
		pz := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[12*i+8:])))
		if !finite([]float64{x, y, z, pz}) || z <= 0 || pz <= 0 {
			return errors.New("geometry archive contains nonfinite valid depth or point")
		}
		px, py := (float64(i%w)+.5)/float64(w), (float64(i/w)+.5)/float64(h)
		expectedX := (px - k[2]) / k[0] * pz
		expectedY := (py - k[5]) / k[4] * pz
		if math.Abs(pz-z) > 1e-4*math.Max(z, 1) || math.Abs(x-expectedX) > 2e-3*math.Max(math.Abs(x), 1) || math.Abs(y-expectedY) > 2e-3*math.Max(math.Abs(y), 1) {
			return errors.New("geometry points fail camera-K/depth projection consistency")
		}
		checked++
	}
	if checked < 100 {
		return errors.New("geometry archive contains too few valid camera-projection samples")
	}
	return nil
}

func makeBackgroundGLB(b *bundle, geo geometryDoc, arrays map[string]numpyArray, texture []byte) ([]byte, Report, error) {
	img, _, err := image.DecodeConfig(bytes.NewReader(texture))
	if err != nil || img.Width != int(b.manifest.Source.Width) || img.Height != int(b.manifest.Source.Height) {
		return nil, Report{}, errors.New("clean plate dimensions differ from oriented source raster")
	}
	w, h := geo.Raster.Width, geo.Raster.Height
	if uint64(w)*uint64(h) > maxGeometryCells {
		return nil, Report{}, errors.New("geometry grid exceeds conversion cell limit")
	}
	var remove []bool
	if b.manifest.Route == preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL {
		remove, err = decodeMask(b.files[b.manifest.Background.RemovalMask.Path], int(b.manifest.Source.Width), int(b.manifest.Source.Height))
		if err != nil {
			return nil, Report{}, err
		}
	} else {
		remove = make([]bool, int(b.manifest.Source.Width)*int(b.manifest.Source.Height))
	}
	validity, points := arrays["validity.npy"].data, arrays["points.npy"].data
	k := b.manifest.Geometry.Camera.NormalizedIntrinsics
	n := b.manifest.Geometry.SupportPlane.Normal
	d := b.manifest.Geometry.SupportPlane.Offset
	vertices := make([][8]float32, w*h) // position, camera-projected UV, normal
	valid := make([]bool, w*h)
	var replaced uint32
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			// Map the low-resolution geometry sample center to source raster.
			sx := minInt(int((float64(x)+.5)*float64(b.manifest.Source.Width)/float64(w)), int(b.manifest.Source.Width)-1)
			sy := minInt(int((float64(y)+.5)*float64(b.manifest.Source.Height)/float64(h)), int(b.manifest.Source.Height)-1)
			removed := remove[sy*int(b.manifest.Source.Width)+sx]
			var px, py, pz float64
			if !removed && validity[i] == 1 {
				px = float64(math.Float32frombits(binary.LittleEndian.Uint32(points[12*i:])))
				py = float64(math.Float32frombits(binary.LittleEndian.Uint32(points[12*i+4:])))
				pz = float64(math.Float32frombits(binary.LittleEndian.Uint32(points[12*i+8:])))
			} else if removed {
				// Excluded device pixels are reconstructed on the stable table plane,
				// rather than simply deleted and left as a mesh hole.
				u, v := (float64(x)+.5)/float64(w), (float64(y)+.5)/float64(h)
				rx, ry := (u-k[2])/k[0], (v-k[5])/k[4]
				denom := n[0]*rx + n[1]*ry + n[2]
				if math.Abs(denom) < 1e-7 {
					continue
				}
				t := -d / denom
				if !finite([]float64{t}) || t <= 0 {
					continue
				}
				px, py, pz = rx*t, ry*t, t
				replaced++
			} else {
				continue
			}
			if !finite([]float64{px, py, pz}) || pz <= 1e-5 {
				continue
			}
			// OpenCV camera (right, down, forward) to Device Studio camera local
			// (right, up, back). Projected UVs retain the original photo framing.
			nx, ny, nz := float64(0), float64(-1), float64(0)
			if removed {
				nx, ny, nz = n[0], -n[1], -n[2]
			} else {
				normalData := arrays["normals.npy"].data
				nx = float64(math.Float32frombits(binary.LittleEndian.Uint32(normalData[12*i:])))
				ny = -float64(math.Float32frombits(binary.LittleEndian.Uint32(normalData[12*i+4:])))
				nz = -float64(math.Float32frombits(binary.LittleEndian.Uint32(normalData[12*i+8:])))
			}
			norm := math.Sqrt(nx*nx + ny*ny + nz*nz)
			if norm < 1e-8 || !finite([]float64{nx, ny, nz}) {
				continue
			}
			// glTF stores texture V from the image's top edge; Blender's glTF
			// importer converts that to its bottom-origin UV convention. Keep
			// the source-camera projection here without a second V inversion.
			vertices[i] = [8]float32{float32(px), float32(-py), float32(-pz), float32(k[0]*px/pz + k[2]), float32(k[4]*py/pz + k[5]), float32(nx / norm), float32(ny / norm), float32(nz / norm)}
			valid[i] = true
		}
	}
	indices := make([]uint32, 0, (w-1)*(h-1)*6)
	medianEdge := estimateGridEdge(vertices, validity, w, h)
	maxEdge := math.Max(medianEdge*8, .02)
	for y := 0; y < h-1; y++ {
		for x := 0; x < w-1; x++ {
			a, c, d0, e := y*w+x, y*w+x+1, (y+1)*w+x, (y+1)*w+x+1
			if !valid[a] || !valid[c] || !valid[d0] || !valid[e] {
				continue
			}
			// Refuse long discontinuity bridges, particularly across object edges.
			if distance3(vertices[a], vertices[c]) > maxEdge || distance3(vertices[a], vertices[d0]) > maxEdge || distance3(vertices[c], vertices[e]) > maxEdge || distance3(vertices[d0], vertices[e]) > maxEdge {
				continue
			}
			indices = append(indices, uint32(a), uint32(d0), uint32(c), uint32(c), uint32(d0), uint32(e))
		}
	}
	if len(indices) < 300 {
		return nil, Report{}, errors.New("background mesh has insufficient connected surface")
	}
	remap := make([]uint32, len(vertices))
	for i := range remap {
		remap[i] = ^uint32(0)
	}
	compact := make([][8]float32, 0, len(indices)/2)
	for i, index := range indices {
		if remap[index] == ^uint32(0) {
			remap[index] = uint32(len(compact))
			compact = append(compact, vertices[index])
		}
		indices[i] = remap[index]
	}
	glb, err := encodeGLB(compact, indices)
	if err != nil {
		return nil, Report{}, err
	}
	if len(glb) > maxGLBBytes {
		return nil, Report{}, errors.New("background GLB exceeds library geometry limit")
	}
	var normal [3]float64
	copy(normal[:], b.manifest.Geometry.SupportPlane.Normal)
	report := Report{
		ContractVersion: "hybrid-device-scene/v1", Route: b.manifest.Route.String(), SourceBundleSHA256: b.sha,
		SourceRasterSHA256: b.manifest.Source.Original.Sha256, GeometryArchiveSHA256: b.manifest.Geometry.Archive.Sha256,
		CleanPlateSHA256: b.manifest.Background.CleanPlate.Sha256,
		SourceReviewer:   b.manifest.Review.ReviewedBy, SourceReviewRationale: b.manifest.Review.Rationale,
		SourceReviewEvidence: append([]string(nil), b.manifest.Review.EvidenceRefs...), GeometryModel: geo.Model.ID,
		CheckpointSHA256: geo.Model.CheckpointSHA256, Postprocess: geo.Postprocess.Method, ScaleStatus: b.manifest.ScaleStatus,
		EstimatedScaleMeters: geo.Units.MetricScale, SupportPlaneNormal: normal, SupportPlaneOffsetM: geo.SupportPlane.Offset,
		SupportPlaneInliers: geo.SupportPlane.Inliers, SupportPlaneResidual95: b.manifest.Geometry.SupportPlane.ResidualP95,
		RasterWidth: uint32(w), RasterHeight: uint32(h), VertexCount: uint32(len(compact)), TriangleCount: uint32(len(indices) / 3),
		DeviceMaskExcluded: b.manifest.Route == preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL, RemovedPixelsReplaced: replaced, UVSource: "gltf_top_origin_original_source_camera_pixel_centers_v1",
	}
	if b.manifest.Background.DeviceMask != nil {
		report.DeviceMaskSHA256 = b.manifest.Background.DeviceMask.Sha256
	}
	if b.manifest.Background.RemovalMask != nil {
		report.RemovalMaskSHA256 = b.manifest.Background.RemovalMask.Sha256
	}
	if b.manifest.Background.RepairProvenance != nil {
		report.RepairProvenanceSHA256 = b.manifest.Background.RepairProvenance.Sha256
	}
	for i := range report.Intrinsics {
		report.Intrinsics[i] = k[i]
	}
	reviewBytes, err := protojson.Marshal(b.manifest.Review)
	if err != nil {
		return nil, Report{}, err
	}
	report.SourceReviewSHA256 = digest(reviewBytes)
	return glb, report, nil
}

func encodeGLB(vertices [][8]float32, indices []uint32) ([]byte, error) {
	const posBytes = 12
	const uvBytes = 8
	const normalBytes = 12
	pos := make([]byte, len(vertices)*posBytes)
	uv := make([]byte, len(vertices)*uvBytes)
	normals := make([]byte, len(vertices)*normalBytes)
	for i, v := range vertices {
		for c := 0; c < 3; c++ {
			binary.LittleEndian.PutUint32(pos[(i*3+c)*4:], math.Float32bits(v[c]))
		}
		for c := 0; c < 2; c++ {
			binary.LittleEndian.PutUint32(uv[(i*2+c)*4:], math.Float32bits(v[3+c]))
		}
		for c := 0; c < 3; c++ {
			binary.LittleEndian.PutUint32(normals[(i*3+c)*4:], math.Float32bits(v[5+c]))
		}
	}
	indexData := make([]byte, len(indices)*4)
	for i, value := range indices {
		binary.LittleEndian.PutUint32(indexData[i*4:], value)
	}
	bin := append(append(append(append([]byte{}, pos...), uv...), normals...), indexData...)
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	posOffset, uvOffset, normalOffset, indexOffset := 0, len(pos), len(pos)+len(uv), len(pos)+len(uv)+len(normals)
	minP, maxP := [3]float32{float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(1))}, [3]float32{float32(math.Inf(-1)), float32(math.Inf(-1)), float32(math.Inf(-1))}
	for _, v := range vertices {
		for c := 0; c < 3; c++ {
			minP[c] = minFloat32(minP[c], v[c])
			maxP[c] = maxFloat32(maxP[c], v[c])
		}
	}
	doc := map[string]any{
		"asset": map[string]any{"version": "2.0", "generator": "device-studio-photographic-background-v1"},
		"scene": 0, "scenes": []any{map[string]any{"nodes": []int{0}}}, "nodes": []any{map[string]any{"name": "ReconstructedPhotographicBackground", "mesh": 0, "translation": []float64{0, 0, 0}, "rotation": []float64{0, 0, 0, 1}, "scale": []float64{1, 1, 1}}},
		"meshes": []any{map[string]any{"primitives": []any{map[string]any{"attributes": map[string]int{"POSITION": 0, "TEXCOORD_0": 1, "NORMAL": 2}, "indices": 3, "material": 0, "mode": 4}}}},
		// The candidate scene binds primitive slot 0 to its source-image
		// emission material. Keep the embedded GLB independently valid for
		// the physical renderer's strict inventory check; the neutral base
		// factors are superseded by that immutable candidate binding.
		"materials": []any{map[string]any{"name": "photographic-background-slot-0", "pbrMetallicRoughness": map[string]any{"baseColorFactor": []float64{1, 1, 1, 1}, "metallicFactor": 0, "roughnessFactor": 1}}},
		"buffers":   []any{map[string]any{"byteLength": len(bin)}},
		"bufferViews": []any{
			map[string]any{"buffer": 0, "byteOffset": posOffset, "byteLength": len(pos), "target": 34962},
			map[string]any{"buffer": 0, "byteOffset": uvOffset, "byteLength": len(uv), "target": 34962},
			map[string]any{"buffer": 0, "byteOffset": normalOffset, "byteLength": len(normals), "target": 34962},
			map[string]any{"buffer": 0, "byteOffset": indexOffset, "byteLength": len(indexData), "target": 34963},
		},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": len(vertices), "type": "VEC3", "min": minP, "max": maxP},
			map[string]any{"bufferView": 1, "componentType": 5126, "count": len(vertices), "type": "VEC2"},
			map[string]any{"bufferView": 2, "componentType": 5126, "count": len(vertices), "type": "VEC3"},
			map[string]any{"bufferView": 3, "componentType": 5125, "count": len(indices), "type": "SCALAR"},
		},
	}
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	for len(jsonData)%4 != 0 {
		jsonData = append(jsonData, ' ')
	}
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	length := 12 + 8 + len(jsonData) + 8 + len(bin)
	if length > maxGLBBytes {
		return nil, errors.New("GLB exceeds size limit")
	}
	out := make([]byte, length)
	binary.LittleEndian.PutUint32(out[0:], 0x46546c67)
	binary.LittleEndian.PutUint32(out[4:], 2)
	binary.LittleEndian.PutUint32(out[8:], uint32(length))
	binary.LittleEndian.PutUint32(out[12:], uint32(len(jsonData)))
	binary.LittleEndian.PutUint32(out[16:], 0x4e4f534a)
	copy(out[20:], jsonData)
	binary.LittleEndian.PutUint32(out[20+len(jsonData):], uint32(len(bin)))
	binary.LittleEndian.PutUint32(out[24+len(jsonData):], 0x004e4942)
	copy(out[28+len(jsonData):], bin)
	return out, nil
}

func estimateGridEdge(vertices [][8]float32, validity []byte, w, h int) float64 {
	values := make([]float64, 0, 4096)
	step := maxInt(1, (w*h)/4096)
	for i := 0; i < w*h; i += step {
		if i%w+1 < w && validity[i] == 1 && validity[i+1] == 1 {
			values = append(values, distance3(vertices[i], vertices[i+1]))
		}
	}
	if len(values) == 0 {
		return .01
	}
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
	return values[len(values)/2]
}

func distance3(a, b [8]float32) float64 {
	x, y, z := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(x*x + y*y + z*z)
}
func digest(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func validSHA(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil && len(s) == 64 && s == strings.ToLower(s)
}
func portablePath(s string) bool {
	return s != "" && len(s) <= 128 && !strings.ContainsAny(s, "\\:\x00") && !strings.HasPrefix(s, "/") && path.Clean(s) == s && s != "." && s != ".." && !strings.HasPrefix(s, "../")
}
func finite(v []float64) bool {
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return false
		}
	}
	return true
}
func norm3(a []float64) float64 { return math.Sqrt(a[0]*a[0] + a[1]*a[1] + a[2]*a[2]) }
func rigid(m []float64) bool {
	if len(m) != 16 || !finite(m) || math.Abs(m[3])+math.Abs(m[7])+math.Abs(m[11])+math.Abs(m[15]-1) > 1e-8 {
		return false
	}
	x, y, z := m[:3], m[4:7], m[8:11]
	dot := func(a, b []float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
	if math.Abs(dot(x, x)-1) > 1e-6 || math.Abs(dot(y, y)-1) > 1e-6 || math.Abs(dot(z, z)-1) > 1e-6 || math.Abs(dot(x, y)) > 1e-6 || math.Abs(dot(x, z)) > 1e-6 || math.Abs(dot(y, z)) > 1e-6 {
		return false
	}
	cross := [3]float64{x[1]*y[2] - x[2]*y[1], x[2]*y[0] - x[0]*y[2], x[0]*y[1] - x[1]*y[0]}
	return math.Abs(cross[0]*z[0]+cross[1]*z[1]+cross[2]*z[2]-1) < 1e-6
}
func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func float32Values(data []byte) []float32 {
	out := make([]float32, len(data)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return out
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
