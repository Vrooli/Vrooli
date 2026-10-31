import type { TerrainVisualTuning } from '../config'

export { terrainTintVariation } from '../sim/terrain/colour'

export function terrainMaterialSettings(wetness: number, settings: TerrainVisualTuning) {
  return {
    color: wetness > 0 ? settings.wetColor : settings.dryColor,
    roughness: Math.max(settings.minimumRoughness, 1 - wetness * settings.wetRoughnessScale),
  }
}
