import type { BiomeSet, TerrainTuning } from '../../config'
import { heightAt, moistureAt, slopeAt, type TerrainField } from './field'

function within(value: number, bounds: readonly [number, number]): boolean {
  return value >= bounds[0] && value <= bounds[1]
}

export function classify(field: TerrainField, _tuning: TerrainTuning, biomeSet: BiomeSet, x: number, z: number): string {
  const height = heightAt(field, x, z)
  const moisture = moistureAt(field, x, z)
  const slope = slopeAt(field, x, z)
  const match = biomeSet.biomes.find((biome) => within(height, biome.bounds.height) && within(moisture, biome.bounds.moisture) && within(slope, biome.bounds.slope))
  return match?.id ?? biomeSet.biomes[biomeSet.biomes.length - 1]?.id ?? 'unknown'
}

export function biomeGrid(field: TerrainField, tuning: TerrainTuning, biomeSet: BiomeSet): Uint8Array {
  const indices = new Map(biomeSet.biomes.map((biome, index) => [biome.id, index]))
  const result = new Uint8Array(field.cols * field.rows)
  for (let row = 0; row < field.rows; row += 1) {
    for (let col = 0; col < field.cols; col += 1) {
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      result[row * field.cols + col] = indices.get(classify(field, tuning, biomeSet, x, z)) ?? biomeSet.biomes.length - 1
    }
  }
  return result
}
