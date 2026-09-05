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

export function buildTerrain({ seed, tuning: resolver }: BuildTerrainInput): TerrainField {
  const tuning = resolver.base()
  const cellSize = tuning.cellSize
  const cols = Math.ceil((tuning.radius * 2) / tuning.cellSize) + 1
  const rows = cols
  const originX = -tuning.radius
  const originZ = -tuning.radius
  const height = new Float32Array(cols * rows)
  const moisture = new Float32Array(cols * rows)
  const terrainSeed = hashString(`terrain:${seed}`)
  const moistureSeed = hashString(`terrain-moisture:${seed}`)
  const detailSeed = hashString(`terrain-detail:${seed}`)
  const warpSeed = hashString(`terrain-warp:${seed}`)

  for (let row = 0; row < rows; row += 1) {
    const z = originZ + row * tuning.cellSize
    for (let col = 0; col < cols; col += 1) {
      const x = originX + col * cellSize
      const index = row * cols + col
      const tuning = resolver.at(x, z)
      const falloffRadius = tuning.radius * tuning.falloffStart
      const radius = Math.hypot(x, z)
      if (radius >= tuning.radius) continue
      const falloff = 1 - smoothstep(falloffRadius, tuning.radius, radius)
      const landform = tuning.amplitude * fbm(x * tuning.frequency, z * tuning.frequency, terrainSeed, tuning.octaves, tuning.lacunarity, tuning.gain)
      const detail = tuning.detailAmplitude * fbm(x * tuning.detailFrequency, z * tuning.detailFrequency, detailSeed, tuning.octaves, tuning.lacunarity, tuning.gain)
      height[index] = (landform + detail) * falloff
      const warpX = fbm(x * tuning.moistureFrequency, z * tuning.moistureFrequency, warpSeed, tuning.octaves, tuning.lacunarity, tuning.gain) * tuning.moistureWarp
      const warpZ = fbm(x * tuning.moistureFrequency, z * tuning.moistureFrequency, warpSeed + 1, tuning.octaves, tuning.lacunarity, tuning.gain) * tuning.moistureWarp
      const wetness = fbm((x + warpX) * tuning.moistureFrequency, (z + warpZ) * tuning.moistureFrequency, moistureSeed, tuning.octaves, tuning.lacunarity, tuning.gain)
      moisture[index] = Math.max(0, Math.min(1, wetness * 0.5 + 0.5))
    }
  }

  return { radius: tuning.radius, cellSize: tuning.cellSize, cols, rows, originX, originZ, height, moisture }
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
