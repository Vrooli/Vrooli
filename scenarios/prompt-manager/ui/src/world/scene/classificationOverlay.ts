import { Color } from 'three'
import type { BiomeSet } from '../config'
import { MAX_TERRAIN_CELLS, type TerrainField } from '../sim/terrain/field'
import { HABITAT_IDS } from '../config/habitats'

export function habitatLegend() {
  return HABITAT_IDS.map((label, index) => ({ label,
    color: label === 'none' ? '#777777' : `#${new Color().setHSL(index / HABITAT_IDS.length, 0.85, 0.5).getHexString()}` }))
}

export function habitatOverlaySteps(field: TerrainField, habitats: Uint8Array) {
  return classificationOverlaySteps(field, habitats, habitatLegend(), 'habitat')
}

export function biomeLegend(set: BiomeSet) {
  return set.biomes.map((biome, index) => ({ label: biome.id,
    color: `#${new Color().setHSL(index / set.biomes.length, 0.85, 0.5).getHexString()}` }))
}

/** Biome IDs refer to terrain samples, not navigation cell centers. */
export function biomeOverlaySteps(field: TerrainField, biomes: Uint8Array, set: BiomeSet) {
  return classificationOverlaySteps(field, biomes, biomeLegend(set), 'biome')
}

function* classificationOverlaySteps(field: TerrainField, biomes: Uint8Array, legend: Array<{ label: string; color: string }>, kind: string) {
  const count = field.cols * field.rows
  if (!Number.isSafeInteger(count) || count < 0 || count > MAX_TERRAIN_CELLS || biomes.length !== count || field.height.length !== count) throw new Error(`Invalid ${kind} overlay field`)
  const palette = legend.map(entry => new Color(entry.color))
  const counts = legend.map(() => 0)
  const positions = new Float32Array(count * 3)
  const colors = new Float32Array(count * 3)
  for (let index = 0; index < count; index++) {
    if (index % 128 === 0) yield { completed: index, total: count }
    const biome = biomes[index] ?? -1
    const color = palette[biome]
    if (!color) throw new Error(`Classification field references an unknown ${kind}`)
    counts[biome] = (counts[biome] ?? 0) + 1
    positions.set([field.originX + index % field.cols * field.cellSize, (field.height[index] ?? 0) + 0.05,
      field.originZ + Math.floor(index / field.cols) * field.cellSize], index * 3)
    colors.set([color.r, color.g, color.b], index * 3)
  }
  return { positions, colors, count, bytes: positions.byteLength + colors.byteLength,
    summary: legend.map((entry, index) => `${entry.label}: ${(counts[index] ?? 0).toLocaleString()}`).join(' · '), counts }
}
