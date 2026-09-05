import type { BiomeSet, TerrainResolver } from '../../config'
import { moistureAt, slopeAt, type TerrainField } from './field'
import { wetHeight } from './water'

function within(value: number, bounds: readonly [number, number]): boolean {
  return value >= bounds[0] && value <= bounds[1]
}

export function classify(field: TerrainField, tuning: TerrainResolver, biomeSet: BiomeSet, x: number, z: number): string {
  const height = wetHeight(field, tuning, x, z)
  // Water follows the active level, including scene overrides. A fixed biome
  // height bound cannot become a second authority when that level changes.
  if (height <= tuning.at(x, z).waterLevel) return 'water'
  const moisture = moistureAt(field, x, z)
  const slope = slopeAt(field, x, z)
  const match = biomeSet.biomes.find((biome) => biome.id !== 'water' && within(height, biome.bounds.height) && within(moisture, biome.bounds.moisture) && within(slope, biome.bounds.slope))
  return match?.id ?? biomeSet.biomes[biomeSet.biomes.length - 1]?.id ?? 'unknown'
}

export function biomeGrid(field: TerrainField, tuning: TerrainResolver, biomeSet: BiomeSet, at: (x: number, z: number) => BiomeSet = () => biomeSet): Uint8Array {
  const indices = new Map(biomeSet.biomes.map((biome, index) => [biome.id, index]))
  const result = new Uint8Array(field.cols * field.rows)
  for (let row = 0; row < field.rows; row += 1) {
    for (let col = 0; col < field.cols; col += 1) {
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      result[row * field.cols + col] = indices.get(classify(field, tuning, at(x, z), x, z)) ?? biomeSet.biomes.length - 1
    }
  }
  return result
}
