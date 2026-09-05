import { describe, expect, it } from 'vitest'
import { SpacingIndex } from '../scatter'
import { Rng } from '../../rng'
import type { Vec2 } from '../../model'
import { SWEEP_SEEDS } from '../../__tests__/seeds'
import { VegetationEntrySchema } from '../../../config/biomes.schema'

describe('scatter spacing', () => {
  it.each(SWEEP_SEEDS)('matches exhaustive spacing for variable biome radii at seed %i', (seed) => {
    const rng = new Rng(seed)
    const index = new SpacingIndex(12)
    const accepted: Vec2[] = []
    for (let i = 0; i < 2000; i += 1) {
      const point: Vec2 = [rng.range(-150, 150), rng.range(-150, 150)]
      const radius = rng.range(2, 12)
      const expected = accepted.some((other) => (point[0] - other[0]) ** 2 + (point[1] - other[1]) ** 2 < radius ** 2)
      expect(index.overlaps(point, radius)).toBe(expected)
      if (!expected) {
        accepted.push(point)
        index.insert(point)
      }
    }
  })

  it('requires class and scale independently of the asset name', () => {
    expect(VegetationEntrySchema.safeParse({ density: 0.1 }).success).toBe(false)
    expect(VegetationEntrySchema.safeParse({ density: 0.1, class: 'tree' }).success).toBe(false)
    expect(VegetationEntrySchema.safeParse({ density: 0.1, scaleRef: 'tree' }).success).toBe(false)
    expect(VegetationEntrySchema.parse({ density: 0.1, class: 'tree', scaleRef: 'tree' })).toEqual({ density: 0.1, class: 'tree', scaleRef: 'tree' })
  })
})
