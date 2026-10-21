package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"

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
	monitorAssetID         = "innerscene-ultrawide-monitor-screen-uv-v1"
	monitorAssetFile       = "assets/monitor/monitor-screen-uv-v1.glb"
	monitorAssetSHA256     = "eff6e9a20a4fd26003046e7f0c7b5d0ea737d83d4ce9adb66b1c118561155266"
	monitorSourceSHA256    = "67e5c979d6755dd561c291f39ae5748849cb26f7bb78f97821166ecbd05e2bed"
	monitorSourceManifest  = "507989d4027b4a31f18629b03fa67c12eaed5c30bdab5ef8246f716e40c10e54"
	monitorBundleID        = "device-studio-starter-innerscene-ultrawide-monitor"
	monitorDeviceID        = "innerscene-ultrawide-monitor-generic"
	monitorSceneID         = "innerscene-monitor-study-no-cup"
	monitorPresetID        = "innerscene-monitor-study-hero"
	monitorDisplayID       = "main_display"
	monitorRootPartID      = "monitor-root"
	monitorDisplayPartID   = "monitor-display-glass"
	monitorDeviceSupportID = "monitor-root-support"
	monitorCameraID        = "monitor-camera-full-bounds-v1"
	monitorReviewID        = "device-studio-e3-starter-local-render-review-v1-innerscene-monitor"
	monitorV2AssetID       = "innerscene-ultrawide-monitor-screen-split-v2"
	monitorV2AssetFile     = "assets/monitor/monitor-screen-split-v2.glb"
	monitorV2AssetSHA256   = "d077d0e5433f8058f0c7d09c7dfae423af6412ffdedeb6f2af34a2f7e766deb9"
	monitorV2ManifestSHA   = "d9718414b087fe098c6877bece031554e997ea8daa28f851fe446ca45ac1020a"
	monitorV2BundleID      = "device-studio-starter-innerscene-ultrawide-monitor-screenmap-v2"
	monitorV2DeviceID      = "innerscene-ultrawide-monitor-generic-screenmap-v2"
	monitorV2SceneID       = "innerscene-monitor-study-no-cup-screenmap-v2"
	monitorV2PresetID      = "innerscene-monitor-study-hero-screenmap-v2"
	monitorV2ReviewID      = "device-studio-e3-starter-local-render-review-v1-innerscene-monitor-screenmap-v2"
	monitorV2CameraID      = "monitor-camera-full-bounds-screenmap-v2"
	monitorV2SupportID     = "monitor-root-support-screenmap-v2"
	monitorV2ManifestPath  = "scenarios/device-studio/assets/sources/innerscene-ultrawide-monitor-screenmap-v2/monitor-screen-split-v2-manifest.json"
)

type monitorSeedConfig struct {
	AssetID, AssetFile, AssetSHA, BundleID, DeviceID, SceneID, PresetID  string
	ReviewID, SourceManifestSHA, SourceManifestPath, CameraID, SupportID string
	DisplayPartID, RootPartID, DisplayID                                 string
	DefinitionDescription, PresetName, Adaptation, ProvenanceQualifier   string
	ScreenPrimitiveIndex, ScreenMaterialIndex                            *uint32
}

func monitorV1Seed() monitorSeedConfig {
	return monitorSeedConfig{AssetID: monitorAssetID, AssetFile: monitorAssetFile, AssetSHA: monitorAssetSHA256, BundleID: monitorBundleID,
		DeviceID: monitorDeviceID, SceneID: monitorSceneID, PresetID: monitorPresetID, ReviewID: monitorReviewID,
		SourceManifestSHA: monitorSourceManifest, SourceManifestPath: "scenarios/device-studio/assets/sources/innerscene-ultrawide-monitor/monitor-source-manifest.json",
		CameraID: monitorCameraID, SupportID: monitorDeviceSupportID, DisplayPartID: monitorDisplayPartID, RootPartID: monitorRootPartID, DisplayID: monitorDisplayID,
		DefinitionDescription: "Generic creator model with a neutral dark display texture. No manufacturer identity, active-pixel bounds, wall mount, or physical qualification is claimed.",
		ProvenanceQualifier:   "pinned original/UV-normalized bytes and creator-page limitations",
		PresetName:            "Generic ultrawide monitor on cup-free study table",
		Adaptation:            "Screen UV0 normalization only. Original source GLB SHA-256 " + monitorSourceSHA256 + "; source manifest SHA-256 " + monitorSourceManifest + "; derivative SHA-256 " + monitorAssetSHA256 + ". See scenarios/device-studio/assets/sources/innerscene-ultrawide-monitor/monitor-source-manifest.json."}
}

func monitorV2Seed() monitorSeedConfig {
	primitive, material := uint32(0), uint32(3)
	return monitorSeedConfig{AssetID: monitorV2AssetID, AssetFile: monitorV2AssetFile, AssetSHA: monitorV2AssetSHA256, BundleID: monitorV2BundleID,
		DeviceID: monitorV2DeviceID, SceneID: monitorV2SceneID, PresetID: monitorV2PresetID, ReviewID: monitorV2ReviewID,
		SourceManifestSHA: monitorV2ManifestSHA, SourceManifestPath: monitorV2ManifestPath,
		CameraID: monitorV2CameraID, SupportID: monitorV2SupportID, DisplayPartID: monitorDisplayPartID, RootPartID: monitorRootPartID, DisplayID: monitorDisplayID,
		ScreenPrimitiveIndex: &primitive, ScreenMaterialIndex: &material,
		DefinitionDescription: "Generic creator model; Source-preserving screen split: only the planar display primitive receives the project screen image; original bevel glass and chassis retain their neutral source material. No manufacturer identity, active-pixel bounds, wall mount, or physical qualification is claimed.",
		ProvenanceQualifier:   "pinned original/v1/v2 bytes, exact display/bevel triangle split, and creator-page limitations",
		PresetName:            "Generic screen-mapped ultrawide monitor on cup-free study table",
		Adaptation:            "The 300-triangle display-glass primitive is partitioned into two planar display triangles and 298 original bevel triangles. Only the front receives a new unit-rectangle TEXCOORD_0 accessor and a cloned screen material; POSITION/NORMAL, source BIN prefix, original UV accessor, and all bezel UV/material references are retained. See " + monitorV2ManifestPath + " (SHA-256 " + monitorV2ManifestSHA + "). Original source GLB SHA-256 " + monitorSourceSHA256 + "."}
}

type MonitorStarterState struct {
	Bundle    library.BundleRevision
	Device    devices.DeviceRevision
	Scene     scenes.Revision
	Motion    motions.Revision
	Placement ReviewedMonitorPlacement
	Preset    shots.PhysicalPresetRevision
}

type ReviewedMonitorPlacement struct {
	DisplayID, DeviceSupportID, SceneSupportID, CameraID string
	SceneSupportWorld, DeviceRootRest                    physical.TRS
	SupportPivotM                                        physical.Vec3
}

// initializeMonitorCandidatePinned adds one candidate-only monitor and setup
// using the existing Library, device, scene, motion, and preset owners. It
// reuses the already published cup-free room/table assets and static motion.
func (s *starterInitializer) initializeMonitorCandidatePinned(ctx context.Context, room scenes.Revision, still motions.Revision) (MonitorStarterState, error) {
	return s.initializeMonitorCandidateWithSeed(ctx, room, still, monitorV1Seed())
}

func (s *starterInitializer) initializeMonitorScreenMappedCandidatePinned(ctx context.Context, room scenes.Revision, still motions.Revision) (MonitorStarterState, error) {
	return s.initializeMonitorCandidateWithSeed(ctx, room, still, monitorV2Seed())
}

func (s *starterInitializer) initializeMonitorCandidateWithSeed(ctx context.Context, room scenes.Revision, still motions.Revision, seed monitorSeedConfig) (MonitorStarterState, error) {
	if room.SceneID != laptopFramedSceneID || still.MotionID != "phone-still-identity" {
		return MonitorStarterState{}, errors.New("monitor seed requires the pinned cup-free study room and identity still motion")
	}
	bundle, err := s.publishMonitorBundle(ctx, seed)
	if err != nil {
		return MonitorStarterState{}, err
	}
	device, err := s.saveMonitorDevice(ctx, bundle, seed)
	if err != nil {
		return MonitorStarterState{}, err
	}
	placement := monitorPlacementFor(seed)
	sceneCandidate, err := buildMonitorSceneCandidate(room.Candidate, device.Candidate, placement, seed)
	if err != nil {
		return MonitorStarterState{}, err
	}
	scene, err := saveSceneCandidateIfUnchanged(ctx, s.scenes, sceneCandidate)
	if err != nil {
		return MonitorStarterState{}, err
	}
	if err := validateMonitorPlacement(device, scene, still, placement, seed); err != nil {
		return MonitorStarterState{}, err
	}
	preset, err := s.ensureMonitorPreset(ctx, device, scene, still, placement, seed)
	if err != nil {
		return MonitorStarterState{}, err
	}
	return MonitorStarterState{Bundle: bundle, Device: device, Scene: scene, Motion: still, Placement: placement, Preset: preset}, nil
}

func (s *starterInitializer) publishMonitorBundle(ctx context.Context, seed monitorSeedConfig) (library.BundleRevision, error) {
	files := []starterFile{{
		ID: seed.AssetID, File: seed.AssetFile, MIME: "model/gltf-binary", SHA: seed.AssetSHA,
		Role: library.RoleGeometryGLB, License: "CC0-1.0", Attribution: "Innerscene, Wide View - Ultrawide Monitor (CC0-1.0; retained as provenance).",
		Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "Generic Innerscene Wide View - Ultrawide Monitor; no manufacturer identity is claimed.", SourceSHA256: monitorSourceSHA256,
			Adaptation: seed.Adaptation},
	}}
	manifest, sources, err := s.buildManifest(ctx, seed.BundleID, files)
	if err != nil {
		return library.BundleRevision{}, err
	}
	return s.publish(ctx, manifest, sources, seed.ReviewID)
}

func (s *starterInitializer) saveMonitorDevice(ctx context.Context, bundle library.BundleRevision, seed monitorSeedConfig) (devices.DeviceRevision, error) {
	ref := artifactRef(bundle, seed.AssetID)
	data, err := fs.ReadFile(s.fs, seed.AssetFile)
	if err != nil {
		return devices.DeviceRevision{}, err
	}
	if ref.SHA256 != seed.AssetSHA || ref.SizeBytes != int64(len(data)) || sha256Hex(data) != seed.AssetSHA {
		return devices.DeviceRevision{}, errors.New("embedded monitor bytes differ from the pinned Library artifact")
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("inspect pinned monitor GLB: %w", err)
	}
	if len(inventory.Nodes) != 6 || len(inventory.Meshes) != 5 {
		return devices.DeviceRevision{}, fmt.Errorf("monitor GLB inventory changed: nodes=%d meshes=%d", len(inventory.Nodes), len(inventory.Meshes))
	}
	nodes := make(map[string]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		if _, exists := nodes[node.Name]; exists {
			return devices.DeviceRevision{}, fmt.Errorf("monitor GLB repeats semantic node name %q", node.Name)
		}
		nodes[node.Name] = node
	}
	root, rootOK := nodes["Wide View"]
	display, displayOK := nodes["Display glass"]
	if !rootOK || !displayOK || root.Index != 5 || display.Index != 1 || root.Parent != nil || display.Parent == nil || *display.Parent != root.Index || display.Mesh == nil || *display.Mesh != 1 {
		return devices.DeviceRevision{}, errors.New("monitor GLB root/display hierarchy differs from its source manifest")
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("derive monitor bounds from pinned GLB: %w", err)
	}
	displayBounds, err := physicalGLBNodeMeshWorldBounds(inventory, display.Index)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("derive monitor glass proxy metric: %w", err)
	}
	width, height, depth := bounds.max.X-bounds.min.X, bounds.max.Y-bounds.min.Y, bounds.max.Z-bounds.min.Z
	displayWidth, displayHeight := displayBounds.max.X-displayBounds.min.X, displayBounds.max.Y-displayBounds.min.Y
	if !positiveFinite(width) || !positiveFinite(height) || !positiveFinite(depth) || !positiveFinite(displayWidth) || !positiveFinite(displayHeight) {
		return devices.DeviceRevision{}, errors.New("monitor GLB envelope, display plane, or source-feet support is invalid")
	}
	evidence := []devices.Evidence{{Source: "embedded " + path.Base(seed.AssetFile) + " SHA-256 " + seed.AssetSHA, Kind: "derived_asset_geometry", Qualifier: "creator GLB mesh span only; not manufacturer dimensions, active pixels, or physical qualification"}}
	parts := make([]devices.SemanticPart, 0, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		partID, role := monitorPartID(node.Index), "enclosure"
		switch node.Name {
		case "Wide View":
			partID, role = seed.RootPartID, "device_root"
		case "Display glass":
			partID, role = seed.DisplayPartID, "display_surface"
		case "Display neck":
			role = "stand_neck"
		case "Split display foot", "Split display foot.001":
			role = "stand_foot"
		}
		parent := ""
		if node.Parent != nil {
			parent = monitorPartID(*node.Parent)
			if *node.Parent == root.Index {
				parent = seed.RootPartID
			}
		}
		parts = append(parts, devices.SemanticPart{ID: partID, Role: role, ParentID: parent, NodeIndex: node.Index, NodeName: node.Name, LocalFrame: node.Frame})
	}
	dimensions := devices.Dimensions{
		Width:  monitorMeasurement(width*1000, "source GLB world-space X span", seed),
		Height: monitorMeasurement(height*1000, "source GLB world-space Y span", seed),
		Depth:  monitorMeasurement(depth*1000, "source GLB world-space Z span", seed),
	}
	definition := devices.Definition{
		SchemaVersion: "device-definition/v1", ID: seed.DeviceID, Group: devices.GroupMonitorTV, Kind: "generic_desktop_monitor",
		IdentityStatus: devices.IdentityGeneric, Model: "Generic Innerscene Wide View ultrawide monitor", Variant: "source mesh candidate; not a named product",
		Description:     seed.DefinitionDescription,
		Dimensions:      map[string]devices.Dimensions{"source_geometry": dimensions},
		Displays:        []devices.DisplaySurface{{ID: seed.DisplayID, Topology: "planar", CoordinateMetric: &devices.DisplayMetric{WidthMM: displayWidth * 1000, HeightMM: displayHeight * 1000, Evidence: evidence}, PhysicalBoundary: unknownRegion(), ActiveArea: unknownRegion(), AppSafeArea: unknownRegion(), Evidence: evidence}},
		SupportedMotion: []devices.Capability{{Name: "rigid_root_transform", Status: "candidate_only", MotionKind: "rigid_body", Evidence: evidence}},
		Supports:        []devices.SupportCapability{{Kind: "tabletop", InterfaceID: "horizontal_surface", Status: "candidate_only", Evidence: evidence}},
		ModelCandidates: []devices.ModelCandidate{{ID: "innerscene-wide-view-ultrawide-monitor", Status: "candidate_only", SHA256: monitorSourceSHA256, ListingURL: "https://www.innerscene.com/tools/library/3d-parts/wide-view-ultrawide-monitor-e43c45f7", Author: "Innerscene", License: "CC0-1.0", Attribution: "Innerscene, Wide View - Ultrawide Monitor; retained as provenance.", RightsNotes: "Generic source model only. No manufacturer/model identity or endorsement is asserted."}},
		Provenance:      []devices.Evidence{{Source: "https://www.innerscene.com/tools/library/3d-parts/wide-view-ultrawide-monitor-e43c45f7", Kind: "creator_listing", Qualifier: "Innerscene original model; CC0-1.0 as stated by source; no product identity"}, {Source: seed.SourceManifestPath + " SHA-256 " + seed.SourceManifestSHA, Kind: "source_and_derivation_manifest", Qualifier: seed.ProvenanceQualifier}},
		Qualification:   "unqualified",
	}
	candidate := devices.DeviceCandidate{
		Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention, SourceToCanonical: identityTRS(), Parts: parts,
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: seed.DisplayID, PartID: seed.DisplayPartID, LocalFrame: identityTRS(), ScreenPrimitiveIndex: seed.ScreenPrimitiveIndex, ScreenMaterialIndex: seed.ScreenMaterialIndex}},
		SupportFrames:   []devices.DeviceSupportFrame{{ID: seed.SupportID, InterfaceID: "horizontal_surface", PartID: seed.RootPartID, LocalFrame: identityTRS()}},
	}
	if _, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, seed.DisplayID); err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("preflight monitor display mapping: %w", err)
	}
	return saveDeviceCandidateIfUnchanged(ctx, s.devices, candidate)
}

func buildMonitorSceneCandidate(base scenes.SceneCandidate, device devices.DeviceCandidate, placement ReviewedMonitorPlacement, seed monitorSeedConfig) (scenes.SceneCandidate, error) {
	if base.SceneID != laptopFramedSceneID || device.Definition.ID != seed.DeviceID || len(base.SupportFrames) == 0 || len(base.CameraFrames) != 1 || base.CameraFrames[0].ID != laptopFramedCameraID {
		return scenes.SceneCandidate{}, errors.New("monitor scene requires the pinned cup-free room/table candidate and one framed source camera")
	}
	for _, part := range base.Parts {
		if part.NodeName == "TeaCup" || part.NodeName == "TeaCupHandle" || part.NodeName == "TeaSurface" {
			return scenes.SceneCandidate{}, errors.New("monitor scene source contains a cup/tea prop")
		}
	}
	hasTableSupport := false
	for _, frame := range base.SupportFrames {
		hasTableSupport = hasTableSupport || frame.ID == laptopTableSupportID && frame.InterfaceID == "horizontal_surface"
	}
	if !hasTableSupport {
		return scenes.SceneCandidate{}, errors.New("monitor scene source has no compatible tabletop support")
	}
	geometry, err := fs.ReadFile(starterAssets, seed.AssetFile)
	if err != nil {
		return scenes.SceneCandidate{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(geometry)
	if err != nil {
		return scenes.SceneCandidate{}, err
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return scenes.SceneCandidate{}, err
	}
	points, err := monitorPlacedBounds(bounds, device.SourceToCanonical, placement)
	if err != nil {
		return scenes.SceneCandidate{}, err
	}
	candidate := base
	candidate.SceneID = seed.SceneID
	candidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), base.CameraFrames...)
	camera := candidate.CameraFrames[0]
	camera.ID = seed.CameraID
	camera.SensorFit = scenes.SensorFitHorizontal
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusSelectedDevice}
	fitted, err := fitPerspectiveCameraPoints(points, camera, 16.0/9.0, .10)
	if err != nil {
		return scenes.SceneCandidate{}, fmt.Errorf("fit monitor camera to full source GLB bounds: %w", err)
	}
	camera.Frame = fitted
	candidate.CameraFrames[0] = camera
	if _, _, _, err := scenes.CanonicalizeCandidate(candidate); err != nil {
		return scenes.SceneCandidate{}, fmt.Errorf("canonicalize monitor no-cup scene candidate: %w", err)
	}
	return candidate, nil
}

func monitorPlacedBounds(bounds physicalBounds, sourceToCanonical physical.TRS, placement ReviewedMonitorPlacement) ([]physical.Vec3, error) {
	scene, err := placement.SceneSupportWorld.Matrix()
	if err != nil {
		return nil, err
	}
	pivot, err := (physical.TRS{Translation: placement.SupportPivotM, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}).Matrix()
	if err != nil {
		return nil, err
	}
	rest, err := placement.DeviceRootRest.Matrix()
	if err != nil {
		return nil, err
	}
	source, err := sourceToCanonical.Matrix()
	if err != nil {
		return nil, err
	}
	world, err := physical.Compose(scene, pivot)
	if err == nil {
		world, err = physical.Compose(world, rest)
	}
	if err == nil {
		world, err = physical.Compose(world, source)
	}
	if err != nil {
		return nil, err
	}
	points := make([]physical.Vec3, 0, 8)
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
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

func monitorPlacement() ReviewedMonitorPlacement {
	return monitorPlacementFor(monitorV1Seed())
}

func monitorPlacementFor(seed monitorSeedConfig) ReviewedMonitorPlacement {
	identity := identityTRS()
	return ReviewedMonitorPlacement{
		DisplayID: seed.DisplayID, DeviceSupportID: seed.SupportID, SceneSupportID: laptopTableSupportID, CameraID: seed.CameraID,
		SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: identity.Rotation, Scale: identity.Scale},
		SupportPivotM:     physical.Vec3{}, DeviceRootRest: identity,
	}
}

func validateMonitorPlacement(device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedMonitorPlacement, seed monitorSeedConfig) error {
	if device.DeviceID != seed.DeviceID || scene.SceneID != seed.SceneID || motion.MotionID != "phone-still-identity" {
		return errors.New("monitor placement is bound to different immutable owner revisions")
	}
	display, support, camera, sceneSupport := false, false, false, false
	for _, value := range device.Candidate.DisplayBindings {
		display = display || value.DisplayID == placement.DisplayID && value.PartID == seed.DisplayPartID
	}
	for _, value := range device.Candidate.SupportFrames {
		support = support || value.ID == placement.DeviceSupportID && value.InterfaceID == "horizontal_surface" && value.PartID == seed.RootPartID
	}
	for _, value := range scene.Candidate.CameraFrames {
		camera = camera || value.ID == placement.CameraID
	}
	for _, value := range scene.Candidate.SupportFrames {
		sceneSupport = sceneSupport || value.ID == placement.SceneSupportID && value.InterfaceID == "horizontal_surface"
	}
	if !display || !support || !camera || !sceneSupport {
		return errors.New("monitor placement IDs are absent from the pinned candidate revisions")
	}
	return nil
}

func (s *starterInitializer) ensureMonitorPreset(ctx context.Context, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedMonitorPlacement, seed monitorSeedConfig) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{
		Device:    &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256},
		Scene:     &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256},
		Motion:    &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256},
		DisplayId: placement.DisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.CameraID,
		SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest),
		StillProfiles:  []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1, shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1},
		MotionProfiles: []shotsv1.PhysicalMotionOutputProfile{shotsv1.PhysicalMotionOutputProfile_PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V1, shotsv1.PhysicalMotionOutputProfile_PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2},
	}
	if current, err := s.presets.Head(ctx, seed.PresetID); err == nil {
		if current.Name == seed.PresetName && proto.Equal(current.Setup, setup) {
			return current, nil
		}
		return shots.PhysicalPresetRevision{}, fmt.Errorf("monitor preset differs from its fixed candidate; refusing implicit revision: %w", shots.ErrPhysicalPresetConflict)
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, seed.PresetID, 0, seed.PresetName, setup)
}

func monitorPartID(index uint32) string { return fmt.Sprintf("monitor-node-%02d", index) }

func monitorMeasurement(valueMM float64, axis string, seed monitorSeedConfig) devices.Measurement {
	value := valueMM
	return devices.Measurement{Value: &value, Unit: "mm", Meaning: "creator GLB " + axis, Method: "derived", Uncertainty: devices.Uncertainty{Status: "unknown"}, Evidence: []devices.Evidence{{Source: path.Base(seed.SourceManifestPath) + " SHA-256 " + seed.SourceManifestSHA, Kind: "derived_asset_geometry", Qualifier: "source GLB geometry only; not measured physical hardware"}}}
}
