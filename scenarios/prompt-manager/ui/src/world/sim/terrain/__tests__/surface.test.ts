import { describe, expect, it } from 'vitest'
import { scenes, tuning } from '../../../config'
import { sceneBiomes } from '../../../config/biomes'
import { terrainForBounds } from '../../layout/centre'
import { makeWorldInput } from '../../__tests__/fixtures'
import { generateWorld } from '../../world'
import { runCooperatively } from '../../cooperative'
import { bakeVertexColour } from '../colour'
import { heightAt, moistureAt } from '../field'
import { shoreDistance } from '../water'
import { terrainSurfaceSteps } from '../surface'
import { terrainMeshSteps, type TerrainMeshInput } from '../mesh'

describe('prepared terrain surface', () => {
  it('prepares flat terrain mesh attributes and allows cancellation without changing source arrays', async () => {
    const cells = 257
    const input: TerrainMeshInput = {
      field: { cols: cells, rows: 1, radius: cells, cellSize: 1, originX: 0, originZ: 0, height: new Float32Array(cells), moisture: new Float32Array(cells) },
      surface: { normals: Float32Array.from({ length: cells * 3 }, (_, index) => index % 3 === 1 ? 1 : 0), colours: new Float64Array(cells * 3).fill(0.5), aoStrength: new Float64Array(cells).fill(1), wetShade: new Float64Array(cells).fill(1) },
      innerRadiusSetting: 10, profile: { terrainInnerRadius: 10, terrainCellScale: 1 }, visual: tuning.visual.terrain,
      weather: { terrainTintMix: 0, terrainTintVariation: 0 }, tint: [1, 1, 1], shadowTint: [0, 0, 0],
    }
    const before = JSON.stringify(input)
    const mesh = await runCooperatively(terrainMeshSteps(input))
    expect(mesh.vertices.length).toBe(cells * 3)
    expect(mesh.vertices[3]).toBe(1)
    expect(mesh.normals).toEqual(input.surface.normals)
    expect([...mesh.colours].every(value => value === 0.5)).toBe(true)
    expect(mesh.indices.length).toBe(0)
    expect(mesh.sphere).toEqual({ center: [128, 0, 0], radius: 128 })
    const controller = new AbortController()
    const steps = terrainMeshSteps(input)
    let checkpoints = 0
    await expect(runCooperatively(steps, { signal: controller.signal, onProgress: () => { if (++checkpoints === 2) controller.abort(new Error('cancel mesh')) } })).rejects.toThrow('cancel mesh')
    expect(steps.next().done).toBe(true)
    expect(JSON.stringify(input)).toBe(before)
  })

  it.each(['park', 'office'] as const)('matches the original per-vertex shading calculations for every %s sample', scene => {
    const world = generateWorld(makeWorldInput({ teams: 1, agents: 2, scene }), tuning)
    const field = world.terrain
    const surface = world.terrainSurface
    const resolver = terrainForBounds(scenes[scene], tuning.terrain, world.bounds)
    const set = sceneBiomes(scenes[scene])
    let normalError = 0
    let colourError = 0
    for (let row = 0; row < field.rows; row++) for (let col = 0; col < field.cols; col++) {
      const index = row * field.cols + col
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      const half = field.cellSize * 0.5
      const dx = (heightAt(field, x + half, z) - heightAt(field, x - half, z)) / (half * 2)
      const dz = (heightAt(field, x, z + half) - heightAt(field, x, z - half)) / (half * 2)
      const length = Math.hypot(dx, 1, dz)
      for (const [axis, value] of [-dx / length, 1 / length, -dz / length].entries()) normalError = Math.max(normalError, Math.abs((surface.normals[index * 3 + axis] ?? NaN) - Math.fround(value)))
      const biome = set.biomes[world.biomes[index] ?? set.biomes.length - 1] ?? set.biomes[set.biomes.length - 1]
      if (!biome) throw new Error('Missing biome')
      const local = resolver.at(x, z)
      const shore = shoreDistance(field, resolver, x, z)
      const ao = (index % 11) / 10
      const expected = bakeVertexColour({ moisture: moistureAt(field, x, z), path: world.pathMask[index] ?? 0, ao, wetShore: shore < 0 ? Math.max(0, 1 + shore / local.wetShoreWidth) : 0, wetShoreDarkening: local.wetShoreDarkening }, biome)
      for (let axis = 0; axis < 3; axis++) {
        const actual = (surface.colours[index * 3 + axis] ?? NaN) * (1 - ao * (surface.aoStrength[index] ?? NaN)) * (surface.wetShade[index] ?? NaN)
        colourError = Math.max(colourError, Math.abs(actual - (expected[axis] ?? NaN)))
      }
    }
    expect(normalError).toBe(0)
    expect(colourError).toBeLessThan(1e-12)
  })

  it('cancels surface preparation without modifying source terrain', async () => {
    const world = generateWorld(makeWorldInput({ teams: 1, agents: 2 }), tuning)
    const original = world.terrain.height.slice()
    const resolver = terrainForBounds(scenes.park, tuning.terrain, world.bounds)
    const steps = terrainSurfaceSteps(world.terrain, resolver, world.biomes, world.pathMask, sceneBiomes(scenes.park))
    const controller = new AbortController()
    let count = 0
    await expect(runCooperatively(steps, { signal: controller.signal, yieldTask: () => Promise.resolve(), onProgress: () => { if (++count === 8) controller.abort(new Error('cancel surface')) } })).rejects.toThrow('cancel surface')
    expect(world.terrain.height).toEqual(original)
    expect(steps.next().done).toBe(true)
  })
})
