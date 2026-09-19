import { describe, expect, it } from 'vitest'
import { uniformTerrain, biomeSets, tuning } from '../../../config'
import { bakeVertexColour, buildTerrain, heightFieldAo } from '..'

describe('terrain colour', () => {
  it('blends biome ramp, path and occlusion into bounded RGB', () => {
    const biome = biomeSets.park.biomes[0]
    if (!biome) throw new Error('missing biome')
    const colour = bakeVertexColour({ moisture: 0.5, path: 0.8, ao: 0.25 }, biome)
    expect(colour.every((channel) => channel >= 0 && channel <= 1)).toBe(true)
    expect(colour).not.toEqual(bakeVertexColour({ moisture: 0.5, path: 0, ao: 0 }, biome))
    const wet = bakeVertexColour({ moisture: 0.5, path: 0, ao: 0, wetShore: 1, wetShoreDarkening: tuning.terrain.wetShoreDarkening }, biome)
    const dry = bakeVertexColour({ moisture: 0.5, path: 0, ao: 0, wetShore: 0 }, biome)
    expect(wet.every((channel, index) => channel < (dry[index] ?? 0))).toBe(true)
  })

  it('returns a bounded height-field occlusion term', () => {
    const field = buildTerrain({ seed: 7, tuning: uniformTerrain(tuning.terrain) })
    expect(heightFieldAo(field, 0, 0, 3, 8)).toBeGreaterThanOrEqual(0)
    expect(heightFieldAo(field, 0, 0, 3, 8)).toBeLessThanOrEqual(1)
  })
})
