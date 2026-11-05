package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

const (
	foldUVAssetID   = "vrooli-generic-book-fold-concept-v3-left-screen-vflip"
	foldUVAssetFile = "assets/foldable-v3/foldable-concept-v3.glb"
	foldUVAssetSHA  = "5f16cb936cb7ed63e55be84cf00ab642c350d87479662b88e86515996498a6cb"
	foldUVBundleID  = "device-studio-starter-generic-book-fold-concept-v3-left-screen-vflip"
	foldUVDeviceID  = "generic-dual-rigid-book-fold-concept-v3-left-screen-vflip"
	foldUVPresetID  = "generic-book-fold-still-v3-left-screen-vflip"
	foldUVReviewID  = "device-studio-e4-foldable-left-screen-vflip-v3"
)

// InitializeReviewedFoldableUVV3 publishes a separate source-mapped device
// revision and preset, preserving the original v2 candidate and its owners.
func (o Owners) InitializeReviewedFoldableUVV3(ctx context.Context) (FoldableStarterState, error) {
	base, err := o.InitializeReviewedFoldableCandidate(ctx)
	if err != nil {
		return FoldableStarterState{}, err
	}
	file := starterFile{ID: foldUVAssetID, File: foldUVAssetFile, MIME: "model/gltf-binary", SHA: foldUVAssetSHA, Role: library.RoleGeometryGLB, License: "AGPL-3.0-only", Attribution: "Vrooli-authored generic dual-rigid book-fold concept; screen-mapping derivative.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "Vrooli generic book-fold v2; private screen UV view derivative.", SourceSHA256: foldAssetSHA, Adaptation: "Append a private TEXCOORD_0 accessor for LeftInnerDisplay mesh 1 primitive 0 and map V to 1-V. Source BIN bytes, all geometry, hierarchy, materials, and other screen mappings remain unchanged."}}
	manifest, sources, err := o.Starter.buildManifest(ctx, foldUVBundleID, []starterFile{file})
	if err != nil {
		return FoldableStarterState{}, err
	}
	bundle, err := o.Starter.publish(ctx, manifest, sources, foldUVReviewID)
	if err != nil {
		return FoldableStarterState{}, err
	}
	ref := artifactRef(bundle, foldUVAssetID)
	data, err := fs.ReadFile(o.Starter.fs, foldUVAssetFile)
	if err != nil {
		return FoldableStarterState{}, err
	}
	if ref.SHA256 != foldUVAssetSHA || ref.SizeBytes != int64(len(data)) || sha256Hex(data) != foldUVAssetSHA {
		return FoldableStarterState{}, errors.New("foldable v3 screen derivative differs from its pinned Library artifact")
	}
	candidate := cloneFoldableCandidate(base.Device.Candidate)
	if candidate.Definition.ID != foldDeviceID || len(candidate.Definition.ModelCandidates) != 1 {
		return FoldableStarterState{}, errors.New("foldable v2 candidate identity or provenance inventory differs")
	}
	candidate.Geometry = ref
	candidate.Definition.ID = foldUVDeviceID
	candidate.Definition.Variant = "Vrooli generic dual-rigid book-fold concept with selected left-screen V-only UV correction"
	candidate.Definition.Description = "Unqualified source-mapped generic foldable candidate. Only the selected left inner display UV mapping differs from v2; scene, pose, motion, geometry and other display mappings are unchanged."
	candidate.Definition.ModelCandidates[0].ID = foldUVAssetID
	candidate.Definition.ModelCandidates[0].SHA256 = foldUVAssetSHA
	candidate.Definition.ModelCandidates[0].Attribution = "Vrooli-authored generic book-fold v2 with private left inner display UV derivative."
	candidate.Definition.Provenance = append(candidate.Definition.Provenance, devices.Evidence{Source: "assets/foldable-v3/asset-manifest-v3.json SHA-256 f7d84855526ac5244b1d0bcc7a166a3bfb53f97db08874fade0febc5bc52d287", Kind: "display_uv_mapping", Qualifier: "only LeftInnerDisplay mesh 1 primitive 0 TEXCOORD_0 maps (u,v) to (u,1-v); no geometry or hinge claims change"})
	if _, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, "inner-left"); err != nil {
		return FoldableStarterState{}, fmt.Errorf("preflight foldable v3 active display binding: %w", err)
	}
	device, err := saveDeviceCandidateIfUnchanged(ctx, o.Starter.devices, candidate)
	if err != nil {
		return FoldableStarterState{}, err
	}
	preset, err := o.Starter.ensureFoldableUVV3Preset(ctx, device, base.Scene, base.Motion, base.Placement)
	if err != nil {
		return FoldableStarterState{}, err
	}
	base.Bundle, base.Device, base.Preset = bundle, device, preset
	return base, nil
}

func cloneFoldableCandidate(source devices.DeviceCandidate) devices.DeviceCandidate {
	out := source
	out.Definition.ModelCandidates = append([]devices.ModelCandidate(nil), source.Definition.ModelCandidates...)
	out.Definition.Provenance = append([]devices.Evidence(nil), source.Definition.Provenance...)
	out.Parts = append([]devices.SemanticPart(nil), source.Parts...)
	out.DisplayBindings = append([]devices.DisplayPartBinding(nil), source.DisplayBindings...)
	out.SupportFrames = append([]devices.DeviceSupportFrame(nil), source.SupportFrames...)
	return out
}

func (s *starterInitializer) ensureFoldableUVV3Preset(ctx context.Context, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedMonitorPlacement) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{Device: &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256}, Scene: &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256}, Motion: &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256}, DisplayId: "inner-left", DeviceSupportId: foldDeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: foldCameraID, SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest), StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1}}
	if current, err := s.presets.Head(ctx, foldUVPresetID); err == nil {
		if current.Name == "Generic book-fold concept v3 left-screen UV review still" && proto.Equal(current.Setup, setup) {
			return current, nil
		}
		return shots.PhysicalPresetRevision{}, shots.ErrPhysicalPresetConflict
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, foldUVPresetID, 0, "Generic book-fold concept v3 left-screen UV review still", setup)
}

const foldUVStaticPresetID = "generic-book-fold-still-v3-left-screen-vflip-static"

// InitializeReviewedFoldableUVV3Static binds a still capture to the shared
// static identity motion while preserving the existing v3 hinge-motion owners.
func (o Owners) InitializeReviewedFoldableUVV3Static(ctx context.Context) (FoldableStarterState, error) {
	state, err := o.InitializeReviewedFoldableUVV3(ctx)
	if err != nil {
		return FoldableStarterState{}, err
	}
	pinned, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return FoldableStarterState{}, err
	}
	still, err := o.Starter.saveStillMotion(pinned)
	if err != nil {
		return FoldableStarterState{}, fmt.Errorf("resolve shared static still motion: %w", err)
	}
	preset, err := o.Starter.ensureFoldableUVV3StaticPreset(ctx, state.Device, state.Scene, still, state.Placement)
	if err != nil {
		return FoldableStarterState{}, err
	}
	state.Motion, state.Preset = still, preset
	return state, nil
}

func (s *starterInitializer) ensureFoldableUVV3StaticPreset(ctx context.Context, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedMonitorPlacement) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{Device: &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256}, Scene: &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256}, Motion: &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256}, DisplayId: "inner-left", DeviceSupportId: foldDeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: foldCameraID, SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest), StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1}}
	if current, err := s.presets.Head(ctx, foldUVStaticPresetID); err == nil {
		if current.Name == "Generic book-fold concept v3 static still review" && proto.Equal(current.Setup, setup) {
			return current, nil
		}
		return shots.PhysicalPresetRevision{}, shots.ErrPhysicalPresetConflict
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, foldUVStaticPresetID, 0, "Generic book-fold concept v3 static still review", setup)
}
