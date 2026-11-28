package physicalruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
)

const (
	ipadBundleID                   = "device-studio-starter-ipad-mini-6-datsketch-neutral-uv-v1"
	ipadSourceSHA256               = "a462088b7bd84b7752802c109beee04110047c3b1a4f355624c863f790e93d5e"
	ipadNeutralDerivativeSHA256    = "8cd65349dcd38cb4f47fd897779db97dbbbf0fc7f6c81cd6c2ab05eaa6b47b5e"
	ipadDerivativeSHA256           = "16d88943240dffae0e14e4743beefa1ed61e7efb4b5aa6cb00ecf2ac4b63749c"
	ipadCandidateID                = "datsketch-ipad-mini-6-neutral-uv-normalized-v1"
	ReviewedIPadPlacementProfileID = "device-studio-reviewed-ipad-mini-candidate-placement-v1"
	ipadSceneID                    = "ipad-study-environment"
	ipadCameraID                   = "ipad-study-camera-16x9-fit-v1"
)

// IPadStarterState is deliberately separate from StarterState so the existing
// phone initialization response and startup contract remain unchanged.
type IPadStarterState struct {
	Bundle    library.BundleRevision
	Device    devices.DeviceRevision
	Scene     scenes.Revision
	Motion    motions.Revision
	Placement ReviewedIPadPlacement
	Preset    shots.PhysicalPresetRevision
}

// ReviewedIPadPlacement is a candidate-only rest pose derived from the pinned
// source mesh. It records a tabletop hypothesis, not a support qualification.
type ReviewedIPadPlacement struct {
	ProfileID         string
	ProfileVersion    uint32
	DisplayID         string
	DeviceSupportID   string
	SceneSupportID    string
	SelectedCameraID  string
	SceneSupportWorld physical.TRS
	SupportPivotM     physical.Vec3
	DeviceRootRest    physical.TRS
}

// InitializeReviewedIPadStarter publishes the UV-normalized neutral DatSketch derivative
// and saves its immutable device candidate through the existing owners. The
// phone starter supplies the shared scene and identity motion candidates.
func (o Owners) InitializeReviewedIPadStarter(ctx context.Context) (IPadStarterState, error) {
	if o.Starter == nil {
		return IPadStarterState{}, errors.New("reviewed starter initializer is unavailable")
	}
	ctx, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return IPadStarterState{}, err
	}
	base, err := o.Starter.Initialize(ctx)
	if err != nil {
		return IPadStarterState{}, err
	}
	bundle, err := o.Starter.publishIPadBundle(ctx)
	if err != nil {
		return IPadStarterState{}, err
	}
	device, err := o.Starter.saveIPadCandidate(ctx, bundle)
	if err != nil {
		return IPadStarterState{}, err
	}
	placement, err := reviewedIPadPlacement(device, base.Scene)
	if err != nil {
		return IPadStarterState{}, err
	}
	ipadScene, err := o.Starter.saveIPadScene(ctx, base.Scene, device, placement)
	if err != nil {
		return IPadStarterState{}, err
	}
	return IPadStarterState{Bundle: bundle, Device: device, Scene: ipadScene, Motion: base.Motion, Placement: placement}, nil
}

func (s *starterInitializer) saveIPadScene(ctx context.Context, source scenes.Revision, device devices.DeviceRevision, placement ReviewedIPadPlacement) (scenes.Revision, error) {
	if source.SceneID != "phone-study-environment" || device.DeviceID != ipadCandidateID {
		return scenes.Revision{}, errors.New("iPad camera scene is bound to unexpected starter candidates")
	}
	data, err := s.readPinnedIPad(device.Candidate.Geometry)
	if err != nil {
		return scenes.Revision{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("inspect iPad geometry for camera fit: %w", err)
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("measure iPad geometry for camera fit: %w", err)
	}
	candidate := source.Candidate
	candidate.SceneID = ipadSceneID
	candidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), source.Candidate.CameraFrames...)
	if len(candidate.CameraFrames) != 1 || candidate.CameraFrames[0].ID != "phone-study-camera" {
		return scenes.Revision{}, errors.New("phone scene camera inventory differs from the pinned starter")
	}
	camera := candidate.CameraFrames[0]
	camera.ID = ipadCameraID
	camera.SensorFit = scenes.SensorFitHorizontal
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusSelectedDevice}
	fitFrame, err := fitIPadPerspectiveCamera(bounds, device.Candidate.SourceToCanonical, placement, camera, 16.0/9.0, .10)
	if err != nil {
		return scenes.Revision{}, err
	}
	camera.Frame = fitFrame
	candidate.CameraFrames[0] = camera
	existing, err := s.scenes.GetHead(ctx, ipadSceneID)
	if err == nil {
		_, _, digest, canonErr := scenes.CanonicalizeCandidate(candidate)
		if canonErr != nil {
			return scenes.Revision{}, canonErr
		}
		if existing.SHA256 == digest {
			return existing, nil
		}
		return scenes.Revision{}, fmt.Errorf("iPad scene head differs from its fixed camera-fit candidate; refusing implicit revision: %w", scenes.ErrRevisionConflict)
	}
	if !errors.Is(err, scenes.ErrRevisionNotFound) {
		return scenes.Revision{}, err
	}
	return s.scenes.SaveCandidate(ctx, candidate, 0)
}

// fitIPadPerspectiveCamera fits conservative transformed geometry bounds in
// the pinned camera direction. Exact vertices are checked by the source test.
func fitIPadPerspectiveCamera(bounds physicalBounds, sourceToCanonical physical.TRS, placement ReviewedIPadPlacement, camera scenes.SceneCameraFrame, aspect, margin float64) (physical.TRS, error) {
	if !bounds.valid() || camera.Projection != scenes.ProjectionPerspective || camera.FocalLengthMM == nil || camera.SensorWidthMM == nil || camera.SensorHeightMM == nil || !positiveFinite(*camera.FocalLengthMM) || !positiveFinite(*camera.SensorWidthMM) || !positiveFinite(*camera.SensorHeightMM) || !positiveFinite(aspect) || margin < 0 || margin >= .5 {
		return physical.TRS{}, errors.New("iPad camera fit requires bounded geometry, a valid perspective sensor, aspect, and margin")
	}
	sceneMatrix, err := placement.SceneSupportWorld.Matrix()
	if err != nil {
		return physical.TRS{}, err
	}
	pivotMatrix, err := (physical.TRS{Translation: placement.SupportPivotM, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}).Matrix()
	if err != nil {
		return physical.TRS{}, err
	}
	rootMatrix, err := placement.DeviceRootRest.Matrix()
	if err != nil {
		return physical.TRS{}, err
	}
	sourceMatrix, err := sourceToCanonical.Matrix()
	if err != nil {
		return physical.TRS{}, err
	}
	worldMatrix, err := physical.Compose(sceneMatrix, pivotMatrix)
	if err == nil {
		worldMatrix, err = physical.Compose(worldMatrix, rootMatrix)
	}
	if err == nil {
		worldMatrix, err = physical.Compose(worldMatrix, sourceMatrix)
	}
	if err != nil {
		return physical.TRS{}, err
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
		point, err = physical.TransformPoint(worldMatrix, point)
		if err != nil {
			return physical.TRS{}, err
		}
		points = append(points, point)
	}
	return fitPerspectiveCameraPoints(points, camera, aspect, margin)
}

// fitPerspectiveCameraPoints centers a perspective camera on already placed
// world-space geometry. Callers supply conservative points covering every
// supported pose; this helper changes only camera translation and preserves
// the supplied view direction.
func fitPerspectiveCameraPoints(points []physical.Vec3, camera scenes.SceneCameraFrame, aspect, margin float64) (physical.TRS, error) {
	if len(points) == 0 || camera.Projection != scenes.ProjectionPerspective || camera.FocalLengthMM == nil || camera.SensorWidthMM == nil || camera.SensorHeightMM == nil || !positiveFinite(*camera.FocalLengthMM) || !positiveFinite(*camera.SensorWidthMM) || !positiveFinite(*camera.SensorHeightMM) || !positiveFinite(aspect) || margin < 0 || margin >= .5 {
		return physical.TRS{}, errors.New("camera fit requires bounded world points, a valid perspective sensor, aspect, and margin")
	}
	center := physical.Vec3{X: (minAxis(points, 0) + maxAxis(points, 0)) / 2, Y: (minAxis(points, 1) + maxAxis(points, 1)) / 2, Z: (minAxis(points, 2) + maxAxis(points, 2)) / 2}
	cameraMatrix, err := camera.Frame.Matrix()
	if err != nil {
		return physical.TRS{}, err
	}
	origin, err := physical.TransformPoint(cameraMatrix, physical.Vec3{})
	if err != nil {
		return physical.TRS{}, err
	}
	axis := func(v physical.Vec3) (physical.Vec3, error) {
		p, err := physical.TransformPoint(cameraMatrix, v)
		if err != nil {
			return physical.Vec3{}, err
		}
		return normalize(physical.Vec3{X: p.X - origin.X, Y: p.Y - origin.Y, Z: p.Z - origin.Z}), nil
	}
	right, err := axis(physical.Vec3{X: 1})
	if err != nil {
		return physical.TRS{}, err
	}
	up, err := axis(physical.Vec3{Y: 1})
	if err != nil {
		return physical.TRS{}, err
	}
	forward, err := axis(physical.Vec3{Z: -1})
	if err != nil {
		return physical.TRS{}, err
	}
	coords := make([][3]float64, len(points))
	minDepth := math.Inf(1)
	for i, p := range points {
		d := subVec(p, center)
		coords[i] = [3]float64{dotVec(d, right), dotVec(d, up), dotVec(d, forward)}
		minDepth = math.Min(minDepth, coords[i][2])
	}
	sensorWidth := *camera.SensorWidthMM
	sensorHeight := sensorWidth / aspect
	if camera.SensorFit == scenes.SensorFitVertical {
		sensorHeight = *camera.SensorHeightMM
		sensorWidth = sensorHeight * aspect
	}
	if camera.SensorFit != scenes.SensorFitHorizontal && camera.SensorFit != scenes.SensorFitVertical {
		return physical.TRS{}, errors.New("iPad camera fit requires an explicit sensor fit")
	}
	allowed := .5 - margin
	centerAxis := func(distance float64, component int) float64 {
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, c := range coords {
			lo = math.Min(lo, c[component])
			hi = math.Max(hi, c[component])
		}
		span := hi - lo + 1
		lo, hi = lo-span*4, hi+span*4
		for n := 0; n < 80; n++ {
			mid := (lo + hi) / 2
			mn, mx := math.Inf(1), math.Inf(-1)
			for _, c := range coords {
				q := (c[component] - mid) / (distance + c[2])
				mn = math.Min(mn, q)
				mx = math.Max(mx, q)
			}
			if mn+mx > 0 {
				lo = mid
			} else {
				hi = mid
			}
		}
		return (lo + hi) / 2
	}
	fits := func(distance float64) (float64, float64, float64, float64) {
		cx, cy := centerAxis(distance, 0), centerAxis(distance, 1)
		maxX, maxY := 0.0, 0.0
		for _, c := range coords {
			depth := distance + c[2]
			maxX = math.Max(maxX, math.Abs((c[0]-cx)*(*camera.FocalLengthMM/sensorWidth)/depth))
			maxY = math.Max(maxY, math.Abs((c[1]-cy)*(*camera.FocalLengthMM/sensorHeight)/depth))
		}
		return cx, cy, maxX, maxY
	}
	low := math.Max(.01, .01-minDepth)
	high := math.Max(.75, low*2)
	for n := 0; n < 32; n++ {
		_, _, x, y := fits(high)
		if x <= allowed && y <= allowed {
			break
		}
		high *= 2
	}
	_, _, x, y := fits(high)
	if x > allowed || y > allowed || !positiveFinite(high) {
		return physical.TRS{}, errors.New("iPad camera bounds cannot fit within the pinned safe frame")
	}
	for n := 0; n < 72; n++ {
		mid := (low + high) / 2
		_, _, x, y := fits(mid)
		if x <= allowed && y <= allowed {
			high = mid
		} else {
			low = mid
		}
	}
	cx, cy, _, _ := fits(high)
	position := addVec(center, addVec(scaleVec(right, cx), addVec(scaleVec(up, cy), scaleVec(forward, -high))))
	camera.Frame.Translation = position
	return camera.Frame, nil
}

func minAxis(points []physical.Vec3, axis int) float64 {
	v := math.Inf(1)
	for _, p := range points {
		x := []float64{p.X, p.Y, p.Z}[axis]
		v = math.Min(v, x)
	}
	return v
}
func maxAxis(points []physical.Vec3, axis int) float64 {
	v := math.Inf(-1)
	for _, p := range points {
		x := []float64{p.X, p.Y, p.Z}[axis]
		v = math.Max(v, x)
	}
	return v
}
func dotVec(a, b physical.Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func subVec(a, b physical.Vec3) physical.Vec3 {
	return physical.Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}
func addVec(a, b physical.Vec3) physical.Vec3 {
	return physical.Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}
func scaleVec(a physical.Vec3, s float64) physical.Vec3 {
	return physical.Vec3{X: a.X * s, Y: a.Y * s, Z: a.Z * s}
}

func (s *starterInitializer) publishIPadBundle(ctx context.Context) (library.BundleRevision, error) {
	file := starterFile{
		ID: "datsketch-ipad-mini-6-neutral-uv-normalized-glb", File: "assets/ipad/ipad-mini-screen-uv-normalized.glb", MIME: "model/gltf-binary",
		SHA: ipadDerivativeSHA256, Role: library.RoleGeometryGLB, License: "CC-BY-4.0",
		Attribution: "iPad mini 6 2021 model by DatSketch, adapted under CC BY 4.0; source screen image replaced with a neutral placeholder and screen UV chart normalized.",
		Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted,
			SourceLabel:  "DatSketch, IPad Mini 6 2021, Sketchfab model f363c2b1c74a462588db184bd4fb024e; CC BY 4.0.",
			SourceSHA256: ipadSourceSHA256,
			Adaptation:   "Starting from the neutral-screen derivative, added a private normalized TEXCOORD_0 buffer view for the display primitive. Existing BIN bytes, mesh positions, other UVs, materials, images, sampler bindings, and node transforms are preserved; the source wallpaper is excluded.",
		},
	}
	manifest, sources, err := s.buildManifest(ctx, ipadBundleID, []starterFile{file})
	if err != nil {
		return library.BundleRevision{}, err
	}
	return s.publish(ctx, manifest, sources, "device-studio-e3-starter-local-render-review-v1-ipad-uv-normalized")
}

func (s *starterInitializer) saveIPadCandidate(ctx context.Context, bundle library.BundleRevision) (devices.DeviceRevision, error) {
	ref := artifactRef(bundle, "datsketch-ipad-mini-6-neutral-uv-normalized-glb")
	data, err := s.readPinnedIPad(ref)
	if err != nil {
		return devices.DeviceRevision{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("inspect neutral iPad starter: %w", err)
	}
	if len(inventory.Nodes) != 18 || len(inventory.Meshes) != 13 || inventory.TriangleCount != 39462 || len(inventory.Extensions) != 0 {
		return devices.DeviceRevision{}, errors.New("neutral iPad GLB differs from the pinned 18-node source hierarchy")
	}
	nodes := make(map[uint32]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodes[node.Index] = node
	}
	root, rootOK := nodes[17]
	screen, screenOK := nodes[14]
	if !rootOK || root.Name != "sceneview_root" || root.Parent != nil || !screenOK || screen.Name != "iPad Mini_Screen_0" || screen.Mesh == nil || *screen.Mesh != 10 {
		return devices.DeviceRevision{}, errors.New("neutral iPad root or screen binding differs from the reviewed GLB inventory")
	}
	meshBounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("derive neutral iPad bounds: %w", err)
	}
	screenBounds, err := physicalGLBNodeMeshWorldBounds(inventory, screen.Index)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("derive neutral iPad screen metric: %w", err)
	}
	widthMM := (meshBounds.max.X - meshBounds.min.X) * 1000
	heightMM := (meshBounds.max.Y - meshBounds.min.Y) * 1000
	depthMM := (meshBounds.max.Z - meshBounds.min.Z) * 1000
	screenWidthMM := (screenBounds.max.X - screenBounds.min.X) * 1000
	screenHeightMM := (screenBounds.max.Y - screenBounds.min.Y) * 1000
	if !positiveFinite(widthMM) || !positiveFinite(heightMM) || !positiveFinite(depthMM) || !positiveFinite(screenWidthMM) || !positiveFinite(screenHeightMM) || math.Abs(widthMM-134.423) > .02 || math.Abs(heightMM-195) > .02 || math.Abs(depthMM-8.775) > .02 || math.Abs(screenWidthMM-117.281) > .02 || math.Abs(screenHeightMM-177.229) > .02 {
		return devices.DeviceRevision{}, fmt.Errorf("neutral iPad geometry metrics differ from retained source measurements: body=%.6fx%.6fx%.6fmm screen=%.6fx%.6fmm", widthMM, heightMM, depthMM, screenWidthMM, screenHeightMM)
	}
	evidence := []devices.Evidence{{Source: "DatSketch iPad Mini 6 GLB source SHA-256 " + ipadSourceSHA256 + "; neutral derivative SHA-256 " + ref.SHA256, Kind: "derived_asset_geometry", Qualifier: "glTF metre assumption; not an independent device measurement or physical qualification"}}
	definition := devices.Definition{
		SchemaVersion: "device-definition/v1", ID: ipadCandidateID, Group: devices.GroupTablet,
		Kind: "generic_tablet", IdentityStatus: devices.IdentityGeneric, Model: "iPad mini 6 — DatSketch model", Variant: "neutral-screen source-preserving adaptation with normalized display UVs",
		Description: "DatSketch's iPad mini 6 model adapted with a neutral screen and normalized display UV chart. Source geometry measurements only; not manufacturer CAD, independently measured hardware, or Apple-endorsed.",
		Dimensions: map[string]devices.Dimensions{"portrait_unrotated": {
			Width: derivedGeometryMeasurement(widthMM, "neutral derivative world-space X span"), Height: derivedGeometryMeasurement(heightMM, "neutral derivative world-space Y span"), Depth: derivedGeometryMeasurement(depthMM, "neutral derivative world-space Z span; includes protruding features"),
		}},
		Displays:        []devices.DisplaySurface{{ID: "main_display", Topology: "planar", CoordinateMetric: &devices.DisplayMetric{WidthMM: screenWidthMM, HeightMM: screenHeightMM, Evidence: evidence}, PhysicalBoundary: unknownRegion(), ActiveArea: unknownRegion(), AppSafeArea: unknownRegion()}},
		SupportedMotion: []devices.Capability{{Name: "rigid_root_transform", Status: "candidate_only", MotionKind: "rigid_body"}},
		Supports:        []devices.SupportCapability{{Kind: "tabletop", InterfaceID: "horizontal_surface", Status: "candidate_only", Evidence: evidence}},
		ModelCandidates: []devices.ModelCandidate{{ID: "datsketch-ipad-mini-6-neutral-uv-normalized-v1", Status: "candidate_only", ListingURL: "https://sketchfab.com/3d-models/ipad-mini-6-2021-f363c2b1c74a462588db184bd4fb024e", SHA256: ref.SHA256, Author: "DatSketch", License: "CC BY 4.0", Attribution: "iPad mini 6 2021 by DatSketch, adapted under CC BY 4.0; embedded screen image replaced and display UVs normalized.", RightsNotes: "Candidate-only local rendering/editing; portable geometry is bound by its Library artifact reference; no manufacturer endorsement or physical measurement claim."}},
		Provenance:      []devices.Evidence{{Source: "https://sketchfab.com/3d-models/ipad-mini-6-2021-f363c2b1c74a462588db184bd4fb024e (model f363c2b1c74a462588db184bd4fb024e)", Kind: "creator_listing", Qualifier: "DatSketch; CC BY 4.0; screen image 1 replaced by neutral raster; creator attribution retained"}, {Source: "ipad-screen-uv-normalization-manifest.json and ipad-screen-uv-normalization.json", Kind: "source_geometry_and_derivation", Qualifier: "display UV values affinely normalized from source chart bounds; geometry metrics remain GLB-derived; no physical qualification"}},
		Qualification:   "unqualified",
	}
	parts := make([]devices.SemanticPart, 0, len(inventory.Nodes))
	partIDs := make(map[uint32]string, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		partIDs[node.Index] = ipadPartID(node.Index)
	}
	for _, node := range inventory.Nodes {
		role := "body_component"
		if node.Index == 17 {
			role = "device_root"
		}
		if node.Index == 14 {
			role = "display_surface"
		}
		if node.Index >= 6 && node.Index <= 12 || node.Index == 15 {
			role = "camera_hardware"
		}
		parent := ""
		if node.Parent != nil {
			parent = partIDs[*node.Parent]
		}
		parts = append(parts, devices.SemanticPart{ID: partIDs[node.Index], Role: role, ParentID: parent, NodeIndex: node.Index, NodeName: node.Name, LocalFrame: node.Frame})
	}
	candidate := devices.DeviceCandidate{
		Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention,
		SourceToCanonical: physical.TRS{Rotation: physical.QuatXYZW{Y: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
		Parts:             parts,
		DisplayBindings:   []devices.DisplayPartBinding{{DisplayID: "main_display", PartID: partIDs[14], LocalFrame: identityTRS()}},
		SupportFrames:     []devices.DeviceSupportFrame{{ID: "device-root-support", InterfaceID: "horizontal_surface", PartID: partIDs[17], LocalFrame: identityTRS()}},
	}
	if _, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, "main_display"); err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("preflight normalized iPad display geometry: %w", err)
	}
	existing, err := s.devices.GetHead(ctx, candidate.Definition.ID)
	if err == nil {
		_, _, digest, canonErr := devices.CanonicalizeDeviceCandidate(candidate)
		if canonErr != nil {
			return devices.DeviceRevision{}, canonErr
		}
		if existing.SHA256 == digest {
			return existing, nil
		}
		return devices.DeviceRevision{}, fmt.Errorf("iPad starter head differs from its fixed candidate; refusing implicit revision: %w", devices.ErrRevisionConflict)
	}
	if !errors.Is(err, devices.ErrRevisionNotFound) {
		return devices.DeviceRevision{}, err
	}
	return s.devices.SaveCandidate(ctx, candidate, 0)
}

func (s *starterInitializer) readPinnedIPad(ref physical.ArtifactRef) ([]byte, error) {
	data, err := fs.ReadFile(s.fs, "assets/ipad/ipad-mini-screen-uv-normalized.glb")
	if err != nil {
		return nil, err
	}
	if len(data) != 1938936 || sha256Hex(data) != ipadDerivativeSHA256 || ref.SHA256 != ipadDerivativeSHA256 || ref.SizeBytes != int64(len(data)) || ref.Role != library.RoleGeometryGLB {
		return nil, errors.New("embedded iPad derivative differs from its pinned Library artifact")
	}
	return data, nil
}

func reviewedIPadPlacement(device devices.DeviceRevision, scene scenes.Revision) (ReviewedIPadPlacement, error) {
	if device.DeviceID != ipadCandidateID || scene.SceneID != "phone-study-environment" {
		return ReviewedIPadPlacement{}, errors.New("iPad candidate placement is bound to different starter candidates")
	}
	root, screenPart, displayBinding := false, false, false
	deviceSupport, sceneSupport := false, false
	for _, part := range device.Candidate.Parts {
		root = root || part.ID == ipadPartID(17) && part.NodeIndex == 17
		screenPart = screenPart || part.ID == ipadPartID(14) && part.NodeIndex == 14
	}
	for _, binding := range device.Candidate.DisplayBindings {
		displayBinding = displayBinding || binding.DisplayID == "main_display" && binding.PartID == ipadPartID(14)
	}
	for _, frame := range device.Candidate.SupportFrames {
		deviceSupport = deviceSupport || frame.ID == "device-root-support" && frame.PartID == ipadPartID(17)
	}
	for _, frame := range scene.Candidate.SupportFrames {
		sceneSupport = sceneSupport || frame.ID == "table-top-flat"
	}
	if !root || !screenPart || !displayBinding || !deviceSupport || !sceneSupport {
		return ReviewedIPadPlacement{}, errors.New("candidate-only iPad placement bindings differ from the saved candidates")
	}
	angle := -89.430831 * math.Pi / 180
	return ReviewedIPadPlacement{
		ProfileID: ReviewedIPadPlacementProfileID, ProfileVersion: 1, DisplayID: "main_display", DeviceSupportID: "device-root-support", SceneSupportID: "table-top-flat", SelectedCameraID: ipadCameraID,
		SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
		SupportPivotM:     physical.Vec3{},
		DeviceRootRest:    physical.TRS{Translation: physical.Vec3{Y: .002631086703401778}, Rotation: physical.QuatXYZW{X: math.Sin(angle / 2), W: math.Cos(angle / 2)}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
	}, nil
}

func ipadPartID(index uint32) string { return fmt.Sprintf("ipad-node-%02d", index) }

func positiveFinite(v float64) bool { return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }

func physicalGLBNodeMeshWorldBounds(inventory rendering.PhysicalGLBInventory, index uint32) (physicalBounds, error) {
	nodes := make(map[uint32]rendering.PhysicalGLBNode, len(inventory.Nodes))
	meshes := make(map[uint32]rendering.PhysicalGLBMesh, len(inventory.Meshes))
	for _, node := range inventory.Nodes {
		nodes[node.Index] = node
	}
	for _, mesh := range inventory.Meshes {
		meshes[mesh.Index] = mesh
	}
	world := make(map[uint32]physical.Matrix4)
	var matrixFor func(uint32) (physical.Matrix4, error)
	matrixFor = func(nodeIndex uint32) (physical.Matrix4, error) {
		if cached, ok := world[nodeIndex]; ok {
			return cached, nil
		}
		node, ok := nodes[nodeIndex]
		if !ok {
			return physical.Matrix4{}, fmt.Errorf("GLB node %d is missing", nodeIndex)
		}
		local, err := node.Frame.Matrix()
		if err != nil {
			return physical.Matrix4{}, err
		}
		if node.Parent != nil {
			parent, err := matrixFor(*node.Parent)
			if err != nil {
				return physical.Matrix4{}, err
			}
			local, err = physical.Compose(parent, local)
			if err != nil {
				return physical.Matrix4{}, err
			}
		}
		world[nodeIndex] = local
		return local, nil
	}
	node, ok := nodes[index]
	if !ok || node.Mesh == nil {
		return physicalBounds{}, errors.New("screen node has no mesh")
	}
	mesh, ok := meshes[*node.Mesh]
	if !ok {
		return physicalBounds{}, errors.New("screen node references an absent mesh")
	}
	matrix, err := matrixFor(index)
	if err != nil {
		return physicalBounds{}, err
	}
	bounds := physicalBounds{min: physical.Vec3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}, max: physical.Vec3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}}
	for _, primitive := range mesh.Primitives {
		for mask := 0; mask < 8; mask++ {
			p := physical.Vec3{}
			for axis := 0; axis < 3; axis++ {
				v := primitive.PositionMin[axis]
				if mask&(1<<axis) != 0 {
					v = primitive.PositionMax[axis]
				}
				switch axis {
				case 0:
					p.X = v
				case 1:
					p.Y = v
				case 2:
					p.Z = v
				}
			}
			p, err = physical.TransformPoint(matrix, p)
			if err != nil {
				return physicalBounds{}, err
			}
			bounds.include(p)
		}
	}
	if !bounds.valid() {
		return physicalBounds{}, errors.New("screen mesh has no bounded position primitives")
	}
	return bounds, nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
