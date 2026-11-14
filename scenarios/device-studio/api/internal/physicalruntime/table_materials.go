package physicalruntime

import (
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
)

const (
	starterTableBaseColorID = "travertine-color"
	starterTableNormalID    = "travertine-normal-gl"
	starterTableRoughnessID = "travertine-roughness"
	starterTableHeightID    = "travertine-height"
)

// starterTableMaterials is the pinned Travertine001 material recipe shared by
// the approved phone/iPad scenes and the separate framed laptop candidate.
func starterTableMaterials(inventory []rendering.PhysicalGLBMaterial, materialID func(uint32) string, tableIndex, windowIndex uint32) []scenes.SceneMaterial {
	materials := make([]scenes.SceneMaterial, 0, len(inventory))
	for _, item := range inventory {
		material := scenes.SceneMaterial{ID: materialID(item.Index), SurfaceProfile: scenes.MaterialPrincipledV1, BaseColorLinearRGBA: item.BaseColor, Metallic: item.Metallic, Roughness: item.Roughness, AlphaMode: scenes.AlphaMode(item.AlphaMode)}
		if item.Index == tableIndex {
			material.Grade = &scenes.SceneMaterialGrade{Recipe: scenes.GradeHueSaturationLinearMixV1, Saturation: .88, Value: .98, MixFactor: .55, MixLinearRGB: [3]float64{.65, .60, .50}}
			material.Inputs = []scenes.SceneMaterialInput{
				{Socket: scenes.SocketBaseColor, TextureBindingID: starterTableBaseColorID, Semantic: "base_color", ColorSpace: scenes.ColorSpaceSRGB, Channel: scenes.ChannelRGB, Sampler: scenes.SamplerRepeat, Coordinates: scenes.CoordinatesGeneratedBounds, PhysicalRepeatM: &[2]float64{1.2, 1.2}},
				{Socket: scenes.SocketRoughness, TextureBindingID: starterTableRoughnessID, Semantic: "roughness", ColorSpace: scenes.ColorSpaceNonColor, Channel: scenes.ChannelRed, Sampler: scenes.SamplerRepeat, Coordinates: scenes.CoordinatesGeneratedBounds, PhysicalRepeatM: &[2]float64{1.2, 1.2}},
				{Socket: scenes.SocketNormal, TextureBindingID: starterTableNormalID, Semantic: "normal_gl", ColorSpace: scenes.ColorSpaceNonColor, Channel: scenes.ChannelRGB, Sampler: scenes.SamplerRepeat, Coordinates: scenes.CoordinatesGeneratedBounds, PhysicalRepeatM: &[2]float64{1.2, 1.2}, NormalConvention: "OPENGL_TANGENT_PLUS_Y_V1", NormalStrength: floatPtr(.45)},
				{Socket: scenes.SocketBump, TextureBindingID: starterTableHeightID, Semantic: "bump_height", ColorSpace: scenes.ColorSpaceNonColor, Channel: scenes.ChannelRed, Sampler: scenes.SamplerRepeat, Coordinates: scenes.CoordinatesGeneratedBounds, PhysicalRepeatM: &[2]float64{1.2, 1.2}, BumpStrength: floatPtr(.08), BumpDistanceM: floatPtr(.00012)},
			}
		}
		if item.Index == windowIndex {
			material.SurfaceProfile = scenes.MaterialEmissionV1
			material.BaseColorLinearRGBA = [4]float64{.624, .656, .704, 1}
			material.Metallic, material.Roughness = 0, 1
		}
		materials = append(materials, material)
	}
	return materials
}
