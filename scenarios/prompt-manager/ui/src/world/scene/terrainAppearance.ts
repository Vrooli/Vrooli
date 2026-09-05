import type { TerrainVisualTuning } from '../config'

export function terrainTintVariation(x: number, z: number, strength: number, settings: TerrainVisualTuning): number {
  return strength * (settings.tintBase
    + Math.sin(x * settings.tintFrequencyX1 + z * settings.tintFrequencyZ1) * settings.tintAmplitude
    + Math.sin(x * settings.tintFrequencyX2 - z * settings.tintFrequencyZ2) * settings.tintAmplitude)
}

export function terrainMaterialSettings(wetness: number, settings: TerrainVisualTuning) {
  return {
    color: wetness > 0 ? settings.wetColor : settings.dryColor,
    roughness: Math.max(settings.minimumRoughness, 1 - wetness * settings.wetRoughnessScale),
  }
}
