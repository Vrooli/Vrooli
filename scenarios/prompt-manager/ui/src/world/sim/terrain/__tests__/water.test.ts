import { describe, expect, it } from 'vitest'
import { uniformTerrain, tuning, biomeSets, type TerrainResolver } from '../../../config'
import type { WorldBounds } from '../../model'
import { findPath } from '../../nav/astar'
import { buildNavGrid, isWalkable } from '../../nav/grid'
import { buildTerrain, classify, isWater, shoreDistance, waterCells } from '..'

describe('terrain water', () => {
  it('shares position-specific water policy with biomes and navigation', () => {
    const dry = { ...tuning.terrain, amplitude: 0, detailAmplitude: 0, moistureBasinDepth: 0, waterLevel: -1 }
    const wet = { ...dry, waterLevel: 1 }
    const resolver: TerrainResolver = { base: () => dry, at: (x) => x < 0 ? wet : dry }
    const field = buildTerrain({ seed: 1, tuning: resolver })
    const bounds: WorldBounds = { width: 20, depth: 20, center: [0, 0], footprint: { width: 1, depth: 1, center: [0, 0] }, outline: [] }
    const grid = buildNavGrid(bounds, [], [], 1, 0.5, 0.4, field, resolver)
    expect(isWater(field, resolver, -3, 0)).toBe(true)
    expect(isWater(field, resolver, 3, 0)).toBe(false)
    expect(classify(field, resolver, biomeSets.park, -3, 0)).toBe('water')
    expect(classify(field, resolver, biomeSets.park, 3, 0)).not.toBe('water')
    expect(shoreDistance(field, resolver, -3, 0)).toBeLessThan(0)
    expect(shoreDistance(field, resolver, 3, 0)).toBeGreaterThan(dry.shoreMargin)
    expect(isWalkable(grid, [-3, 0])).toBe(false)
    expect(isWalkable(grid, [3, 0])).toBe(true)
  })

  it('forms optional deterministic ponds across the seed matrix', () => {
    const counts = [1, 7, 99, 12345].map((seed) => waterCells(buildTerrain({ seed, tuning: uniformTerrain(tuning.terrain) }), uniformTerrain(tuning.terrain)))
    expect(counts.filter((entry) => entry.count > 0).length).toBeGreaterThanOrEqual(2)
    const dry = uniformTerrain({ ...tuning.terrain, waterLevel: -20 })
    expect(waterCells(buildTerrain({ seed: 1, tuning: dry }), dry)).toEqual({ count: 0, components: 0 })
  })

  it('marks water and its dry-side shore margin unwalkable', () => {
    const field = buildTerrain({ seed: 1, tuning: uniformTerrain(tuning.terrain) })
    const bounds: WorldBounds = { width: 80, depth: 80, center: [0, 0], footprint: { width: 1, depth: 1, center: [0, 0] }, outline: [] }
    const grid = buildNavGrid(bounds, [], [], 1, 0.5, 0.4, field, uniformTerrain(tuning.terrain))
    let sawWater = false
    let sawShore = false
    for (let row = 0; row < grid.rows; row += 1) {
      for (let col = 0; col < grid.cols; col += 1) {
        const point: [number, number] = [grid.originX + (col + 0.5) * grid.cellSize, grid.originZ + (row + 0.5) * grid.cellSize]
        const distance = shoreDistance(field, uniformTerrain(tuning.terrain), point[0], point[1])
        if (isWater(field, uniformTerrain(tuning.terrain), point[0], point[1])) sawWater = true
        if (distance >= 0 && distance < tuning.terrain.shoreMargin) sawShore = true
        if (distance < tuning.terrain.shoreMargin) expect(isWalkable(grid, point)).toBe(false)
      }
    }
    expect(sawWater).toBe(true)
    expect(sawShore).toBe(true)
  })

  it('routes between dry points around a pond', () => {
    const field = buildTerrain({ seed: 1, tuning: uniformTerrain(tuning.terrain) })
    const bounds: WorldBounds = { width: 70, depth: 70, center: [0, 0], footprint: { width: 1, depth: 1, center: [0, 0] }, outline: [] }
    const grid = buildNavGrid(bounds, [], [], 1, 0.5, 0.4, field, uniformTerrain(tuning.terrain))
    const wetIndex = field.height.findIndex((_, index) => {
      const x = field.originX + (index % field.cols) * field.cellSize
      const z = field.originZ + Math.floor(index / field.cols) * field.cellSize
      return Math.abs(x) < 25 && Math.abs(z) < 25 && isWater(field, uniformTerrain(tuning.terrain), x, z)
    })
    expect(wetIndex).toBeGreaterThanOrEqual(0)
    if (wetIndex < 0) return
    const wx = field.originX + (wetIndex % field.cols) * field.cellSize
    const wz = field.originZ + Math.floor(wetIndex / field.cols) * field.cellSize
    const from: [number, number] = [wx - 8, wz]
    const to: [number, number] = [wx + 8, wz]
    if (!isWalkable(grid, from) || !isWalkable(grid, to)) return
    const path = findPath(grid, from, to)
    expect(path).not.toBeNull()
    expect(path?.some((point) => Math.abs(point[1] - wz) > 0.5)).toBe(true)
  })
})
