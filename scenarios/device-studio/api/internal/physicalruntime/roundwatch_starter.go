package physicalruntime

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"

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
	roundWatchAssetID      = "grab3d-round-smartwatch-neutral-screen-v1"
	roundWatchAssetFile    = "assets/roundwatch/round-smartwatch-neutral.glb"
	roundWatchAssetSHA     = "216bfc3c6664ca044c1690446aa7d73587e9f1caed865cd5d168aaca2010dd9d"
	roundWatchSourceSHA    = "763ba7570daad6e1024ab8242efb7c3dccb9a183b56782a564bdc748dba3d589"
	roundWatchBundleID     = "device-studio-starter-grab3d-round-smartwatch-neutral-v1"
	roundWatchDeviceID     = "grab3d-round-smartwatch-generic-candidate-v1"
	roundWatchSceneID      = "grab3d-round-smartwatch-study-candidate-v1"
	roundWatchPresetID     = "grab3d-round-smartwatch-study-still-v1"
	roundWatchReviewID     = "device-studio-e4-grab3d-round-smartwatch-source-derivative-v1"
	roundWatchDisplayID    = "main_display"
	roundWatchRootPartID   = "watch-source-root"
	roundWatchBodyPartID   = "watch-body"
	roundWatchScreenPartID = "watch-round-display"
	roundWatchSupportID    = "watch-closed-strap-tabletop-candidate"
	roundWatchCameraID     = "round-watch-camera-full-source-bounds-v1"
)

type RoundWatchStarterState struct {
	Bundle library.BundleRevision
	Device devices.DeviceRevision
	Scene  scenes.Revision
	Motion motions.Revision
	Preset shots.PhysicalPresetRevision
}

// InitializeReviewedRoundWatchCandidate is an explicit local opt-in. It never
// replaces or changes the existing rectangular-watch starter.
func (o Owners) InitializeReviewedRoundWatchCandidate(ctx context.Context) (RoundWatchStarterState, error) {
	if o.Starter == nil {
		return RoundWatchStarterState{}, errors.New("round-watch candidate initializer is unavailable")
	}
	pinned, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return RoundWatchStarterState{}, err
	}
	base, err := o.Starter.initializePinned(pinned)
	if err != nil {
		return RoundWatchStarterState{}, err
	}
	room, err := o.Starter.initializeLaptopCandidatePinned(pinned)
	if err != nil {
		return RoundWatchStarterState{}, err
	}
	return o.Starter.initializeRoundWatchPinned(pinned, room.FramedScene, base.Motion)
}

func (s *starterInitializer) initializeRoundWatchPinned(ctx context.Context, room scenes.Revision, still motions.Revision) (RoundWatchStarterState, error) {
	if room.SceneID != laptopFramedSceneID || still.MotionID != "phone-still-identity" {
		return RoundWatchStarterState{}, errors.New("round-watch candidate requires the pinned cup-free room and identity still motion")
	}
	bundle, err := s.publishRoundWatchBundle(ctx)
	if err != nil {
		return RoundWatchStarterState{}, err
	}
	device, bounds, err := s.saveRoundWatchDevice(ctx, bundle)
	if err != nil {
		return RoundWatchStarterState{}, err
	}
	scene, err := s.saveRoundWatchScene(ctx, room, device, bounds)
	if err != nil {
		return RoundWatchStarterState{}, err
	}
	preset, err := s.ensureRoundWatchPreset(ctx, device, scene, still)
	if err != nil {
		return RoundWatchStarterState{}, err
	}
	return RoundWatchStarterState{Bundle: bundle, Device: device, Scene: scene, Motion: still, Preset: preset}, nil
}

func (s *starterInitializer) publishRoundWatchBundle(ctx context.Context) (library.BundleRevision, error) {
	file := starterFile{
		ID: roundWatchAssetID, File: roundWatchAssetFile, MIME: "model/gltf-binary", SHA: roundWatchAssetSHA,
		Role: library.RoleGeometryGLB, License: "CC0-1.0",
		Attribution: "Grab3D procedural generic smartwatch (CC0 1.0); adapted to remove analog face overlays and normalize the separate display UVs.",
		Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "Grab3D Cozy Electronics Kit smartwatch; procedural, AI-assisted scripts, no AI-generated geometry or textures.", SourceSHA256: roundWatchSourceSHA,
			Adaptation: "Retains source hierarchy and original BIN payload; keeps smartwatch_body and the separate smartwatch_display_glass primitive 0, removes dial tick/hand/dot primitives 1-4, and normalizes only primitive 0 TEXCOORD_0 to [0,1]. See roundwatch-source-derivative.json."},
	}
	manifest, sources, err := s.buildManifest(ctx, roundWatchBundleID, []starterFile{file})
	if err != nil {
		return library.BundleRevision{}, err
	}
	return s.publish(ctx, manifest, sources, roundWatchReviewID)
}

func (s *starterInitializer) saveRoundWatchDevice(ctx context.Context, bundle library.BundleRevision) (devices.DeviceRevision, physicalBounds, error) {
	data, err := fs.ReadFile(s.fs, roundWatchAssetFile)
	if err != nil {
		return devices.DeviceRevision{}, physicalBounds{}, err
	}
	ref := artifactRef(bundle, roundWatchAssetID)
	if ref.SHA256 != roundWatchAssetSHA || ref.SizeBytes != int64(len(data)) || fmt.Sprintf("%x", sha256.Sum256(data)) != roundWatchAssetSHA {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("embedded round-watch derivative differs from its pinned Library artifact")
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("inspect round-watch derivative: %w", err)
	}
	if len(inventory.Nodes) != 3 || len(inventory.Meshes) != 2 || inventory.TriangleCount != 5956 || len(inventory.Extensions) != 0 {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("round-watch derivative differs from reviewed source hierarchy/topology (nodes=%d meshes=%d triangles=%d extensions=%v)", len(inventory.Nodes), len(inventory.Meshes), inventory.TriangleCount, inventory.Extensions)
	}
	nodes := map[string]rendering.PhysicalGLBNode{}
	for _, node := range inventory.Nodes {
		nodes[node.Name] = node
	}
	body, bodyOK := nodes["smartwatch_body"]
	display, displayOK := nodes["smartwatch_display_glass"]
	root, rootOK := nodes["smartwatch"]
	if !bodyOK || body.Index != 0 || body.Parent == nil || *body.Parent != root.Index || body.Mesh == nil || *body.Mesh != 0 ||
		!displayOK || display.Index != 1 || display.Parent == nil || *display.Parent != root.Index || display.Mesh == nil || *display.Mesh != 1 || !rootOK || root.Parent != nil || len(inventory.Meshes[1].Primitives) != 1 {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("round-watch source body/display hierarchy changed")
	}
	primitive := inventory.Meshes[1].Primitives[0]
	if primitive.Material != 1 || primitive.UVMin != [2]float64{} || primitive.UVMax != [2]float64{1, 1} {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("round-watch screen is not the normalized neutral display-glass primitive")
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil || !bounds.valid() {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("derive round-watch source envelope: %w", err)
	}
	evidence := []devices.Evidence{
		{Source: "Grab3D smartwatch.glb SHA-256 " + roundWatchSourceSHA, Kind: "source_geometry_and_license", Qualifier: "CC0 1.0; creator-declared closed-strap dimensions approx. 39 x 66 x 63 mm and 36 x 43 mm case; measurement and variant uncertainty retained"},
		{Source: "assets/roundwatch/roundwatch-source-derivative.json SHA-256 9b1a2eda89df387d0aed5c56f3de0e949b61917ac4300dbf32131f88c1ade391", Kind: "reproducible_derivation", Qualifier: "only analog dial overlays removed and separate display UV range normalized; source geometry, textures, transforms preserved"},
	}
	width, height, depth := (bounds.max.X-bounds.min.X)*1000, (bounds.max.Y-bounds.min.Y)*1000, (bounds.max.Z-bounds.min.Z)*1000
	displayEvidence := []devices.Evidence{{Source: "Grab3D smartwatch.glb SHA-256 " + roundWatchSourceSHA, Kind: "source_display_geometry", Qualifier: "separate glass primitive bounds; normalized display image area is a conservative candidate circle, not physical active-pixel qualification"}}
	definition := devices.Definition{
		SchemaVersion: "device-definition/v1", ID: roundWatchDeviceID, Group: devices.GroupWatch, Kind: "generic_round_smartwatch_style",
		IdentityStatus: devices.IdentityGeneric, Model: "Generic round smartwatch-style design", Variant: "unknown source variant",
		Description: "Candidate-only round smartwatch derived from a CC0 procedural model. The analog dial marks and hands were removed; display geometry is isolated for app content. Closed-strap contact, exact product variant, fit, dimensions and motion are unqualified.",
		Dimensions:  map[string]devices.Dimensions{"source_geometry_envelope": {Width: roundWatchMeasurement(width, "source geometry X span"), Height: roundWatchMeasurement(height, "source geometry Y span including closed strap"), Depth: roundWatchMeasurement(depth, "source geometry Z span")}},
		Displays: []devices.DisplaySurface{{ID: roundWatchDisplayID, Topology: "planar", CoordinateMetric: &devices.DisplayMetric{WidthMM: 31.5, HeightMM: 38.5, Evidence: displayEvidence},
			PhysicalBoundary: unknownRegion(),
			ActiveArea:       devices.Region{Status: "candidate_only", Shape: "circle", Bounds: &devices.Rect{X: 0, Y: .09090909, Width: 1, Height: .81818182}, Evidence: displayEvidence},
			AppSafeArea:      devices.Region{Status: "candidate_only", Shape: "circle", Bounds: &devices.Rect{X: .05, Y: .13181818, Width: .9, Height: .73636364}, Evidence: displayEvidence}, Evidence: displayEvidence}},
		SupportedPoses:  []devices.Pose{{Name: "closed-strap-tabletop-hypothesis", Status: "candidate_only", Evidence: evidence}},
		SupportedMotion: []devices.Capability{{Name: "rigid_root_transform", Status: "candidate_only", MotionKind: "rigid_body", Evidence: evidence}},
		Supports:        []devices.SupportCapability{{Kind: "tabletop", InterfaceID: "horizontal_surface", Status: "candidate_only", Evidence: evidence}},
		ModelCandidates: []devices.ModelCandidate{{ID: roundWatchAssetID, Status: "candidate_only", SHA256: ref.SHA256, Author: "Grab3D", License: "CC0-1.0", Attribution: "Grab3D Cozy Electronics Kit smartwatch; CC0 1.0.", RightsNotes: "Source license permits reuse. Candidate is unqualified; source-stated dimensions and closed-strap pose are creator claims, not measurements."}},
		Provenance:      evidence, Qualification: "unqualified",
	}
	parts := []devices.SemanticPart{
		{ID: roundWatchRootPartID, Role: "device_root", NodeIndex: root.Index, NodeName: root.Name, LocalFrame: root.Frame},
		{ID: roundWatchBodyPartID, Role: "body_component", ParentID: roundWatchRootPartID, NodeIndex: body.Index, NodeName: body.Name, LocalFrame: body.Frame},
		{ID: roundWatchScreenPartID, Role: "display_surface", ParentID: roundWatchRootPartID, NodeIndex: display.Index, NodeName: display.Name, LocalFrame: display.Frame},
	}
	zero, material := uint32(0), uint32(1)
	candidate := devices.DeviceCandidate{Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention, SourceToCanonical: identityTRS(), Parts: parts,
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: roundWatchDisplayID, PartID: roundWatchScreenPartID, LocalFrame: identityTRS(), ScreenPrimitiveIndex: &zero, ScreenMaterialIndex: &material}},
		SupportFrames:   []devices.DeviceSupportFrame{{ID: roundWatchSupportID, InterfaceID: "horizontal_surface", PartID: roundWatchRootPartID, LocalFrame: identityTRS()}},
	}
	mapping, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, roundWatchDisplayID)
	if err != nil {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("round-watch display binding failed preflight: %w", err)
	}
	if mapping.NodeIndex != display.Index || mapping.MeshIndex != 1 || mapping.PrimitiveIndex != 0 || mapping.MaterialIndex != 1 || mapping.UVMin != [2]float64{} || mapping.UVMax != [2]float64{1, 1} {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("round-watch display preflight selected different source geometry")
	}
	revision, err := saveDeviceCandidateIfUnchanged(ctx, s.devices, candidate)
	return revision, bounds, err
}

func (s *starterInitializer) saveRoundWatchScene(ctx context.Context, room scenes.Revision, device devices.DeviceRevision, bounds physicalBounds) (scenes.Revision, error) {
	if room.SceneID != laptopFramedSceneID || device.DeviceID != roundWatchDeviceID || len(room.Candidate.CameraFrames) != 1 {
		return scenes.Revision{}, errors.New("round-watch scene requires its pinned cup-free room and framed camera")
	}
	candidate := room.Candidate
	candidate.SceneID = roundWatchSceneID
	camera := candidate.CameraFrames[0]
	camera.ID = roundWatchCameraID
	fitted, err := fitIPadPerspectiveCamera(bounds, identityTRS(), ReviewedIPadPlacement{SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: identityTRS().Rotation, Scale: identityTRS().Scale}, DeviceRootRest: identityTRS()}, camera, 16.0/9.0, .12)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("fit round-watch camera to source bounds: %w", err)
	}
	camera.Frame = fitted
	center := physical.Vec3{X: 0, Y: .75 + (bounds.min.Y+bounds.max.Y)/2, Z: (bounds.min.Z + bounds.max.Z) / 2}
	distance, err := watchCameraAxisDistance(camera.Frame, center)
	if err != nil {
		return scenes.Revision{}, err
	}
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusDistance, DistanceM: &distance}
	candidate.CameraFrames = []scenes.SceneCameraFrame{camera}
	return saveSceneCandidateIfUnchanged(ctx, s.scenes, candidate)
}

func (s *starterInitializer) ensureRoundWatchPreset(ctx context.Context, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{Device: &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256}, Scene: &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256}, Motion: &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256}, DisplayId: roundWatchDisplayID, DeviceSupportId: roundWatchSupportID, SceneSupportId: laptopTableSupportID, CameraId: roundWatchCameraID,
		SceneSupportWorld: starterTRS(physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: identityTRS().Rotation, Scale: identityTRS().Scale}), SupportPivotM: starterVec3(physical.Vec3{}), DeviceRootRest: starterTRS(identityTRS()), StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1}}
	if current, err := s.presets.Head(ctx, roundWatchPresetID); err == nil {
		if current.Name == "Generic round smartwatch candidate on study table" && proto.Equal(current.Setup, setup) {
			return current, nil
		}
		return shots.PhysicalPresetRevision{}, fmt.Errorf("round-watch preset differs from fixed candidate: %w", shots.ErrPhysicalPresetConflict)
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, roundWatchPresetID, 0, "Generic round smartwatch candidate on study table", setup)
}

func roundWatchMeasurement(valueMM float64, axis string) devices.Measurement {
	value := valueMM
	evidence := []devices.Evidence{{Source: "Grab3D smartwatch.glb SHA-256 " + roundWatchSourceSHA, Kind: "derived_asset_geometry", Qualifier: "GLB span only; creator-declared approximate size is separately recorded and is not a measurement"}}
	return devices.Measurement{Value: &value, Unit: "mm", Meaning: axis, Configuration: "full source hierarchy; original transforms retained", Method: "derived", Uncertainty: devices.Uncertainty{Status: "unknown"}, Evidence: evidence}
}
