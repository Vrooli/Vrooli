package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"device-studio/internal/composition"
	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
)

const (
	laptopCandidateBundleID = "device-studio-laptop-derive16-candidate-v1"
	laptopDeviceID          = "jack-baeten-macbook-pro-m3-16-derive16-candidate"
	laptopSceneID           = "laptop-study-room-no-cup-candidate"
	laptopMotionID          = "laptop-lid-open-to-closed-derive16-candidate"
	laptopDeviceAssetID     = "jack-baeten-laptop-neutral-rig-glb"
	laptopSceneAssetID      = "laptop-study-room-no-cup-glb"
	laptopJointID           = "laptop_lid_hinge"
	laptopDisplayID         = "main_display"
	laptopRootPartID        = "laptop-root"
	laptopBasePartID        = "laptop-base"
	laptopBaseGeometryID    = "laptop-base-geometry"
	laptopReviewPartID      = "laptop-hinge-review"
	laptopReviewGeometryID  = "laptop-hinge-review-geometry"
	laptopLidPivotPartID    = "laptop-lid-pivot"
	laptopLidGeometryID     = "laptop-lid-geometry"
	laptopScreenPartID      = "laptop-screen-surface"
	laptopTableSupportID    = "laptop-tabletop"
	laptopCameraID          = "laptop-candidate-camera"
	laptopFramedSceneID     = "laptop-study-room-no-cup-framed-candidate-v2"
	laptopFramedCameraID    = "laptop-candidate-camera-framed-v2"
	laptopGeometrySHA256    = "3005e1e8b12e2017565292712aaad20dfb35cda2c42519a937d2c74afb6f5925"
	laptopRoomSHA256        = "68a8ad34f5af76f681e9aab7337ce80514221bda333c6713b58f624556660f56"
	laptopSourceSHA256      = "4b7a6736e12e105b00556ca6397f3dffe559a1c518954209df08828f186d8a78"
	laptopRoomSourceSHA256  = "c0f1450af8cc83b237159f2c5208bd964c48fc5dc9fe7f2ac2b2ee5409447f4d"
	laptopJointPivotY       = -0.0007076454348862171
	laptopJointPivotZ       = -0.12193954735994339
	laptopClosedAngleRad    = 1.919
)

// LaptopStarterState exposes the immutable owner revisions needed by an
// explicit local review render. The candidate and room remain unqualified.
type LaptopStarterState struct {
	Bundle          library.BundleRevision
	Device          devices.DeviceRevision
	Scene           scenes.Revision
	FramedScene     scenes.Revision
	Motion          motions.Revision
	Placement       LaptopCandidatePlacement
	FramedPlacement LaptopCandidatePlacement
	Preset          shots.PhysicalPresetRevision
}

type LaptopCandidatePlacement struct {
	DisplayID         string
	DeviceSupportID   string
	SceneSupportID    string
	CameraID          string
	SceneSupportWorld physical.TRS
	SupportPivotM     physical.Vec3
	DeviceRootRest    physical.TRS
}

// InitializeReviewedLaptopStarter publishes the fixed derive16 bytes and
// saves a candidate-only device, room, and motion through their owning services.
func (o Owners) InitializeReviewedLaptopStarter(ctx context.Context) (LaptopStarterState, error) {
	if o.Starter == nil {
		return LaptopStarterState{}, errors.New("laptop candidate initializer is unavailable")
	}
	return o.Starter.initializeLaptopCandidate(ctx)
}

func (s *starterInitializer) initializeLaptopCandidate(ctx context.Context) (LaptopStarterState, error) {
	ctx, err := s.pinRoute(ctx)
	if err != nil {
		return LaptopStarterState{}, err
	}
	return s.initializeLaptopCandidatePinned(ctx)
}

func (s *starterInitializer) initializeLaptopCandidatePinned(ctx context.Context) (LaptopStarterState, error) {
	files := []starterFile{
		{ID: laptopDeviceAssetID, File: "assets/laptop/laptop-neutral-rig.glb", MIME: "model/gltf-binary", SHA: laptopGeometrySHA256, Role: library.RoleGeometryGLB, License: "CC BY 4.0", Attribution: "MacBook Pro M3 16 inch 2024 creator model by Jack Baeten, adapted under CC BY 4.0; neutral screen and candidate-only part/joint interpretation. Not manufacturer CAD or endorsement.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "Jack Baeten MacBook Pro M3 16 inch 2024 creator model; derive16 neutral-screen candidate with display-only material and UV adaptations.", SourceSHA256: laptopSourceSHA256, Adaptation: "Derivation chain: source GLB 4b7a6736e12e105b00556ca6397f3dffe559a1c518954209df08828f186d8a78 -> derive16 candidate c3eef69aa2a0c88a444a5736f5bee4eb62ba6f5228d2e95ff15f897e78e51651 -> final laptop-neutral-rig.glb 3005e1e8b12e2017565292712aaad20dfb35cda2c42519a937d2c74afb6f5925. The material-only stage removes emissiveFactor, emissiveTexture, and KHR_materials_emissive_strength from the project-replaced display slot. The final stage appends a private UV buffer view/accessor and affine-normalizes only the exclusive display UV0 chart from its measured inset bounds to [0,1]; the preceding GLB BIN bytes remain an exact prefix, not the entire final BIN. Positions, normals, indices, other node transforms, and non-screen material JSON remain unchanged. Keep candidate hinge hierarchy/fitted pivot; unresolved trim and continuous clearance remain unqualified."}},
		{ID: laptopSceneAssetID, File: "assets/laptop-study-environment/laptop-study-environment-no-cup.glb", MIME: "model/gltf-binary", SHA: laptopRoomSHA256, Role: library.RoleGeometryGLB, License: "AGPL-3.0-only", Attribution: "Vrooli-authored room/table candidate derived from the phone-study environment; cup props omitted from the active scene. Not measured room or furniture CAD.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "Vrooli-authored phone-study room/table scene; cup-free candidate derivative.", SourceSHA256: laptopRoomSourceSHA256, Adaptation: "Derive from phone-study scene c0f1450af8cc83b237159f2c5208bd964c48fc5dc9fe7f2ac2b2ee5409447f4d to laptop-study-environment-no-cup.glb 68a8ad34f5af76f681e9aab7337ce80514221bda333c6713b58f624556660f56. Remove the three cup leaf nodes TeaCup, TeaCupHandle, and TeaSurface from the node array and active scene, remap retained node indices, and preserve mesh/material/accessor arrays and the complete BIN chunk."}},
	}
	manifest, sources, err := s.buildManifest(ctx, laptopCandidateBundleID, files)
	if err != nil {
		return LaptopStarterState{}, err
	}
	bundle, err := s.publish(ctx, manifest, sources, "device-studio-e3-laptop-derive16-local-candidate-v1")
	if err != nil {
		return LaptopStarterState{}, err
	}
	device, err := s.saveLaptopDevice(ctx, bundle)
	if err != nil {
		return LaptopStarterState{}, err
	}
	scene, err := s.saveLaptopScene(ctx, bundle)
	if err != nil {
		return LaptopStarterState{}, err
	}
	motion, err := s.saveLaptopMotion(ctx)
	if err != nil {
		return LaptopStarterState{}, err
	}
	placement, err := laptopCandidatePlacement(device, scene, motion)
	if err != nil {
		return LaptopStarterState{}, err
	}
	framedScene, err := s.saveLaptopFramedScene(ctx, scene, device, motion)
	if err != nil {
		return LaptopStarterState{}, err
	}
	framedPlacement, err := laptopFramedCandidatePlacement(device, framedScene, motion)
	if err != nil {
		return LaptopStarterState{}, err
	}
	return LaptopStarterState{Bundle: bundle, Device: device, Scene: scene, FramedScene: framedScene, Motion: motion, Placement: placement, FramedPlacement: framedPlacement}, nil
}

func (s *starterInitializer) saveLaptopDevice(ctx context.Context, bundle library.BundleRevision) (devices.DeviceRevision, error) {
	ref := artifactRef(bundle, laptopDeviceAssetID)
	data, err := fs.ReadFile(s.fs, "assets/laptop/laptop-neutral-rig.glb")
	if err != nil {
		return devices.DeviceRevision{}, err
	}
	if len(data) != 4638060 || sha256Hex(data) != laptopGeometrySHA256 || ref.SHA256 != laptopGeometrySHA256 {
		return devices.DeviceRevision{}, errors.New("embedded derive16 laptop bytes differ from their fixed Library artifact")
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("inspect derive16 laptop candidate: %w", err)
	}
	nodes := make(map[string]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodes[node.Name] = node
	}
	root, rootOK := nodes["LaptopStarterRoot"]
	base, baseOK := nodes["LaptopBaseCandidate"]
	baseGeometry, baseGeometryOK := nodes["LaptopBaseGeometry"]
	review, reviewOK := nodes["LaptopHingeReview"]
	reviewGeometry, reviewGeometryOK := nodes["LaptopHingeReviewGeometry"]
	pivot, pivotOK := nodes["LaptopLidPivotCandidate"]
	lid, lidOK := nodes["LaptopLidGeometry"]
	screen, screenOK := nodes["LaptopScreenSurface"]
	if !rootOK || !baseOK || !baseGeometryOK || !reviewOK || !reviewGeometryOK || !pivotOK || !lidOK || !screenOK ||
		root.Parent != nil || base.Parent == nil || *base.Parent != root.Index || pivot.Parent == nil || *pivot.Parent != root.Index ||
		review.Parent == nil || *review.Parent != root.Index || lid.Parent == nil || *lid.Parent != pivot.Index ||
		screen.Parent == nil || *screen.Parent != pivot.Index || baseGeometry.Parent == nil || *baseGeometry.Parent != base.Index ||
		reviewGeometry.Parent == nil || *reviewGeometry.Parent != review.Index || pivot.Frame.Translation != (physical.Vec3{Y: laptopJointPivotY, Z: laptopJointPivotZ}) {
		return devices.DeviceRevision{}, errors.New("derive16 laptop node names, hierarchy, or fitted hinge pivot differ from the pinned candidate inventory")
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("derive laptop candidate bounds: %w", err)
	}
	screenBounds, err := physicalGLBNodeMeshWorldBounds(inventory, screen.Index)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("derive laptop screen metrics: %w", err)
	}
	widthMM := (bounds.max.X - bounds.min.X) * 1000
	heightMM := (bounds.max.Y - bounds.min.Y) * 1000
	depthMM := (bounds.max.Z - bounds.min.Z) * 1000
	screenWidthMM := (screenBounds.max.X - screenBounds.min.X) * 1000
	screenHeightMM := (screenBounds.max.Y - screenBounds.min.Y) * 1000
	if !positiveFinite(widthMM) || !positiveFinite(heightMM) || !positiveFinite(depthMM) || !positiveFinite(screenWidthMM) || !positiveFinite(screenHeightMM) {
		return devices.DeviceRevision{}, errors.New("derive16 laptop bounds or display metrics are invalid")
	}
	evidence := []devices.Evidence{{Source: "E3 laptop derive16 inventory and exact candidate GLB SHA-256 " + laptopGeometrySHA256, Kind: "derived_asset_geometry", Qualifier: "creator-model geometry values only; not independently measured laptop dimensions or physical qualification"}}
	parts := []devices.SemanticPart{
		{ID: laptopRootPartID, Role: "device_root", NodeIndex: root.Index, NodeName: root.Name, LocalFrame: root.Frame},
		{ID: laptopBasePartID, Role: "base", ParentID: laptopRootPartID, NodeIndex: base.Index, NodeName: base.Name, LocalFrame: base.Frame},
		{ID: laptopBaseGeometryID, Role: "base_geometry", ParentID: laptopBasePartID, NodeIndex: baseGeometry.Index, NodeName: baseGeometry.Name, LocalFrame: baseGeometry.Frame},
		{ID: laptopReviewPartID, Role: "unresolved_hinge_review", ParentID: laptopRootPartID, NodeIndex: review.Index, NodeName: review.Name, LocalFrame: review.Frame},
		{ID: laptopReviewGeometryID, Role: "unresolved_hinge_geometry", ParentID: laptopReviewPartID, NodeIndex: reviewGeometry.Index, NodeName: reviewGeometry.Name, LocalFrame: reviewGeometry.Frame},
		{ID: laptopLidPivotPartID, Role: "hinge_child", ParentID: laptopRootPartID, NodeIndex: pivot.Index, NodeName: pivot.Name, LocalFrame: pivot.Frame},
		{ID: laptopLidGeometryID, Role: "lid_geometry", ParentID: laptopLidPivotPartID, NodeIndex: lid.Index, NodeName: lid.Name, LocalFrame: lid.Frame},
		{ID: laptopScreenPartID, Role: "display_surface", ParentID: laptopLidPivotPartID, NodeIndex: screen.Index, NodeName: screen.Name, LocalFrame: screen.Frame},
	}
	lower, upper := 0.0, laptopClosedAngleRad
	definition := devices.Definition{
		SchemaVersion: "device-definition/v1", ID: laptopDeviceID, Group: devices.GroupLaptop, Kind: "generic_laptop",
		IdentityStatus: devices.IdentityGeneric, Model: "Jack Baeten MacBook Pro M3 16 inch 2024 creator model",
		Variant:     "derive16 neutral-screen candidate; heuristic base/lid ownership and fitted hinge remain unqualified",
		Description: "A source-attributed creator model adapted for local candidate review. Not manufacturer CAD, an independently measured laptop, or manufacturer-endorsed.",
		Dimensions: map[string]devices.Dimensions{"source_geometry": {
			Width: laptopGeometryMeasurement(widthMM, "source GLB world-space X span"), Height: laptopGeometryMeasurement(heightMM, "source GLB world-space Y span"), Depth: laptopGeometryMeasurement(depthMM, "source GLB world-space Z span"),
		}},
		Displays:        []devices.DisplaySurface{{ID: laptopDisplayID, Topology: "planar", CoordinateMetric: &devices.DisplayMetric{WidthMM: screenWidthMM, HeightMM: screenHeightMM, Evidence: evidence}, PhysicalBoundary: unknownRegion(), ActiveArea: unknownRegion(), AppSafeArea: unknownRegion()}},
		Joints:          []devices.Joint{{ID: laptopJointID, ParentPart: laptopRootPartID, ChildPart: laptopLidPivotPartID, Axis: &devices.Vector3{X: 1}, PivotMeters: &devices.Vector3{Y: laptopJointPivotY, Z: laptopJointPivotZ}, LowerRadians: &lower, UpperRadians: &upper, Status: "candidate_only", Evidence: []devices.Evidence{{Source: "E3 derive16 screen/deck-plane fit", Kind: "fitted_hinge_candidate", Qualifier: "X-parallel screen/deck visual fit only; not a measured mechanism, official travel limit, or continuous-clearance result"}}}},
		SupportedPoses:  []devices.Pose{{Name: "source_open", Status: "candidate_only", JointState: map[string]float64{laptopJointID: 0}}, {Name: "intermediate", Status: "candidate_only", JointState: map[string]float64{laptopJointID: laptopClosedAngleRad / 2}}, {Name: "near_closed", Status: "candidate_only", JointState: map[string]float64{laptopJointID: laptopClosedAngleRad}}},
		SupportedMotion: []devices.Capability{{Name: "laptop_lid_hinge_review", Status: "candidate_only", MotionKind: "articulated", JointIDs: []string{laptopJointID}, Evidence: evidence}},
		Supports:        []devices.SupportCapability{{Kind: "tabletop", InterfaceID: "horizontal_surface", Status: "candidate_only", Evidence: evidence}},
		ModelCandidates: []devices.ModelCandidate{{ID: "jack-baeten-macbook-pro-m3-16-derived-neutral-screen", Status: "candidate_only", SHA256: laptopGeometrySHA256, ListingURL: "https://sketchfab.com/3d-models/macbook-pro-m3-16-inch-2024-8e34fc2b303144f78490007d91ff57c4", Author: "Jack Baeten", License: "CC BY 4.0", Attribution: "MacBook Pro M3 16 inch 2024 by Jack Baeten; adapted with neutral display and candidate-only hinge hierarchy.", RightsNotes: "CC BY 4.0 attribution required. Creator model adaptation only; no manufacturer CAD or endorsement."}},
		Provenance:      []devices.Evidence{{Source: "https://sketchfab.com/3d-models/macbook-pro-m3-16-inch-2024-8e34fc2b303144f78490007d91ff57c4", Kind: "creator_listing", Qualifier: "Jack Baeten; CC BY 4.0; creator attribution retained; not a manufacturer source"}, {Source: "E3 scene-studies/laptop derive16 retained source/inventory receipts", Kind: "source_geometry_and_derivation", Qualifier: "Neutral-screen derivative, candidate semantic role split and fitted hinge; unresolved rear components and protrusions remain unqualified"}},
		Qualification:   "unqualified",
	}
	candidate := devices.DeviceCandidate{
		Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention,
		SourceToCanonical: identityTRS(), Parts: parts,
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: laptopDisplayID, PartID: laptopScreenPartID, LocalFrame: identityTRS()}},
		SupportFrames:   []devices.DeviceSupportFrame{{ID: "laptop-root-support", InterfaceID: "horizontal_surface", PartID: laptopRootPartID, LocalFrame: identityTRS()}},
	}
	if _, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, laptopDisplayID); err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("preflight derive16 laptop screen mapping: %w", err)
	}
	return saveDeviceCandidateIfUnchanged(ctx, s.devices, candidate)
}

func (s *starterInitializer) saveLaptopScene(ctx context.Context, bundle library.BundleRevision) (scenes.Revision, error) {
	ref := artifactRef(bundle, laptopSceneAssetID)
	data, err := fs.ReadFile(s.fs, "assets/laptop-study-environment/laptop-study-environment-no-cup.glb")
	if err != nil {
		return scenes.Revision{}, err
	}
	if len(data) != 420960 || sha256Hex(data) != laptopRoomSHA256 || ref.SHA256 != laptopRoomSHA256 {
		return scenes.Revision{}, errors.New("embedded laptop room bytes differ from their fixed Library artifact")
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("inspect laptop room candidate: %w", err)
	}
	active := make(map[uint32]bool, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		active[node.Index] = true
		if node.Name == "TeaCup" || node.Name == "TeaCupHandle" || node.Name == "TeaSurface" {
			return scenes.Revision{}, errors.New("cup-free laptop room contains a retained cup node")
		}
	}
	materialID := func(index uint32) string { return fmt.Sprintf("laptop-room-material-%d", index) }
	materials := make([]scenes.SceneMaterial, 0, len(inventory.Materials))
	for _, item := range inventory.Materials {
		material := scenes.SceneMaterial{ID: materialID(item.Index), SurfaceProfile: scenes.MaterialPrincipledV1, BaseColorLinearRGBA: item.BaseColor, Metallic: item.Metallic, Roughness: item.Roughness, AlphaMode: scenes.AlphaMode(item.AlphaMode)}
		if item.Name == "Soft luminous window" {
			material.SurfaceProfile = scenes.MaterialEmissionV1
			material.BaseColorLinearRGBA = [4]float64{.624, .656, .704, 1}
			material.Metallic, material.Roughness = 0, 1
		}
		materials = append(materials, material)
	}
	partID := func(index uint32) string { return fmt.Sprintf("laptop-scene-part-%d", index) }
	parts := make([]scenes.ScenePart, 0, len(active))
	for _, node := range inventory.Nodes {
		if !active[node.Index] {
			continue
		}
		role := "scene_geometry"
		if node.Name == "TableTop" {
			role = "support_surface"
		}
		parent := ""
		if node.Parent != nil {
			parent = partID(*node.Parent)
		}
		parts = append(parts, scenes.ScenePart{ID: partID(node.Index), Role: role, ArtifactBindingID: "laptop-room-geometry", ParentID: parent, NodeIndex: node.Index, NodeName: node.Name, LocalFrame: node.Frame})
	}
	slots := make([]scenes.SceneMaterialSlot, 0)
	for _, node := range inventory.Nodes {
		if !active[node.Index] || node.Mesh == nil {
			continue
		}
		mesh := inventory.Meshes[*node.Mesh]
		for _, primitive := range mesh.Primitives {
			slots = append(slots, scenes.SceneMaterialSlot{PartID: partID(node.Index), PrimitiveIndex: primitive.Index, SlotIndex: primitive.Material, MaterialID: materialID(primitive.Material)})
		}
	}
	identity := identityTRS()
	candidate := scenes.SceneCandidate{
		Version: scenes.CandidateVersion, SceneID: laptopSceneID, CoordinateConvention: physical.CoordinateConvention,
		Artifacts: []scenes.SceneArtifactBinding{{ID: "laptop-room-geometry", Ref: ref, SourceUnit: "m", SourceToCanonical: ptrTRS(identity)}},
		Materials: materials, MaterialSlots: slots, Parts: parts,
		SupportFrames: []scenes.SceneSupportFrame{{ID: laptopTableSupportID, InterfaceID: "horizontal_surface", FramePartID: partID(5), LocalFrame: physical.TRS{Translation: physical.Vec3{Y: .0170000196}, Rotation: identity.Rotation, Scale: identity.Scale}, ContactRegionPartID: partID(5), LoadDirectionLocal: &physical.Vec3{Y: -1}}},
		CameraFrames:  []scenes.SceneCameraFrame{{ID: laptopCameraID, Frame: lookAtFrame(physical.Vec3{X: .43, Y: 1.12, Z: .72}, physical.Vec3{Y: .88}), Projection: scenes.ProjectionPerspective, FocalLengthMM: floatPtr(50), SensorWidthMM: floatPtr(36), SensorHeightMM: floatPtr(24), SensorFit: scenes.SensorFitHorizontal, NearClipM: .1, FarClipM: 1000, Focus: &scenes.FocusTarget{Mode: scenes.FocusSelectedDevice}, DOFFNumber: floatPtr(8)}},
		LightFrames:   starterLights(), Environment: scenes.SceneEnvironment{Model: "CONSTANT_V1", LinearRGB: [3]float64{.53, .56, .60}, Strength: .2},
	}
	if _, err := rendering.ValidateScenePhysicalGLB(data, candidate); err != nil {
		return scenes.Revision{}, fmt.Errorf("preflight cup-free laptop room candidate: %w", err)
	}
	return saveSceneCandidateIfUnchanged(ctx, s.scenes, candidate)
}

func (s *starterInitializer) saveLaptopFramedScene(ctx context.Context, base scenes.Revision, device devices.DeviceRevision, motion motions.Revision) (scenes.Revision, error) {
	if base.SceneID != laptopSceneID || device.DeviceID != laptopDeviceID || motion.MotionID != laptopMotionID {
		return scenes.Revision{}, errors.New("framed laptop scene is bound to unexpected immutable candidates")
	}
	textureBundle, err := s.publishSceneBundle(ctx)
	if err != nil {
		return scenes.Revision{}, err
	}
	data, err := fs.ReadFile(s.fs, "assets/laptop-study-environment/laptop-study-environment-no-cup.glb")
	if err != nil {
		return scenes.Revision{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return scenes.Revision{}, err
	}
	tableIndex, tableOK := namedMaterialIndex(inventory.Materials, "Travertine001 top material slot")
	windowIndex, windowOK := namedMaterialIndex(inventory.Materials, "Soft luminous window")
	if !tableOK || !windowOK {
		return scenes.Revision{}, errors.New("framed laptop room lacks the pinned table or window material")
	}
	candidate := base.Candidate
	candidate.SceneID = laptopFramedSceneID
	candidate.Artifacts = append([]scenes.SceneArtifactBinding(nil), base.Candidate.Artifacts...)
	for _, id := range []string{"travertine-color", "travertine-normal-gl", "travertine-roughness", "travertine-height"} {
		ref := artifactRef(textureBundle, id)
		if ref.ArtifactID == "" || ref.Role != library.RoleTextureImage {
			return scenes.Revision{}, fmt.Errorf("framed laptop texture %q is not pinned by the shared scene bundle", id)
		}
		candidate.Artifacts = append(candidate.Artifacts, scenes.SceneArtifactBinding{ID: id, Ref: ref})
	}
	materialID := func(index uint32) string { return fmt.Sprintf("laptop-room-material-%d", index) }
	candidate.Materials = starterTableMaterials(inventory.Materials, materialID, tableIndex, windowIndex)
	candidate.MaterialSlots = append([]scenes.SceneMaterialSlot(nil), base.Candidate.MaterialSlots...)
	candidate.Parts = append([]scenes.ScenePart(nil), base.Candidate.Parts...)
	candidate.SupportFrames = append([]scenes.SceneSupportFrame(nil), base.Candidate.SupportFrames...)
	candidate.LightFrames = append([]scenes.SceneLightFrame(nil), base.Candidate.LightFrames...)
	candidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), base.Candidate.CameraFrames...)
	if len(candidate.CameraFrames) != 1 || candidate.CameraFrames[0].ID != laptopCameraID {
		return scenes.Revision{}, errors.New("framed laptop source camera differs from the fixed candidate")
	}
	camera := candidate.CameraFrames[0]
	camera.ID = laptopFramedCameraID
	camera.SensorFit = scenes.SensorFitHorizontal
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusSelectedDevice}
	placement, err := laptopCandidatePlacement(device, base, motion)
	if err != nil {
		return scenes.Revision{}, err
	}
	deviceData, err := fs.ReadFile(s.fs, "assets/laptop/laptop-neutral-rig.glb")
	if err != nil || sha256Hex(deviceData) != laptopGeometrySHA256 {
		return scenes.Revision{}, errors.New("framed laptop sweep source differs from the pinned device candidate")
	}
	deviceInventory, err := rendering.InspectPhysicalGLB(deviceData)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("inspect laptop geometry for framed camera: %w", err)
	}
	bounds, err := physicalGLBJointSweepBounds(deviceInventory, device.Candidate, motion.Candidate, laptopJointID)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("bound complete laptop hinge sweep for camera fit: %w", err)
	}
	fitted, err := fitIPadPerspectiveCamera(bounds, device.Candidate.SourceToCanonical, ReviewedIPadPlacement{SceneSupportWorld: placement.SceneSupportWorld, SupportPivotM: placement.SupportPivotM, DeviceRootRest: placement.DeviceRootRest}, camera, 16.0/9.0, .10)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("fit framed laptop camera to full hinge sweep: %w", err)
	}
	camera.Frame = fitted
	candidate.CameraFrames[0] = camera
	if _, err := rendering.ValidateScenePhysicalGLB(data, candidate); err != nil {
		return scenes.Revision{}, fmt.Errorf("preflight framed laptop scene: %w", err)
	}
	return saveSceneCandidateIfUnchanged(ctx, s.scenes, candidate)
}

func (s *starterInitializer) saveLaptopMotion(ctx context.Context) (motions.Revision, error) {
	open, middle, closed := 0.0, laptopClosedAngleRad/2, laptopClosedAngleRad
	candidate := motions.MotionCandidate{
		Version: motions.CandidateVersion, MotionID: laptopMotionID, Timebase: composition.Rational{Num: 24, Den: 1}, DurationTicks: 48,
		RequiredCapabilities: []string{"laptop_lid_hinge_review"},
		Tracks:               []motions.MotionTrack{{Target: motions.TrackTarget{JointID: laptopJointID}, Interpolation: motions.Linear, Keys: []motions.MotionKey{{Tick: 0, AngleRad: &open}, {Tick: 24, AngleRad: &middle}, {Tick: 47, AngleRad: &closed}, {Tick: 48, AngleRad: &closed}}}},
	}
	return saveLaptopMotionCandidate(ctx, s.motions, candidate)
}

func saveDeviceCandidateIfUnchanged(ctx context.Context, owner *devices.DeviceRevisionService, candidate devices.DeviceCandidate) (devices.DeviceRevision, error) {
	existing, err := owner.GetHead(ctx, candidate.Definition.ID)
	if err == nil {
		_, _, digest, canonErr := devices.CanonicalizeDeviceCandidate(candidate)
		if canonErr != nil {
			return devices.DeviceRevision{}, canonErr
		}
		if existing.SHA256 == digest {
			return existing, nil
		}
		return devices.DeviceRevision{}, fmt.Errorf("device candidate head differs; refusing implicit revision: %w", devices.ErrRevisionConflict)
	}
	if !errors.Is(err, devices.ErrRevisionNotFound) {
		return devices.DeviceRevision{}, err
	}
	return owner.SaveCandidate(ctx, candidate, 0)
}

func saveSceneCandidateIfUnchanged(ctx context.Context, owner *scenes.RevisionService, candidate scenes.SceneCandidate) (scenes.Revision, error) {
	_, _, digest, err := scenes.CanonicalizeCandidate(candidate)
	if err != nil {
		return scenes.Revision{}, err
	}
	existing, err := owner.GetHead(ctx, candidate.SceneID)
	if err == nil {
		if existing.SHA256 == digest {
			return existing, nil
		}
		return scenes.Revision{}, fmt.Errorf("scene candidate head differs; refusing implicit revision: %w", scenes.ErrRevisionConflict)
	}
	if !errors.Is(err, scenes.ErrRevisionNotFound) {
		return scenes.Revision{}, err
	}
	return owner.SaveCandidate(ctx, candidate, 0)
}

func saveLaptopMotionCandidate(ctx context.Context, owner *motions.RevisionService, candidate motions.MotionCandidate) (motions.Revision, error) {
	_, _, digest, err := motions.CanonicalizeCandidate(candidate)
	if err != nil {
		return motions.Revision{}, err
	}
	existing, err := owner.GetHead(ctx, candidate.MotionID)
	if err == nil {
		if existing.SHA256 == digest {
			return existing, nil
		}
		return motions.Revision{}, fmt.Errorf("laptop motion head differs; refusing implicit revision: %w", motions.ErrRevisionConflict)
	}
	if !errors.Is(err, motions.ErrRevisionNotFound) {
		return motions.Revision{}, err
	}
	return owner.SaveCandidate(ctx, candidate, 0)
}

func laptopCandidatePlacement(device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision) (LaptopCandidatePlacement, error) {
	return laptopPlacementForScene(device, scene, motion, laptopSceneID, laptopCameraID)
}

func laptopPlacementForScene(device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, sceneID, cameraID string) (LaptopCandidatePlacement, error) {
	if device.DeviceID != laptopDeviceID || scene.SceneID != sceneID || motion.MotionID != laptopMotionID {
		return LaptopCandidatePlacement{}, errors.New("laptop candidate placement is bound to different immutable owner revisions")
	}
	rootSupport, display := false, false
	for _, frame := range device.Candidate.SupportFrames {
		rootSupport = rootSupport || frame.ID == "laptop-root-support" && frame.PartID == laptopRootPartID
	}
	for _, binding := range device.Candidate.DisplayBindings {
		display = display || binding.DisplayID == laptopDisplayID && binding.PartID == laptopScreenPartID
	}
	sceneSupport, camera := false, false
	for _, frame := range scene.Candidate.SupportFrames {
		sceneSupport = sceneSupport || frame.ID == laptopTableSupportID
	}
	for _, frame := range scene.Candidate.CameraFrames {
		camera = camera || frame.ID == cameraID
	}
	if !rootSupport || !display || !sceneSupport || !camera {
		return LaptopCandidatePlacement{}, errors.New("laptop placement IDs are absent from pinned candidate revisions")
	}
	// The fitted model has a derived underside extent below its root origin.
	// This candidate placement aligns its lowest source bound to the table plane;
	// it does not claim a measured device support/contact interface.
	return LaptopCandidatePlacement{DisplayID: laptopDisplayID, DeviceSupportID: "laptop-root-support", SceneSupportID: laptopTableSupportID, CameraID: cameraID,
		SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
		SupportPivotM:     physical.Vec3{Y: .01068804}, DeviceRootRest: identityTRS()}, nil
}

func laptopFramedCandidatePlacement(device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision) (LaptopCandidatePlacement, error) {
	return laptopPlacementForScene(device, scene, motion, laptopFramedSceneID, laptopFramedCameraID)
}

func laptopGeometryMeasurement(valueMM float64, axis string) devices.Measurement {
	value := valueMM
	return devices.Measurement{Value: &value, Unit: "mm", Meaning: "derive16 creator GLB " + axis, Method: "derived", Uncertainty: devices.Uncertainty{Status: "unknown"}, Evidence: []devices.Evidence{{Source: "embedded laptop-neutral-rig.glb SHA-256 " + laptopGeometrySHA256, Kind: "derived_asset_geometry", Qualifier: "creator-model geometry span only; not measured hardware or qualification"}}}
}
