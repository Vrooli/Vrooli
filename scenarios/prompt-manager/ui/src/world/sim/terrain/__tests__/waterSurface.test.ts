import { BufferAttribute, BufferGeometry } from 'three'
import { runCooperatively } from '../../cooperative'
import { describe, expect, it } from 'vitest'
import { uniformTerrain, tuning } from '../../../config'
import { buildTerrain, type TerrainField } from '../field'
import { waterCells } from '../water'
import { waterSurfaceComponents, waterGeometrySteps } from '../waterSurface'

function field(heights: number[]): TerrainField {
  return { radius: 10, cellSize: 1, cols: 2, rows: 2, originX: 0, originZ: 0, height: new Float32Array(heights), moisture: new Float32Array(4) }
}

describe('waterSurfaceComponents', () => {
  const waterTuning = uniformTerrain({ ...tuning.terrain, waterLevel: 0, moistureBasinDepth: 0 })

  it('prepares transferable geometry with the same normals as the rendering oracle', async () => {
    const terrain = buildTerrain({ seed: 1, tuning: uniformTerrain(tuning.terrain) })
    const resolver = uniformTerrain(tuning.terrain)
    const surfaces = waterSurfaceComponents(terrain, resolver)
    const prepared = await runCooperatively(waterGeometrySteps(terrain, resolver), { yieldTask: () => Promise.resolve() })
    expect(prepared).toHaveLength(surfaces.length)
    for (const [index, actual] of prepared.entries()) {
      const expected = surfaces[index]
      if (!expected) throw new Error('Missing reference pond')
      expect(actual.positions).toEqual(new Float32Array(expected.positions))
      expect(actual.shore).toEqual(new Float32Array(expected.shore))
      expect(Array.from(actual.indices)).toEqual(expected.indices)
      const geometry = new BufferGeometry()
      geometry.setAttribute('position', new BufferAttribute(new Float32Array(expected.positions), 3))
      geometry.setIndex(expected.indices)
      geometry.computeVertexNormals()
      geometry.computeBoundingSphere()
      expect(actual.sphere.center).toEqual(geometry.boundingSphere?.center.toArray())
      expect(actual.sphere.radius).toBeCloseTo(geometry.boundingSphere?.radius ?? 0, 8)
      const normals = geometry.getAttribute('normal').array
      for (let i = 0; i < normals.length; i++) expect(actual.normals[i]).toBeCloseTo(normals[i] ?? 0, 6)
      geometry.dispose()
    }
  })

  it('cancels pond preparation without publishing meshes or changing the source field', async () => {
    const terrain = buildTerrain({ seed: 1, tuning: uniformTerrain(tuning.terrain) })
    const before = terrain.height.slice()
    const controller = new AbortController()
    const steps = waterGeometrySteps(terrain, uniformTerrain(tuning.terrain))
    let checkpoints = 0
    await expect(runCooperatively(steps, {
      signal: controller.signal, yieldTask: () => Promise.resolve(),
      onProgress: () => { if (++checkpoints === 10) controller.abort(new Error('cancel water')) },
    })).rejects.toThrow('cancel water')
    expect(terrain.height).toEqual(before)
    expect(steps.next().done).toBe(true)
  })

  it('clips a one-corner wet cell to a shoreline triangle', () => {
    const surfaces = waterSurfaceComponents(field([-1, 1, 1, 1]), waterTuning)
    expect(surfaces).toHaveLength(1)
    expect(surfaces[0]?.indices).toHaveLength(3)
    expect(surfaces[0]?.positions).toHaveLength(9)
  })

  it('keeps disconnected saddle corners in separate component meshes', () => {
    const surfaces = waterSurfaceComponents(field([-1, 1, 1, -1]), waterTuning)
    expect(surfaces).toHaveLength(2)
    expect(surfaces.every((surface) => surface.indices.length === 3)).toBe(true)
  })

  it('emits one lower-triangle buffer per connected pond on generated terrain', () => {
    const terrain = buildTerrain({ seed: 1, tuning: uniformTerrain(tuning.terrain) })
    const cells = waterCells(terrain, uniformTerrain(tuning.terrain))
    const surfaces = waterSurfaceComponents(terrain, uniformTerrain(tuning.terrain))
    const triangles = surfaces.reduce((sum, surface) => sum + surface.indices.length / 3, 0)
    expect(surfaces).toHaveLength(cells.components)
    expect(triangles).toBeLessThan(7_862)
  })
})
