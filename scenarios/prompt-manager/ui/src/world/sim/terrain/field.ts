import type { TerrainResolver } from '../../config'
import { smoothstep } from '../../config/regions'
import { hashString } from '../rng'
import { fbm } from './noise'

export interface TerrainField {
  radius: number
  cellSize: number
  cols: number
  rows: number
  originX: number
  originZ: number
  height: Float32Array
  moisture: Float32Array
}

export interface GroundSampler {
  heightAt(x: number, z: number): number
}

export function groundSampler(field: TerrainField): GroundSampler {
  return { heightAt: (x, z) => heightAt(field, x, z) }
}

export interface BuildTerrainInput {
  seed: number
  tuning: TerrainResolver
}

export interface TerrainProgress { completed: number; total: number }
/** Hard allocation ceiling, independent of the visual quality preset. */
export const MAX_TERRAIN_CELLS = 1_048_576

/** The synchronous API and cooperative generation consume the same row algorithm. */
export function buildTerrain(input: BuildTerrainInput): TerrainField {
  const steps = buildTerrainSteps(input)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* buildTerrainSteps({ seed, tuning: resolver }: BuildTerrainInput): Generator<TerrainProgress, TerrainField> {
  const tuning = resolver.base()
  const cellSize = tuning.cellSize
  if (!Number.isFinite(tuning.radius) || tuning.radius <= 0 || !Number.isFinite(cellSize) || cellSize <= 0) throw new Error('Terrain extent and cell size must be finite and positive')
  const cols = Math.ceil((tuning.radius * 2) / tuning.cellSize) + 1
  const rows = cols
  if (!Number.isSafeInteger(cols) || cols * rows > MAX_TERRAIN_CELLS) throw new Error('Terrain grid exceeds the allocation budget')
  const originX = -tuning.radius
  const originZ = -tuning.radius
  const height = new Float32Array(cols * rows)
  const moisture = new Float32Array(cols * rows)
  const field = { radius: tuning.radius, cellSize, cols, rows, originX, originZ, height, moisture }
  return yield* rebuildTerrainRegionSteps(field, { seed, tuning: resolver })
}

/** Recompute only selected samples of generation-owned terrain. The caller must
 * retain the original seed and select every point affected by a resolver change.
 * Never use on published/cache-owned fields; cancellation discards this field.
 */
export function* rebuildTerrainRegionSteps(field: TerrainField, { seed, tuning: resolver }: BuildTerrainInput, includes?: (x: number, z: number) => boolean): Generator<TerrainProgress, TerrainField> {
  const base = resolver.base()
  if (base.radius !== field.radius || base.cellSize !== field.cellSize) throw new Error('Regional terrain rebuild cannot change grid dimensions')
  const terrainSeed = hashString(`terrain:${seed}`)
  const moistureSeed = hashString(`terrain-moisture:${seed}`)
  const detailSeed = hashString(`terrain-detail:${seed}`)
  const warpSeed = hashString(`terrain-warp:${seed}`)

  const { cols, rows, originX, originZ, cellSize, height, moisture } = field
  for (let row = 0; row < rows; row += 1) {
    const z = originZ + row * cellSize
    for (let col = 0; col < cols; col += 1) {
      const x = originX + col * cellSize
      if (includes && !includes(x, z)) continue
      const index = row * cols + col
      const tuning = resolver.at(x, z)
      const falloffRadius = tuning.radius * tuning.falloffStart
      const radius = Math.hypot(x, z)
      if (radius >= tuning.radius) { height[index] = 0; moisture[index] = 0; continue }
      const falloff = 1 - smoothstep(falloffRadius, tuning.radius, radius)
      const landform = tuning.amplitude * fbm(x * tuning.frequency, z * tuning.frequency, terrainSeed, tuning.octaves, tuning.lacunarity, tuning.gain)
      const detail = tuning.detailAmplitude * fbm(x * tuning.detailFrequency, z * tuning.detailFrequency, detailSeed, tuning.octaves, tuning.lacunarity, tuning.gain)
      height[index] = (landform + detail) * falloff
      const warpX = fbm(x * tuning.moistureFrequency, z * tuning.moistureFrequency, warpSeed, tuning.octaves, tuning.lacunarity, tuning.gain) * tuning.moistureWarp
      const warpZ = fbm(x * tuning.moistureFrequency, z * tuning.moistureFrequency, warpSeed + 1, tuning.octaves, tuning.lacunarity, tuning.gain) * tuning.moistureWarp
      const wetness = fbm((x + warpX) * tuning.moistureFrequency, (z + warpZ) * tuning.moistureFrequency, moistureSeed, tuning.octaves, tuning.lacunarity, tuning.gain)
      moisture[index] = Math.max(0, Math.min(1, wetness * 0.5 + 0.5))
      // Basin depth is real ground geometry. All ground consumers (rendering,
      // navigation, vegetation and water classification) must share this height.
      height[index] = (height[index] ?? 0) - (moisture[index] ?? 0) * tuning.moistureBasinDepth * falloff
    }
    yield { completed: row + 1, total: rows }
  }

  return field
}

function sample(field: TerrainField, values: Float32Array, x: number, z: number): number {
  if (Math.hypot(x, z) >= field.radius) return 0
  const fx = (x - field.originX) / field.cellSize
  const fz = (z - field.originZ) / field.cellSize
  const x0 = Math.max(0, Math.min(field.cols - 1, Math.floor(fx)))
  const z0 = Math.max(0, Math.min(field.rows - 1, Math.floor(fz)))
  const x1 = Math.min(field.cols - 1, x0 + 1)
  const z1 = Math.min(field.rows - 1, z0 + 1)
  const tx = fx - x0
  const tz = fz - z0
  const a = values[z0 * field.cols + x0] ?? 0
  const b = values[z0 * field.cols + x1] ?? a
  const c = values[z1 * field.cols + x0] ?? a
  const d = values[z1 * field.cols + x1] ?? c
  return (a + (b - a) * tx) + ((c + (d - c) * tx) - (a + (b - a) * tx)) * tz
}

export function heightAt(field: TerrainField, x: number, z: number): number {
  return sample(field, field.height, x, z)
}

/** Conservative ceiling of every bilinear terrain cell touched by a rectangle. */
export function maximumHeightInRegion(field: TerrainField, minX: number, minZ: number, maxX: number, maxZ: number): number {
  if (![minX, minZ, maxX, maxZ].every(Number.isFinite) || minX > maxX || minZ > maxZ) throw new Error('Invalid terrain query bounds')
  if (maxX < field.originX || maxZ < field.originZ || minX > field.originX + (field.cols - 1) * field.cellSize || minZ > field.originZ + (field.rows - 1) * field.cellSize) return 0
  const x0 = Math.max(0, Math.floor((minX - field.originX) / field.cellSize))
  const z0 = Math.max(0, Math.floor((minZ - field.originZ) / field.cellSize))
  const x1 = Math.min(field.cols - 1, Math.floor((maxX - field.originX) / field.cellSize) + 1)
  const z1 = Math.min(field.rows - 1, Math.floor((maxZ - field.originZ) / field.cellSize) + 1)
  // The radial field's exterior has height zero; including it is conservative.
  let maximum = 0
  for (let z = z0; z <= z1; z++) for (let x = x0; x <= x1; x++) maximum = Math.max(maximum, field.height[z * field.cols + x] ?? 0)
  return maximum
}

export function moistureAt(field: TerrainField, x: number, z: number): number {
  return sample(field, field.moisture, x, z)
}

/** Ground incline in radians, derived by central difference. */
export function slopeAt(field: TerrainField, x: number, z: number): number {
  const half = field.cellSize * 0.5
  const dx = (heightAt(field, x + half, z) - heightAt(field, x - half, z)) / (half * 2)
  const dz = (heightAt(field, x, z + half) - heightAt(field, x, z - half)) / (half * 2)
  return Math.atan(Math.hypot(dx, dz))
}
