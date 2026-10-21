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
	laptopOrientedBundleID    = "device-studio-laptop-screen-upright-180-candidate-v1"
	laptopOrientedAssetID     = "jack-baeten-laptop-neutral-screen-upright-180-glb"
	laptopOrientedDeviceID    = "jack-baeten-macbook-pro-m3-16-screen-upright-180-candidate-v1"
	laptopOrientedSceneID     = "laptop-study-room-screen-upright-180-candidate-v1"
	laptopOrientedCameraID    = "laptop-candidate-camera-screen-upright-180-v1"
	laptopOrientedPresetID    = "laptop-screen-upright-180-review-still-v2"
	laptopOrientedReviewID    = "device-studio-e3-laptop-display-orientation-review-v1"
	laptopOrientedAssetFile   = "assets/laptop/laptop-neutral-rig-screen-upright-180-v1.glb"
	laptopOrientedAssetSHA256 = "20c9224deafefa48af2d2be5e48ca4fce484d167522c4762d2f1669cec78883b"
)

type laptopScreenViewSpec struct {
	bundleID, assetID, deviceID, sceneID, cameraID, presetID, reviewID string
	assetFile, assetSHA, orientation, correction                       string
}

func laptopUpright180View() laptopScreenViewSpec {
	return laptopScreenViewSpec{bundleID: laptopOrientedBundleID, assetID: laptopOrientedAssetID, deviceID: laptopOrientedDeviceID,
		sceneID: laptopOrientedSceneID, cameraID: laptopOrientedCameraID, presetID: laptopOrientedPresetID, reviewID: laptopOrientedReviewID,
		assetFile: laptopOrientedAssetFile, assetSHA: laptopOrientedAssetSHA256, orientation: "180-degree UV view", correction: "(u,v)->(1-u,1-v)"}
}

func laptopScreenVFlipView() laptopScreenViewSpec {
	return laptopScreenViewSpec{bundleID: "device-studio-laptop-screen-vflip-candidate-v1", assetID: "jack-baeten-laptop-neutral-screen-vflip-glb",
		deviceID: "jack-baeten-macbook-pro-m3-16-screen-vflip-candidate-v1", sceneID: "laptop-study-room-screen-vflip-candidate-v1",
		cameraID: "laptop-candidate-camera-screen-vflip-v1", presetID: "laptop-screen-vflip-review-still-v1",
		reviewID: "device-studio-e3-laptop-display-vflip-review-v1", assetFile: "assets/laptop/laptop-neutral-rig-screen-vflip-v1.glb",
		assetSHA: "1d833e3d08d0722952ecd9b6616cf933af11fca5716cfe4bc67aa110850272bd", orientation: "vertical-only screen UV flip",
		correction: "(u,v)->(u,1-v)"}
}

// InitializeReviewedLaptopScreenUpright preserves the existing laptop
// candidate and publishes a separate display-only UV orientation revision.
func (o Owners) InitializeReviewedLaptopScreenUpright(ctx context.Context) (LaptopStarterState, error) {
	return o.initializeReviewedLaptopScreenView(ctx, laptopUpright180View())
}

// InitializeReviewedLaptopScreenVFlip adds a distinct vertical-only screen
// view candidate while preserving the source laptop's horizontal orientation.
func (o Owners) InitializeReviewedLaptopScreenVFlip(ctx context.Context) (LaptopStarterState, error) {
	return o.initializeReviewedLaptopScreenView(ctx, laptopScreenVFlipView())
}

func (o Owners) initializeReviewedLaptopScreenView(ctx context.Context, spec laptopScreenViewSpec) (LaptopStarterState, error) {
	if o.Starter == nil {
		return LaptopStarterState{}, errors.New("oriented laptop candidate initializer is unavailable")
	}
	base, err := o.InitializeReviewedLaptopStarter(ctx)
	if err != nil {
		return LaptopStarterState{}, err
	}
	file := starterFile{ID: spec.assetID, File: spec.assetFile, MIME: "model/gltf-binary", SHA: spec.assetSHA, Role: library.RoleGeometryGLB, License: "CC BY 4.0", Attribution: "MacBook Pro M3 16 inch 2024 creator model by Jack Baeten, adapted under CC BY 4.0; display-only UV orientation candidate. Not manufacturer CAD or endorsement.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "Jack Baeten derive16 neutral-screen laptop candidate.", SourceSHA256: laptopGeometrySHA256, Adaptation: "Append a private TEXCOORD_0 accessor for LaptopScreenSurface primitive 0, mapping only its UV values " + spec.correction + ". The original GLB BIN chunk remains an exact prefix; positions, normals, indices, all other accessors, hierarchy and transforms are unchanged."}}
	manifest, sources, err := o.Starter.buildManifest(ctx, spec.bundleID, []starterFile{file})
	if err != nil {
		return LaptopStarterState{}, err
	}
	bundle, err := o.Starter.publish(ctx, manifest, sources, spec.reviewID)
	if err != nil {
		return LaptopStarterState{}, err
	}
	ref := artifactRef(bundle, spec.assetID)
	data, err := fs.ReadFile(o.Starter.fs, spec.assetFile)
	if err != nil {
		return LaptopStarterState{}, err
	}
	if len(data) != 4638984 || ref.SHA256 != spec.assetSHA || sha256Hex(data) != spec.assetSHA {
		return LaptopStarterState{}, errors.New("oriented laptop GLB differs from its pinned Library artifact")
	}
	candidate := base.Device.Candidate
	candidate.Definition.ModelCandidates = append([]devices.ModelCandidate(nil), candidate.Definition.ModelCandidates...)
	candidate.Definition.Provenance = append([]devices.Evidence(nil), candidate.Definition.Provenance...)
	if candidate.Definition.ID != laptopDeviceID || len(candidate.Definition.ModelCandidates) != 1 {
		return LaptopStarterState{}, errors.New("derive16 laptop candidate identity or provenance inventory differs")
	}
	candidate.Definition.ID = spec.deviceID
	candidate.Definition.Variant = "derive16 creator model with a display-only " + spec.orientation
	candidate.Definition.Description = "A distinct unqualified candidate derived from the pinned laptop model. Only the display UV view changes; source geometry, scale, hierarchy, pose and motion remain unchanged."
	candidate.Definition.ModelCandidates[0].ID = spec.deviceID
	candidate.Definition.ModelCandidates[0].SHA256 = ref.SHA256
	candidate.Definition.ModelCandidates[0].Attribution = "MacBook Pro M3 16 inch 2024 creator model by Jack Baeten, adapted under CC BY 4.0; display UV orientation " + spec.orientation + "."
	candidate.Definition.Provenance = append(candidate.Definition.Provenance, devices.Evidence{Source: spec.assetFile + " SHA-256 " + spec.assetSHA, Kind: "display_uv_orientation", Qualifier: "only LaptopScreenSurface primitive 0 TEXCOORD_0 is remapped " + spec.correction + "; source geometry and pose remain unchanged"})
	candidate.Geometry = ref
	if _, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, laptopDisplayID); err != nil {
		return LaptopStarterState{}, fmt.Errorf("preflight oriented laptop display binding: %w", err)
	}
	device, err := saveDeviceCandidateIfUnchanged(ctx, o.Starter.devices, candidate)
	if err != nil {
		return LaptopStarterState{}, err
	}
	sceneCandidate := base.FramedScene.Candidate
	sceneCandidate.SceneID = spec.sceneID
	sceneCandidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), sceneCandidate.CameraFrames...)
	if len(sceneCandidate.CameraFrames) != 1 || sceneCandidate.CameraFrames[0].ID != laptopFramedCameraID {
		return LaptopStarterState{}, errors.New("derive16 framed scene camera inventory differs")
	}
	sceneCandidate.CameraFrames[0].ID = spec.cameraID
	scene, err := saveSceneCandidateIfUnchanged(ctx, o.Starter.scenes, sceneCandidate)
	if err != nil {
		return LaptopStarterState{}, err
	}
	placement := base.FramedPlacement
	placement.CameraID = spec.cameraID
	pinned, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return LaptopStarterState{}, err
	}
	still, err := o.Starter.saveStillMotion(pinned)
	if err != nil {
		return LaptopStarterState{}, fmt.Errorf("resolve shared identity still motion: %w", err)
	}
	preset, err := o.Starter.ensureLaptopOrientedPreset(pinned, spec, device, scene, still, placement)
	if err != nil {
		return LaptopStarterState{}, err
	}
	base.Bundle, base.Device, base.FramedScene, base.FramedPlacement = bundle, device, scene, placement
	base.Preset = preset
	return base, nil
}

func (s *starterInitializer) ensureLaptopOrientedPreset(ctx context.Context, spec laptopScreenViewSpec, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement LaptopCandidatePlacement) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{
		Device:    &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256},
		Scene:     &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256},
		Motion:    &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256},
		DisplayId: laptopDisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.CameraID,
		SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest),
		StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1, shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1},
	}
	if existing, err := s.presets.Head(ctx, spec.presetID); err == nil {
		if proto.Equal(existing.Setup, setup) {
			return existing, nil
		}
		return shots.PhysicalPresetRevision{}, shots.ErrPhysicalPresetConflict
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, spec.presetID, 0, "Laptop screen orientation review still", setup)
}
