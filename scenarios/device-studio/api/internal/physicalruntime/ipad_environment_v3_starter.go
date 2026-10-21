package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"

	"device-studio/internal/library"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
)

const (
	ipadEnvironmentV3BundleID   = "device-studio-starter-ipad-mini-6-ipad-environment-v2"
	ipadEnvironmentV3ArtifactID = "ipad-study-environment-v2-glb"
	ipadEnvironmentV3SceneID    = "ipad-study-environment-screen-uv-oriented-180-v2-cup128-fit-camera-v3"
	ipadEnvironmentV3CameraID   = "ipad-study-camera-16x9-fit-screen-uv-oriented-180-v3"
	ipadEnvironmentV3SHA256     = "598c50a9bea67b56f9911c2601e3bed8d11428e171725315b0f382fa92462c5f"
	ipadEnvironmentV3Size       = 289224
)

// IPadEnvironmentV3State keeps the scene bundle distinct from the oriented
// device bundle while returning the exact immutable device/scene revisions.
type IPadEnvironmentV3State struct {
	IPadStarterState
	SceneBundle library.BundleRevision
}

// InitializeReviewedIPadEnvironmentV3 saves a separate iPad-specific scene and
// camera. Existing phone and iPad v1/v2 candidates remain unchanged.
func (o Owners) InitializeReviewedIPadEnvironmentV3(ctx context.Context) (IPadEnvironmentV3State, error) {
	if o.Starter == nil {
		return IPadEnvironmentV3State{}, errors.New("reviewed starter initializer is unavailable")
	}
	ctx, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return IPadEnvironmentV3State{}, err
	}
	base, err := o.InitializeReviewedIPadOrientedStarter(ctx)
	if err != nil {
		return IPadEnvironmentV3State{}, err
	}
	bundle, err := o.Starter.publishIPadEnvironmentV3Bundle(ctx)
	if err != nil {
		return IPadEnvironmentV3State{}, err
	}
	data, err := fs.ReadFile(o.Starter.fs, "assets/ipad-environment-v3/ipad-study-environment.glb")
	if err != nil {
		return IPadEnvironmentV3State{}, err
	}
	if len(data) != ipadEnvironmentV3Size || sha256Hex(data) != ipadEnvironmentV3SHA256 {
		return IPadEnvironmentV3State{}, errors.New("embedded iPad environment v3 differs from its pinned GLB")
	}
	candidate, err := buildIPadEnvironmentV3Candidate(bundle, data, base.Scene.Candidate)
	if err != nil {
		return IPadEnvironmentV3State{}, err
	}
	if _, err := rendering.ValidateScenePhysicalGLB(data, candidate); err != nil {
		return IPadEnvironmentV3State{}, fmt.Errorf("validate iPad environment v3 against its exact GLB inventory: %w", err)
	}
	_, _, digest, err := scenes.CanonicalizeCandidate(candidate)
	if err != nil {
		return IPadEnvironmentV3State{}, err
	}
	scene, err := saveIPadSceneCandidate(ctx, o.Starter.scenes, candidate, digest)
	if err != nil {
		return IPadEnvironmentV3State{}, err
	}
	placement := base.Placement
	placement.SelectedCameraID = ipadEnvironmentV3CameraID
	return IPadEnvironmentV3State{
		IPadStarterState: IPadStarterState{
			Bundle: base.Bundle, Device: base.Device, Scene: scene, Motion: base.Motion, Placement: placement,
		},
		SceneBundle: bundle,
	}, nil
}

func (s *starterInitializer) publishIPadEnvironmentV3Bundle(ctx context.Context) (library.BundleRevision, error) {
	files := []starterFile{
		{ID: ipadEnvironmentV3ArtifactID, File: "assets/ipad-environment-v3/ipad-study-environment.glb", MIME: "model/gltf-binary", SHA: ipadEnvironmentV3SHA256, Role: library.RoleGeometryGLB, License: "AGPL-3.0-only", Attribution: "Vrooli-authored iPad-specific room, table, and cup scene; not measured furniture or CAD.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAuthored, SourceLabel: "Vrooli-authored iPad study scene; native export6 candidate-only output. The scene was authored separately from the iPad model and embeds no device or private app pixels."}},
		{ID: "travertine-color", File: "assets/phone-table/textures/travertine-color.jpg", MIME: "image/jpeg", SHA: "e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8", Role: library.RoleTextureImage, License: "CC0-1.0", Attribution: "ambientCG Travertine001 color texture, CC0-1.0.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAcquired, SourceLabel: "ambientCG Travertine001; acquisition record 4268b08091b930230a90de78c6e82f793c887fb870ec0adc28bf923317153f4a.", SourceSHA256: "e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8"}},
		{ID: "travertine-normal-gl", File: "assets/phone-table/textures/travertine-normal-gl.jpg", MIME: "image/jpeg", SHA: "68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b", Role: library.RoleTextureImage, License: "CC0-1.0", Attribution: "ambientCG Travertine001 OpenGL tangent +Y normal map, CC0-1.0.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAcquired, SourceLabel: "ambientCG Travertine001; acquisition record 4268b08091b930230a90de78c6e82f793c887fb870ec0adc28bf923317153f4a.", SourceSHA256: "68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b"}},
		{ID: "travertine-roughness", File: "assets/phone-table/textures/travertine-roughness.jpg", MIME: "image/jpeg", SHA: "a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90", Role: library.RoleTextureImage, License: "CC0-1.0", Attribution: "ambientCG Travertine001 roughness map, CC0-1.0.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAcquired, SourceLabel: "ambientCG Travertine001; acquisition record 4268b08091b930230a90de78c6e82f793c887fb870ec0adc28bf923317153f4a.", SourceSHA256: "a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90"}},
		{ID: "travertine-height", File: "assets/phone-table/textures/travertine-height.jpg", MIME: "image/jpeg", SHA: "64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf", Role: library.RoleTextureImage, License: "CC0-1.0", Attribution: "ambientCG Travertine001 height map used only for subtle bump, CC0-1.0.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAcquired, SourceLabel: "ambientCG Travertine001; acquisition record 4268b08091b930230a90de78c6e82f793c887fb870ec0adc28bf923317153f4a.", SourceSHA256: "64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf"}},
	}
	manifest, sources, err := s.buildManifest(ctx, ipadEnvironmentV3BundleID, files)
	if err != nil {
		return library.BundleRevision{}, err
	}
	return s.publish(ctx, manifest, sources, "device-studio-e3-starter-local-render-review-v1-ipad-environment-v3")
}

func buildIPadEnvironmentV3Candidate(bundle library.BundleRevision, data []byte, base scenes.SceneCandidate) (scenes.SceneCandidate, error) {
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return scenes.SceneCandidate{}, err
	}
	if len(inventory.Nodes) != 14 || len(inventory.Materials) != 7 {
		return scenes.SceneCandidate{}, fmt.Errorf("iPad scene v3 inventory changed: nodes=%d materials=%d", len(inventory.Nodes), len(inventory.Materials))
	}
	geometry := artifactRef(bundle, ipadEnvironmentV3ArtifactID)
	if geometry.SHA256 != ipadEnvironmentV3SHA256 || geometry.SizeBytes != ipadEnvironmentV3Size {
		return scenes.SceneCandidate{}, errors.New("iPad scene v3 Library geometry binding is incomplete")
	}
	textureIDs := []string{"travertine-color", "travertine-normal-gl", "travertine-roughness", "travertine-height"}
	bindings := []scenes.SceneArtifactBinding{{ID: "ipad-environment-v3-geometry", Ref: geometry, SourceUnit: "m", SourceToCanonical: ptrTRS(identityTRS())}}
	for _, id := range textureIDs {
		ref := artifactRef(bundle, id)
		if ref.ArtifactID == "" || ref.Role != library.RoleTextureImage {
			return scenes.SceneCandidate{}, fmt.Errorf("iPad scene v3 texture %q is not published in its scene bundle", id)
		}
		bindings = append(bindings, scenes.SceneArtifactBinding{ID: id, Ref: ref})
	}
	materialID := func(index uint32) string { return fmt.Sprintf("ipad-environment-v3-material-%d", index) }
	topMaterial, topOK := namedMaterialIndex(inventory.Materials, "Travertine001 top material slot")
	windowMaterial, windowOK := namedMaterialIndex(inventory.Materials, "Soft luminous window")
	if !topOK || !windowOK {
		return scenes.SceneCandidate{}, errors.New("iPad scene v3 named material bindings differ from its exported inventory")
	}
	materials := starterTableMaterials(inventory.Materials, materialID, topMaterial, windowMaterial)
	partID := func(index uint32) string { return fmt.Sprintf("ipad-environment-v3-node-%d", index) }
	parts := make([]scenes.ScenePart, 0, len(inventory.Nodes))
	tablePartID := ""
	var tableTopY float64
	for _, node := range inventory.Nodes {
		role := "scene_geometry"
		switch node.Name {
		case "RoomBackWall", "WindowFrameRailLower", "WindowFrameRailUpper", "WindowFrameUprightLeft", "WindowFrameUprightRight", "WindowLightPanel":
			role = "environment_backdrop"
		case "TableTop":
			role, tablePartID = "support_surface", partID(node.Index)
			if node.Mesh == nil {
				return scenes.SceneCandidate{}, errors.New("iPad scene v3 TableTop node has no mesh")
			}
			mesh := meshByIndex(inventory.Meshes, *node.Mesh)
			if mesh == nil || len(mesh.Primitives) == 0 {
				return scenes.SceneCandidate{}, errors.New("iPad scene v3 TableTop mesh is absent")
			}
			tableTopY = math.Inf(-1)
			for _, primitive := range mesh.Primitives {
				if primitive.PositionMax[1] > tableTopY {
					tableTopY = primitive.PositionMax[1]
				}
			}
		}
		parent := ""
		if node.Parent != nil {
			if int(*node.Parent) >= len(inventory.Nodes) {
				return scenes.SceneCandidate{}, errors.New("iPad scene v3 node has an invalid parent index")
			}
			parent = partID(*node.Parent)
		}
		parts = append(parts, scenes.ScenePart{ID: partID(node.Index), Role: role, ArtifactBindingID: "ipad-environment-v3-geometry", ParentID: parent, NodeIndex: node.Index, NodeName: node.Name, LocalFrame: node.Frame})
	}
	if tablePartID == "" || !positiveFinite(tableTopY) {
		return scenes.SceneCandidate{}, errors.New("iPad scene v3 has no measured tabletop support frame")
	}
	slots := make([]scenes.SceneMaterialSlot, 0)
	for _, node := range inventory.Nodes {
		if node.Mesh == nil {
			continue
		}
		mesh := meshByIndex(inventory.Meshes, *node.Mesh)
		if mesh == nil {
			return scenes.SceneCandidate{}, fmt.Errorf("iPad scene v3 node %q references an absent mesh", node.Name)
		}
		for _, primitive := range mesh.Primitives {
			slots = append(slots, scenes.SceneMaterialSlot{PartID: partID(node.Index), PrimitiveIndex: primitive.Index, SlotIndex: primitive.Material, MaterialID: materialID(primitive.Material)})
		}
	}
	identity := identityTRS()
	candidate := scenes.SceneCandidate{
		Version: scenes.CandidateVersion, SceneID: ipadEnvironmentV3SceneID, CoordinateConvention: physical.CoordinateConvention,
		Artifacts: bindings, Materials: materials, MaterialSlots: slots, Parts: parts,
		SupportFrames: []scenes.SceneSupportFrame{{ID: "table-top-flat", InterfaceID: "horizontal_surface", FramePartID: tablePartID, LocalFrame: physical.TRS{Translation: physical.Vec3{Y: tableTopY}, Rotation: identity.Rotation, Scale: identity.Scale}, ContactRegionPartID: tablePartID, LoadDirectionLocal: &physical.Vec3{Y: -1}}},
		CameraFrames:  []scenes.SceneCameraFrame{{ID: ipadEnvironmentV3CameraID, Frame: physical.TRS{Translation: physical.Vec3{X: .1002994491448368, Y: 1.409043851488113, Z: .5300879317248927}, Rotation: physical.QuatXYZW{X: -.3913243654372847, Y: .14705677223763933, Z: .06350320239003839, W: .9062046623364743}, Scale: identity.Scale}, Projection: scenes.ProjectionPerspective, FocalLengthMM: floatPtr(78), SensorWidthMM: floatPtr(36), SensorHeightMM: floatPtr(20.25), SensorFit: scenes.SensorFitHorizontal, NearClipM: .1, FarClipM: 1000, Focus: &scenes.FocusTarget{Mode: scenes.FocusSelectedDevice}, DOFFNumber: floatPtr(11)}},
		LightFrames:   append([]scenes.SceneLightFrame(nil), base.LightFrames...), Environment: base.Environment,
	}
	if _, _, _, err := scenes.CanonicalizeCandidate(candidate); err != nil {
		return scenes.SceneCandidate{}, fmt.Errorf("canonicalize iPad environment v3 candidate: %w", err)
	}
	return candidate, nil
}

func meshByIndex(meshes []rendering.PhysicalGLBMesh, index uint32) *rendering.PhysicalGLBMesh {
	for i := range meshes {
		if meshes[i].Index == index {
			return &meshes[i]
		}
	}
	return nil
}
