import { describe, expect, it } from 'vitest'
import { scenes, tuning } from '../../../config'
import { SceneSchema } from '../../../config/scenes.schema'
import { checkIndoorTerrain } from '../../invariants'
import { makeWorld, makeWorldInput } from '../../__tests__/fixtures'
import { SWEEP_SEEDS } from '../../__tests__/seeds'
import { buildTerrain, heightAt, isWater } from '../../terrain'
import { resolveTerrain } from '../../../config'
import { rebuildLayout } from '../../world'
import { centreRegion, centreWeight, regionForBounds, terrainForBounds } from '../centre'

describe('scene centre', () => {
  it('derives its extent from the plate and blends monotonically without an edge jump', () => {
    const region = centreRegion(scenes.office, { x: 3, z: 2, width: 20, depth: 10 })
    if (!region) throw new Error('office centre is missing')
    expect(region).toEqual({ x: 3, z: 2, width: 32, depth: 22, blend: 4 })
    expect(centreWeight(region, 3, 2)).toBe(1)
    const edge = region.x + region.width / 2
    const weights = Array.from({ length: 41 }, (_, i) => centreWeight(region, edge + i / 10, 2))
    expect(weights[0]).toBe(1)
    expect(weights[weights.length - 1]).toBe(0)
    weights.slice(1).forEach((weight, i) => expect(weight).toBeLessThanOrEqual(weights[i] ?? Number.NEGATIVE_INFINITY))
    expect(centreWeight(region, edge + 1e-5, 2)).toBeCloseTo(1, 8)
    expect(centreWeight({ ...region, blend: 0 }, edge + 0.01, 2)).toBe(0)
    expect(centreRegion(scenes.park, region)).toBeUndefined()
  })

  it('rejects unsupported geometry and unbounded region settings', () => {
    for (const override of [{ source: 'circle' }, { margin: -1 }, { margin: 41 }, { blend: 41 }, { maxBoundaryGrade: 0 }]) {
      expect(SceneSchema.safeParse({ ...scenes.office, centre: { ...scenes.office.centre, ...override } }).success).toBe(false)
    }
  })

  it.each(SWEEP_SEEDS)('keeps a flat dry centre, natural exterior and bounded transition at seed %i', (seed) => {
    const state = makeWorld({ teams: 5, agents: 25, ...{ scene: 'office', seed }, treeVariants: 3 })
    const region = regionForBounds(scenes.office, state.bounds)
    if (!region || !scenes.office.centre) throw new Error('office centre is missing')
    const resolver = terrainForBounds(scenes.office, tuning.terrain, state.bounds)
    const natural = buildTerrain({ seed, tuning: resolveTerrain(scenes.park, tuning) })
    expect(checkIndoorTerrain(state, resolver)).toEqual([])
    const centreHeight = heightAt(state.terrain, region.x, region.z)
    for (const sign of [-1, 1]) {
      expect(heightAt(state.terrain, region.x + sign * region.width / 2, region.z)).toBeCloseTo(centreHeight, 6)
      expect(heightAt(state.terrain, region.x, region.z + sign * region.depth / 2)).toBeCloseTo(centreHeight, 6)
      for (const otherSign of [-1, 1]) expect(heightAt(state.terrain, region.x + sign * region.width / 2, region.z + otherSign * region.depth / 2)).toBeCloseTo(centreHeight, 6)
    }
    let inside = 0
    let exteriorWater = 0
    let maximumGrade = 0
    const field = state.terrain
    for (let row = 0; row < field.rows; row += 1) for (let col = 0; col < field.cols; col += 1) {
      const index = row * field.cols + col
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      const weight = centreWeight(region, x, z)
      if (weight === 1) { inside += 1; expect(isWater(field, resolver, x, z)).toBe(false) }
      if (weight === 0) {
        expect(field.height[index]).toBe(natural.height[index])
        if (isWater(field, resolver, x, z)) exteriorWater += 1
      }
      if (weight > 0 && weight < 1) {
        const dx = (heightAt(field, x + field.cellSize, z) - heightAt(field, x, z)) / field.cellSize
        const dz = (heightAt(field, x, z + field.cellSize) - heightAt(field, x, z)) / field.cellSize
        maximumGrade = Math.max(maximumGrade, Math.hypot(dx, dz))
      }
    }
    expect(inside).toBeGreaterThan(0)
    expect(exteriorWater).toBeGreaterThan(0)
    expect(maximumGrade).toBeLessThanOrEqual(scenes.office.centre.maxBoundaryGrade)
    expect(state.decor.filter((spot) => !spot.roomId).every((spot) => centreWeight(region, ...spot.position) < 1)).toBe(true)
    console.log(JSON.stringify({ seed, inside, exteriorWater, maximumGrade, exteriorVegetation: state.decor.filter((spot) => !spot.roomId).length }))
  })

  it('rebuilds the centre when roster growth changes the floorplate', () => {
    const before = makeWorld({ teams: 2, agents: 6, ...{ scene: 'office' }, treeVariants: 3 })
    const after = rebuildLayout(before, makeWorldInput({ teams: 4, agents: 24, scene: 'office' }), tuning, 3)
    expect(after.bounds.footprint).not.toEqual(before.bounds.footprint)
    expect(after.terrain).not.toBe(before.terrain)
    expect(checkIndoorTerrain(after, terrainForBounds(scenes.office, tuning.terrain, after.bounds))).toEqual([])
  })

  it('detects missing exterior vegetation and damaged centre ground', () => {
    const state = makeWorld({ teams: 2, agents: 6, ...{ scene: 'office' }, treeVariants: 3 })
    const resolver = terrainForBounds(scenes.office, tuning.terrain, state.bounds)
    state.decor = state.decor.filter((spot) => Boolean(spot.roomId))
    expect(checkIndoorTerrain(state, resolver).map((violation) => violation.rule)).toContain('indoor-has-landscape')
    const field = state.terrain
    const cell = Math.round(-field.originZ / field.cellSize) * field.cols + Math.round(-field.originX / field.cellSize)
    field.height[cell] = resolver.at(0, 0).waterLevel - 1
    const rules = checkIndoorTerrain(state, resolver).map((violation) => violation.rule)
    expect(rules).toContain('indoor-is-flat')
    expect(rules).toContain('indoor-has-no-water')
  })
})
