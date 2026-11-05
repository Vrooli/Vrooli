package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"

	"device-studio/internal/composition"
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
	foldAssetID         = "vrooli-generic-book-fold-concept-glb-v2"
	foldAssetFile       = "assets/foldable/foldable-concept.glb"
	foldAssetSHA        = "90e1e5195ee88018e37b98d8d08af4d1407d8161e6216cf3fb699921d77ce4ac"
	foldManifestPath    = "scenarios/device-studio/assets/sources/foldable-concept/foldable-manifest.json"
	foldManifestSHA     = "cd9bc7b43cf1a8ae3ce3a0ac76161fbcfde2a0d7b1712a92b1240bf29d4c6192"
	foldBundleID        = "device-studio-starter-generic-book-fold-concept-v2"
	foldDeviceID        = "generic-dual-rigid-book-fold-concept-v2"
	foldSceneID         = "generic-book-fold-study-no-cup-v2"
	foldMotionID        = "generic-book-fold-hinge-motion-v2"
	foldPresetID        = "generic-book-fold-still-v2"
	foldReviewID        = "device-studio-e3-foldable-authored-candidate-v2"
	foldRootPart        = "fold-root"
	foldLeftPart        = "fold-left-housing"
	foldPivotPart       = "fold-hinge-pivot"
	foldRightPart       = "fold-right-housing"
	foldLeftInnerPart   = "fold-left-inner-display"
	foldRightInnerPart  = "fold-right-inner-display"
	foldOuterPart       = "fold-outer-display"
	foldHingePart       = "fold-hinge-barrel"
	foldJointID         = "concept-book-fold-hinge"
	foldCameraID        = "fold-camera-open-envelope-v2"
	foldDeviceSupportID = "fold-root-tabletop-v1"
	foldStillName       = "Generic book-fold concept v2 on study table"
)

const (
	foldSupportLiftM = 0.00282
	foldClosedAngle  = math.Pi
)

type FoldableStarterState struct {
	Bundle    library.BundleRevision
	Device    devices.DeviceRevision
	Scene     scenes.Revision
	Motion    motions.Revision
	Placement ReviewedMonitorPlacement
	Preset    shots.PhysicalPresetRevision
}

// InitializeReviewedFoldableCandidate is opt-in and keeps the authored concept
// unqualified until its separate visual and geometry reviews are complete.
func (o Owners) InitializeReviewedFoldableCandidate(ctx context.Context) (FoldableStarterState, error) {
	if o.Starter == nil {
		return FoldableStarterState{}, errors.New("foldable candidate initializer is unavailable")
	}
	pinned, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return FoldableStarterState{}, err
	}
	base, err := o.Starter.initializePinned(pinned)
	if err != nil {
		return FoldableStarterState{}, err
	}
	room, err := o.Starter.initializeLaptopCandidatePinned(pinned)
	if err != nil {
		return FoldableStarterState{}, err
	}
	return o.Starter.initializeFoldableCandidatePinned(pinned, room.FramedScene, base.Motion)
}

func (s *starterInitializer) initializeFoldableCandidatePinned(ctx context.Context, room scenes.Revision, still motions.Revision) (FoldableStarterState, error) {
	if room.SceneID != laptopFramedSceneID || still.MotionID != "phone-still-identity" {
		return FoldableStarterState{}, errors.New("foldable candidate requires the pinned cup-free room and identity still motion")
	}
	file := starterFile{ID: foldAssetID, File: foldAssetFile, MIME: "model/gltf-binary", SHA: foldAssetSHA, Role: library.RoleGeometryGLB, License: "AGPL-3.0-only", Attribution: "Vrooli-authored generic dual-rigid book-fold concept; Fold6 nominal envelope is a proportion reference only.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAuthored, SourceLabel: "Vrooli-authored neutral book-fold concept; no product geometry copied."}}
	manifest, sources, err := s.buildManifest(ctx, foldBundleID, []starterFile{file})
	if err != nil {
		return FoldableStarterState{}, err
	}
	bundle, err := s.publish(ctx, manifest, sources, foldReviewID)
	if err != nil {
		return FoldableStarterState{}, err
	}
	device, err := s.saveFoldableDevice(ctx, bundle)
	if err != nil {
		return FoldableStarterState{}, err
	}
	motion, err := s.saveFoldableMotion(ctx)
	if err != nil {
		return FoldableStarterState{}, err
	}
	placement := foldPlacement()
	scene, err := s.saveFoldableScene(ctx, room, device, motion, placement)
	if err != nil {
		return FoldableStarterState{}, err
	}
	preset, err := s.ensureFoldablePreset(ctx, device, scene, still, placement)
	if err != nil {
		return FoldableStarterState{}, err
	}
	return FoldableStarterState{Bundle: bundle, Device: device, Scene: scene, Motion: motion, Placement: placement, Preset: preset}, nil
}

func (s *starterInitializer) saveFoldableDevice(ctx context.Context, bundle library.BundleRevision) (devices.DeviceRevision, error) {
	ref := artifactRef(bundle, foldAssetID)
	data, err := fs.ReadFile(s.fs, foldAssetFile)
	if err != nil {
		return devices.DeviceRevision{}, err
	}
	if ref.SHA256 != foldAssetSHA || ref.SizeBytes != 22480 || sha256Hex(data) != foldAssetSHA {
		return devices.DeviceRevision{}, errors.New("embedded foldable concept differs from its pinned Library artifact")
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("inspect foldable concept GLB: %w", err)
	}
	if len(inventory.Nodes) != 8 || len(inventory.Meshes) != 6 || inventory.TriangleCount != 622 {
		return devices.DeviceRevision{}, fmt.Errorf("foldable concept inventory changed: nodes=%d meshes=%d triangles=%d", len(inventory.Nodes), len(inventory.Meshes), inventory.TriangleCount)
	}
	nodes := make(map[string]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodes[node.Name] = node
	}
	root, rootOK := nodes["FoldConceptRoot"]
	left, leftOK := nodes["LeftHousing"]
	pivot, pivotOK := nodes["MainHingePivot"]
	right, rightOK := nodes["RightHousing"]
	leftScreen, leftScreenOK := nodes["LeftInnerDisplay"]
	rightScreen, rightScreenOK := nodes["RightInnerDisplay"]
	outer, outerOK := nodes["OuterDisplay"]
	hinge, hingeOK := nodes["HingeBarrel"]
	if !rootOK || !leftOK || !pivotOK || !rightOK || !leftScreenOK || !rightScreenOK || !outerOK || !hingeOK || root.Parent != nil || left.Parent == nil || *left.Parent != root.Index || pivot.Parent == nil || *pivot.Parent != root.Index || right.Parent == nil || *right.Parent != pivot.Index || leftScreen.Parent == nil || *leftScreen.Parent != left.Index || rightScreen.Parent == nil || *rightScreen.Parent != right.Index || outer.Parent == nil || *outer.Parent != right.Index || hinge.Parent == nil || *hinge.Parent != root.Index {
		return devices.DeviceRevision{}, errors.New("foldable concept no longer has separate rigid halves, hinge hierarchy, or three display nodes")
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return devices.DeviceRevision{}, err
	}
	evidence := []devices.Evidence{
		{Source: foldManifestPath + " SHA-256 " + foldManifestSHA, Kind: "authored_design_manifest", Qualifier: "generic concept; nominal Samsung open envelope used for proportions only; not manufacturer geometry"},
		{Source: "https://www.samsung.com/ph/smartphones/galaxy-z-fold6/specs/", Kind: "manufacturer_reference", Qualifier: "open dimensions inform the generic concept envelope; no Samsung geometry, hinge axis, active boundary, or limits are asserted"},
	}
	displays := make([]devices.DisplaySurface, 0, 3)
	bindings := make([]devices.DisplayPartBinding, 0, 3)
	for _, spec := range []struct {
		id, part string
		node     rendering.PhysicalGLBNode
		material uint32
	}{{"inner-left", foldLeftInnerPart, leftScreen, 1}, {"inner-right", foldRightInnerPart, rightScreen, 2}, {"outer-cover", foldOuterPart, outer, 3}} {
		b, err := physicalGLBNodeMeshWorldBounds(inventory, spec.node.Index)
		if err != nil {
			return devices.DeviceRevision{}, err
		}
		w, h := (b.max.X-b.min.X)*1000, (b.max.Y-b.min.Y)*1000
		if !positiveFinite(w) || !positiveFinite(h) {
			return devices.DeviceRevision{}, errors.New("foldable display geometry has invalid dimensions")
		}
		metric := &devices.DisplayMetric{WidthMM: w, HeightMM: h, Evidence: evidence}
		displays = append(displays, devices.DisplaySurface{ID: spec.id, Topology: "planar", CoordinateMetric: metric, PhysicalBoundary: unknownRegion(), ActiveArea: unknownRegion(), AppSafeArea: unknownRegion(), Evidence: evidence})
		primitive, material := uint32(0), spec.material
		bindings = append(bindings, devices.DisplayPartBinding{DisplayID: spec.id, PartID: spec.part, LocalFrame: identityTRS(), ScreenPrimitiveIndex: &primitive, ScreenMaterialIndex: &material})
	}
	lower, upper := 0.0, foldClosedAngle
	dimensions := devices.Dimensions{Width: foldMeasurement((bounds.max.X-bounds.min.X)*1000, "source open-configuration X envelope"), Height: foldMeasurement((bounds.max.Y-bounds.min.Y)*1000, "source Y envelope"), Depth: foldMeasurement((bounds.max.Z-bounds.min.Z)*1000, "source open-configuration Z envelope")}
	definition := devices.Definition{
		SchemaVersion: "device-definition/v1", ID: foldDeviceID, Group: devices.GroupFoldable, Kind: "generic_dual_rigid_book_fold_concept",
		IdentityStatus: devices.IdentityConcept, Model: "Generic dual-rigid book-fold concept", Variant: "Vrooli-authored reference-proportioned geometry; not a named product",
		Description: "Unqualified local review candidate. It has two rigid panels, separate inner and outer neutral display surfaces, and an authored hinge interval. The flexible display, crease, Fold6 hinge, physical limits, product fit, and manufacturing dimensions are not represented or validated.",
		Dimensions:  map[string]devices.Dimensions{"source_open_geometry": dimensions}, Displays: displays,
		Joints:          []devices.Joint{{ID: foldJointID, ParentPart: foldRootPart, ChildPart: foldPivotPart, Axis: &devices.Vector3{Y: 1}, PivotMeters: &devices.Vector3{}, LowerRadians: &lower, UpperRadians: &upper, Status: "candidate_only", Evidence: []devices.Evidence{{Source: foldManifestPath, Kind: "authored_joint_assumption", Qualifier: "concept-only Y-axis pivot and 0–180-degree interval; not a measured hinge or Fold6 limits"}}}},
		SupportedPoses:  []devices.Pose{{Name: "open-flat", Status: "candidate_only", JointState: map[string]float64{foldJointID: 0}, Evidence: evidence}, {Name: "folded-closed", Status: "candidate_only", JointState: map[string]float64{foldJointID: foldClosedAngle}, Evidence: evidence}},
		SupportedMotion: []devices.Capability{{Name: "concept-book-fold-hinge", Status: "candidate_only", MotionKind: "articulated", JointIDs: []string{foldJointID}, Evidence: evidence}},
		Supports:        []devices.SupportCapability{{Kind: "tabletop", InterfaceID: "horizontal_surface", Status: "candidate_only", Evidence: evidence}},
		ModelCandidates: []devices.ModelCandidate{{ID: foldAssetID, Status: "candidate_only", SHA256: foldAssetSHA, Author: "Vrooli", License: "AGPL-3.0-only", Attribution: "Vrooli-authored generic book-fold concept; Samsung Fold6 dimensions are a design reference only.", RightsNotes: "Internal candidate review only. No Fold6 geometry, manufacturer CAD, brand endorsement, physical-device measurement, hinge qualification, or commercial asset clearance is claimed."}},
		Provenance:      evidence, Qualification: "unqualified",
	}
	parts := []devices.SemanticPart{
		{ID: foldRootPart, Role: "device_root", NodeIndex: root.Index, NodeName: root.Name, LocalFrame: root.Frame},
		{ID: foldLeftPart, Role: "rigid_panel", ParentID: foldRootPart, NodeIndex: left.Index, NodeName: left.Name, LocalFrame: left.Frame},
		{ID: foldPivotPart, Role: "hinge_pivot", ParentID: foldRootPart, NodeIndex: pivot.Index, NodeName: pivot.Name, LocalFrame: pivot.Frame},
		{ID: foldRightPart, Role: "rigid_panel", ParentID: foldPivotPart, NodeIndex: right.Index, NodeName: right.Name, LocalFrame: right.Frame},
		{ID: foldLeftInnerPart, Role: "display_surface", ParentID: foldLeftPart, NodeIndex: leftScreen.Index, NodeName: leftScreen.Name, LocalFrame: leftScreen.Frame},
		{ID: foldRightInnerPart, Role: "display_surface", ParentID: foldRightPart, NodeIndex: rightScreen.Index, NodeName: rightScreen.Name, LocalFrame: rightScreen.Frame},
		{ID: foldOuterPart, Role: "display_surface", ParentID: foldRightPart, NodeIndex: outer.Index, NodeName: outer.Name, LocalFrame: outer.Frame},
		{ID: foldHingePart, Role: "hinge_barrel", ParentID: foldRootPart, NodeIndex: hinge.Index, NodeName: hinge.Name, LocalFrame: hinge.Frame},
	}
	candidate := devices.DeviceCandidate{Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention, SourceToCanonical: identityTRS(), Parts: parts, DisplayBindings: bindings, SupportFrames: []devices.DeviceSupportFrame{{ID: foldDeviceSupportID, InterfaceID: "horizontal_surface", PartID: foldRootPart, LocalFrame: identityTRS()}}}
	for _, display := range displays {
		if _, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, display.ID); err != nil {
			return devices.DeviceRevision{}, fmt.Errorf("preflight %s binding: %w", display.ID, err)
		}
	}
	return saveDeviceCandidateIfUnchanged(ctx, s.devices, candidate)
}

func (s *starterInitializer) saveFoldableMotion(ctx context.Context) (motions.Revision, error) {
	open, middle, closed := 0.0, foldClosedAngle/2, foldClosedAngle
	candidate := motions.MotionCandidate{Version: motions.CandidateVersion, MotionID: foldMotionID, Timebase: composition.Rational{Num: 24, Den: 1}, DurationTicks: 48, RequiredCapabilities: []string{"concept-book-fold-hinge"}, Tracks: []motions.MotionTrack{{Target: motions.TrackTarget{JointID: foldJointID}, Interpolation: motions.Linear, Keys: []motions.MotionKey{{Tick: 0, AngleRad: &open}, {Tick: 24, AngleRad: &middle}, {Tick: 47, AngleRad: &closed}, {Tick: 48, AngleRad: &closed}}}}}
	return saveLaptopMotionCandidate(ctx, s.motions, candidate)
}

func (s *starterInitializer) saveFoldableScene(ctx context.Context, room scenes.Revision, device devices.DeviceRevision, motion motions.Revision, placement ReviewedMonitorPlacement) (scenes.Revision, error) {
	if room.SceneID != laptopFramedSceneID || device.DeviceID != foldDeviceID || motion.MotionID != foldMotionID || len(room.Candidate.CameraFrames) != 1 {
		return scenes.Revision{}, errors.New("foldable scene is not bound to the pinned study room, device, and motion")
	}
	data, err := fs.ReadFile(s.fs, foldAssetFile)
	if err != nil {
		return scenes.Revision{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return scenes.Revision{}, err
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return scenes.Revision{}, err
	}
	points, err := monitorPlacedBounds(bounds, device.Candidate.SourceToCanonical, placement)
	if err != nil {
		return scenes.Revision{}, err
	}
	candidate := room.Candidate
	candidate.SceneID = foldSceneID
	candidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), candidate.CameraFrames...)
	camera := candidate.CameraFrames[0]
	camera.ID, camera.SensorFit = foldCameraID, scenes.SensorFitHorizontal
	displayNode := uint32(0)
	for _, node := range inventory.Nodes {
		if node.Name == "LeftInnerDisplay" {
			displayNode = node.Index
			break
		}
	}
	if displayNode == 0 {
		return scenes.Revision{}, errors.New("foldable focus display node is missing")
	}
	displayBounds, err := physicalGLBNodeMeshWorldBounds(inventory, displayNode)
	if err != nil {
		return scenes.Revision{}, err
	}
	displayPoints, err := monitorPlacedBounds(displayBounds, device.Candidate.SourceToCanonical, placement)
	if err != nil {
		return scenes.Revision{}, err
	}
	target := meanPhysicalPoints(displayPoints)
	camera.Frame, err = fitPerspectiveCameraPoints(points, camera, 16.0/9.0, .10)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("fit foldable camera to open source envelope: %w", err)
	}
	focusDistance, err := watchCameraAxisDistance(camera.Frame, target)
	if err != nil {
		return scenes.Revision{}, err
	}
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusDistance, DistanceM: &focusDistance}
	candidate.CameraFrames[0] = camera
	return saveSceneCandidateIfUnchanged(ctx, s.scenes, candidate)
}

func (s *starterInitializer) ensureFoldablePreset(ctx context.Context, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedMonitorPlacement) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{Device: &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256}, Scene: &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256}, Motion: &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256}, DisplayId: "inner-left", DeviceSupportId: foldDeviceSupportID, SceneSupportId: laptopTableSupportID, CameraId: foldCameraID, SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest), StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1}}
	if current, err := s.presets.Head(ctx, foldPresetID); err == nil {
		if current.Name == foldStillName && proto.Equal(current.Setup, setup) {
			return current, nil
		}
		return shots.PhysicalPresetRevision{}, fmt.Errorf("foldable preset differs from its fixed candidate; refusing implicit revision: %w", shots.ErrPhysicalPresetConflict)
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, foldPresetID, 0, foldStillName, setup)
}

func foldPlacement() ReviewedMonitorPlacement {
	identity := identityTRS()
	flatOnTable := physical.TRS{Translation: physical.Vec3{Y: foldSupportLiftM}, Rotation: physical.QuatXYZW{X: -math.Sqrt(.5), W: math.Sqrt(.5)}, Scale: identity.Scale}
	return ReviewedMonitorPlacement{DisplayID: "inner-left", DeviceSupportID: foldDeviceSupportID, SceneSupportID: laptopTableSupportID, CameraID: foldCameraID, SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: identity.Rotation, Scale: identity.Scale}, DeviceRootRest: flatOnTable}
}

func foldMeasurement(mm float64, meaning string) devices.Measurement {
	value := mm
	return devices.Measurement{Value: &value, Unit: "mm", Meaning: meaning, Method: "derived", Uncertainty: devices.Uncertainty{Status: "unknown"}, Evidence: []devices.Evidence{{Source: foldManifestPath, Kind: "derived_concept_geometry", Qualifier: "authored mesh envelope only; not device measurement"}}}
}

func meanPhysicalPoints(points []physical.Vec3) physical.Vec3 {
	mean := physical.Vec3{}
	for _, point := range points {
		mean.X += point.X
		mean.Y += point.Y
		mean.Z += point.Z
	}
	if len(points) == 0 {
		return mean
	}
	count := float64(len(points))
	return physical.Vec3{X: mean.X / count, Y: mean.Y / count, Z: mean.Z / count}
}
