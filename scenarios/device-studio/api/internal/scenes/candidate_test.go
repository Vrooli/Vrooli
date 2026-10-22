package scenes

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"device-studio/internal/library"
	"device-studio/internal/physical"
)

func approvedPhoneStudyMetadata() SceneCandidate {
	geometry := binding("scene-glb", "geometry_glb", "a")
	geometry.SourceUnit = "m"
	geometry.SourceToCanonical = &physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	texture := func(id, sum string) SceneArtifactBinding { return binding(id, "texture_image", sum) }
	return SceneCandidate{
		Version: CandidateVersion, SceneID: "phone-table-study", CoordinateConvention: physical.CoordinateConvention,
		Artifacts: []SceneArtifactBinding{geometry, texture("travertine-color", "b"), texture("travertine-roughness", "c"), texture("travertine-normal", "d"), texture("travertine-height", "e"), texture("screen-raster", "f")},
		Materials: []SceneMaterial{
			{ID: "travertine", SurfaceProfile: MaterialPrincipledV1, BaseColorLinearRGBA: [4]float64{0.8, 0.75, 0.65, 1}, Roughness: 0.5, AlphaMode: AlphaOpaque,
				Inputs: []SceneMaterialInput{
					{Socket: SocketBaseColor, TextureBindingID: "travertine-color", Semantic: "base_color", ColorSpace: ColorSpaceSRGB, Channel: ChannelRGB, Sampler: SamplerRepeat, Coordinates: CoordinatesGeneratedBounds, PhysicalRepeatM: ptr2(1.2, 1.2)},
					{Socket: SocketRoughness, TextureBindingID: "travertine-roughness", Semantic: "roughness", ColorSpace: ColorSpaceNonColor, Channel: ChannelRed, Sampler: SamplerRepeat, Coordinates: CoordinatesGeneratedBounds, PhysicalRepeatM: ptr2(1.2, 1.2)},
					{Socket: SocketNormal, TextureBindingID: "travertine-normal", Semantic: "normal_gl", ColorSpace: ColorSpaceNonColor, Channel: ChannelRGB, Sampler: SamplerRepeat, Coordinates: CoordinatesGeneratedBounds, PhysicalRepeatM: ptr2(1.2, 1.2), NormalConvention: "OPENGL_TANGENT_PLUS_Y_V1", NormalStrength: ptr(0.45)},
					{Socket: SocketBump, TextureBindingID: "travertine-height", Semantic: "bump_height", ColorSpace: ColorSpaceNonColor, Channel: ChannelRed, Sampler: SamplerRepeat, Coordinates: CoordinatesGeneratedBounds, PhysicalRepeatM: ptr2(1.2, 1.2), BumpStrength: ptr(0.08), BumpDistanceM: ptr(0.00012)},
				},
				Grade: &SceneMaterialGrade{Recipe: GradeHueSaturationLinearMixV1, Saturation: 0.88, Value: 0.98, MixFactor: 0.55, MixLinearRGB: [3]float64{0.65, 0.60, 0.50}},
			},
			{ID: "screen-emission", SurfaceProfile: MaterialEmissionV1, BaseColorLinearRGBA: [4]float64{1, 1, 1, 1}, Roughness: 1, AlphaMode: AlphaOpaque,
				Inputs: []SceneMaterialInput{{Socket: SocketBaseColor, TextureBindingID: "screen-raster", Semantic: "base_color", ColorSpace: ColorSpaceSRGB, Channel: ChannelRGB, Sampler: SamplerClamp, Coordinates: CoordinatesUVSet, UVSet: "screen_uv", AffineUV: ptr6(1, 0, 0, 1, 0, 0)}},
			},
		},
		MaterialSlots: []SceneMaterialSlot{{PartID: "table-top", PrimitiveIndex: 0, SlotIndex: 0, MaterialID: "travertine"}, {PartID: "phone-screen", PrimitiveIndex: 0, SlotIndex: 1, MaterialID: "screen-emission"}},
		Parts: []ScenePart{
			{ID: "table", Role: "environment", ArtifactBindingID: "scene-glb", NodeIndex: 0, NodeName: "Table", LocalFrame: identityTRS()},
			{ID: "table-top", Role: "surface", ArtifactBindingID: "scene-glb", ParentID: "table", NodeIndex: 1, NodeName: "TableTop", LocalFrame: identityTRS()},
			{ID: "phone-screen", Role: "screen", ArtifactBindingID: "scene-glb", ParentID: "table", NodeIndex: 2, NodeName: "PhoneScreenProxy", LocalFrame: identityTRS()},
		},
		SupportFrames: []SceneSupportFrame{{ID: "tabletop", InterfaceID: "horizontal_surface", FramePartID: "table-top", LocalFrame: identityTRS(), LoadDirectionLocal: &physical.Vec3{Y: -1}}},
		CameraFrames:  []SceneCameraFrame{{ID: "hero-camera", Frame: identityTRS(), Projection: ProjectionPerspective, FocalLengthMM: ptr(85), SensorWidthMM: ptr(36), SensorHeightMM: ptr(24), SensorFit: SensorFitHorizontal, NearClipM: 0.01, FarClipM: 100, Focus: &FocusTarget{Mode: FocusSelectedDevice}, DOFFNumber: ptr(11)}},
		LightFrames: []SceneLightFrame{
			{ID: "warm-key", Frame: identityTRS(), LocalAxisProfile: "AREA_XY_MINUS_Z_V1", Shape: LightRectangle, DimensionsM: [2]float64{0.95, 1.25}, PowerWatts: 12, LinearRGB: [3]float64{1, 0.91, 0.8}},
			{ID: "neutral-fill", Frame: identityTRS(), LocalAxisProfile: "AREA_XY_MINUS_Z_V1", Shape: LightRectangle, DimensionsM: [2]float64{0.75, 0.55}, PowerWatts: 2.8, LinearRGB: [3]float64{0.91, 0.94, 1}},
			{ID: "cool-edge", Frame: identityTRS(), LocalAxisProfile: "AREA_XY_MINUS_Z_V1", Shape: LightRectangle, DimensionsM: [2]float64{0.24, 0.85}, PowerWatts: 1.6, LinearRGB: [3]float64{0.8, 0.88, 1}},
		},
		Environment: SceneEnvironment{Model: "CONSTANT_V1", LinearRGB: [3]float64{0.53, 0.56, 0.6}, Strength: 0.2},
		Regions:     []SceneRegion{{ID: "table-contact", PartID: "table-top", Purpose: RegionContact}, {ID: "phone-clearance", PartID: "table", Purpose: RegionClearance}},
	}
}

func TestCanonicalizeApprovedPhoneStudyMetadata(t *testing.T) {
	input := approvedPhoneStudyMetadata()
	original := cloneCandidate(input)
	got, canonical, digest, err := CanonicalizeCandidate(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatal("canonicalization mutated caller-owned input")
	}
	_, again, digestAgain, err := CanonicalizeCandidate(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonical, again) || digest != digestAgain {
		t.Fatal("canonical bytes or digest changed on replay")
	}
	if got.CameraFrames[0].Focus.Mode != FocusSelectedDevice || materialByID(got.Materials, "travertine").Grade.Recipe != GradeHueSaturationLinearMixV1 {
		t.Fatal("approved-scene metadata lost typed focus or grade")
	}
}

func TestSupportSlotRequiresDeclaredFitAndCanonicalizesGroups(t *testing.T) {
	input := approvedPhoneStudyMetadata()
	input.Version = CandidateVersionSlots
	input.SupportSlots = []SceneSupportSlot{{ID: "phone-table-fit", FrameID: "tabletop", Kind: "tabletop", RequiredInterface: "horizontal_surface", AllowedGroups: []string{"tablet", "phone"}, DimensionState: "source_geometry_envelope", RequiredPose: "screen_up", MaxWidthM: 0.09, MaxHeightM: 0.18, MaxDepthM: 0.02}}
	got, encoded, digest, err := CanonicalizeCandidate(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.SupportSlots[0].AllowedGroups[0] != "phone" {
		t.Fatalf("support groups not sorted canonically: %+v", got.SupportSlots[0].AllowedGroups)
	}
	if !bytes.Contains(encoded, []byte(`"support_slots"`)) || digest == "" {
		t.Fatal("support slot did not participate in canonical revision identity")
	}
	if input.SupportSlots[0].AllowedGroups[0] != "tablet" {
		t.Fatal("canonicalization mutated caller-owned support slot")
	}

	invalid := input
	invalid.SupportSlots = append([]SceneSupportSlot(nil), input.SupportSlots...)
	invalid.SupportSlots[0].MaxDepthM = 0
	if _, _, _, err := CanonicalizeCandidate(invalid); err == nil || !strings.Contains(err.Error(), "maximum envelope") {
		t.Fatalf("incomplete slot fit must fail closed, got %v", err)
	}
}

func TestCanonicalizeSortsCollectionsAndInputsWithoutMutating(t *testing.T) {
	first := approvedPhoneStudyMetadata()
	second := cloneCandidate(first)
	second.Artifacts[0], second.Artifacts[1] = second.Artifacts[1], second.Artifacts[0]
	second.Parts[0], second.Parts[2] = second.Parts[2], second.Parts[0]
	second.MaterialSlots[0], second.MaterialSlots[1] = second.MaterialSlots[1], second.MaterialSlots[0]
	second.Materials[0].Inputs[0], second.Materials[0].Inputs[3] = second.Materials[0].Inputs[3], second.Materials[0].Inputs[0]
	original := cloneCandidate(second)
	_, a, da, err := CanonicalizeCandidate(first)
	if err != nil {
		t.Fatal(err)
	}
	_, b, db, err := CanonicalizeCandidate(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) || da != db {
		t.Fatal("equivalent collection order did not canonicalize identically")
	}
	if !reflect.DeepEqual(second, original) {
		t.Fatal("sorting mutated caller-owned slices")
	}
}

func TestCanonicalizeRejectsInvalidLocalSceneContracts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*SceneCandidate)
	}{
		{"part cycle", func(c *SceneCandidate) { c.Parts[0].ParentID = "table-top" }},
		{"duplicate node binding", func(c *SceneCandidate) { c.Parts[2].NodeIndex = c.Parts[1].NodeIndex }},
		{"unknown material slot", func(c *SceneCandidate) { c.MaterialSlots[0].MaterialID = "missing" }},
		{"slot collision", func(c *SceneCandidate) {
			c.MaterialSlots[1].PartID = c.MaterialSlots[0].PartID
			c.MaterialSlots[1].PrimitiveIndex = c.MaterialSlots[0].PrimitiveIndex
			c.MaterialSlots[1].SlotIndex = c.MaterialSlots[0].SlotIndex
		}},
		{"texture role mismatch", func(c *SceneCandidate) { c.Materials[0].Inputs[0].TextureBindingID = "scene-glb" }},
		{"emission principled response", func(c *SceneCandidate) { c.Materials[1].Roughness = 0.5 }},
		{"wrong channel space", func(c *SceneCandidate) { c.Materials[0].Inputs[0].ColorSpace = ColorSpaceNonColor }},
		{"overflowing affine determinant", func(c *SceneCandidate) { c.Materials[1].Inputs[0].AffineUV = ptr6(1e308, 1e308, 1e308, 1e308, 0, 0) }},
		{"generated bounds clamp", func(c *SceneCandidate) { c.Materials[0].Inputs[0].Sampler = SamplerClamp }},
		{"socket union extra field", func(c *SceneCandidate) { c.Materials[0].Inputs[0].NormalStrength = ptr(0.5) }},
		{"rigid correction scale", func(c *SceneCandidate) { c.Artifacts[0].SourceToCanonical.Scale.X = 2 }},
		{"perspective orthographic mixture", func(c *SceneCandidate) { c.CameraFrames[0].OrthographicSpanM = ptr(2) }},
		{"focus union mixture", func(c *SceneCandidate) { c.CameraFrames[0].Focus.DistanceM = ptr(1) }},
		{"light axis profile", func(c *SceneCandidate) { c.LightFrames[0].LocalAxisProfile = "host-default" }},
		{"support frame scale", func(c *SceneCandidate) { c.SupportFrames[0].LocalFrame.Scale.X = 2 }},
		{"region references missing part", func(c *SceneCandidate) { c.Regions[0].PartID = "missing" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := approvedPhoneStudyMetadata()
			test.mutate(&candidate)
			if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
				t.Fatal("invalid candidate was accepted")
			}
		})
	}
}

func TestCanonicalizeRejectsInvalidArtifactIdentityAndBounds(t *testing.T) {
	t.Run("bad pinned digest", func(t *testing.T) {
		candidate := approvedPhoneStudyMetadata()
		candidate.Artifacts[0].Ref.ReviewSHA256 = "bad"
		if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
			t.Fatal("invalid digest was accepted")
		}
	})
	t.Run("unsupported source unit", func(t *testing.T) {
		candidate := approvedPhoneStudyMetadata()
		candidate.Artifacts[0].SourceUnit = "ft"
		if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
			t.Fatal("unsupported units were accepted")
		}
	})
	t.Run("role-specific size cap", func(t *testing.T) {
		candidate := approvedPhoneStudyMetadata()
		candidate.Artifacts[0].Ref.SizeBytes = library.MaxGeometryBytes + 1
		if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
			t.Fatal("oversized geometry reference was accepted")
		}
		candidate = approvedPhoneStudyMetadata()
		candidate.Artifacts[1].Ref.SizeBytes = library.MaxTextureBytes + 1
		if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
			t.Fatal("oversized texture reference was accepted")
		}
	})
	t.Run("too many parts", func(t *testing.T) {
		candidate := approvedPhoneStudyMetadata()
		for i := 0; i <= MaxParts; i++ {
			candidate.Parts = append(candidate.Parts, ScenePart{ID: stringID(i), Role: "part", ArtifactBindingID: "scene-glb", NodeIndex: uint32(10 + i), NodeName: "Generated", LocalFrame: identityTRS()})
		}
		if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
			t.Fatal("part bound was not enforced")
		}
	})
	t.Run("too many material inputs", func(t *testing.T) {
		candidate := approvedPhoneStudyMetadata()
		candidate.Materials[0].Inputs = append(candidate.Materials[0].Inputs, candidate.Materials[0].Inputs[0])
		if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
			t.Fatal("material input bound was not enforced")
		}
	})
	t.Run("alpha mask requires explicit cutoff", func(t *testing.T) {
		candidate := approvedPhoneStudyMetadata()
		candidate.Materials[0].AlphaMode = AlphaMask
		if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
			t.Fatal("alpha mask without a cutoff was accepted")
		}
		candidate.Materials[0].AlphaCutoff = ptr(0.5)
		if _, _, _, err := CanonicalizeCandidate(candidate); err != nil {
			t.Fatalf("explicit alpha cutoff rejected: %v", err)
		}
		candidate.Materials[0].AlphaMode = AlphaOpaque
		if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
			t.Fatal("irrelevant alpha cutoff was accepted")
		}
		candidate.Materials[0].AlphaMode = AlphaMask
		candidate.Materials[0].AlphaCutoff = ptr(1.1)
		if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
			t.Fatal("out-of-range alpha cutoff was accepted")
		}
	})
}

func binding(id, role, digestPrefix string) SceneArtifactBinding {
	return SceneArtifactBinding{ID: id, Ref: physical.ArtifactRef{BundleID: "phone-study", BundleRevision: 1, ManifestSHA256: stringsRepeat(digestPrefix, 64), ReviewSHA256: stringsRepeat("1", 64), ArtifactID: id, Role: library.ArtifactRole(role), SizeBytes: 1024, SHA256: stringsRepeat(digestPrefix, 64)}}
}
func identityTRS() physical.TRS {
	return physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}
func ptr(value float64) *float64                { return &value }
func ptr2(a, b float64) *[2]float64             { return &[2]float64{a, b} }
func ptr6(a, b, c, d, e, f float64) *[6]float64 { return &[6]float64{a, b, c, d, e, f} }
func stringsRepeat(value string, count int) string {
	out := make([]byte, count)
	for i := range out {
		out[i] = value[0]
	}
	return string(out)
}
func stringID(value int) string { return "part-" + strconv.Itoa(value) }
func materialByID(materials []SceneMaterial, id string) SceneMaterial {
	for _, material := range materials {
		if material.ID == id {
			return material
		}
	}
	return SceneMaterial{}
}
