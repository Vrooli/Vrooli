import { describe, expect, it } from 'vitest'
import { biomeSets, tuning } from '../../../config'
import { bakeVertexColour, buildTerrain, heightFieldAo } from '..'

describe('terrain colour', () => {
  it('blends biome ramp, path and occlusion into bounded RGB', () => {
    const biome = biomeSets.park.biomes[0]
    if (!biome) throw new Error('missing biome')
    const colour = bakeVertexColour({ moisture: 0.5, path: 0.8, ao: 0.25 }, biome)
    expect(colour.every((channel) => channel >= 0 && channel <= 1)).toBe(true)
    expect(colour).not.toEqual(bakeVertexColour({ moisture: 0.5, path: 0, ao: 0 }, biome))
  })

  it('returns a bounded height-field occlusion term', () => {
    const field = buildTerrain({ seed: 7, tuning: tuning.terrain })
    expect(heightFieldAo(field, 0, 0, 3, 8)).toBeGreaterThanOrEqual(0)
    expect(heightFieldAo(field, 0, 0, 3, 8)).toBeLessThanOrEqual(1)
  })
})
