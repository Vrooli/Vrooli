import { describe, expect, it } from 'vitest'
import { uniformTerrain, tuning } from '../../../config'
import { buildTerrain, type TerrainField } from '../field'
import { waterCells } from '../water'
import { waterSurfaceComponents } from '../waterSurface'

function field(heights: number[]): TerrainField {
  return { radius: 10, cellSize: 1, cols: 2, rows: 2, originX: 0, originZ: 0, height: new Float32Array(heights), moisture: new Float32Array(4) }
}

describe('waterSurfaceComponents', () => {
  const waterTuning = uniformTerrain({ ...tuning.terrain, waterLevel: 0, moistureBasinDepth: 0 })

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
