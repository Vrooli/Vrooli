import { describe, expect, it } from 'vitest'
import { biomeSets, tuning } from '../../../config'
import { biomeGrid, buildTerrain, classify } from '..'

describe('biomes', () => {
  it('classifies every cell deterministically and shows variety outdoors', () => {
    const field = buildTerrain({ seed: 1, tuning: tuning.terrain })
    const first = biomeGrid(field, tuning.terrain, biomeSets.park)
    const again = biomeGrid(field, tuning.terrain, biomeSets.park)
    expect(first).toEqual(again)
    expect(new Set(first).size).toBeGreaterThanOrEqual(3)
    expect([...first].every((index) => index < biomeSets.park.biomes.length)).toBe(true)
  })

  it('uses the floor biome everywhere indoors', () => {
    const field = buildTerrain({ seed: 99, tuning: tuning.terrain })
    expect(classify(field, tuning.terrain, biomeSets.office, 0, 0)).toBe('floor')
    expect(new Set(biomeGrid(field, tuning.terrain, biomeSets.office))).toEqual(new Set([0]))
  })
})
