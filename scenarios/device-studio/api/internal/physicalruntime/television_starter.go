package physicalruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"device-studio/internal/devices"
	"device-studio/internal/geometry"
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
	televisionAssetID       = "vrooli-authored-generic-television-concept-v2-screen-vflip"
	televisionBundleID      = "device-studio-starter-authored-generic-television-v2-screen-vflip"
	televisionDeviceID      = "authored-generic-television-candidate-v2-screen-vflip"
	televisionSceneID       = "authored-generic-television-study-candidate-v2-screen-vflip"
	televisionPresetID      = "authored-generic-television-study-stand-v2-screen-vflip"
	televisionReviewID      = "device-studio-e4-authored-generic-television-candidate-v2-screen-vflip"
	televisionDisplayID     = "main_display"
	televisionRootPartID    = "television-root"
	televisionDisplayPartID = "television-screen"
	televisionSupportID     = "television-stand-support-candidate"
	televisionWallSupportID = "television-wall-mount-candidate"
	televisionCameraID      = "television-camera-full-bounds-v1"
	televisionSourceSHA     = "d1a8507fe9cd524f4aad5fbdef2dd7533f0ba83d9f1b4bdddb82f00bda712dce"
	televisionAssetSHA      = "137638c936842757e33d763d930e663140479083ccd3e82f339a64b8db89c554"
)

type TelevisionStarterState struct {
	Bundle library.BundleRevision
	Device devices.DeviceRevision
	Scene  scenes.Revision
	Motion motions.Revision
	Preset shots.PhysicalPresetRevision
}

// InitializeReviewedTelevisionCandidate opts into an authored generic model;
// it does not imply Samsung identity, VESA compliance, or physical qualification.
func (o Owners) InitializeReviewedTelevisionCandidate(ctx context.Context) (TelevisionStarterState, error) {
	if o.Starter == nil {
		return TelevisionStarterState{}, errors.New("television candidate initializer is unavailable")
	}
	pinned, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return TelevisionStarterState{}, err
	}
	base, err := o.Starter.initializePinned(pinned)
	if err != nil {
		return TelevisionStarterState{}, err
	}
	room, err := o.Starter.initializeLaptopCandidatePinned(pinned)
	if err != nil {
		return TelevisionStarterState{}, err
	}
	return o.Starter.initializeTelevisionPinned(pinned, room.FramedScene, base.Motion)
}

func televisionRecipe() geometry.Recipe {
	return geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{930, 600, 250},
		Parts: []geometry.Part{
			{ID: "housing", Kind: "rounded_box", Role: "housing", Center: [3]float64{0, 20, 0}, Size: [3]float64{920, 520, 36}, Bevel: 7, Material: "matte_polymer"},
			{ID: "bezel", Kind: "rounded_box", Role: "bezel", Center: [3]float64{0, 20, 20}, Size: [3]float64{850, 468, 4}, Bevel: 1, Material: "gloss_polymer"},
			{ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 20, 22.5}, Size: [3]float64{820, 438, 1}, Material: "glass"},
			{ID: "stand_neck", Kind: "rounded_box", Role: "control", Center: [3]float64{0, -250, -28}, Size: [3]float64{116, 100, 64}, Bevel: 8, Material: "anodized_metal"},
			{ID: "stand_foot_left", Kind: "rounded_box", Role: "detail", Center: [3]float64{-300, -285, -35}, Size: [3]float64{96, 30, 180}, Bevel: 10, Material: "anodized_metal"},
			{ID: "stand_foot_right", Kind: "rounded_box", Role: "detail", Center: [3]float64{300, -285, -35}, Size: [3]float64{96, 30, 180}, Bevel: 10, Material: "anodized_metal"},
			{ID: "wall_mount_plate", Kind: "rounded_box", Role: "detail", Center: [3]float64{0, 20, -23}, Size: [3]float64{220, 220, 8}, Bevel: 3, Material: "neutral_metal"},
			{ID: "mount_pad_ul", Kind: "box", Role: "detail", Center: [3]float64{-100, 120, -30}, Size: [3]float64{20, 20, 6}, Material: "neutral_metal"},
			{ID: "mount_pad_ur", Kind: "box", Role: "detail", Center: [3]float64{100, 120, -30}, Size: [3]float64{20, 20, 6}, Material: "neutral_metal"},
			{ID: "mount_pad_ll", Kind: "box", Role: "detail", Center: [3]float64{-100, -80, -30}, Size: [3]float64{20, 20, 6}, Material: "neutral_metal"},
			{ID: "mount_pad_lr", Kind: "box", Role: "detail", Center: [3]float64{100, -80, -30}, Size: [3]float64{20, 20, 6}, Material: "neutral_metal"},
		},
		Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.96, .96}},
	}
}

// televisionThinBezelRecipe is a separate immutable v3 design. The v2 recipe
// remains byte-pinned for its retained rejection evidence.
func televisionThinBezelRecipe() geometry.Recipe {
	return geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{970, 700, 300},
		Parts: []geometry.Part{
			{ID: "housing", Kind: "rounded_box", Role: "housing", Center: [3]float64{0, 0, 0}, Size: [3]float64{965, 559, 36}, Bevel: 12, Material: "matte_polymer"},
			{ID: "bezel", Kind: "rounded_box", Role: "bezel", Center: [3]float64{0, 0, 20}, Size: [3]float64{957, 551, 3}, Bevel: 1.2, Material: "anodized_metal"},
			{ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 22.2}, Size: [3]float64{951, 536, 1}, Material: "glass"},
			{ID: "stand_neck", Kind: "rounded_box", Role: "control", Center: [3]float64{0, -290, -26}, Size: [3]float64{84, 100, 44}, Bevel: 10, Material: "anodized_metal"},
			{ID: "stand_leg_left", Kind: "rounded_box", Role: "detail", Center: [3]float64{-285, -285, -32}, Size: [3]float64{22, 100, 24}, Bevel: 5, Material: "anodized_metal"},
			{ID: "stand_leg_right", Kind: "rounded_box", Role: "detail", Center: [3]float64{285, -285, -32}, Size: [3]float64{22, 100, 24}, Bevel: 5, Material: "anodized_metal"},
			{ID: "stand_foot_left", Kind: "rounded_box", Role: "detail", Center: [3]float64{-285, -335, -45}, Size: [3]float64{250, 18, 160}, Bevel: 8, Material: "anodized_metal"},
			{ID: "stand_foot_right", Kind: "rounded_box", Role: "detail", Center: [3]float64{285, -335, -45}, Size: [3]float64{250, 18, 160}, Bevel: 8, Material: "anodized_metal"},
			{ID: "wall_mount_plate", Kind: "rounded_box", Role: "detail", Center: [3]float64{0, 0, -23}, Size: [3]float64{200, 200, 8}, Bevel: 3, Material: "neutral_metal"},
			{ID: "mount_pad_ul", Kind: "box", Role: "detail", Center: [3]float64{-80, 80, -30}, Size: [3]float64{18, 18, 6}, Material: "neutral_metal"},
			{ID: "mount_pad_ur", Kind: "box", Role: "detail", Center: [3]float64{80, 80, -30}, Size: [3]float64{18, 18, 6}, Material: "neutral_metal"},
			{ID: "mount_pad_ll", Kind: "box", Role: "detail", Center: [3]float64{-80, -80, -30}, Size: [3]float64{18, 18, 6}, Material: "neutral_metal"},
			{ID: "mount_pad_lr", Kind: "box", Role: "detail", Center: [3]float64{80, -80, -30}, Size: [3]float64{18, 18, 6}, Material: "neutral_metal"},
		},
		Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.997, .997}},
	}
}

func compileTelevisionAsset() ([]byte, string, error) {
	data, digest, err := compileTelevisionRecipe(televisionRecipe())
	if err != nil {
		return nil, "", err
	}
	if digest != televisionAssetSHA {
		return nil, "", fmt.Errorf("authored television bytes changed without a versioned source update: %s", digest)
	}
	return data, digest, nil
}

func compileTelevisionV3Asset() ([]byte, string, error) {
	return compileTelevisionRecipe(televisionThinBezelRecipe())
}

func compileTelevisionRecipe(recipe geometry.Recipe) ([]byte, string, error) {
	if err := geometry.Validate(recipe); err != nil {
		return nil, "", fmt.Errorf("validate authored television recipe: %w", err)
	}
	data, err := geometry.Compile(recipe)
	if err != nil {
		return nil, "", fmt.Errorf("compile authored television: %w", err)
	}
	data, err = flipScreenUVV(data, "display", 1)
	if err != nil {
		return nil, "", fmt.Errorf("derive television screen-only UV view: %w", err)
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	return data, digest, nil
}

func (s *starterInitializer) initializeTelevisionPinned(ctx context.Context, room scenes.Revision, still motions.Revision) (TelevisionStarterState, error) {
	if room.SceneID != laptopFramedSceneID || still.MotionID != "phone-still-identity" {
		return TelevisionStarterState{}, errors.New("television requires the pinned cup-free room and identity still motion")
	}
	data, digest, err := compileTelevisionAsset()
	if err != nil {
		return TelevisionStarterState{}, err
	}
	bundle, err := s.publishTelevisionBundle(ctx, data, digest)
	if err != nil {
		return TelevisionStarterState{}, err
	}
	device, bounds, err := s.saveTelevisionDevice(ctx, bundle, data, digest)
	if err != nil {
		return TelevisionStarterState{}, err
	}
	scene, err := s.saveTelevisionScene(ctx, room, device, bounds)
	if err != nil {
		return TelevisionStarterState{}, err
	}
	preset, err := s.ensureTelevisionPreset(ctx, device, scene, still, bounds)
	if err != nil {
		return TelevisionStarterState{}, err
	}
	return TelevisionStarterState{Bundle: bundle, Device: device, Scene: scene, Motion: still, Preset: preset}, nil
}

func (s *starterInitializer) publishTelevisionBundle(ctx context.Context, data []byte, digest string) (library.BundleRevision, error) {
	manifest := library.BundleManifest{Version: library.ManifestVersion, BundleID: televisionBundleID, Revision: 1, Artifacts: []library.Artifact{{
		ID: televisionAssetID, Role: library.RoleGeometryGLB, MIME: "model/gltf-binary", SizeBytes: int64(len(data)), SHA256: digest, LicenseID: "AGPL-3.0-only",
		Attribution: "Vrooli-authored generic television concept; not a manufacturer model or CAD.",
		Provenance:  library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "Deterministic finite generic-television geometry recipe in television_starter.go; includes separate front screen, tabletop stand, and candidate-only rear plate/pads.", SourceSHA256: televisionSourceSHA, Adaptation: "Versioned screen-only UV derivative: a private TEXCOORD_0 accessor flips V for display primitive 1; source geometry and other BIN bytes are retained. Samsung DU8000 dimensions are comparison-only; no product identity or VESA compliance is claimed."},
		Rights:      []library.Use{library.UseEdit, library.UseRender},
	}}}
	sources := map[string]io.Reader{televisionAssetID: bytes.NewReader(data)}
	return s.publish(ctx, manifest, sources, televisionReviewID)
}

func (s *starterInitializer) saveTelevisionDevice(ctx context.Context, bundle library.BundleRevision, data []byte, digest string) (devices.DeviceRevision, physicalBounds, error) {
	ref := artifactRef(bundle, televisionAssetID)
	if ref.SHA256 != digest || ref.SizeBytes != int64(len(data)) {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("compiled television differs from its pinned Library artifact")
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("inspect authored television GLB: %w", err)
	}
	if len(inventory.Nodes) != len(televisionRecipe().Parts) || len(inventory.Meshes) != len(televisionRecipe().Parts) || len(inventory.Extensions) != 0 {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("authored television GLB inventory differs from its finite recipe")
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil || !bounds.valid() {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("derive authored television bounds: %w", err)
	}
	nodesByName := make(map[string]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodesByName[node.Name] = node
	}
	display, ok := nodesByName["display"]
	if !ok || display.Mesh == nil {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("television screen part is absent")
	}
	mesh := inventory.Meshes[*display.Mesh]
	if len(mesh.Primitives) != 2 {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("television display mesh must retain one housing face and one exact screen plane")
	}
	screenPrimitive := mesh.Primitives[1]
	materials := map[uint32]rendering.PhysicalGLBMaterial{}
	for _, material := range inventory.Materials {
		materials[material.Index] = material
	}
	screenMaterial, ok := materials[screenPrimitive.Material]
	if !ok || screenMaterial.Name != "screen-glass" {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("television screen plane does not use the compiler's screen-glass material")
	}
	evidence := []devices.Evidence{
		{Source: "television_starter.go finite recipe; compiled GLB SHA-256 " + digest, Kind: "authored_candidate_geometry", Qualifier: "authored generic concept dimensions derived from mesh, not a measured or manufacturer model"},
		{Source: "https://www.samsung.com/ca/tvs/uhd-4k-tv/du8000-55-inch-crystal-uhd-4k-tizen-os-smart-tv-un55du8000fxzc/", Kind: "comparison_reference", Qualifier: "Samsung Canada lists 1232.1 x 708.8 x 25.7 mm without stand and 1232.1 x 747.8 x 228.8 mm with stand; comparison only, no identity or design derivation"},
	}
	wallMountEvidence := []devices.Evidence{{Source: "television_starter.go finite recipe; compiled GLB SHA-256 " + digest, Kind: "authored_mount_concept", Qualifier: "four rear plate pads are placed on an authored 200 x 200 mm pattern; not VESA asserted, verified, or certified"}}
	width, height, depth := (bounds.max.X-bounds.min.X)*1000, (bounds.max.Y-bounds.min.Y)*1000, (bounds.max.Z-bounds.min.Z)*1000
	displayNode, rootNode := display, nodesByName["housing"]
	if rootNode.Mesh == nil {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("television housing root has no mesh")
	}
	displayWidth, displayHeight := (mesh.Primitives[0].PositionMax[0]-mesh.Primitives[0].PositionMin[0])*1000, (mesh.Primitives[0].PositionMax[1]-mesh.Primitives[0].PositionMin[1])*1000
	displayEvidence := []devices.Evidence{{Source: "compiled television GLB SHA-256 " + digest, Kind: "derived_display_geometry", Qualifier: "separate unit-UV screen plane inside an authored flat face; candidate geometry only"}}
	definition := devices.Definition{SchemaVersion: "device-definition/v1", ID: televisionDeviceID, Group: devices.GroupMonitorTV, Kind: "generic_television_concept", IdentityStatus: devices.IdentityConcept, Model: "Generic authored flat-panel television concept", Variant: "authored concept; no product variant",
		Description:     "Neutral authored generic television candidate with a replaceable front screen, tabletop stand, and rear mounting concept. It is not Samsung CAD; DU8000 measurements are only a comparison reference. Stand stability, mount pattern, dimensions, active area, and physical qualification are unverified.",
		Dimensions:      map[string]devices.Dimensions{"authored_geometry_envelope": {Width: televisionMeasurement(width, "full authored mesh X span", evidence), Height: televisionMeasurement(height, "full authored mesh Y span including stand", evidence), Depth: televisionMeasurement(depth, "full authored mesh Z span including stand feet", evidence)}},
		Displays:        []devices.DisplaySurface{{ID: televisionDisplayID, Topology: "planar", CoordinateMetric: &devices.DisplayMetric{WidthMM: displayWidth * .96, HeightMM: displayHeight * .96, Evidence: displayEvidence}, PhysicalBoundary: devices.Region{Status: "candidate_only", Shape: "rect", Bounds: &devices.Rect{X: .02, Y: .02, Width: .96, Height: .96}, Evidence: displayEvidence}, ActiveArea: devices.Region{Status: "candidate_only", Shape: "rect", Bounds: &devices.Rect{X: .02, Y: .02, Width: .96, Height: .96}, Evidence: displayEvidence}, AppSafeArea: devices.Region{Status: "candidate_only", Shape: "rect", Bounds: &devices.Rect{X: .04, Y: .04, Width: .92, Height: .92}, Evidence: displayEvidence}, Evidence: displayEvidence}},
		SupportedPoses:  []devices.Pose{{Name: "stand-supported-tabletop-landscape", Status: "candidate_only", Evidence: evidence}, {Name: "rear-mount-wall-supported-landscape", Status: "candidate_only", Evidence: evidence}},
		Supports:        []devices.SupportCapability{{Kind: "tabletop", InterfaceID: "horizontal_surface", Status: "candidate_only", Evidence: evidence}, {Kind: "wall_mount", InterfaceID: "authored-rear-plate-pattern-unverified", Status: "candidate_only", Evidence: wallMountEvidence}},
		ModelCandidates: []devices.ModelCandidate{{ID: televisionAssetID, Status: "candidate_only", SHA256: digest, Author: "Vrooli", License: "AGPL-3.0-only", Attribution: "Vrooli-authored generic concept.", RightsNotes: "Not a Samsung product model or CAD. Rear mounting pattern is an authored concept, not VESA-compatible or certified."}}, Provenance: evidence, Qualification: "unqualified"}
	parts := make([]devices.SemanticPart, 0, len(inventory.Nodes))
	partIDs := map[uint32]string{}
	for _, node := range inventory.Nodes {
		partIDs[node.Index] = "television-" + node.Name
	}
	for _, node := range inventory.Nodes {
		role := "housing_component"
		if node.Name == "housing" {
			role = "device_root"
		}
		if node.Name == "display" {
			role = "display_surface"
		}
		if strings.HasPrefix(node.Name, "stand_") {
			role = "stand_component"
		}
		if strings.HasPrefix(node.Name, "wall_mount") || strings.HasPrefix(node.Name, "mount_pad") {
			role = "wall_mount_concept"
		}
		parent := ""
		if node.Parent != nil {
			parent = partIDs[*node.Parent]
		}
		parts = append(parts, devices.SemanticPart{ID: partIDs[node.Index], Role: role, ParentID: parent, NodeIndex: node.Index, NodeName: node.Name, LocalFrame: node.Frame})
	}
	primitive, material := screenPrimitive.Index, screenPrimitive.Material
	candidate := devices.DeviceCandidate{Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention, SourceToCanonical: identityTRS(), Parts: parts,
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: televisionDisplayID, PartID: partIDs[displayNode.Index], LocalFrame: identityTRS(), ScreenPrimitiveIndex: &primitive, ScreenMaterialIndex: &material}},
		SupportFrames:   []devices.DeviceSupportFrame{{ID: televisionSupportID, InterfaceID: "horizontal_surface", PartID: partIDs[rootNode.Index], LocalFrame: physical.TRS{Translation: physical.Vec3{Y: bounds.min.Y}, Rotation: identityTRS().Rotation, Scale: identityTRS().Scale}}, {ID: televisionWallSupportID, InterfaceID: "authored-rear-plate-pattern-unverified", PartID: partIDs[displayNode.Index], LocalFrame: physical.TRS{Translation: physical.Vec3{Z: bounds.min.Z}, Rotation: identityTRS().Rotation, Scale: identityTRS().Scale}}},
	}
	mapping, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, televisionDisplayID)
	if err != nil {
		return devices.DeviceRevision{}, physicalBounds{}, fmt.Errorf("television screen binding failed preflight: %w", err)
	}
	if mapping.NodeIndex != displayNode.Index || mapping.PrimitiveIndex != screenPrimitive.Index || mapping.MaterialIndex != screenPrimitive.Material || mapping.UVMin != [2]float64{} || mapping.UVMax != [2]float64{1, 1} {
		return devices.DeviceRevision{}, physicalBounds{}, errors.New("television screen preflight selected different authored geometry")
	}
	revision, err := saveDeviceCandidateIfUnchanged(ctx, s.devices, candidate)
	return revision, bounds, err
}

func (s *starterInitializer) saveTelevisionScene(ctx context.Context, room scenes.Revision, device devices.DeviceRevision, bounds physicalBounds) (scenes.Revision, error) {
	if room.SceneID != laptopFramedSceneID || device.DeviceID != televisionDeviceID || len(room.Candidate.CameraFrames) != 1 {
		return scenes.Revision{}, errors.New("television scene requires the pinned cup-free room and framed camera")
	}
	candidate := room.Candidate
	candidate.SceneID = televisionSceneID
	camera := candidate.CameraFrames[0]
	camera.ID = televisionCameraID
	placement := ReviewedIPadPlacement{SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: identityTRS().Rotation, Scale: identityTRS().Scale}, DeviceRootRest: physical.TRS{Translation: physical.Vec3{Y: -bounds.min.Y}, Rotation: identityTRS().Rotation, Scale: identityTRS().Scale}}
	fitted, err := fitIPadPerspectiveCamera(bounds, identityTRS(), placement, camera, 16.0/9.0, .10)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("fit television camera to authored envelope: %w", err)
	}
	camera.Frame = fitted
	center := physical.Vec3{Y: .75 + (bounds.max.Y-bounds.min.Y)/2, Z: (bounds.min.Z + bounds.max.Z) / 2}
	distance, err := watchCameraAxisDistance(camera.Frame, center)
	if err != nil {
		return scenes.Revision{}, err
	}
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusDistance, DistanceM: &distance}
	candidate.CameraFrames = []scenes.SceneCameraFrame{camera}
	return saveSceneCandidateIfUnchanged(ctx, s.scenes, candidate)
}

func (s *starterInitializer) ensureTelevisionPreset(ctx context.Context, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, bounds physicalBounds) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{Device: &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256}, Scene: &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256}, Motion: &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256}, DisplayId: televisionDisplayID, DeviceSupportId: televisionSupportID, SceneSupportId: laptopTableSupportID, CameraId: televisionCameraID,
		SceneSupportWorld: starterTRS(physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: identityTRS().Rotation, Scale: identityTRS().Scale}), SupportPivotM: starterVec3(physical.Vec3{}), DeviceRootRest: starterTRS(physical.TRS{Translation: physical.Vec3{Y: -bounds.min.Y}, Rotation: identityTRS().Rotation, Scale: identityTRS().Scale}), StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1}}
	if current, err := s.presets.Head(ctx, televisionPresetID); err == nil {
		if current.Name == "Generic television concept on two-foot stand" && proto.Equal(current.Setup, setup) {
			return current, nil
		}
		return shots.PhysicalPresetRevision{}, fmt.Errorf("television preset differs from fixed candidate: %w", shots.ErrPhysicalPresetConflict)
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, televisionPresetID, 0, "Generic television concept on two-foot stand", setup)
}

func televisionMeasurement(valueMM float64, meaning string, evidence []devices.Evidence) devices.Measurement {
	value := valueMM
	return devices.Measurement{Value: &value, Unit: "mm", Meaning: meaning, Configuration: "authored generic recipe with stand included", Method: "derived", Uncertainty: devices.Uncertainty{Status: "unknown"}, Evidence: evidence}
}

var _ = io.EOF
