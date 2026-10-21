package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

const (
	watchAssetID       = "sceneview-series7-generic-neutral-watch-v1"
	watchAssetFile     = "assets/watch/series7-generic-neutral-screen.glb"
	watchAssetSHA256   = "929de3091a06c42dbded6c88312d3285d2ed73857fbecaf188cb94f1ae42b522"
	watchBundleID      = "device-studio-starter-generic-rectangular-watch-v1"
	watchDeviceID      = "generic-rectangular-smartwatch-source-candidate-v1"
	watchSceneID       = "generic-rectangular-watch-study-no-cup-v1"
	watchPresetID      = "generic-rectangular-watch-study-still-v1"
	watchReviewID      = "device-studio-e4-series7-neutral-watch-candidate-v1"
	watchDisplayID     = "main_display"
	watchRootPartID    = "watch-root"
	watchScreenPartID  = "watch-screen"
	watchSupportID     = "watch-root-tabletop-candidate"
	watchCameraID      = "watch-camera-full-source-bounds-v1"
	watchManifestPath  = "scenarios/device-studio/api/internal/physicalruntime/assets/watch/asset-manifest.json"
	watchManifestSHA   = "a02481d03ffb051e900b3bd548d973871e54a891adf82adeb92f2c4b9552d7f9"
	watchSourceSHA256  = "ac3c33a32bc27c562e6655245e20327d17c77c988f11aab3016496d851e4e5e3"
	watchCreatorSHA256 = "900a4e5d90c06106979b2c2bc8431f43e4311e0426df43cea65ce95b7f60051a"
)

type WatchStarterState struct {
	Bundle    library.BundleRevision
	Device    devices.DeviceRevision
	Scene     scenes.Revision
	Motion    motions.Revision
	Placement ReviewedWatchPlacement
	Preset    shots.PhysicalPresetRevision
}

// ReviewedWatchPlacement is a conservative candidate pose, not contact proof.
type ReviewedWatchPlacement struct {
	DisplayID, DeviceSupportID, SceneSupportID, CameraID string
	SceneSupportWorld, DeviceRootRest                    physical.TRS
	SupportPivotM                                        physical.Vec3
}

// InitializeReviewedWatchCandidate is opt-in; normal starter initialization does not publish it.
func (o Owners) InitializeReviewedWatchCandidate(ctx context.Context) (WatchStarterState, error) {
	if o.Starter == nil {
		return WatchStarterState{}, errors.New("watch candidate initializer is unavailable")
	}
	pinned, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return WatchStarterState{}, err
	}
	base, err := o.Starter.initializePinned(pinned)
	if err != nil {
		return WatchStarterState{}, err
	}
	room, err := o.Starter.initializeLaptopCandidatePinned(pinned)
	if err != nil {
		return WatchStarterState{}, err
	}
	return o.Starter.initializeWatchCandidatePinned(pinned, room.FramedScene, base.Motion)
}

func (s *starterInitializer) initializeWatchCandidatePinned(ctx context.Context, room scenes.Revision, still motions.Revision) (WatchStarterState, error) {
	if room.SceneID != laptopFramedSceneID || still.MotionID != "phone-still-identity" {
		return WatchStarterState{}, errors.New("watch candidate requires the pinned cup-free room and identity still motion")
	}
	bundle, err := s.publishWatchBundle(ctx)
	if err != nil {
		return WatchStarterState{}, err
	}
	device, bounds, err := s.saveWatchDevice(ctx, bundle)
	if err != nil {
		return WatchStarterState{}, err
	}
	placement := watchPlacement(bounds)
	scene, err := s.saveWatchScene(ctx, room, device, bounds, placement)
	if err != nil {
		return WatchStarterState{}, err
	}
	if err := validateWatchPlacement(device, scene, still, placement); err != nil {
		return WatchStarterState{}, err
	}
	preset, err := s.ensureWatchPreset(ctx, device, scene, still, placement)
	if err != nil {
		return WatchStarterState{}, err
	}
	return WatchStarterState{Bundle: bundle, Device: device, Scene: scene, Motion: still, Placement: placement, Preset: preset}, nil
}

func (s *starterInitializer) publishWatchBundle(ctx context.Context) (library.BundleRevision, error) {
	file := starterFile{
		ID: watchAssetID, File: watchAssetFile, MIME: "model/gltf-binary", SHA: watchAssetSHA256,
		Role: library.RoleGeometryGLB, License: "CC BY 4.0",
		Attribution: "Apple Watch Series 7 - Free Watch-Face SDC™ by SDC PERFORMANCE™; base model attribution to bharathsrk13; adapted under CC BY 4.0. Generic candidate, no manufacturer endorsement.",
		Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted,
			SourceLabel:  "SceneView mirror of SDC PERFORMANCE™ Apple Watch Series 7 - Free Watch-Face; base model attribution to bharathsrk13.",
			SourceSHA256: watchSourceSHA256,
			Adaptation:   "Reproducible derivative manifest " + watchManifestPath + " SHA-256 " + watchManifestSHA + ": replace branded image 0 with neutral PNG, normalize only node 27 / mesh 14 / primitive 0 UV0, and set material 11 to neutral replaceable-screen values. Original hierarchy and sceneview_root scale are preserved; no nominal-size fit."},
	}
	manifest, sources, err := s.buildManifest(ctx, watchBundleID, []starterFile{file})
	if err != nil {
		return library.BundleRevision{}, err
	}
	return s.publish(ctx, manifest, sources, watchReviewID)
}

func (s *starterInitializer) saveWatchDevice(ctx context.Context, bundle library.BundleRevision) (devices.DeviceRevision, physicalBounds, error) {
	ref := artifactRef(bundle, watchAssetID)
	data, err := fs.ReadFile(s.fs, watchAssetFile)
	if err != nil {
		return devices.DeviceRevision{}, physicalBounds{}, err
	}
	if ref.SHA256 != watchAssetSHA256 || ref.SizeBytes != int64(len(data)) || sha256Hex(data) != watchAssetSHA256 {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("embedded watch derivative differs from its pinned Library artifact")
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("inspect neutral watch derivative: %w", err)
	}
	if len(inventory.Nodes) != 29 || len(inventory.Meshes) != 15 || inventory.TriangleCount != 156139 || len(inventory.Extensions) != 0 {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("neutral watch GLB differs from the pinned source inventory")
	}
	nodes := make(map[uint32]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodes[node.Index] = node
	}
	root, rootOK := nodes[28]
	screen, screenOK := nodes[27]
	if !rootOK || root.Name != "sceneview_root" || root.Parent != nil || math.Abs(root.Frame.Scale.X-.6430670822618203) > 1e-9 ||
		math.Abs(root.Frame.Scale.Y-.6430670822618203) > 1e-9 || math.Abs(root.Frame.Scale.Z-.6430670822618203) > 1e-9 ||
		!screenOK || screen.Name != "Object_27" || screen.Parent == nil || *screen.Parent != 26 || screen.Mesh == nil || *screen.Mesh != 14 {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("watch source hierarchy, root scale, or exact screen binding changed")
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil || !bounds.valid() {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("derive conservative watch source envelope: %w", err)
	}
	evidence := []devices.Evidence{
		{Source: watchManifestPath + " SHA-256 " + watchManifestSHA, Kind: "source_geometry_and_derivation", Qualifier: "preserved glTF hierarchy including root scale; geometry-derived only, variant and physical dimensions unknown"},
		{Source: "https://sketchfab.com/3d-models/apple-watch-series-7-free-watch-face-sdctm-cf15ecbf159f42a28df5e56bdfa666bb", Kind: "creator_listing", Qualifier: "SDC PERFORMANCE™; CC BY 4.0; description attributes base model to bharathsrk13; branded embedded image excluded"},
		{Source: "watch-creator-api.json SHA-256 " + watchCreatorSHA256, Kind: "creator_license_record", Qualifier: "creator SDC PERFORMANCE™; CC BY 4.0; base-model attribution to bharathsrk13"},
	}
	width, height, depth := (bounds.max.X-bounds.min.X)*1000, (bounds.max.Y-bounds.min.Y)*1000, (bounds.max.Z-bounds.min.Z)*1000
	displayEvidence := []devices.Evidence{{Source: watchManifestPath + " SHA-256 " + watchManifestSHA, Kind: "derived_display_geometry", Qualifier: "node 27 source mesh plane projection; not an active-pixel or physical qualification"}}
	definition := devices.Definition{
		SchemaVersion: "device-definition/v1", ID: watchDeviceID, Group: devices.GroupWatch, Kind: "generic_rectangular_smartwatch_style",
		IdentityStatus: devices.IdentityGeneric, Model: "Generic rectangular smartwatch-style design", Variant: "unknown; source hierarchy scale preserved",
		Description: "Candidate-only rectangular smartwatch-style model. Source geometry and screen mapping are preserved by the pinned derivative; physical size, product variant, wrist support, contact, motion, and qualification are unknown.",
		Dimensions: map[string]devices.Dimensions{"source_hierarchy_axis_envelope": {
			Width: watchGeometryMeasurement(width, "full source hierarchy X envelope"), Height: watchGeometryMeasurement(height, "full source hierarchy Y envelope"), Depth: watchGeometryMeasurement(depth, "full source hierarchy Z envelope; includes band"),
		}},
		Displays:        []devices.DisplaySurface{{ID: watchDisplayID, Topology: "planar", CoordinateMetric: &devices.DisplayMetric{WidthMM: 25.071, HeightMM: 21.404, Evidence: displayEvidence}, PhysicalBoundary: unknownRegion(), ActiveArea: unknownRegion(), AppSafeArea: unknownRegion(), Evidence: displayEvidence}},
		SupportedPoses:  []devices.Pose{{Name: "source_rest_tabletop_hypothesis", Status: "candidate_only", Evidence: evidence}},
		SupportedMotion: []devices.Capability{{Name: "rigid_root_transform", Status: "candidate_only", MotionKind: "rigid_body", Evidence: evidence}},
		Supports:        []devices.SupportCapability{{Kind: "tabletop", InterfaceID: "horizontal_surface", Status: "candidate_only", Evidence: []devices.Evidence{{Source: watchManifestPath, Kind: "placement_hypothesis", Qualifier: "AABB-derived camera/placement envelope only; not grounding/contact proof"}}}},
		ModelCandidates: []devices.ModelCandidate{{ID: watchAssetID, Status: "candidate_only", SHA256: ref.SHA256,
			ListingURL: "https://sketchfab.com/3d-models/apple-watch-series-7-free-watch-face-sdctm-cf15ecbf159f42a28df5e56bdfa666bb",
			Author:     "SDC PERFORMANCE™; base model attribution to bharathsrk13", License: "CC BY 4.0",
			Attribution: "Apple Watch Series 7 - Free Watch-Face SDC™ by SDC PERFORMANCE™; base model attribution to bharathsrk13; adapted under CC BY 4.0.",
			RightsNotes: "Candidate-only local review. Branded source image excluded. No manufacturer endorsement or trademark clearance; variant, physical measurement, wrist support and round-watch-family suitability are not claimed."}},
		Provenance: evidence, Qualification: "unqualified",
	}
	partIDs := make(map[uint32]string, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		partIDs[node.Index] = watchPartID(node.Index)
	}
	parts := make([]devices.SemanticPart, 0, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		role := "body_component"
		if node.Index == 28 {
			role = "device_root"
		} else if node.Index == 27 {
			role = "display_surface"
		} else if node.Index == 8 || node.Index == 9 || node.Index == 10 {
			role = "band_component"
		} else if node.Index == 6 || node.Index == 7 || node.Index == 17 || node.Index == 18 {
			role = "case_component"
		} else if node.Index == 25 {
			role = "display_glass"
		}
		parent := ""
		if node.Parent != nil {
			parent = partIDs[*node.Parent]
		}
		parts = append(parts, devices.SemanticPart{ID: partIDs[node.Index], Role: role, ParentID: parent, NodeIndex: node.Index, NodeName: node.Name, LocalFrame: node.Frame})
	}
	zero, material := uint32(0), uint32(11)
	candidate := devices.DeviceCandidate{
		Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention, SourceToCanonical: identityTRS(), Parts: parts,
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: watchDisplayID, PartID: partIDs[27], LocalFrame: identityTRS(), ScreenPrimitiveIndex: &zero, ScreenMaterialIndex: &material}},
		SupportFrames:   []devices.DeviceSupportFrame{{ID: watchSupportID, InterfaceID: "horizontal_surface", PartID: partIDs[28], LocalFrame: identityTRS()}},
	}
	screenGeometry, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, watchDisplayID)
	if err != nil {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("preflight exact watch screen binding: %w", err)
	}
	if screenGeometry.NodeIndex != 27 || screenGeometry.MeshIndex != 14 || screenGeometry.PrimitiveIndex != 0 || screenGeometry.MaterialIndex != 11 || screenGeometry.UVMin != [2]float64{} || screenGeometry.UVMax != [2]float64{1, 1} {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("preflight resolved a different watch display geometry binding")
	}
	device, err := saveDeviceCandidateIfUnchanged(ctx, s.devices, candidate)
	return device, bounds, err
}

func (s *starterInitializer) saveWatchScene(ctx context.Context, room scenes.Revision, device devices.DeviceRevision, bounds physicalBounds, placement ReviewedWatchPlacement) (scenes.Revision, error) {
	if room.SceneID != laptopFramedSceneID || device.DeviceID != watchDeviceID || len(room.Candidate.CameraFrames) != 1 || room.Candidate.CameraFrames[0].ID != laptopFramedCameraID {
		return scenes.Revision{}, errors.New("watch scene requires the pinned cup-free room and framed camera")
	}
	hasSupport := false
	for _, support := range room.Candidate.SupportFrames {
		hasSupport = hasSupport || support.ID == laptopTableSupportID && support.InterfaceID == "horizontal_surface"
	}
	if !hasSupport {
		return scenes.Revision{}, errors.New("watch scene has no candidate tabletop support frame")
	}
	candidate := room.Candidate
	candidate.SceneID = watchSceneID
	candidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), room.Candidate.CameraFrames...)
	camera := candidate.CameraFrames[0]
	camera.ID = watchCameraID
	camera.SensorFit = scenes.SensorFitHorizontal
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusSelectedDevice}
	fitted, err := fitIPadPerspectiveCamera(bounds, identityTRS(), ReviewedIPadPlacement{SceneSupportWorld: placement.SceneSupportWorld, SupportPivotM: placement.SupportPivotM, DeviceRootRest: placement.DeviceRootRest}, camera, 16.0/9.0, .10)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("fit watch camera to conservative full-source bounds: %w", err)
	}
	camera.Frame = fitted
	candidate.CameraFrames[0] = camera
	return saveSceneCandidateIfUnchanged(ctx, s.scenes, candidate)
}

func watchPlacement(bounds physicalBounds) ReviewedWatchPlacement {
	identity := identityTRS()
	identity.Translation.Y = -bounds.min.Y
	return ReviewedWatchPlacement{
		DisplayID: watchDisplayID, DeviceSupportID: watchSupportID, SceneSupportID: laptopTableSupportID, CameraID: watchCameraID,
		SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: identity.Rotation, Scale: identity.Scale}, DeviceRootRest: identity,
	}
}

func validateWatchPlacement(device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedWatchPlacement) error {
	if device.DeviceID != watchDeviceID || scene.SceneID != watchSceneID || motion.MotionID != "phone-still-identity" ||
		placement.DisplayID != watchDisplayID || placement.DeviceSupportID != watchSupportID || placement.SceneSupportID != laptopTableSupportID || placement.CameraID != watchCameraID {
		return errors.New("watch placement is bound to different candidate revisions or semantic roles")
	}
	return nil
}

func (s *starterInitializer) ensureWatchPreset(ctx context.Context, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedWatchPlacement) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{
		Device:    &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256},
		Scene:     &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256},
		Motion:    &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256},
		DisplayId: placement.DisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.CameraID,
		SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest),
		StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1},
	}
	if current, err := s.presets.Head(ctx, watchPresetID); err == nil {
		if current.Name == "Generic rectangular watch candidate on cup-free study table" && proto.Equal(current.Setup, setup) {
			return current, nil
		}
		return shots.PhysicalPresetRevision{}, fmt.Errorf("watch preset differs from its fixed candidate; refusing implicit revision: %w", shots.ErrPhysicalPresetConflict)
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, watchPresetID, 0, "Generic rectangular watch candidate on cup-free study table", setup)
}

func watchPartID(index uint32) string { return fmt.Sprintf("watch-node-%02d", index) }

func watchGeometryMeasurement(valueMM float64, axis string) devices.Measurement {
	value := valueMM
	return devices.Measurement{Value: &value, Unit: "mm", Meaning: "GLB-derived " + axis, Configuration: "full source hierarchy including sceneview_root scale 0.6430670822618203", Method: "derived", Uncertainty: devices.Uncertainty{Status: "unknown"}, Evidence: []devices.Evidence{{Source: watchManifestPath + " SHA-256 " + watchManifestSHA, Kind: "derived_asset_geometry", Qualifier: "source geometry only; not physical hardware, nominal dimensions, or product variant"}}}
}
