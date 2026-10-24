package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"math"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

const (
	ipadLandscapeBundleID   = "device-studio-starter-ipad-mini-6-datsketch-landscape-quarter-turn-v1"
	ipadLandscapeArtifactID = "datsketch-ipad-mini-6-neutral-uv-landscape-quarter-turn-v1-glb"
	ipadLandscapeDeviceID   = "datsketch-ipad-mini-6-neutral-uv-landscape-quarter-turn-v1"
	ipadLandscapeSceneID    = "ipad-study-room-no-cup-landscape-quarter-turn-v1"
	ipadLandscapeCameraID   = "ipad-study-camera-no-cup-landscape-quarter-turn-v1"
	ipadLandscapePresetID   = "ipad-landscape-no-cup-quarter-turn-review-still-v1"
	ipadLandscapeGLBSHA256  = "3d69757aa94ff7b1e7abe3ef180f233cd263228fcfc72bec999fa5f946ac6b19"
)

const (
	ipadLandscapeV2BundleID   = "device-studio-starter-ipad-mini-6-datsketch-landscape-quarter-turn-v2"
	ipadLandscapeV2ArtifactID = "datsketch-ipad-mini-6-neutral-uv-landscape-quarter-turn-v2-glb"
	ipadLandscapeV2DeviceID   = "datsketch-ipad-mini-6-neutral-uv-landscape-quarter-turn-v2"
	ipadLandscapeV2SceneID    = "ipad-study-room-no-cup-landscape-quarter-turn-v2"
	ipadLandscapeV2CameraID   = "ipad-study-camera-no-cup-landscape-quarter-turn-v2"
	ipadLandscapeV2PresetID   = "ipad-landscape-no-cup-quarter-turn-review-still-v2"
	ipadLandscapeV2GLBSHA256  = "1b13a2d730add6f45328a0518fcdd804ea69c5dfa611d2310468534c20eb6255"
)

type ipadLandscapeVariant struct {
	bundleID, artifactID, deviceID, sceneID, cameraID, presetID, glbSHA string
	derive                                                              func([]byte, string, int) ([]byte, error)
	assetName, qualificationID, description, adaptation                 string
}

var ipadLandscapeV1 = ipadLandscapeVariant{
	bundleID: ipadLandscapeBundleID, artifactID: ipadLandscapeArtifactID, deviceID: ipadLandscapeDeviceID,
	sceneID: ipadLandscapeSceneID, cameraID: ipadLandscapeCameraID, presetID: ipadLandscapePresetID, glbSHA: ipadLandscapeGLBSHA256,
	derive: rotateScreenUVQuarterTurn, assetName: "ipad-mini-landscape-uv-quarter-v1.glb", qualificationID: "device-studio-e4-ipad-landscape-quarter-turn-v1",
	description: "quarter-turned display UVs", adaptation: "map (u,v) to (v,1-u)",
}

var ipadLandscapeV2 = ipadLandscapeVariant{
	bundleID: ipadLandscapeV2BundleID, artifactID: ipadLandscapeV2ArtifactID, deviceID: ipadLandscapeV2DeviceID,
	sceneID: ipadLandscapeV2SceneID, cameraID: ipadLandscapeV2CameraID, presetID: ipadLandscapeV2PresetID, glbSHA: ipadLandscapeV2GLBSHA256,
	derive: rotateScreenUVQuarterTurnCounterClockwise, assetName: "ipad-mini-landscape-uv-quarter-v2.glb", qualificationID: "device-studio-e4-ipad-landscape-quarter-turn-v2",
	description: "inverse quarter-turned display UVs, corrected after native orientation review", adaptation: "map (u,v) to (1-v,u)",
}

// InitializeReviewedIPadLandscapeStarter creates a versioned pose candidate.
// The source mesh and dimensions stay unchanged; only the active display UVs,
// display axes, and root rotation around the screen normal change.
func (o Owners) InitializeReviewedIPadLandscapeStarter(ctx context.Context) (IPadStarterState, error) {
	return o.initializeReviewedIPadLandscapeStarter(ctx, ipadLandscapeV1)
}

// InitializeReviewedIPadLandscapeV2Starter keeps the failed v1 source and
// starter immutable while publishing the corrected inverse UV composition.
func (o Owners) InitializeReviewedIPadLandscapeV2Starter(ctx context.Context) (IPadStarterState, error) {
	return o.initializeReviewedIPadLandscapeStarter(ctx, ipadLandscapeV2)
}

func (o Owners) initializeReviewedIPadLandscapeStarter(ctx context.Context, variant ipadLandscapeVariant) (IPadStarterState, error) {
	if o.Starter == nil {
		return IPadStarterState{}, errors.New("iPad landscape candidate initializer is unavailable")
	}
	base, err := o.InitializeReviewedIPadStarter(ctx)
	if err != nil {
		return IPadStarterState{}, err
	}
	laptop, err := o.InitializeReviewedLaptopStarter(ctx)
	if err != nil {
		return IPadStarterState{}, err
	}
	source, err := o.Starter.readPinnedIPad(base.Device.Candidate.Geometry)
	if err != nil {
		return IPadStarterState{}, err
	}
	data, err := variant.derive(source, "iPad Mini_Screen_0", 0)
	if err != nil {
		return IPadStarterState{}, fmt.Errorf("derive landscape iPad display UVs: %w", err)
	}
	if len(data) != 1939668 || variant.glbSHA == "" || sha256Hex(data) != variant.glbSHA {
		return IPadStarterState{}, fmt.Errorf("landscape iPad GLB differs from pinned %s derivation (actual SHA %s)", variant.deviceID, sha256Hex(data))
	}
	manifest, sources, err := o.Starter.buildManifest(ctx, variant.bundleID, []starterFile{{
		ID: variant.artifactID, File: "assets/ipad/" + variant.assetName, Data: data, MIME: "model/gltf-binary", SHA: variant.glbSHA,
		Role: library.RoleGeometryGLB, License: "CC-BY-4.0",
		Attribution: "iPad mini 6 2021 model by DatSketch, adapted under CC BY 4.0; neutral screen and display UV quarter-turn. Geometry remains unchanged.",
		Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "DatSketch neutral-screen iPad mini 6 UV-normalized candidate.", SourceSHA256: ipadDerivativeSHA256,
			Adaptation: "Append a private TEXCOORD_0 view for iPad Mini_Screen_0 primitive 0 and " + variant.adaptation + ". Preserve source positions, normals, indices, hierarchy, materials, images, transforms, and existing BIN bytes."},
	}})
	if err != nil {
		return IPadStarterState{}, err
	}
	bundle, err := o.Starter.publish(ctx, manifest, sources, variant.qualificationID)
	if err != nil {
		return IPadStarterState{}, err
	}
	ref := artifactRef(bundle, variant.artifactID)
	candidate := cloneIPadCandidate(base.Device.Candidate)
	candidate.Definition.Dimensions = cloneDimensions(candidate.Definition.Dimensions)
	candidate.Definition.Displays = append([]devices.DisplaySurface(nil), candidate.Definition.Displays...)
	if len(candidate.Definition.Displays) != 1 || candidate.Definition.Displays[0].CoordinateMetric == nil {
		return IPadStarterState{}, errors.New("landscape iPad source display metric is incomplete")
	}
	metric := *candidate.Definition.Displays[0].CoordinateMetric
	candidate.Definition.Displays[0].CoordinateMetric = &metric
	candidate.Geometry = ref
	candidate.Definition.ID = variant.deviceID
	candidate.Definition.Variant = "neutral-screen source-preserving landscape pose with " + variant.description
	candidate.Definition.Description = "DatSketch iPad mini 6 source model in a versioned landscape pose. The mesh remains unchanged; " + variant.description + " and tabletop root orientation are candidate-only. App pixels depend on the landscape source capture."
	candidate.Definition.ModelCandidates[0].ID = variant.deviceID
	candidate.Definition.ModelCandidates[0].SHA256 = ref.SHA256
	candidate.Definition.ModelCandidates[0].Attribution = "iPad mini 6 2021 by DatSketch, adapted under CC BY 4.0; neutral display and UV quarter-turn for landscape pose."
	candidate.Definition.Displays[0].CoordinateMetric.WidthMM, candidate.Definition.Displays[0].CoordinateMetric.HeightMM = base.Device.Candidate.Definition.Displays[0].CoordinateMetric.HeightMM, base.Device.Candidate.Definition.Displays[0].CoordinateMetric.WidthMM
	candidate.Definition.Dimensions["landscape_pose"] = devices.Dimensions{
		Width:  base.Device.Candidate.Definition.Dimensions["portrait_unrotated"].Height,
		Height: base.Device.Candidate.Definition.Dimensions["portrait_unrotated"].Width,
		Depth:  base.Device.Candidate.Definition.Dimensions["portrait_unrotated"].Depth,
	}
	candidate.Definition.Provenance = append(candidate.Definition.Provenance, devices.Evidence{Source: variant.assetName + "; " + variant.glbSHA, Kind: "display_uv_and_pose_derivative", Qualifier: "only the active screen TEXCOORD_0 is remapped; the root is rotated 90 degrees around the screen normal; source geometry is unchanged; candidate-only"})
	device, err := saveIPadCandidate(ctx, o.Devices, candidate)
	if err != nil {
		return IPadStarterState{}, err
	}
	placement := base.Placement
	placement.ProfileID = "device-studio-reviewed-ipad-mini-landscape-candidate-" + variant.deviceID
	placement.ProfileVersion = 1
	placement.SceneSupportID = laptop.FramedPlacement.SceneSupportID
	placement.SceneSupportWorld = laptop.FramedPlacement.SceneSupportWorld
	placement.SelectedCameraID = variant.cameraID
	placement.DeviceRootRest, err = landscapeRootRest(source, base.Device.Candidate.SourceToCanonical, placement)
	if err != nil {
		return IPadStarterState{}, err
	}
	sceneCandidate := laptop.FramedScene.Candidate
	sceneCandidate.SceneID = variant.sceneID
	sceneCandidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), sceneCandidate.CameraFrames...)
	if len(sceneCandidate.CameraFrames) != 1 {
		return IPadStarterState{}, errors.New("cup-free landscape scene requires one framed camera")
	}
	camera := sceneCandidate.CameraFrames[0]
	camera.ID = variant.cameraID
	camera.FocalLengthMM = floatPtr(85)
	camera.SensorFit = scenes.SensorFitHorizontal
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusSelectedDevice}
	inventory, err := rendering.InspectPhysicalGLB(source)
	if err != nil {
		return IPadStarterState{}, err
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return IPadStarterState{}, err
	}
	camera.Frame, err = fitIPadPerspectiveCamera(bounds, candidate.SourceToCanonical, placement, camera, 16.0/9.0, .16)
	if err != nil {
		return IPadStarterState{}, fmt.Errorf("fit landscape iPad camera to cup-free setup: %w", err)
	}
	sceneCandidate.CameraFrames[0] = camera
	scene, err := saveSceneCandidateIfUnchanged(ctx, o.Scenes, sceneCandidate)
	if err != nil {
		return IPadStarterState{}, err
	}
	preset, err := o.Starter.ensureIPadLandscapePreset(ctx, variant.presetID, device, scene, base.Motion, placement)
	if err != nil {
		return IPadStarterState{}, err
	}
	return IPadStarterState{Bundle: bundle, Device: device, Scene: scene, Motion: base.Motion, Placement: placement, Preset: preset}, nil
}

func landscapeRootRest(source []byte, sourceToCanonical physical.TRS, placement ReviewedIPadPlacement) (physical.TRS, error) {
	inventory, err := rendering.InspectPhysicalGLB(source)
	if err != nil {
		return physical.TRS{}, err
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return physical.TRS{}, err
	}
	base := placement.DeviceRootRest
	quarterTurn := physical.QuatXYZW{Y: math.Sin(math.Pi / 4), W: math.Cos(math.Pi / 4)}
	base.Rotation = multiplyQuaternion(quarterTurn, base.Rotation)
	base.Translation.Y = 0
	sceneMatrix, err := placement.SceneSupportWorld.Matrix()
	if err != nil {
		return physical.TRS{}, err
	}
	pivotMatrix, err := (physical.TRS{Translation: placement.SupportPivotM, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}).Matrix()
	if err != nil {
		return physical.TRS{}, err
	}
	rootMatrix, err := base.Matrix()
	if err != nil {
		return physical.TRS{}, err
	}
	sourceMatrix, err := sourceToCanonical.Matrix()
	if err != nil {
		return physical.TRS{}, err
	}
	world, err := physical.Compose(sceneMatrix, pivotMatrix)
	if err == nil {
		world, err = physical.Compose(world, rootMatrix)
	}
	if err == nil {
		world, err = physical.Compose(world, sourceMatrix)
	}
	if err != nil {
		return physical.TRS{}, err
	}
	minY := math.Inf(1)
	for mask := 0; mask < 8; mask++ {
		point := physical.Vec3{X: bounds.min.X, Y: bounds.min.Y, Z: bounds.min.Z}
		if mask&1 != 0 {
			point.X = bounds.max.X
		}
		if mask&2 != 0 {
			point.Y = bounds.max.Y
		}
		if mask&4 != 0 {
			point.Z = bounds.max.Z
		}
		point, err = physical.TransformPoint(world, point)
		if err != nil {
			return physical.TRS{}, err
		}
		minY = math.Min(minY, point.Y)
	}
	if !positiveFinite(math.Abs(minY)) && minY != 0 {
		return physical.TRS{}, errors.New("landscape iPad support clearance is non-finite")
	}
	targetY := placement.SceneSupportWorld.Translation.Y + placement.SupportPivotM.Y
	base.Translation.Y = placement.DeviceRootRest.Translation.Y
	if !positiveFinite(math.Abs(minY-targetY)) && minY != targetY {
		return physical.TRS{}, errors.New("landscape iPad conservative bounds are non-finite")
	}
	return base, nil
}

func multiplyQuaternion(a, b physical.QuatXYZW) physical.QuatXYZW {
	q := physical.QuatXYZW{X: a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y, Y: a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X, Z: a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W, W: a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z}
	canonical, err := physical.CanonicalizeQuaternion(q)
	if err != nil {
		return q
	}
	return canonical
}

func (s *starterInitializer) ensureIPadLandscapePreset(ctx context.Context, presetID string, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedIPadPlacement) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{
		Device: &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256}, Scene: &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256}, Motion: &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256},
		DisplayId: placement.DisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.SelectedCameraID,
		SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest), StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1},
	}
	return s.presets.Save(ctx, presetID, 0, "iPad landscape cup-free review still", setup)
}

func cloneDimensions(source map[string]devices.Dimensions) map[string]devices.Dimensions {
	out := make(map[string]devices.Dimensions, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}
