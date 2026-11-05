// Package sceneprep owns the immutable photographic scene exchange. Inference
// jobs and Device Studio revisions retain their own execution/state ownership.
package sceneprep

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	"google.golang.org/protobuf/proto"
)

const (
	ContractVersion         = "hybrid-device-scene/v1"
	CameraBasis             = "opencv-camera-x-right-y-down-z-forward-v1"
	WorldBasis              = "device-studio-right-handed-meters-x-right-y-up-z-front-v1"
	MaxBundleBytes   int64  = 256 << 20
	MaxExpandedBytes uint64 = 192 << 20
	MaxManifestBytes int64  = 64 << 10
)

// Bundle has verified member sizes/digests. It never extracts paths to disk.
// The caller owns the ReaderAt's lifetime; OpenArtifact readers must be closed.
type Bundle struct {
	Manifest *preparationv1.PreparedPhotograph
	members  map[string]*zip.File
}

func Open(reader io.ReaderAt, size int64) (*Bundle, error) {
	if size <= 0 || size > MaxBundleBytes {
		return nil, fmt.Errorf("scene preparation bundle must be 1..%d bytes", MaxBundleBytes)
	}
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("scene preparation ZIP: %w", err)
	}
	members := map[string]*zip.File{}
	var expanded uint64
	for _, member := range archive.File {
		if len(members) >= 17 || !portablePath(member.Name) || member.FileInfo().IsDir() || !member.Mode().IsRegular() {
			return nil, fmt.Errorf("invalid scene member %q", member.Name)
		}
		if _, exists := members[member.Name]; exists {
			return nil, fmt.Errorf("duplicate scene member %q", member.Name)
		}
		if member.UncompressedSize64 > MaxExpandedBytes-expanded {
			return nil, fmt.Errorf("scene members exceed expanded byte limit")
		}
		expanded += member.UncompressedSize64
		members[member.Name] = member
	}
	manifestFile := members["manifest.json"]
	if manifestFile == nil || manifestFile.UncompressedSize64 > uint64(MaxManifestBytes) {
		return nil, fmt.Errorf("scene manifest missing or oversized")
	}
	stream, err := manifestFile.Open()
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(stream, MaxManifestBytes+1))
	closeErr := stream.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	manifest := &preparationv1.PreparedPhotograph{}
	if err := protojson.Unmarshal(raw, manifest); err != nil {
		return nil, fmt.Errorf("scene manifest: %w", err)
	}
	artifacts, err := Validate(manifest)
	if err != nil {
		return nil, err
	}
	if len(members) != len(artifacts)+1 {
		return nil, fmt.Errorf("scene ZIP contains undeclared or missing members")
	}
	bundle := &Bundle{Manifest: manifest, members: members}
	for name, artifact := range artifacts {
		member := members[name]
		if member == nil || member.UncompressedSize64 != uint64(artifact.SizeBytes) {
			return nil, fmt.Errorf("scene artifact %q size/member mismatch", name)
		}
		stream, err := member.Open()
		if err != nil {
			return nil, err
		}
		digest := sha256.New()
		count, readErr := io.Copy(digest, io.LimitReader(stream, artifact.SizeBytes+1))
		closeErr := stream.Close()
		if readErr != nil {
			return nil, fmt.Errorf("scene artifact %q: %w", name, readErr)
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if count != artifact.SizeBytes || hex.EncodeToString(digest.Sum(nil)) != artifact.Sha256 {
			return nil, fmt.Errorf("scene artifact %q hash mismatch", name)
		}
	}
	source := manifest.Source
	for _, raster := range []struct {
		artifact      *preparationv1.Artifact
		width, height uint32
	}{
		{source.Original, source.OriginalWidth, source.OriginalHeight},
		{source.Oriented, source.Width, source.Height},
		{manifest.Background.CleanPlate, source.Width, source.Height},
		{manifest.Background.DeviceMask, source.Width, source.Height},
		{manifest.Background.RemovalMask, source.Width, source.Height},
	} {
		if raster.artifact != nil {
			if err := bundle.validateRaster(raster.artifact, raster.width, raster.height); err != nil {
				return nil, err
			}
		}
	}
	for _, mask := range manifest.Background.OccluderMasks {
		if err := bundle.validateRaster(mask, source.Width, source.Height); err != nil {
			return nil, err
		}
	}
	return bundle, nil
}

func (b *Bundle) OpenArtifact(name string) (io.ReadCloser, error) {
	member := b.members[name]
	if member == nil || name == "manifest.json" {
		return nil, fmt.Errorf("unknown scene artifact %q", name)
	}
	return member.Open()
}

func (b *Bundle) validateRaster(artifact *preparationv1.Artifact, width, height uint32) error {
	stream, err := b.OpenArtifact(artifact.Path)
	if err != nil {
		return err
	}
	defer stream.Close()
	config, format, err := image.DecodeConfig(stream)
	if err != nil || config.Width != int(width) || config.Height != int(height) {
		return fmt.Errorf("scene raster %q does not match its declared dimensions", artifact.Path)
	}
	expected := map[string]string{"png": "image/png", "jpeg": "image/jpeg"}[format]
	if expected == "" || expected != artifact.MediaType {
		return fmt.Errorf("scene raster %q media type disagrees with decoded format", artifact.Path)
	}
	return nil
}

// Validate checks the typed contract; Open additionally verifies actual bytes.
// Geometry-array/model consistency is still enforced by the geometry owner's
// archive validator when converting this bundle to a physical scene.
func Validate(m *preparationv1.PreparedPhotograph) (map[string]*preparationv1.Artifact, error) {
	if m == nil || m.ContractVersion != ContractVersion {
		return nil, fmt.Errorf("unsupported prepared scene contract")
	}
	if m.Route != preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_EMPTY_SCENE && m.Route != preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL {
		return nil, fmt.Errorf("a photographic source route is required")
	}
	s, g, b := m.Source, m.Geometry, m.Background
	if s == nil || g == nil || b == nil || g.Camera == nil || g.SupportPlane == nil || s.Original == nil || s.Oriented == nil || g.Archive == nil || b.CleanPlate == nil {
		return nil, fmt.Errorf("source, geometry, camera, support plane and clean plate are required")
	}
	if !rasterSize(s.OriginalWidth, s.OriginalHeight) || !rasterSize(s.Width, s.Height) {
		return nil, fmt.Errorf("source raster exceeds 16 million pixels or has empty dimensions")
	}
	orientation := s.OriginalExifOrientation
	expected, width, height := orientationMap(s.OriginalWidth, s.OriginalHeight, orientation)
	if expected == nil || width != s.Width || height != s.Height || !equalNumbers(s.OriginalToOriented, expected, 0) {
		return nil, fmt.Errorf("source EXIF transform and oriented raster disagree")
	}
	camera := g.Camera
	k := camera.NormalizedIntrinsics
	if camera.Basis != CameraBasis || camera.WorldBasis != WorldBasis || !rasterSize(camera.RasterWidth, camera.RasterHeight) || len(k) != 9 || !finite(k) || k[0] <= 0 || k[4] <= 0 || !equalNumbers([]float64{k[1], k[3], k[6], k[7], k[8]}, []float64{0, 0, 0, 0, 1}, 1e-9) || !rigid(camera.CameraToWorld) {
		return nil, fmt.Errorf("camera requires normalized pinhole intrinsics, explicit bases and a rigid camera-to-world transform")
	}
	if m.ScaleStatus != "estimated_monocular_scale_not_measured" {
		return nil, fmt.Errorf("photographic geometry scale must remain an explicit monocular estimate")
	}
	plane := g.SupportPlane
	if len(plane.Normal) != 3 || !finite(append(append([]float64{}, plane.Normal...), plane.Offset, plane.ResidualP50, plane.ResidualP95)) || math.Abs(dot(plane.Normal, plane.Normal)-1) > 1e-5 || plane.ValidPixelCount == 0 || plane.InlierCount < 3 || plane.InlierCount > plane.ValidPixelCount || plane.ResidualP50 < 0 || plane.ResidualP95 < plane.ResidualP50 {
		return nil, fmt.Errorf("support plane requires finite unit normal, inlier support and ordered residuals")
	}
	envelope := m.CameraEnvelope
	if envelope == nil || !finite([]float64{envelope.MaxTranslationM, envelope.MaxRotationDegrees}) || envelope.MaxTranslationM < 0 || envelope.MaxTranslationM > .25 || envelope.MaxRotationDegrees < 0 || envelope.MaxRotationDegrees > 10 {
		return nil, fmt.Errorf("photographic camera envelope must stay within 0.25 metres and 10 degrees")
	}
	if len(b.OccluderMasks) > 8 {
		return nil, fmt.Errorf("at most eight occluder masks are supported")
	}
	if m.Route == preparationv1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL && (b.DeviceMask == nil || b.RemovalMask == nil || b.RepairProvenance == nil) {
		return nil, fmt.Errorf("device removal requires distinct region/removal masks and retained repair provenance")
	}
	review := m.Review
	if review == nil || (review.Decision != "pending" && review.Decision != "rejected" && review.Decision != "accepted") || len(review.EvidenceRefs) > 16 || len(review.Rationale) > 2048 || len(review.ReviewedBy) > 128 {
		return nil, fmt.Errorf("bounded explicit visual review state is required")
	}
	if review.Decision == "pending" && review.ReviewedAtUnixMs != 0 || review.Decision != "pending" && review.ReviewedAtUnixMs <= 0 {
		return nil, fmt.Errorf("completed visual review requires an owner-assigned timestamp and pending review must not have one")
	}
	if review.Decision != "pending" && (strings.TrimSpace(review.ReviewedBy) == "" || strings.TrimSpace(review.Rationale) == "" || len(review.EvidenceRefs) == 0) {
		return nil, fmt.Errorf("completed visual review requires attribution, rationale and retained evidence")
	}
	for _, ref := range review.EvidenceRefs {
		if strings.TrimSpace(ref) == "" || len(ref) > 512 {
			return nil, fmt.Errorf("invalid visual review evidence reference")
		}
	}
	artifacts := map[string]*preparationv1.Artifact{}
	for _, a := range append([]*preparationv1.Artifact{s.Original, s.Oriented, g.Archive, b.CleanPlate, b.DeviceMask, b.RemovalMask, b.RepairProvenance}, b.OccluderMasks...) {
		if a == nil {
			continue
		}
		if !portablePath(a.Path) || a.Path == "manifest.json" || a.SizeBytes <= 0 || a.SizeBytes > int64(MaxExpandedBytes) || !validSHA(a.Sha256) || len(a.MediaType) > 128 || a.MediaType == "" {
			return nil, fmt.Errorf("invalid scene artifact identity")
		}
		if prior, ok := artifacts[a.Path]; ok && !proto.Equal(prior, a) {
			return nil, fmt.Errorf("conflicting scene artifact %q", a.Path)
		}
		artifacts[a.Path] = a
	}
	if g.Archive.MediaType != "application/zip" {
		return nil, fmt.Errorf("geometry must retain its original ZIP archive")
	}
	if b.RepairProvenance != nil && b.RepairProvenance.MediaType != "application/json" {
		return nil, fmt.Errorf("repair provenance must be JSON")
	}
	return artifacts, nil
}

// ValidateForRender is stricter than importing a pending/rejected draft.
func ValidateForRender(m *preparationv1.PreparedPhotograph) error {
	if _, err := Validate(m); err != nil {
		return err
	}
	if m.Review.Decision != "accepted" {
		return fmt.Errorf("prepared photograph has no accepted visual review")
	}
	if (m.CameraEnvelope.MaxTranslationM > 0 || m.CameraEnvelope.MaxRotationDegrees > 0) && !m.CameraEnvelope.OverscanReviewed {
		return fmt.Errorf("camera movement requires reviewed overscan coverage")
	}
	return nil
}

func portablePath(name string) bool {
	return name != "" && len(name) <= 128 && !strings.ContainsAny(name, "\\:\x00") && !strings.HasPrefix(name, "/") && path.Clean(name) == name && name != "." && name != ".." && !strings.HasPrefix(name, "../")
}
func validSHA(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}
func rasterSize(w, h uint32) bool { return w > 0 && h > 0 && uint64(w)*uint64(h) <= 16_000_000 }
func finite(values []float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
func equalNumbers(a, b []float64, tolerance float64) bool {
	if len(a) != len(b) || !finite(a) || !finite(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > tolerance {
			return false
		}
	}
	return true
}
func dot(a, b []float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func rigid(m []float64) bool {
	if len(m) != 16 || !finite(m) || !equalNumbers([]float64{m[3], m[7], m[11], m[15]}, []float64{0, 0, 0, 1}, 1e-8) {
		return false
	}
	x, y, z := m[:3], m[4:7], m[8:11]
	if math.Abs(dot(x, x)-1) > 1e-6 || math.Abs(dot(y, y)-1) > 1e-6 || math.Abs(dot(z, z)-1) > 1e-6 || math.Abs(dot(x, y)) > 1e-6 || math.Abs(dot(x, z)) > 1e-6 || math.Abs(dot(y, z)) > 1e-6 {
		return false
	}
	cross := []float64{x[1]*y[2] - x[2]*y[1], x[2]*y[0] - x[0]*y[2], x[0]*y[1] - x[1]*y[0]}
	return math.Abs(dot(cross, z)-1) < 1e-6
}
func orientationMap(w, h uint32, o uint32) ([]float64, uint32, uint32) {
	width, height := float64(w), float64(h)
	transforms := [][]float64{nil, {1, 0, 0, 0, 1, 0, 0, 0, 1}, {-1, 0, width, 0, 1, 0, 0, 0, 1}, {-1, 0, width, 0, -1, height, 0, 0, 1}, {1, 0, 0, 0, -1, height, 0, 0, 1}, {0, 1, 0, 1, 0, 0, 0, 0, 1}, {0, -1, height, 1, 0, 0, 0, 0, 1}, {0, -1, height, -1, 0, width, 0, 0, 1}, {0, 1, 0, -1, 0, width, 0, 0, 1}}
	if o < 1 || o > 8 {
		return nil, 0, 0
	}
	if o >= 5 {
		return transforms[o], h, w
	}
	return transforms[o], w, h
}

// DecodeManifest validates a standalone manifest without asserting artifact bytes.
func DecodeManifest(raw []byte) (*preparationv1.PreparedPhotograph, error) {
	if len(raw) > int(MaxManifestBytes) {
		return nil, fmt.Errorf("prepared scene manifest exceeds byte limit")
	}
	result := &preparationv1.PreparedPhotograph{}
	if err := protojson.Unmarshal(bytes.TrimSpace(raw), result); err != nil {
		return nil, err
	}
	_, err := Validate(result)
	return result, err
}
