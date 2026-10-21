package physicalruntime

import (
	"context"
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
	smartDisplayAssetFile       = "assets/smart-display/generic-wedge-smart-display.glb"
	smartDisplayAssetSHA        = "1b1364a137fc99545080dca69104c4f4d2a6b8821ad4ebc709981da4df89830f"
	smartDisplayManifest        = "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display/asset-manifest.json"
	smartDisplayManifestSHA     = "f7f89637e53ba8e2b900466b008d5e52b11185552a6fe537c169f6f71cee7380"
	digitalFrameManifest        = "scenarios/device-studio/api/internal/physicalruntime/assets/digital-frame/asset-manifest-v2.json"
	digitalFrameManifestSHA     = "0bd8c2af563201c7244cfbd17aadfb5150c297dade066f78c32666f98c27decd"
	digitalFrameWallManifest    = "scenarios/device-studio/api/internal/physicalruntime/assets/digital-frame/asset-manifest-v3.json"
	digitalFrameWallManifestSHA = "d7458aa8c2160fc9f3d06f06187fe15ed1ebf4c9359b963bd6955751878f6c10"
	vehicleStackAssetFile       = "assets/vehicle-center-stack/generic-center-stack.glb"
	vehicleStackAssetSHA        = "37fc41e26e5c5f616f78fe68dfbd88cbbce6bfb0038754a65a20783684239ce9"
	vehicleStackManifest        = "scenarios/device-studio/api/internal/physicalruntime/assets/vehicle-center-stack/asset-manifest.json"
	vehicleStackManifestSHA     = "6f123070d68bd9021dbfb07ad01b34eb612a2ae1d4970847efb2bd6d278d84d5"
	familyWallSceneID           = "screen-family-candidate-wall-study-v1"
	familyWallSupportID         = "study-wall-vertical-candidate"
	familyWallSupportPartID     = "laptop-scene-part-0"
	familyWallCameraID          = "screen-family-wall-camera-full-envelope-v1"
)

type ScreenFamilyStarterState struct {
	Bundle library.BundleRevision
	Device devices.DeviceRevision
	Scene  scenes.Revision
	Motion motions.Revision
	Preset shots.PhysicalPresetRevision
}

type screenFamilySpec struct {
	assetID, file, sha, bundleID, deviceID, sceneID, presetID, presetName, reviewID                        string
	manifestPath, manifestSHA, nodeName, displayID, rootName, groupKind, kind, model, variant, description string
	license, attribution, sourceLabel, sourceSHA, adaptation, profile                                      string
	wall                                                                                                   bool
}

func smartDisplaySpec() screenFamilySpec {
	return screenFamilySpec{assetID: "vrooli-generic-wedge-smart-display-concept-v1", file: smartDisplayAssetFile, sha: smartDisplayAssetSHA,
		bundleID: "device-studio-generic-wedge-smart-display-v1", deviceID: "generic-wedge-smart-display-candidate-v1",
		sceneID: "generic-wedge-smart-display-study-candidate-v1", presetID: "generic-wedge-smart-display-tabletop-still-v1",
		presetName: "Generic wedge smart display tabletop review", reviewID: "device-studio-e4-generic-wedge-smart-display-candidate-v1",
		manifestPath: smartDisplayManifest, manifestSHA: smartDisplayManifestSHA, nodeName: "MainDisplay", displayID: "main_display",
		rootName: "Generic Wedge Smart Display", groupKind: string(devices.GroupSmartDisplayFrame), kind: "generic_wedge_smart_display_concept",
		model: "Generic authored wedge smart-display concept", variant: "180 x 125 x 90 mm authored design envelope",
		description: "Unqualified Vrooli-authored wedge display candidate for tabletop composition studies. Nest Hub dimensions are comparison context only; no branded identity, electronics, or product fit is claimed.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic wedge smart-display concept; no manufacturer geometry.",
		sourceLabel: "Deterministic Vrooli GLB derivation script; see the exact asset manifest.", profile: "tabletop"}
}

func digitalFrameSpec(wall bool) screenFamilySpec {
	profile, file, digest, id, deviceID, sceneID, presetID, name := "tabletop", "assets/digital-frame/innerscene-frame-tabletop-v2.glb",
		"bc6b22fb722672b0601dc2ea4424517e6bed589e3ebf40de9ecd0257d20fb4f4", "innerscene-frame-digital-neutral-tabletop-v2",
		"innerscene-digital-frame-tabletop-candidate-v2", "innerscene-digital-frame-tabletop-study-v2", "innerscene-digital-frame-tabletop-still-v2", "Innerscene-derived neutral digital frame tabletop review"
	if wall {
		profile, file, digest, id, deviceID, sceneID, presetID, name = "wall", "assets/digital-frame/innerscene-frame-wall-v3.glb",
			"86c216e9bc574ac076bc78231107cb53b4c178bdea4121239df43ca8963e17bf", "innerscene-frame-digital-neutral-wall-v3",
			"innerscene-digital-frame-wall-candidate-v3", "innerscene-digital-frame-wall-study-v3", "innerscene-digital-frame-wall-still-v3", "Innerscene-derived neutral digital frame wall review"
	}
	manifestPath, manifestSHA := digitalFrameManifest, digitalFrameManifestSHA
	if wall {
		manifestPath, manifestSHA = digitalFrameWallManifest, digitalFrameWallManifestSHA
	}
	return screenFamilySpec{assetID: id, file: file, sha: digest, bundleID: "device-studio-" + id, deviceID: deviceID, sceneID: sceneID,
		presetID: presetID, presetName: name, reviewID: "device-studio-e4-" + id, manifestPath: manifestPath,
		manifestSHA: manifestSHA, nodeName: "Unprinted frame insert", displayID: "main_display", rootName: "Frame Stand",
		groupKind: string(devices.GroupSmartDisplayFrame), kind: "generic_digital_frame_concept", model: "Generic Innerscene-derived digital frame concept",
		variant: "neutral electronic display, " + profile + " support derivative", description: "Unqualified generic electronic-frame candidate adapted from the CC0 Innerscene picture-frame model. The neutral surface and support conversion are candidate interpretations; this is not an Innerscene electronic product or measured digital frame.",
		license: "CC0-1.0", attribution: "Innerscene Frame Stand - Picture Frame, CC0-1.0; Vrooli neutral display UV/material adaptation; wall cleat authored by Vrooli where applicable.",
		sourceLabel: "Innerscene Frame Stand - Picture Frame, source-preserving digital-frame derivative.", sourceSHA: "7c9f5cf3e63c7f01cc911b1db9eea2c95b5e1bdadd9b75c3076b10dca70cef10",
		adaptation: "Only the target display material/UV view changes. Table derivative retains source easel geometry; wall derivative omits the source easel leaves and adds a generic authored rear cleat. Both preserve original BIN bytes as a prefix and keep source geometry untouched.", profile: profile, wall: wall}
}

func vehicleStackSpec(profile string) screenFamilySpec {
	presetID, presetName := "generic-center-stack-carplay-sim-still-v1", "Generic center stack · CarPlay-style simulated screen"
	if profile == "android_auto_simulated" {
		presetID, presetName = "generic-center-stack-android-auto-sim-still-v1", "Generic center stack · Android Auto-style simulated screen"
	}
	return screenFamilySpec{assetID: "vrooli-generic-vehicle-center-stack-concept-v1", file: vehicleStackAssetFile, sha: vehicleStackAssetSHA,
		bundleID: "device-studio-generic-vehicle-center-stack-v1", deviceID: "generic-vehicle-center-stack-candidate-v1",
		sceneID: "generic-vehicle-center-stack-study-candidate-v1", presetID: presetID, presetName: presetName,
		reviewID: "device-studio-e4-generic-vehicle-center-stack-candidate-v1", manifestPath: vehicleStackManifest, manifestSHA: vehicleStackManifestSHA,
		nodeName: "CenterDisplay", displayID: "main_display", rootName: "Generic Vehicle Center Stack Concept",
		groupKind: string(devices.GroupVehicleHost), kind: "generic_vehicle_center_stack_concept", model: "Generic authored vehicle center-stack concept",
		variant:     "380 x 250 x 120 mm authored review geometry; simulated screen-use profiles",
		description: "Unqualified Vrooli-authored center-stack candidate with simulated screen-use profiles. The profile label does not provide Apple CarPlay or Android Auto functionality, vehicle compatibility, safety review, or an in-vehicle fit.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic center-stack concept; no manufacturer geometry.",
		sourceLabel: "Deterministic Vrooli GLB derivation script; see the exact asset manifest.", profile: "simulated_vehicle_screen_use"}
}

func (o Owners) InitializeReviewedSmartDisplayCandidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, smartDisplaySpec())
}

func (o Owners) InitializeReviewedDigitalFrameCandidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, digitalFrameSpec(false))
}

func (o Owners) InitializeReviewedDigitalFrameWallCandidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, digitalFrameSpec(true))
}

func (o Owners) InitializeReviewedVehicleCenterStackCandidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.InitializeReviewedVehicleCenterStackProfile(ctx, "carplay_simulated")
}

func (o Owners) InitializeReviewedVehicleCenterStackProfile(ctx context.Context, profile string) (ScreenFamilyStarterState, error) {
	if profile != "carplay_simulated" && profile != "android_auto_simulated" {
		return ScreenFamilyStarterState{}, errors.New("vehicle screen profile must be carplay_simulated or android_auto_simulated")
	}
	return o.initializeScreenFamilyCandidate(ctx, vehicleStackSpec(profile))
}

func (o Owners) initializeScreenFamilyCandidate(ctx context.Context, spec screenFamilySpec) (ScreenFamilyStarterState, error) {
	if o.Starter == nil {
		return ScreenFamilyStarterState{}, errors.New("screen-family candidate initializer is unavailable")
	}
	pinned, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	base, err := o.Starter.initializePinned(pinned)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	room, err := o.Starter.initializeLaptopCandidatePinned(pinned)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	if base.Motion.MotionID != "phone-still-identity" || room.FramedScene.SceneID != laptopFramedSceneID {
		return ScreenFamilyStarterState{}, errors.New("screen-family candidate requires the pinned cup-free room and shared identity motion")
	}
	return o.Starter.initializeScreenFamilyPinned(pinned, room.FramedScene, base.Motion, spec)
}

func (s *starterInitializer) initializeScreenFamilyPinned(ctx context.Context, room scenes.Revision, still motions.Revision, spec screenFamilySpec) (ScreenFamilyStarterState, error) {
	file := starterFile{ID: spec.assetID, File: spec.file, MIME: "model/gltf-binary", SHA: spec.sha, Role: library.RoleGeometryGLB,
		License: spec.license, Attribution: spec.attribution}
	if spec.sourceSHA == "" {
		file.Provenance = library.ArtifactProvenance{Kind: library.ProvenanceAuthored, SourceLabel: spec.sourceLabel}
	} else {
		file.Provenance = library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: spec.sourceLabel, SourceSHA256: spec.sourceSHA, Adaptation: spec.adaptation + " See " + spec.manifestPath + " SHA-256 " + spec.manifestSHA + "."}
	}
	manifest, sources, err := s.buildManifest(ctx, spec.bundleID, []starterFile{file})
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	bundle, err := s.publish(ctx, manifest, sources, spec.reviewID)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	data, err := fs.ReadFile(s.fs, spec.file)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	if sha256Hex(data) != spec.sha {
		return ScreenFamilyStarterState{}, fmt.Errorf("screen-family bytes differ from pinned candidate %q", spec.assetID)
	}
	ref := artifactRef(bundle, spec.assetID)
	if ref.SHA256 != spec.sha || ref.SizeBytes != int64(len(data)) {
		return ScreenFamilyStarterState{}, errors.New("screen-family Library artifact differs from the pinned candidate bytes")
	}
	device, bounds, inventory, err := s.saveScreenFamilyDevice(ctx, spec, ref, data)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	scene, placement, err := s.saveScreenFamilyScene(ctx, room, device, spec, bounds)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	preset, err := s.ensureScreenFamilyPreset(ctx, device, scene, still, placement, spec)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	_ = inventory
	return ScreenFamilyStarterState{Bundle: bundle, Device: device, Scene: scene, Motion: still, Preset: preset}, nil
}

func (s *starterInitializer) saveScreenFamilyDevice(ctx context.Context, spec screenFamilySpec, ref physical.ArtifactRef, data []byte) (devices.DeviceRevision, physicalBounds, rendering.PhysicalGLBInventory, error) {
	invalid := func(err error) (devices.DeviceRevision, physicalBounds, rendering.PhysicalGLBInventory, error) {
		return devices.DeviceRevision{}, physicalBounds{}, rendering.PhysicalGLBInventory{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return invalid(fmt.Errorf("inspect %s candidate: %w", spec.assetID, err))
	}
	if len(inventory.Nodes) == 0 || len(inventory.Meshes) == 0 {
		return invalid(errors.New("screen-family candidate has no bounded core glTF geometry"))
	}
	nodes := make(map[uint32]rendering.PhysicalGLBNode, len(inventory.Nodes))
	nodesByName := make(map[string]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodes[node.Index], nodesByName[node.Name] = node, node
	}
	var root rendering.PhysicalGLBNode
	rootOK := false
	for _, node := range inventory.Nodes {
		if node.Name == spec.rootName || node.Parent == nil {
			if !rootOK {
				root, rootOK = node, true
			}
		}
	}
	display, displayOK := nodesByName[spec.nodeName]
	if !rootOK || !displayOK || display.Mesh == nil {
		return invalid(fmt.Errorf("screen-family root %q or screen node %q is missing", spec.rootName, spec.nodeName))
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil || !bounds.valid() {
		return invalid(fmt.Errorf("derive %s GLB envelope: %w", spec.assetID, err))
	}
	partID := func(index uint32) string { return "screen-family-node-" + fmt.Sprint(index) }
	parts := make([]devices.SemanticPart, 0, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		role := "housing_component"
		if node.Index == root.Index {
			role = "device_root"
		} else if node.Index == display.Index {
			role = "display_surface"
		}
		parent := ""
		if node.Parent != nil {
			if _, ok := nodes[*node.Parent]; !ok {
				return invalid(errors.New("screen-family node has an unavailable parent"))
			}
			parent = partID(*node.Parent)
		}
		parts = append(parts, devices.SemanticPart{ID: partID(node.Index), Role: role, ParentID: parent, NodeIndex: node.Index, NodeName: node.Name, LocalFrame: node.Frame})
	}
	if display.Mesh == nil || int(*display.Mesh) >= len(inventory.Meshes) {
		return invalid(errors.New("screen-family display node has no mesh"))
	}
	mesh := inventory.Meshes[*display.Mesh]
	if len(mesh.Primitives) == 0 {
		return invalid(errors.New("screen-family display has no primitive"))
	}
	primitive := mesh.Primitives[0]
	materialFound := false
	for _, candidate := range inventory.Materials {
		if candidate.Index == primitive.Material {
			materialFound = true
			break
		}
	}
	if !materialFound || (spec.sourceSHA == "" && (primitive.UVMin != [2]float64{} || primitive.UVMax != [2]float64{1, 1})) ||
		(spec.sourceSHA != "" && (primitive.UVMin != [2]float64{} || primitive.UVMax != [2]float64{1, 1})) {
		return invalid(errors.New("screen-family display primitive is not a bounded unit-UV surface"))
	}
	displayBounds, err := physicalGLBNodeMeshWorldBounds(inventory, display.Index)
	if err != nil {
		return invalid(fmt.Errorf("derive screen plane bounds: %w", err))
	}
	widthMM, heightMM := (displayBounds.max.X-displayBounds.min.X)*1000, (displayBounds.max.Y-displayBounds.min.Y)*1000
	if !positiveFinite(widthMM) || !positiveFinite(heightMM) {
		return invalid(errors.New("screen-family display metric must have finite positive spans"))
	}
	evidence := []devices.Evidence{
		{Source: spec.manifestPath + " SHA-256 " + spec.manifestSHA, Kind: "candidate_asset_manifest", Qualifier: "candidate-only source, rights, geometry, and use-scope record"},
		{Source: spec.sourceLabel, Kind: "source_and_design_basis", Qualifier: spec.attribution},
	}
	measurement := func(value float64, axis string) devices.Measurement {
		return devices.Measurement{Value: &value, Unit: "mm", Meaning: "GLB-derived full-source " + axis + " span", Configuration: spec.profile + " geometry profile", Method: "derived", Uncertainty: devices.Uncertainty{Status: "unknown", Explanation: "geometry bytes are not an independent physical measurement"}, Evidence: evidence}
	}
	displayEvidence := []devices.Evidence{{Source: spec.file + " SHA-256 " + spec.sha, Kind: "derived_display_geometry", Qualifier: "single explicit active screen primitive; no physical active-area or pixel calibration claim"}}
	definition := devices.Definition{SchemaVersion: "device-definition/v1", ID: spec.deviceID, Group: devices.Group(spec.groupKind), Kind: spec.kind,
		IdentityStatus: devices.IdentityConcept, Model: spec.model, Variant: spec.variant, Description: spec.description,
		Dimensions:      map[string]devices.Dimensions{"source_geometry_envelope": {Width: measurement((bounds.max.X-bounds.min.X)*1000, "X"), Height: measurement((bounds.max.Y-bounds.min.Y)*1000, "Y"), Depth: measurement((bounds.max.Z-bounds.min.Z)*1000, "Z")}},
		Displays:        []devices.DisplaySurface{{ID: spec.displayID, Topology: "planar", CoordinateMetric: &devices.DisplayMetric{WidthMM: widthMM, HeightMM: heightMM, Evidence: displayEvidence}, PhysicalBoundary: unknownRegion(), ActiveArea: unknownRegion(), AppSafeArea: unknownRegion(), Evidence: displayEvidence}},
		SupportedPoses:  []devices.Pose{{Name: "candidate-" + spec.profile, Status: "candidate_only", Evidence: evidence}},
		Supports:        []devices.SupportCapability{{Kind: familySupportKind(spec), InterfaceID: familySupportInterface(spec), Status: "candidate_only", Evidence: evidence}},
		ModelCandidates: []devices.ModelCandidate{{ID: spec.assetID, Status: "candidate_only", SHA256: ref.SHA256, Author: familyAuthor(spec), License: spec.license, Attribution: spec.attribution, RightsNotes: spec.description}},
		Provenance:      evidence, Qualification: "unqualified"}
	localSupport := identityTRS()
	localSupport.Translation = physical.Vec3{X: 0, Y: bounds.min.Y - root.Frame.Translation.Y, Z: bounds.min.Z - root.Frame.Translation.Z}
	primitiveIndex, materialIndex := primitive.Index, primitive.Material
	candidate := devices.DeviceCandidate{Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention,
		SourceToCanonical: identityTRS(), Parts: parts,
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: spec.displayID, PartID: partID(display.Index), LocalFrame: identityTRS(), ScreenPrimitiveIndex: &primitiveIndex, ScreenMaterialIndex: &materialIndex}},
		SupportFrames:   []devices.DeviceSupportFrame{{ID: familyDeviceSupportID(spec), InterfaceID: familySupportInterface(spec), PartID: partID(root.Index), LocalFrame: localSupport}}}
	resolved, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, spec.displayID)
	if err != nil || resolved.NodeIndex != display.Index || resolved.PrimitiveIndex != primitive.Index || resolved.MaterialIndex != primitive.Material || resolved.UVMin != [2]float64{} || resolved.UVMax != [2]float64{1, 1} {
		return invalid(fmt.Errorf("preflight screen-family display selector: mapping=%+v err=%w", resolved, err))
	}
	revision, err := saveDeviceCandidateIfUnchanged(ctx, s.devices, candidate)
	if err != nil {
		return invalid(err)
	}
	return revision, bounds, inventory, nil
}

func familySupportKind(spec screenFamilySpec) string {
	if spec.wall {
		return "wall_mount"
	}
	return "tabletop"
}

func familySupportInterface(spec screenFamilySpec) string {
	if spec.wall {
		return "vertical_surface"
	}
	return "horizontal_surface"
}

func familyDeviceSupportID(spec screenFamilySpec) string { return spec.deviceID + "-support" }

func familyAuthor(spec screenFamilySpec) string {
	if spec.sourceSHA != "" {
		return "Innerscene; adapted by Vrooli"
	}
	return "Vrooli"
}

func (s *starterInitializer) saveScreenFamilyScene(ctx context.Context, room scenes.Revision, device devices.DeviceRevision, spec screenFamilySpec, bounds physicalBounds) (scenes.Revision, ReviewedIPadPlacement, error) {
	if room.SceneID != laptopFramedSceneID || len(room.Candidate.CameraFrames) != 1 || room.Candidate.CameraFrames[0].ID != laptopFramedCameraID {
		return scenes.Revision{}, ReviewedIPadPlacement{}, errors.New("screen-family scene requires the pinned cup-free framed study room")
	}
	candidate := room.Candidate
	candidate.SceneID = spec.sceneID
	candidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), room.Candidate.CameraFrames...)
	camera := candidate.CameraFrames[0]
	camera.ID = spec.sceneID + "-camera-v1"
	camera.SensorFit = scenes.SensorFitHorizontal
	identity := identityTRS()
	placement := ReviewedIPadPlacement{SceneSupportID: laptopTableSupportID, SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: identity.Rotation, Scale: identity.Scale}, DeviceRootRest: identity}
	if spec.wall {
		candidate.SceneID = familyWallSceneID
		camera.ID = familyWallCameraID
		placement.SceneSupportWorld = physical.TRS{Translation: physical.Vec3{Y: 1.72, Z: -.46}, Rotation: identity.Rotation, Scale: identity.Scale}
		for _, part := range candidate.Parts {
			if part.ID == familyWallSupportPartID && part.NodeName == "RoomBackWall" {
				candidate.SupportFrames = append(append([]scenes.SceneSupportFrame(nil), candidate.SupportFrames...), scenes.SceneSupportFrame{
					ID: familyWallSupportID, InterfaceID: "vertical_surface", FramePartID: part.ID,
					LocalFrame:          physical.TRS{Translation: physical.Vec3{Z: .02}, Rotation: identity.Rotation, Scale: identity.Scale},
					ContactRegionPartID: part.ID, LoadDirectionLocal: &physical.Vec3{Z: 1},
				})
				break
			}
		}
		if !hasSceneSupport(candidate, familyWallSupportID) {
			return scenes.Revision{}, ReviewedIPadPlacement{}, errors.New("study room has no pinned vertical wall support part")
		}
		placement.SceneSupportID = familyWallSupportID
	}
	placement.DeviceSupportID = familyDeviceSupportID(spec)
	placement.SelectedCameraID = camera.ID
	placement.DisplayID = spec.displayID
	placement.DeviceRootRest.Translation.Y = -bounds.min.Y
	if spec.wall {
		placement.DeviceRootRest.Translation.Y = 0
		placement.DeviceRootRest.Translation.Z = 0
	}
	fitted, err := fitIPadPerspectiveCamera(bounds, identityTRS(), placement, camera, 16.0/9.0, .10)
	if err != nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, fmt.Errorf("fit %s camera to full candidate bounds: %w", spec.assetID, err)
	}
	camera.Frame = fitted
	target := physical.Vec3{X: (bounds.min.X + bounds.max.X) / 2, Y: placement.SceneSupportWorld.Translation.Y + (bounds.min.Y+bounds.max.Y)/2, Z: placement.SceneSupportWorld.Translation.Z + (bounds.min.Z+bounds.max.Z)/2}
	distance, err := watchCameraAxisDistance(camera.Frame, target)
	if err != nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, err
	}
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusDistance, DistanceM: &distance}
	candidate.CameraFrames[0] = camera
	scene, err := saveSceneCandidateIfUnchanged(ctx, s.scenes, candidate)
	return scene, placement, err
}

func hasSceneSupport(candidate scenes.SceneCandidate, id string) bool {
	for _, support := range candidate.SupportFrames {
		if support.ID == id {
			return true
		}
	}
	return false
}

func (s *starterInitializer) ensureScreenFamilyPreset(ctx context.Context, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedIPadPlacement, spec screenFamilySpec) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{
		Device:    &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256},
		Scene:     &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256},
		Motion:    &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256},
		DisplayId: placement.DisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.SelectedCameraID,
		SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest),
		StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1},
	}
	if current, err := s.presets.Head(ctx, spec.presetID); err == nil {
		if current.Name == spec.presetName && proto.Equal(current.Setup, setup) {
			return current, nil
		}
		return shots.PhysicalPresetRevision{}, fmt.Errorf("screen-family preset %q differs from its fixed candidate; refusing implicit revision: %w", spec.presetID, shots.ErrPhysicalPresetConflict)
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, spec.presetID, 0, spec.presetName, setup)
}
