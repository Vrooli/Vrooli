package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
)

const (
	ipadOrientedBundleID    = "device-studio-starter-ipad-mini-6-datsketch-neutral-uv-oriented-180-v2"
	ipadOrientedArtifactID  = "datsketch-ipad-mini-6-neutral-uv-oriented-180-v2-glb"
	ipadOrientedCandidateID = "datsketch-ipad-mini-6-neutral-uv-oriented-180-v2"
	ipadOrientedSceneID     = "ipad-study-environment-screen-uv-oriented-180-v2"
	ipadOrientedCameraID    = "ipad-study-camera-16x9-fit-screen-uv-oriented-180-v2"
	ipadOrientedGLBSHA256   = "0326fd47e7df4a065978eaae76e6789cb3027f54bab6f3f7fdef1bdfb867e52e"
)

// InitializeReviewedIPadOrientedStarter publishes a distinct UV-oriented
// candidate revision. The v1 candidate, asset, scene, and camera remain intact.
func (o Owners) InitializeReviewedIPadOrientedStarter(ctx context.Context) (IPadStarterState, error) {
	if o.Starter == nil {
		return IPadStarterState{}, errors.New("reviewed starter initializer is unavailable")
	}
	ctx, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return IPadStarterState{}, err
	}
	v1, err := o.InitializeReviewedIPadStarter(ctx)
	if err != nil {
		return IPadStarterState{}, err
	}
	bundle, err := o.Starter.publishIPadOrientedBundle(ctx)
	if err != nil {
		return IPadStarterState{}, err
	}
	ref := artifactRef(bundle, ipadOrientedArtifactID)
	data, err := o.Starter.readPinnedIPadOriented(ref)
	if err != nil {
		return IPadStarterState{}, err
	}
	candidate := cloneIPadCandidate(v1.Device.Candidate)
	candidate.Geometry = ref
	candidate.Definition.ID = ipadOrientedCandidateID
	candidate.Definition.Variant = "neutral-screen source-preserving adaptation with normalized display UVs rotated 180 degrees"
	candidate.Definition.Description = "DatSketch's iPad mini 6 model with a neutral display and a 180-degree screen UV correction observed in native hero-4. Geometry and device pose are unchanged; candidate remains unqualified."
	if len(candidate.Definition.ModelCandidates) != 1 {
		return IPadStarterState{}, errors.New("iPad v1 model provenance inventory differs")
	}
	candidate.Definition.ModelCandidates[0].ID = ipadOrientedCandidateID
	candidate.Definition.ModelCandidates[0].SHA256 = ref.SHA256
	candidate.Definition.ModelCandidates[0].Attribution = "iPad mini 6 2021 by DatSketch, adapted under CC BY 4.0; embedded screen image replaced, display UVs normalized and oriented 180 degrees."
	candidate.Definition.Provenance = append(candidate.Definition.Provenance, devices.Evidence{Source: "ipad-screen-uv-orientation-manifest.json and retained hero-4 observation", Kind: "screen_uv_orientation", Qualifier: "only accessor 36 is remapped (u,v) to (1-u,1-v); geometry and candidate remains unqualified"})
	if _, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, "main_display"); err != nil {
		return IPadStarterState{}, fmt.Errorf("preflight 180-degree iPad display binding: %w", err)
	}
	device, err := saveIPadCandidate(ctx, o.Starter.devices, candidate)
	if err != nil {
		return IPadStarterState{}, err
	}
	sceneCandidate := v1.Scene.Candidate
	sceneCandidate.SceneID = ipadOrientedSceneID
	sceneCandidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), sceneCandidate.CameraFrames...)
	if len(sceneCandidate.CameraFrames) != 1 || sceneCandidate.CameraFrames[0].ID != ipadCameraID {
		return IPadStarterState{}, errors.New("iPad v1 scene camera inventory differs")
	}
	sceneCandidate.CameraFrames[0].ID = ipadOrientedCameraID
	_, _, sceneSHA, err := scenes.CanonicalizeCandidate(sceneCandidate)
	if err != nil {
		return IPadStarterState{}, err
	}
	scene, err := saveIPadSceneCandidate(ctx, o.Starter.scenes, sceneCandidate, sceneSHA)
	if err != nil {
		return IPadStarterState{}, err
	}
	placement := v1.Placement
	placement.SelectedCameraID = ipadOrientedCameraID
	return IPadStarterState{Bundle: bundle, Device: device, Scene: scene, Motion: v1.Motion, Placement: placement}, nil
}

func (s *starterInitializer) publishIPadOrientedBundle(ctx context.Context) (library.BundleRevision, error) {
	file := starterFile{ID: ipadOrientedArtifactID, File: "assets/ipad/ipad-mini-screen-uv-oriented-180-v2.glb", MIME: "model/gltf-binary", SHA: ipadOrientedGLBSHA256, Role: library.RoleGeometryGLB, License: "CC-BY-4.0", Attribution: "iPad mini 6 2021 model by DatSketch, adapted under CC BY 4.0; neutral screen, normalized UVs, and screen UV orientation corrected by 180 degrees.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "DatSketch iPad Mini 6 2021; normalized neutral-screen derivative v1.", SourceSHA256: ipadDerivativeSHA256, Adaptation: "Append a private TEXCOORD_0 view and change only accessor 36 to map (u,v) to (1-u,1-v). Preserve all existing BIN bytes, geometry, display raster, hierarchy, materials, and transforms."}}
	manifest, sources, err := s.buildManifest(ctx, ipadOrientedBundleID, []starterFile{file})
	if err != nil {
		return library.BundleRevision{}, err
	}
	return s.publish(ctx, manifest, sources, "device-studio-e3-starter-local-render-review-v1-ipad-uv-oriented-180")
}

func (s *starterInitializer) readPinnedIPadOriented(ref physical.ArtifactRef) ([]byte, error) {
	data, err := fs.ReadFile(s.fs, "assets/ipad/ipad-mini-screen-uv-oriented-180-v2.glb")
	if err != nil {
		return nil, err
	}
	if len(data) != 1939592 || sha256Hex(data) != ipadOrientedGLBSHA256 || ref.SHA256 != ipadOrientedGLBSHA256 || ref.SizeBytes != int64(len(data)) || ref.Role != library.RoleGeometryGLB {
		return nil, errors.New("embedded iPad oriented derivative differs from its pinned Library artifact")
	}
	return data, nil
}

func saveIPadCandidate(ctx context.Context, owner *devices.DeviceRevisionService, candidate devices.DeviceCandidate) (devices.DeviceRevision, error) {
	existing, err := owner.GetHead(ctx, candidate.Definition.ID)
	if err == nil {
		_, _, digest, canonErr := devices.CanonicalizeDeviceCandidate(candidate)
		if canonErr != nil {
			return devices.DeviceRevision{}, canonErr
		}
		if existing.SHA256 == digest {
			return existing, nil
		}
		return devices.DeviceRevision{}, fmt.Errorf("iPad oriented candidate head differs; refusing implicit revision: %w", devices.ErrRevisionConflict)
	}
	if !errors.Is(err, devices.ErrRevisionNotFound) {
		return devices.DeviceRevision{}, err
	}
	return owner.SaveCandidate(ctx, candidate, 0)
}

func saveIPadSceneCandidate(ctx context.Context, owner *scenes.RevisionService, candidate scenes.SceneCandidate, digest string) (scenes.Revision, error) {
	existing, err := owner.GetHead(ctx, candidate.SceneID)
	if err == nil {
		if existing.SHA256 == digest {
			return existing, nil
		}
		return scenes.Revision{}, fmt.Errorf("iPad oriented scene head differs; refusing implicit revision: %w", scenes.ErrRevisionConflict)
	}
	if !errors.Is(err, scenes.ErrRevisionNotFound) {
		return scenes.Revision{}, err
	}
	return owner.SaveCandidate(ctx, candidate, 0)
}

func cloneIPadCandidate(source devices.DeviceCandidate) devices.DeviceCandidate {
	out := source
	out.Definition.ModelCandidates = append([]devices.ModelCandidate(nil), source.Definition.ModelCandidates...)
	out.Definition.Provenance = append([]devices.Evidence(nil), source.Definition.Provenance...)
	out.Parts = append([]devices.SemanticPart(nil), source.Parts...)
	out.DisplayBindings = append([]devices.DisplayPartBinding(nil), source.DisplayBindings...)
	out.SupportFrames = append([]devices.DeviceSupportFrame(nil), source.SupportFrames...)
	return out
}
