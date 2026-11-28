import type { BiomeSet, TerrainResolver } from '../../config'
import { bakeVertexColour } from './colour'
import { heightAt, moistureAt, type TerrainField } from './field'
import { shoreDistance } from './water'

/** Quality/weather-independent shading inputs. Doubles preserve the colour
 * calculation order before the renderer writes its final Float32 colours.
 */
export interface TerrainSurfaceSamples {
  normals: Float32Array
  colours: Float64Array
  wetShade: Float64Array
  aoStrength: Float64Array
}

export function* terrainSurfaceSteps(field: TerrainField, resolver: TerrainResolver, biomes: Uint8Array, path: Float32Array, set: BiomeSet): Generator<{ completed: number; total: number }, TerrainSurfaceSamples> {
  const cells = field.rows * field.cols
  const normals = new Float32Array(cells * 3)
  const colours = new Float64Array(cells * 3)
  const wetShade = new Float64Array(cells)
  const aoStrength = new Float64Array(cells)
  const half = field.cellSize * 0.5
  for (let row = 0; row < field.rows; row++) for (let col = 0; col < field.cols; col++) {
    if (col % 128 === 0) yield { completed: row, total: field.rows }
    const index = row * field.cols + col
    const x = field.originX + col * field.cellSize
    const z = field.originZ + row * field.cellSize
    const dx = (heightAt(field, x + half, z) - heightAt(field, x - half, z)) / (half * 2)
    const dz = (heightAt(field, x, z + half) - heightAt(field, x, z - half)) / (half * 2)
    const length = Math.hypot(dx, 1, dz)
    normals.set([-dx / length, 1 / length, -dz / length], index * 3)
    const biome = set.biomes[biomes[index] ?? set.biomes.length - 1] ?? set.biomes[set.biomes.length - 1]
    if (!biome) continue
    const local = resolver.at(x, z)
    const shore = shoreDistance(field, resolver, x, z)
    const wet = shore < 0 ? Math.max(0, 1 + shore / local.wetShoreWidth) : 0
    wetShade[index] = 1 - Math.max(0, Math.min(1, wet)) * Math.max(0, Math.min(1, local.wetShoreDarkening))
    aoStrength[index] = biome.aoStrength
    colours.set(bakeVertexColour({ moisture: moistureAt(field, x, z), path: path[index] ?? 0, ao: 0 }, biome), index * 3)
  }
  yield { completed: field.rows, total: field.rows }
  return { normals, colours, wetShade, aoStrength }
}
