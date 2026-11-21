package photographic

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"sort"
	"time"
)

type GeometryQA struct {
	InspectionOnly        bool                `json:"inspection_only"`
	Accepted              bool                `json:"accepted"`
	ReviewDecision        string              `json:"review_decision"`
	SourceBundleSHA256    string              `json:"source_bundle_sha256"`
	SourceRasterSHA256    string              `json:"source_raster_sha256"`
	GeometryArchiveSHA256 string              `json:"geometry_archive_sha256"`
	GeometryFormat        string              `json:"geometry_format"`
	Model                 string              `json:"model"`
	CheckpointSHA256      string              `json:"checkpoint_sha256"`
	Postprocess           string              `json:"postprocess"`
	ScaleStatus           string              `json:"scale_status"`
	EstimatedMetricScale  float64             `json:"estimated_metric_scale"`
	SourceRaster          rasterQA            `json:"source_raster"`
	GeometryRaster        rasterQA            `json:"geometry_raster"`
	Camera                cameraQA            `json:"camera"`
	Projection            projectionQA        `json:"source_reprojection_consistency"`
	Plane                 supportPlaneQA      `json:"support_plane"`
	Depth                 depthQA             `json:"depth"`
	ValidatedArrays       []arrayValidationQA `json:"strictly_validated_npy_arrays"`
	PreviewPanels         []string            `json:"preview_panels"`
	Warnings              []string            `json:"warnings"`
}

type rasterQA struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type cameraQA struct {
	Basis          string     `json:"basis"`
	NormalizedK    [9]float64 `json:"normalized_intrinsics"`
	FocalPixels    [2]float64 `json:"focal_pixels"`
	PrincipalPixel [2]float64 `json:"principal_point_pixels"`
}

type projectionQA struct {
	CheckedSamples int     `json:"checked_samples"`
	MaxAbsErrorXpx float64 `json:"max_absolute_x_error_pixels"`
	MaxAbsErrorYpx float64 `json:"max_absolute_y_error_pixels"`
	MaxEuclideanPx float64 `json:"max_euclidean_error_pixels"`
	Method         string  `json:"method"`
}

type supportPlaneQA struct {
	NormalOpenCVCamera    [3]float64 `json:"normal_opencv_camera"`
	OffsetEstimateMeters  float64    `json:"offset_estimate_meters"`
	MetricStatus          string     `json:"metric_status"`
	ThresholdEstimateM    float64    `json:"inlier_threshold_estimate_meters"`
	ROI                   roiQA      `json:"selected_roi_normalized"`
	ROIPixelBounds        roiBounds  `json:"roi_pixel_exclusive_bounds"`
	SelectedValidPoints   uint32     `json:"selected_valid_points"`
	ObservedSelectedValid uint32     `json:"observed_selected_valid_pixels"`
	InlierPoints          uint32     `json:"manifest_inlier_points"`
	ObservedInlierPixels  uint32     `json:"observed_inlier_pixels"`
	ObservedRejected      uint32     `json:"observed_rejected_roi_pixels"`
	ManifestRejected      uint32     `json:"manifest_rejected_roi_pixels"`
	PlanePixelsAcrossView uint32     `json:"observed_plane_consistent_pixels_across_view"`
	ResidualP50EstimateM  float64    `json:"residual_p50_estimate_meters"`
	ResidualP95EstimateM  float64    `json:"residual_p95_estimate_meters"`
	OwnerManifestP95M     float64    `json:"owner_manifest_residual_p95_estimate_meters"`
	OwnerP95Mismatch      bool       `json:"owner_manifest_p95_differs_from_inner_p95"`
}

type roiQA struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type roiBounds struct {
	X0 int `json:"x0"`
	X1 int `json:"x1"`
	Y0 int `json:"y0"`
	Y1 int `json:"y1"`
}

type depthQA struct {
	ValidPixels int     `json:"valid_pixels"`
	MinM        float64 `json:"min_estimate_meters"`
	P02M        float64 `json:"p02_estimate_meters"`
	P98M        float64 `json:"p98_estimate_meters"`
	MaxM        float64 `json:"max_estimate_meters"`
}

type arrayValidationQA struct {
	Member string `json:"member"`
	DType  string `json:"dtype"`
	Shape  []int  `json:"shape"`
	Bytes  int    `json:"validated_data_bytes"`
}

// InspectPending applies the production outer-bundle, inner-geometry, NPY and
// projection validators to one pending EMPTY_SCENE preparation. It only emits
// diagnostics; it never changes review state or generates render assets.
func InspectPending(raw []byte) (GeometryQA, []byte, error) {
	b, err := openBundle(raw)
	if err != nil {
		return GeometryQA{}, nil, err
	}
	if b.manifest.Review == nil || b.manifest.Review.Decision != "pending" {
		return GeometryQA{}, nil, errors.New("geometry QA is restricted to pending preparations")
	}
	if b.manifest.Route.String() != "PHOTOGRAPH_ROUTE_EMPTY_SCENE" {
		return GeometryQA{}, nil, errors.New("geometry QA plate preview only supports original EMPTY_SCENE preparations")
	}
	if err := validatePreparedWithPending(b.manifest, true); err != nil {
		return GeometryQA{}, nil, err
	}
	if !bytes.Equal(b.files[b.manifest.Background.CleanPlate.Path], b.files[b.manifest.Source.Oriented.Path]) {
		return GeometryQA{}, nil, errors.New("EMPTY_SCENE geometry QA requires a clean plate identical to the original oriented source")
	}
	geo, arrays, err := openGeometry(b.files[b.manifest.Geometry.Archive.Path], b.manifest, true)
	if err != nil {
		return GeometryQA{}, nil, err
	}
	qa, err := summarizeGeometryQA(b, geo, arrays)
	if err != nil {
		return GeometryQA{}, nil, err
	}
	preview, err := renderGeometryQAPreview(b, geo, arrays)
	if err != nil {
		return GeometryQA{}, nil, err
	}
	return qa, preview, nil
}

// GeometryQAZIP packages a bounded read-only inspection report and PNG. Its
// fixed member timestamps make repeated inspection of the same bytes stable.
func GeometryQAZIP(qa GeometryQA, preview []byte) ([]byte, error) {
	if !qa.InspectionOnly || qa.Accepted || qa.ReviewDecision != "pending" || len(preview) == 0 || len(preview) > 24<<20 {
		return nil, errors.New("invalid pending-only geometry QA result")
	}
	manifest, err := json.MarshalIndent(qa, "", "  ")
	if err != nil || len(manifest) > 64<<10 {
		return nil, errors.New("geometry QA report is invalid or oversized")
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, item := range []struct {
		name string
		data []byte
	}{{"geometry-qa.json", manifest}, {"geometry-qa.png", preview}} {
		header := &zip.FileHeader{Name: item.name, Method: zip.Deflate}
		header.SetModTime(time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC))
		w, err := zw.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(item.data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if out.Len() > 32<<20 {
		return nil, errors.New("geometry QA archive exceeds output limit")
	}
	return out.Bytes(), nil
}

func summarizeGeometryQA(b *bundle, geo geometryDoc, arrays map[string]numpyArray) (GeometryQA, error) {
	w, h := geo.Raster.Width, geo.Raster.Height
	k := b.manifest.Geometry.Camera.NormalizedIntrinsics
	qa := GeometryQA{
		InspectionOnly: true, Accepted: false, ReviewDecision: b.manifest.Review.Decision,
		SourceBundleSHA256: b.sha, SourceRasterSHA256: b.manifest.Source.Original.Sha256,
		GeometryArchiveSHA256: b.manifest.Geometry.Archive.Sha256, GeometryFormat: geo.Format,
		Model: geo.Model.ID, CheckpointSHA256: geo.Model.CheckpointSHA256, Postprocess: geo.Postprocess.Method,
		ScaleStatus: b.manifest.ScaleStatus, EstimatedMetricScale: geo.Units.MetricScale,
		SourceRaster:   rasterQA{Width: int(b.manifest.Source.Width), Height: int(b.manifest.Source.Height)},
		GeometryRaster: rasterQA{Width: w, Height: h},
		Camera:         cameraQA{Basis: b.manifest.Geometry.Camera.Basis, NormalizedK: [9]float64{k[0], k[1], k[2], k[3], k[4], k[5], k[6], k[7], k[8]}, FocalPixels: [2]float64{k[0] * float64(w), k[4] * float64(h)}, PrincipalPixel: [2]float64{k[2] * float64(w), k[5] * float64(h)}},
		PreviewPanels:  []string{"source with selected support ROI/inlier/rejected samples", "MoGe camera-Z depth estimate", "support-plane residual classification and boundary"},
		Warnings:       []string{"All scene scale, depth, intrinsics and plane distances are monocular estimates; no measured camera calibration or physical distance is established.", "The preview is diagnostic evidence only and does not accept the preparation or approve camera motion."},
	}
	for _, name := range []string{"points.npy", "depth.npy", "normals.npy", "validity.npy", "intrinsics.npy", "support_plane_rejected.npy"} {
		array := arrays[name]
		qa.ValidatedArrays = append(qa.ValidatedArrays, arrayValidationQA{Member: name, DType: array.dtype, Shape: append([]int(nil), array.shape...), Bytes: len(array.data)})
	}
	var projection projectionQA
	projection.Method = "sampled pinhole reprojection from float32 XYZ/depth with pixel-center normalized K"
	stride := maxInt(1, (w*h)/8192)
	points, depths, validity := arrays["points.npy"].data, arrays["depth.npy"].data, arrays["validity.npy"].data
	for i := 0; i < w*h; i += stride {
		if validity[i] == 0 {
			continue
		}
		x := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[i*12:])))
		y := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[i*12+4:])))
		z := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[i*12+8:])))
		depth := float64(math.Float32frombits(binary.LittleEndian.Uint32(depths[i*4:])))
		if z <= 0 || math.Abs(z-depth) > 1e-4*math.Max(depth, 1) {
			return GeometryQA{}, errors.New("QA projection sample has invalid depth/point agreement")
		}
		u := k[0]*x/z + k[2]
		v := k[4]*y/z + k[5]
		expectedU := (float64(i%w) + .5) / float64(w)
		expectedV := (float64(i/w) + .5) / float64(h)
		dx := math.Abs(u-expectedU) * float64(w)
		dy := math.Abs(v-expectedV) * float64(h)
		projection.CheckedSamples++
		projection.MaxAbsErrorXpx = math.Max(projection.MaxAbsErrorXpx, dx)
		projection.MaxAbsErrorYpx = math.Max(projection.MaxAbsErrorYpx, dy)
		projection.MaxEuclideanPx = math.Max(projection.MaxEuclideanPx, math.Hypot(dx, dy))
	}
	if projection.CheckedSamples < 100 {
		return GeometryQA{}, errors.New("QA found too few valid reprojection samples")
	}
	qa.Projection = projection

	roi := geo.SupportPlane.ROI
	bounds := roiPixelCentersBounds(roi.X, roi.Y, roi.Width, roi.Height, w, h)
	x0, x1, y0, y1 := bounds.X0, bounds.X1, bounds.Y0, bounds.Y1
	plane := supportPlaneQA{NormalOpenCVCamera: [3]float64{geo.SupportPlane.Normal[0], geo.SupportPlane.Normal[1], geo.SupportPlane.Normal[2]}, OffsetEstimateMeters: geo.SupportPlane.Offset, MetricStatus: geo.SupportPlane.MetricStatus, ThresholdEstimateM: geo.SupportPlane.DistanceThreshold, ROI: roiQA{X: roi.X, Y: roi.Y, Width: roi.Width, Height: roi.Height}, ROIPixelBounds: roiBounds{X0: x0, X1: x1, Y0: y0, Y1: y1}, SelectedValidPoints: geo.SupportPlane.SelectedValidPoints, InlierPoints: geo.SupportPlane.Inliers, ManifestRejected: geo.SupportPlane.RejectedROIPixels}
	if len(geo.SupportPlane.Residuals) >= 4 {
		plane.ResidualP50EstimateM = geo.SupportPlane.Residuals[0]
		plane.ResidualP95EstimateM = geo.SupportPlane.Residuals[2]
	}
	plane.OwnerManifestP95M = b.manifest.Geometry.SupportPlane.ResidualP95
	plane.OwnerP95Mismatch = geo.SupportPlane.OuterResidualP95Mismatch
	if plane.OwnerP95Mismatch {
		qa.Warnings = append(qa.Warnings, "The immutable Image Tools owner manifest retained the inner P90 residual in its P95 field; this inspection reports the inner archive P95 and owner value separately. Import remains blocked by strict conversion validation until a corrected preparation is created.")
	}
	rejected := arrays["support_plane_rejected.npy"].data
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if x >= x0 && x < x1 && y >= y0 && y < y1 {
				if validity[i] == 1 {
					plane.ObservedSelectedValid++
				}
				if rejected[i] == 1 {
					plane.ObservedRejected++
				} else if validity[i] == 1 {
					plane.ObservedInlierPixels++
				}
			}
			if validity[i] == 0 {
				continue
			}
			px := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[i*12:])))
			py := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[i*12+4:])))
			pz := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[i*12+8:])))
			residual := math.Abs(geo.SupportPlane.Normal[0]*px + geo.SupportPlane.Normal[1]*py + geo.SupportPlane.Normal[2]*pz + geo.SupportPlane.Offset)
			if residual <= geo.SupportPlane.DistanceThreshold {
				plane.PlanePixelsAcrossView++
			}
		}
	}
	if plane.ObservedRejected != plane.ManifestRejected || plane.ObservedSelectedValid != plane.SelectedValidPoints {
		return GeometryQA{}, errors.New("support-plane selected/rejected-pixel counts differ from the retained manifest")
	}
	if plane.ObservedInlierPixels != plane.InlierPoints {
		return GeometryQA{}, fmt.Errorf("support-plane inlier count mismatch: NPY=%d manifest=%d", plane.ObservedInlierPixels, plane.InlierPoints)
	}
	qa.Plane = plane
	depthValues := make([]float64, 0, w*h)
	for i := 0; i < w*h; i++ {
		if validity[i] == 0 {
			continue
		}
		z := float64(math.Float32frombits(binary.LittleEndian.Uint32(depths[i*4:])))
		if !finite([]float64{z}) || z <= 0 {
			return GeometryQA{}, errors.New("depth QA found a nonpositive or nonfinite valid sample")
		}
		depthValues = append(depthValues, z)
	}
	if len(depthValues) < 100 {
		return GeometryQA{}, errors.New("depth QA found too few valid samples")
	}
	sort.Float64s(depthValues)
	qa.Depth = depthQA{ValidPixels: len(depthValues), MinM: depthValues[0], P02M: depthValues[int(float64(len(depthValues)-1)*.02)], P98M: depthValues[int(float64(len(depthValues)-1)*.98)], MaxM: depthValues[len(depthValues)-1]}
	return qa, nil
}

func renderGeometryQAPreview(b *bundle, geo geometryDoc, arrays map[string]numpyArray) ([]byte, error) {
	w, h := geo.Raster.Width, geo.Raster.Height
	if w <= 0 || h <= 0 || w > 2048 || h > 2048 {
		return nil, errors.New("geometry QA preview raster is out of bounds")
	}
	decoded, _, err := image.Decode(bytes.NewReader(b.files[b.manifest.Source.Oriented.Path]))
	if err != nil {
		return nil, fmt.Errorf("decode QA source raster: %w", err)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, w*3, h))
	points, depths, validity, rejected := arrays["points.npy"].data, arrays["depth.npy"].data, arrays["validity.npy"].data, arrays["support_plane_rejected.npy"].data
	roi := geo.SupportPlane.ROI
	bounds := roiPixelCentersBounds(roi.X, roi.Y, roi.Width, roi.Height, w, h)
	x0, x1, y0, y1 := bounds.X0, bounds.X1, bounds.Y0, bounds.Y1
	depthValues := make([]float64, 0, w*h)
	for i := 0; i < w*h; i++ {
		if validity[i] == 1 {
			depthValues = append(depthValues, float64(math.Float32frombits(binary.LittleEndian.Uint32(depths[4*i:]))))
		}
	}
	if len(depthValues) == 0 {
		return nil, errors.New("no valid depth samples for QA preview")
	}
	sort.Float64s(depthValues)
	low, high := depthValues[int(float64(len(depthValues)-1)*.02)], depthValues[int(float64(len(depthValues)-1)*.98)]
	if high <= low {
		high = low + 1e-6
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			sx := minInt(int((float64(x)+.5)*float64(decoded.Bounds().Dx())/float64(w)), decoded.Bounds().Dx()-1)
			sy := minInt(int((float64(y)+.5)*float64(decoded.Bounds().Dy())/float64(h)), decoded.Bounds().Dy()-1)
			base := color.RGBAModel.Convert(decoded.At(decoded.Bounds().Min.X+sx, decoded.Bounds().Min.Y+sy)).(color.RGBA)
			left := base
			inside := x >= x0 && x < x1 && y >= y0 && y < y1
			if inside {
				if rejected[i] != 0 || validity[i] == 0 {
					left = blendQA(base, color.RGBA{R: 255, G: 38, B: 42, A: 255}, 0.62)
				} else {
					left = blendQA(base, color.RGBA{R: 38, G: 235, B: 100, A: 255}, 0.38)
				}
			}
			canvas.SetRGBA(x, y, left)
			if validity[i] == 1 {
				z := float64(math.Float32frombits(binary.LittleEndian.Uint32(depths[4*i:])))
				t := clamp01((z - low) / (high - low))
				canvas.SetRGBA(w+x, y, depthColor(t))
				px := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[12*i:])))
				py := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[12*i+4:])))
				pz := float64(math.Float32frombits(binary.LittleEndian.Uint32(points[12*i+8:])))
				residual := math.Abs(geo.SupportPlane.Normal[0]*px + geo.SupportPlane.Normal[1]*py + geo.SupportPlane.Normal[2]*pz + geo.SupportPlane.Offset)
				if residual <= geo.SupportPlane.DistanceThreshold {
					canvas.SetRGBA(2*w+x, y, color.RGBA{R: 30, G: 210, B: 245, A: 255})
				} else {
					canvas.SetRGBA(2*w+x, y, residualColor(residual/geo.SupportPlane.DistanceThreshold))
				}
			} else {
				canvas.SetRGBA(w+x, y, color.RGBA{R: 28, G: 30, B: 38, A: 255})
				canvas.SetRGBA(2*w+x, y, color.RGBA{R: 28, G: 30, B: 38, A: 255})
			}
		}
	}
	for x := x0; x < x1; x++ {
		if y0 >= 0 && y0 < h {
			canvas.SetRGBA(x, y0, color.RGBA{R: 255, G: 220, B: 20, A: 255})
			canvas.SetRGBA(w+x, y0, color.RGBA{R: 255, G: 220, B: 20, A: 255})
			canvas.SetRGBA(2*w+x, y0, color.RGBA{R: 255, G: 220, B: 20, A: 255})
		}
		if y1 > 0 && y1 <= h {
			canvas.SetRGBA(x, y1-1, color.RGBA{R: 255, G: 220, B: 20, A: 255})
			canvas.SetRGBA(w+x, y1-1, color.RGBA{R: 255, G: 220, B: 20, A: 255})
			canvas.SetRGBA(2*w+x, y1-1, color.RGBA{R: 255, G: 220, B: 20, A: 255})
		}
	}
	for y := y0; y < y1; y++ {
		if x0 >= 0 && x0 < w {
			canvas.SetRGBA(x0, y, color.RGBA{R: 255, G: 220, B: 20, A: 255})
			canvas.SetRGBA(w+x0, y, color.RGBA{R: 255, G: 220, B: 20, A: 255})
			canvas.SetRGBA(2*w+x0, y, color.RGBA{R: 255, G: 220, B: 20, A: 255})
		}
		if x1 > 0 && x1 <= w {
			canvas.SetRGBA(x1-1, y, color.RGBA{R: 255, G: 220, B: 20, A: 255})
			canvas.SetRGBA(w+x1-1, y, color.RGBA{R: 255, G: 220, B: 20, A: 255})
			canvas.SetRGBA(2*w+x1-1, y, color.RGBA{R: 255, G: 220, B: 20, A: 255})
		}
	}
	var out bytes.Buffer
	if err := pngEncode(&out, canvas); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// roiPixelCentersBounds mirrors the producer's half-open normalized ROI over
// pixel centers: (x+0.5)/width >= start and < end.
func roiPixelCentersBounds(x, y, roiWidth, roiHeight float64, width, height int) roiBounds {
	return roiBounds{
		X0: maxInt(0, minInt(width, int(math.Ceil(x*float64(width)-.5)))),
		X1: maxInt(0, minInt(width, int(math.Ceil((x+roiWidth)*float64(width)-.5)))),
		Y0: maxInt(0, minInt(height, int(math.Ceil(y*float64(height)-.5)))),
		Y1: maxInt(0, minInt(height, int(math.Ceil((y+roiHeight)*float64(height)-.5)))),
	}
}

func blendQA(a, b color.RGBA, alpha float64) color.RGBA {
	return color.RGBA{R: uint8(float64(a.R)*(1-alpha) + float64(b.R)*alpha), G: uint8(float64(a.G)*(1-alpha) + float64(b.G)*alpha), B: uint8(float64(a.B)*(1-alpha) + float64(b.B)*alpha), A: 255}
}

func depthColor(t float64) color.RGBA {
	// Compact blue→cyan→yellow→red map; the report records the quantile range.
	stops := []color.RGBA{{R: 20, G: 30, B: 180, A: 255}, {R: 20, G: 210, B: 220, A: 255}, {R: 240, G: 220, B: 25, A: 255}, {R: 220, G: 35, B: 20, A: 255}}
	return interpolateStops(stops, t)
}

func residualColor(t float64) color.RGBA {
	t = clamp01(t / 4)
	return color.RGBA{R: 220, G: uint8(165 * (1 - t)), B: uint8(40 * (1 - t)), A: 255}
}

func interpolateStops(stops []color.RGBA, t float64) color.RGBA {
	t = clamp01(t) * float64(len(stops)-1)
	index := int(math.Floor(t))
	if index >= len(stops)-1 {
		return stops[len(stops)-1]
	}
	return blendQA(stops[index], stops[index+1], t-float64(index))
}

func clamp01(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}

func pngEncode(out io.Writer, img image.Image) error {
	return png.Encode(out, img)
}
