import type { QualityProfile, TerrainVisualTuning, WeatherPreset } from '../../config'
import { heightFieldAo, type Rgb } from './colour'
import { MAX_TERRAIN_CELLS, type TerrainField } from './field'
import type { TerrainSurfaceSamples } from './surface'
import { terrainTintVariation } from './colour'

export interface TerrainMeshInput {
  field: TerrainField
  surface: TerrainSurfaceSamples
  innerRadiusSetting: number
  profile: Pick<QualityProfile, 'terrainInnerRadius' | 'terrainCellScale'>
  visual: TerrainVisualTuning
  weather: Pick<WeatherPreset, 'terrainTintMix' | 'terrainTintVariation'>
  tint: Rgb
  shadowTint: Rgb
}
export interface TerrainMeshData { vertices: Float32Array; normals: Float32Array; colours: Float32Array; indices: Uint32Array; sphere: { center: [number, number, number]; radius: number } }

export function prepareTerrainMesh(input: TerrainMeshInput): TerrainMeshData {
  const steps = terrainMeshSteps(input)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

/** Rendering inputs are independent of the simulation recipe. Source arrays remain borrowed. */
export function* terrainMeshSteps({ field, surface, innerRadiusSetting, profile, visual, weather, tint, shadowTint }: TerrainMeshInput): Generator<{ completed: number; total: number }, TerrainMeshData> {
    if (!Number.isSafeInteger(field.cols) || !Number.isSafeInteger(field.rows) || field.cols < 1 || field.rows < 1 || field.cols * field.rows > MAX_TERRAIN_CELLS) throw new Error('Terrain mesh grid exceeds allocation budget')
    if (![field.cellSize, field.radius, innerRadiusSetting, profile.terrainInnerRadius, profile.terrainCellScale, visual.aoRadius].every(value => Number.isFinite(value) && value > 0)) throw new Error('Invalid terrain mesh sampling dimensions')
    if (!Number.isInteger(visual.aoSamples) || visual.aoSamples < 1 || visual.aoSamples > 32) throw new Error('Invalid terrain mesh AO sample count')
    const innerRadius = Math.max(field.cellSize, Math.min(innerRadiusSetting, profile.terrainInnerRadius))
    const step = Math.max(1, Math.round(profile.terrainCellScale * innerRadiusSetting / innerRadius))
    const cols = Math.floor((field.cols - 1) / step) + 1
    const rows = Math.floor((field.rows - 1) / step) + 1
    const vertices = new Float32Array(cols * rows * 3)
    const normals = new Float32Array(vertices.length)
    const colours = new Float32Array(vertices.length)
    const indices: number[] = []
    for (let row = 0; row < rows; row += 1) for (let col = 0; col < cols; col += 1) {
      if (col % 128 === 0) yield { completed: row, total: rows }
      const sourceRow = Math.min(field.rows - 1, row * step)
      const sourceCol = Math.min(field.cols - 1, col * step)
      const sourceIndex = sourceRow * field.cols + sourceCol
      const index = row * cols + col
      const x = field.originX + sourceCol * field.cellSize
      const z = field.originZ + sourceRow * field.cellSize
      const y = field.height[sourceIndex] ?? 0
      vertices.set([x, y, z], index * 3)
      normals.set(surface.normals.subarray(sourceIndex * 3, sourceIndex * 3 + 3), index * 3)
      {
        const ao = heightFieldAo(field, x, z, visual.aoRadius, visual.aoSamples)
        const shade = 1 - Math.max(0, Math.min(1, ao)) * (surface.aoStrength[sourceIndex] ?? 0)
        const wetShade = surface.wetShade[sourceIndex] ?? 1
        const colour = [0, 1, 2].map(axis => (surface.colours[sourceIndex * 3 + axis] ?? 0) * shade * wetShade)
        const mix = weather.terrainTintMix
        const variation = terrainTintVariation(x, z, weather.terrainTintVariation, visual)
        const tintR = tint[0] + (shadowTint[0] - tint[0]) * variation
        const tintG = tint[1] + (shadowTint[1] - tint[1]) * variation
        const tintB = tint[2] + (shadowTint[2] - tint[2]) * variation
        colours.set([(colour[0] ?? 0) * (1 - mix) + tintR * mix, (colour[1] ?? 0) * (1 - mix) + tintG * mix, (colour[2] ?? 0) * (1 - mix) + tintB * mix], index * 3)
      }
    }
    for (let row = 0; row < rows - 1; row += 1) for (let col = 0; col < cols - 1; col += 1) {
      if (col % 128 === 0) yield { completed: row, total: rows }
      const x = field.originX + (col + 0.5) * field.cellSize * step
      const z = field.originZ + (row + 0.5) * field.cellSize * step
      if (Math.hypot(x, z) > field.radius) continue
      const a = row * cols + col
      const b = a + 1
      const c = a + cols
      const d = c + 1
      indices.push(a, c, b, b, c, d)
    }
    const typedIndices = new Uint32Array(indices.length)
    for (let index = 0; index < indices.length; index++) {
      if (index % 128 === 0) yield { completed: index, total: indices.length }
      typedIndices[index] = indices[index] ?? 0
    }
    const min = [Infinity, Infinity, Infinity]
    const max = [-Infinity, -Infinity, -Infinity]
    for (let index = 0; index < vertices.length; index += 3) {
      if (index % 384 === 0) yield { completed: index, total: vertices.length }
      for (let axis = 0; axis < 3; axis++) {
        min[axis] = Math.min(min[axis] ?? Infinity, vertices[index + axis] ?? 0)
        max[axis] = Math.max(max[axis] ?? -Infinity, vertices[index + axis] ?? 0)
      }
    }
    const center = [0, 1, 2].map(axis => ((min[axis] ?? 0) + (max[axis] ?? 0)) / 2) as [number, number, number]
    let radiusSquared = 0
    for (let index = 0; index < vertices.length; index += 3) {
      if (index % 384 === 0) yield { completed: index, total: vertices.length }
      const x = (vertices[index] ?? 0) - center[0]
      const y = (vertices[index + 1] ?? 0) - center[1]
      const z = (vertices[index + 2] ?? 0) - center[2]
      radiusSquared = Math.max(radiusSquared, x * x + y * y + z * z)
    }
    return { vertices, normals, colours, indices: typedIndices, sphere: { center, radius: Math.sqrt(radiusSquared) } }
}
