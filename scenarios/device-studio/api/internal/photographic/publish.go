package photographic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"device-studio/internal/library"
	"device-studio/internal/physical"
	"device-studio/internal/scenes"
)

// sceneAssemblyProfile versions candidate identity independently from source
// use rights. A change to materials, lights, or contact compositing must create
// a distinct immutable candidate even when the reviewed source and rights are
// unchanged.
const sceneAssemblyProfile = "photographic-scene-assembly-v5-gltf-top-origin-uv-row-major-affine-identity-emission-estimated-light-shadow-source-camera-plate-no-parallax"

// PublishClaims are explicit operator-supplied rights labels. They do not
// authenticate ownership; the accepted source review controls whether local
// rendering is permitted, independently of scene/motion qualification.
type PublishClaims struct {
	LicenseID          string
	Attribution        string
	SourceLabel        string
	AllowLocalRender   bool
	SourceUseRationale string
}

type Publisher interface {
	Publish(context.Context, library.PublishInput) (library.BundleRevision, error)
}

type SceneSaver interface {
	SaveCandidate(context.Context, scenes.SceneCandidate, uint32) (scenes.Revision, error)
}

type PublishedCandidate struct {
	Scene      scenes.Revision
	Library    library.BundleRevision
	Conversion Report
}

// PublishCandidate converts one accepted prepared photo, publishes immutable
// GLB/texture bytes with rights bounded by its accepted source review, then
// saves its SceneCandidate. Local render permission never qualifies motion or
// publication. The same source bundle, rights claims and owner timestamp replay.
func PublishCandidate(ctx context.Context, raw []byte, claims PublishClaims, libraryOwner Publisher, sceneOwner SceneSaver) (PublishedCandidate, error) {
	if strings.TrimSpace(claims.LicenseID) == "" || strings.TrimSpace(claims.Attribution) == "" || strings.TrimSpace(claims.SourceLabel) == "" || len(claims.LicenseID) > 128 || len(claims.Attribution) > 1024 || len(claims.SourceLabel) > 512 || len(claims.SourceUseRationale) > 1024 || strings.ContainsAny(claims.SourceUseRationale, "\x00\r") || (claims.AllowLocalRender && strings.TrimSpace(claims.SourceUseRationale) == "") || (!claims.AllowLocalRender && strings.TrimSpace(claims.SourceUseRationale) != "") {
		return PublishedCandidate{}, errors.New("explicit license, attribution and source label claims are required")
	}
	if libraryOwner == nil || sceneOwner == nil {
		return PublishedCandidate{}, errors.New("canonical library and scene owners are required")
	}
	claims.LicenseID = strings.TrimSpace(claims.LicenseID)
	claims.Attribution = strings.TrimSpace(claims.Attribution)
	claims.SourceLabel = strings.TrimSpace(claims.SourceLabel)
	claims.SourceUseRationale = strings.TrimSpace(claims.SourceUseRationale)
	converted, err := Convert(raw)
	if err != nil {
		return PublishedCandidate{}, err
	}
	bundle, err := openBundle(raw)
	if err != nil {
		return PublishedCandidate{}, err
	}
	if !library.IsValidStableID(bundle.manifest.Review.ReviewedBy) {
		return PublishedCandidate{}, errors.New("accepted source reviewer must have a stable library reviewer id")
	}
	if len(converted.GeometryGLB) == 0 || len(converted.Texture) == 0 {
		return PublishedCandidate{}, errors.New("conversion did not produce both geometry and clean-plate texture")
	}
	claimsBytes, err := json.Marshal(struct {
		SourceBundleSHA256 string `json:"source_bundle_sha256"`
		LicenseID          string `json:"license_id"`
		Attribution        string `json:"attribution"`
		SourceLabel        string `json:"source_label"`
		AllowLocalRender   bool   `json:"allow_local_render"`
		SourceUseRationale string `json:"source_use_rationale,omitempty"`
	}{bundle.sha, claims.LicenseID, claims.Attribution, claims.SourceLabel, claims.AllowLocalRender, claims.SourceUseRationale})
	if err != nil {
		return PublishedCandidate{}, err
	}
	claimSum := sha256.Sum256(claimsBytes)
	claimSHA := hex.EncodeToString(claimSum[:])
	identitySHA := photographicCandidateIdentity(bundle.sha, claimSHA, sceneAssemblyProfile)
	converted.Report.SourceUseClaimSHA256 = claimSHA
	converted.Report.LocalRenderAuthorized = claims.AllowLocalRender
	converted.Report.SourceUseRationale = claims.SourceUseRationale
	adaptationBytes, err := json.Marshal(struct {
		ContractVersion       string  `json:"contract_version"`
		AssemblyProfile       string  `json:"assembly_profile"`
		SourceBundleSHA256    string  `json:"source_bundle_sha256"`
		GeometryArchiveSHA256 string  `json:"geometry_archive_sha256"`
		Model                 string  `json:"model"`
		CheckpointSHA256      string  `json:"checkpoint_sha256"`
		ScaleStatus           string  `json:"scale_status"`
		ScaleEstimateM        float64 `json:"scale_estimate_m"`
		DeviceMaskSHA256      string  `json:"device_mask_sha256"`
		RepairProvenanceSHA   string  `json:"repair_provenance_sha256"`
		RenderMode            string  `json:"render_mode"`
		SourceUseClaimSHA256  string  `json:"source_use_claim_sha256"`
		LocalRenderAuthorized bool    `json:"local_render_authorized"`
	}{converted.Report.ContractVersion, sceneAssemblyProfile, converted.Report.SourceBundleSHA256, converted.Report.GeometryArchiveSHA256, converted.Report.GeometryModel, converted.Report.CheckpointSHA256, converted.Report.ScaleStatus, converted.Report.EstimatedScaleMeters, converted.Report.DeviceMaskSHA256, converted.Report.RepairProvenanceSHA256, "SOURCE_CAMERA_PLATE_V1", claimSHA, claims.AllowLocalRender})
	if err != nil {
		return PublishedCandidate{}, err
	}
	if len(adaptationBytes) > 1024 {
		return PublishedCandidate{}, errors.New("photographic provenance summary exceeds the library adaptation bound")
	}
	artifactRights := []library.Use{library.UseEdit}
	allowedUses := []library.Use{library.UseEdit}
	if claims.AllowLocalRender {
		artifactRights = append(artifactRights, library.UseRender)
		allowedUses = append(allowedUses, library.UseRender)
	}
	bundleID := "photographic-background-" + bundle.sha[:24] + "-" + identitySHA[:16]
	manifest := library.BundleManifest{Version: library.ManifestVersion, BundleID: bundleID, Revision: 1, Artifacts: []library.Artifact{
		{ID: "background-glb", Role: library.RoleGeometryGLB, MIME: "model/gltf-binary", SizeBytes: int64(len(converted.GeometryGLB)), SHA256: digest(converted.GeometryGLB), LicenseID: claims.LicenseID, Attribution: claims.Attribution,
			Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: claims.SourceLabel, SourceSHA256: bundle.sha, Adaptation: string(adaptationBytes)}, Rights: append([]library.Use(nil), artifactRights...)},
		{ID: "clean-plate", Role: library.RoleTextureImage, MIME: bundle.manifest.Background.CleanPlate.MediaType, SizeBytes: int64(len(converted.Texture)), SHA256: digest(converted.Texture), LicenseID: claims.LicenseID, Attribution: claims.Attribution,
			Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: claims.SourceLabel, SourceSHA256: bundle.sha, Adaptation: string(adaptationBytes)}, Rights: append([]library.Use(nil), artifactRights...)},
	}}
	canonical, _, manifestSHA, err := library.CanonicalizeManifest(manifest)
	if err != nil {
		return PublishedCandidate{}, fmt.Errorf("canonicalize photographic library manifest: %w", err)
	}
	review := library.ReviewBinding{ID: photographicReviewBindingID(bundle.sha, identitySHA), ReviewerID: bundle.manifest.Review.ReviewedBy,
		ReviewedAt: time.UnixMilli(bundle.manifest.Review.ReviewedAtUnixMs).UTC(), Decision: "approved", Rationale: claims.SourceUseRationale, ManifestSHA256: manifestSHA, AllowedUses: allowedUses}
	published, err := libraryOwner.Publish(ctx, library.PublishInput{Manifest: canonical, Review: review, Sources: map[string]io.Reader{"background-glb": bytes.NewReader(converted.GeometryGLB), "clean-plate": bytes.NewReader(converted.Texture)}})
	if err != nil {
		return PublishedCandidate{}, fmt.Errorf("publish photographic assets under the declared source-use claims: %w", err)
	}
	geometryRef, textureRef := artifactRef(published, "background-glb"), artifactRef(published, "clean-plate")
	if geometryRef.BundleID == "" || textureRef.BundleID == "" {
		return PublishedCandidate{}, errors.New("library publication omitted expected artifacts")
	}
	candidate, err := buildSceneCandidate(bundle, converted, geometryRef, textureRef)
	if err != nil {
		return PublishedCandidate{}, err
	}
	saved, err := sceneOwner.SaveCandidate(ctx, candidate, 0)
	if err != nil {
		return PublishedCandidate{}, fmt.Errorf("save photographic scene candidate: %w", err)
	}
	return PublishedCandidate{Scene: saved, Library: published, Conversion: converted.Report}, nil
}

func buildSceneCandidate(b *bundle, output Background, geometryRef, textureRef physical.ArtifactRef) (scenes.SceneCandidate, error) {
	m := b.manifest
	k := m.Geometry.Camera.NormalizedIntrinsics
	if math.Abs(k[2]-.5) > 1e-4 || math.Abs(k[5]-.5) > 1e-4 {
		return scenes.SceneCandidate{}, errors.New("SceneCandidate camera contract cannot represent an off-center principal point")
	}
	cameraFrame, err := cameraTRS(m.Geometry.Camera.CameraToWorld)
	if err != nil {
		return scenes.SceneCandidate{}, err
	}
	fx, fy := k[0], k[4]
	if fx <= 0 || fy <= 0 {
		return scenes.SceneCandidate{}, errors.New("camera focal values must be positive")
	}
	sensorWidth := 36.0
	focal := fx * sensorWidth
	sensorHeight := focal / fy
	if !finite([]float64{sensorHeight, focal}) || sensorHeight <= 0 {
		return scenes.SceneCandidate{}, errors.New("camera sensor dimensions are invalid")
	}
	planeNormal := [3]float64{m.Geometry.SupportPlane.Normal[0], -m.Geometry.SupportPlane.Normal[1], -m.Geometry.SupportPlane.Normal[2]}
	planePoint := physical.Vec3{X: -m.Geometry.SupportPlane.Offset * planeNormal[0], Y: -m.Geometry.SupportPlane.Offset * planeNormal[1], Z: -m.Geometry.SupportPlane.Offset * planeNormal[2]}
	planeRotation, err := planeFrameRotation(planeNormal)
	if err != nil {
		return scenes.SceneCandidate{}, err
	}
	// This is an authored, bounded local-render light guess based on the
	// visible left-side window in the source plate. It is deliberately named
	// as an estimate; monocular reconstruction does not recover illumination.
	photoLightRotation := multiplyQuaternion(cameraFrame.Rotation, multiplyQuaternion(planeRotation, physical.QuatXYZW{X: -math.Sqrt(.5), W: math.Sqrt(.5)}))
	cameraRight := rotateVector(cameraFrame.Rotation, [3]float64{1, 0, 0})
	worldPlaneNormal := rotateVector(cameraFrame.Rotation, planeNormal)
	worldPlanePointOffset := rotateVector(cameraFrame.Rotation, [3]float64{planePoint.X, planePoint.Y, planePoint.Z})
	worldPlanePoint := [3]float64{cameraFrame.Translation.X + worldPlanePointOffset[0], cameraFrame.Translation.Y + worldPlanePointOffset[1], cameraFrame.Translation.Z + worldPlanePointOffset[2]}
	photoLightPosition := physical.Vec3{
		X: worldPlanePoint[0] + worldPlaneNormal[0]*1.1 - cameraRight[0]*.75,
		Y: worldPlanePoint[1] + worldPlaneNormal[1]*1.1 - cameraRight[1]*.75,
		Z: worldPlanePoint[2] + worldPlaneNormal[2]*1.1 - cameraRight[2]*.75,
	}
	photo := &scenes.PhotographicBackground{
		ContractVersion: output.Report.ContractVersion, Route: output.Report.Route, SourceBundleSHA256: output.Report.SourceBundleSHA256, SourceRasterSHA256: output.Report.SourceRasterSHA256,
		GeometryArchiveSHA256: output.Report.GeometryArchiveSHA256, RepairProvenanceSHA256: output.Report.RepairProvenanceSHA256,
		DeviceMaskSHA256: output.Report.DeviceMaskSHA256, RemovalMaskSHA256: output.Report.RemovalMaskSHA256,
		ScaleStatus: output.Report.ScaleStatus, EstimatedScaleMeters: output.Report.EstimatedScaleMeters,
		GeometryModel: output.Report.GeometryModel, CheckpointSHA256: output.Report.CheckpointSHA256, Postprocess: output.Report.Postprocess,
		RenderMode:         "SOURCE_CAMERA_PLATE_V1",
		SupportPlaneNormal: output.Report.SupportPlaneNormal, SupportPlaneOffsetM: output.Report.SupportPlaneOffsetM,
		SupportPlaneInliers: output.Report.SupportPlaneInliers, SupportPlaneResidual95: output.Report.SupportPlaneResidual95,
		CameraEnvelopeTranslationM: m.CameraEnvelope.MaxTranslationM, CameraEnvelopeRotationDegrees: m.CameraEnvelope.MaxRotationDegrees,
		OverscanReviewed: m.CameraEnvelope.OverscanReviewed, SourceReviewSHA256: output.Report.SourceReviewSHA256,
		SourceReviewer: output.Report.SourceReviewer, SourceReviewRationale: output.Report.SourceReviewRationale,
		SourceReviewEvidence: append([]string(nil), output.Report.SourceReviewEvidence...),
		SourceUseClaimSHA256: output.Report.SourceUseClaimSHA256, LocalRenderAuthorized: output.Report.LocalRenderAuthorized,
		SourceUseRationale: output.Report.SourceUseRationale,
	}
	identity := physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	return scenes.SceneCandidate{
		Version: scenes.CandidateVersion, SceneID: "photographic-background-" + output.Report.SourceBundleSHA256[:24] + "-" + photographicCandidateIdentity(output.Report.SourceBundleSHA256, output.Report.SourceUseClaimSHA256, sceneAssemblyProfile)[:16], CoordinateConvention: physical.CoordinateConvention,
		Artifacts: []scenes.SceneArtifactBinding{
			{ID: "background-geometry", Ref: geometryRef, SourceUnit: "m", SourceToCanonical: &identity},
			{ID: "background-texture", Ref: textureRef},
		},
		// The reconstructed plate carries photographed appearance, not a
		// relightable PBR estimate. Emission preserves those source pixels;
		// device lighting and contact shadow are composed in the render owner.
		Materials: []scenes.SceneMaterial{{ID: "clean-plate-material", SurfaceProfile: scenes.MaterialEmissionV1, BaseColorLinearRGBA: [4]float64{1, 1, 1, 1}, Roughness: 1, AlphaMode: scenes.AlphaOpaque,
			Inputs: []scenes.SceneMaterialInput{{Socket: scenes.SocketBaseColor, TextureBindingID: "background-texture", Semantic: "base_color", ColorSpace: scenes.ColorSpaceSRGB, Channel: scenes.ChannelRGB, Sampler: scenes.SamplerClamp, Coordinates: scenes.CoordinatesUVSet, UVSet: "TEXCOORD_0", AffineUV: &[6]float64{1, 0, 0, 0, 1, 0}}}}},
		MaterialSlots: []scenes.SceneMaterialSlot{{PartID: "background-surface", PrimitiveIndex: 0, SlotIndex: 0, MaterialID: "clean-plate-material"}},
		Parts:         []scenes.ScenePart{{ID: "background-surface", Role: "photographic_background", ArtifactBindingID: "background-geometry", NodeIndex: 0, NodeName: "ReconstructedPhotographicBackground", LocalFrame: cameraFrame}},
		SupportFrames: []scenes.SceneSupportFrame{{ID: "photographic-support-plane", InterfaceID: "horizontal_surface", FramePartID: "background-surface", LocalFrame: physical.TRS{Translation: planePoint, Rotation: planeRotation, Scale: identity.Scale}, ContactRegionPartID: "background-surface", LoadDirectionLocal: &physical.Vec3{Y: -1}}},
		CameraFrames:  []scenes.SceneCameraFrame{{ID: "source-camera", Frame: cameraFrame, Projection: scenes.ProjectionPerspective, FocalLengthMM: &focal, SensorWidthMM: &sensorWidth, SensorHeightMM: &sensorHeight, SensorFit: scenes.SensorFitAuto, NearClipM: .001, FarClipM: 1000}},
		LightFrames:   []scenes.SceneLightFrame{{ID: "estimated-left-window-area-v1", Frame: physical.TRS{Translation: photoLightPosition, Rotation: photoLightRotation, Scale: identity.Scale}, LocalAxisProfile: "AREA_XY_MINUS_Z_V1", Shape: scenes.LightRectangle, DimensionsM: [2]float64{.8, .55}, PowerWatts: 220, LinearRGB: [3]float64{1, .94, .84}}},
		Environment:   scenes.SceneEnvironment{Model: "CONSTANT_V1", LinearRGB: [3]float64{0, 0, 0}, Strength: 0}, Photograph: photo,
	}, nil
}

func multiplyQuaternion(a, b physical.QuatXYZW) physical.QuatXYZW {
	return physical.QuatXYZW{
		X: a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y,
		Y: a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X,
		Z: a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W,
		W: a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z,
	}
}

func rotateVector(q physical.QuatXYZW, v [3]float64) [3]float64 {
	// q is normalized by cameraTRS/planeFrameRotation before use.
	x, y, z := q.X, q.Y, q.Z
	w := q.W
	return [3]float64{
		(1-2*(y*y+z*z))*v[0] + 2*(x*y-z*w)*v[1] + 2*(x*z+y*w)*v[2],
		2*(x*y+z*w)*v[0] + (1-2*(x*x+z*z))*v[1] + 2*(y*z-x*w)*v[2],
		2*(x*z-y*w)*v[0] + 2*(y*z+x*w)*v[1] + (1-2*(x*x+y*y))*v[2],
	}
}

func photographicCandidateIdentity(sourceBundleSHA256, sourceUseClaimSHA256, profile string) string {
	identity, _ := json.Marshal(struct {
		SourceBundleSHA256 string `json:"source_bundle_sha256"`
		SourceUseClaimSHA  string `json:"source_use_claim_sha256"`
		AssemblyProfile    string `json:"assembly_profile"`
	}{sourceBundleSHA256, sourceUseClaimSHA256, profile})
	sum := sha256.Sum256(identity)
	return hex.EncodeToString(sum[:])
}

func photographicReviewBindingID(sourceBundleSHA256, candidateIdentitySHA256 string) string {
	return "photo-review-" + sourceBundleSHA256[:24] + "-" + candidateIdentitySHA256[:16]
}

func artifactRef(revision library.BundleRevision, id string) physical.ArtifactRef {
	for _, artifact := range revision.Manifest.Artifacts {
		if artifact.ID == id {
			return physical.ArtifactRef{BundleID: revision.Manifest.BundleID, BundleRevision: revision.Manifest.Revision, ManifestSHA256: revision.ManifestSHA256, ReviewSHA256: revision.ReviewSHA256,
				ArtifactID: artifact.ID, Role: artifact.Role, SizeBytes: artifact.SizeBytes, SHA256: artifact.SHA256}
		}
	}
	return physical.ArtifactRef{}
}

func cameraTRS(m []float64) (physical.TRS, error) {
	if !rigid(m) {
		return physical.TRS{}, errors.New("camera-to-world matrix must be rigid")
	}
	// Convert the column-major 3x3 camera rotation to a canonical quaternion.
	r00, r01, r02 := m[0], m[4], m[8]
	r10, r11, r12 := m[1], m[5], m[9]
	r20, r21, r22 := m[2], m[6], m[10]
	trace := r00 + r11 + r22
	var q physical.QuatXYZW
	switch {
	case trace > 0:
		s := math.Sqrt(trace+1) * 2
		q = physical.QuatXYZW{W: .25 * s, X: (r21 - r12) / s, Y: (r02 - r20) / s, Z: (r10 - r01) / s}
	case r00 > r11 && r00 > r22:
		s := math.Sqrt(1+r00-r11-r22) * 2
		q = physical.QuatXYZW{W: (r21 - r12) / s, X: .25 * s, Y: (r01 + r10) / s, Z: (r02 + r20) / s}
	case r11 > r22:
		s := math.Sqrt(1+r11-r00-r22) * 2
		q = physical.QuatXYZW{W: (r02 - r20) / s, X: (r01 + r10) / s, Y: .25 * s, Z: (r12 + r21) / s}
	default:
		s := math.Sqrt(1+r22-r00-r11) * 2
		q = physical.QuatXYZW{W: (r10 - r01) / s, X: (r02 + r20) / s, Y: (r12 + r21) / s, Z: .25 * s}
	}
	q, err := physical.CanonicalizeQuaternion(q)
	if err != nil {
		return physical.TRS{}, err
	}
	return physical.TRS{Translation: physical.Vec3{X: m[12], Y: m[13], Z: m[14]}, Rotation: q, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, nil
}

func planeFrameRotation(normal [3]float64) (physical.QuatXYZW, error) {
	// Choose camera-right projected onto the table. The resulting local +Y is
	// the stable support normal and local axes form a right-handed frame.
	x := [3]float64{1, 0, 0}
	dot := x[0]*normal[0] + x[1]*normal[1] + x[2]*normal[2]
	for i := range x {
		x[i] -= dot * normal[i]
	}
	norm := math.Sqrt(x[0]*x[0] + x[1]*x[1] + x[2]*x[2])
	if norm < 1e-8 {
		return physical.QuatXYZW{}, errors.New("support plane cannot define a stable local tangent")
	}
	for i := range x {
		x[i] /= norm
	}
	y := normal
	z := [3]float64{x[1]*y[2] - x[2]*y[1], x[2]*y[0] - x[0]*y[2], x[0]*y[1] - x[1]*y[0]}
	// Column vectors map local X/Y/Z to the chosen tangent, normal, bitangent.
	trs, err := cameraTRS([]float64{x[0], x[1], x[2], 0, y[0], y[1], y[2], 0, z[0], z[1], z[2], 0, 0, 0, 0, 1})
	return trs.Rotation, err
}
