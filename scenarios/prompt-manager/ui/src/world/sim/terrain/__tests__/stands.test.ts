import { describe, expect, it } from 'vitest'
import { tuning } from '../../../config'
import { hashString } from '../../rng'
import { SWEEP_SEEDS } from '../../__tests__/seeds'
import { standMask } from '../stands'
import { makeWorld } from '../../__tests__/fixtures'

function neighbourVariance(state: ReturnType<typeof makeWorld>): number {
  const trees = state.decor.filter((spot) => spot.kind === 'tree')
  const distances = trees.map((tree, index) => Math.min(...trees.filter((_, other) => other !== index).map((other) => Math.hypot(tree.position[0] - other.position[0], tree.position[1] - other.position[1]))))
  expect(distances.length).toBeGreaterThan(2)
  const mean = distances.reduce((sum, distance) => sum + distance, 0) / distances.length
  return distances.reduce((sum, distance) => sum + (distance - mean) ** 2, 0) / distances.length
}

describe('vegetation stands', () => {
  it.each(SWEEP_SEEDS)('has more varied tree spacing than uniform scatter at seed %i', (seed) => {
    const clustered = neighbourVariance(makeWorld({ teams: 5, agents: 25, ...{ seed }, treeVariants: 3 }))
    const uniform = neighbourVariance(makeWorld({ teams: 5, agents: 25, ...{ seed }, tuning: { ...tuning, layout: { ...tuning.layout, stands: { ...tuning.layout.stands, floor: 1 } } }, treeVariants: 3 }))
    console.info(JSON.stringify({ seed, clusteredVariance: clustered, uniformVariance: uniform, factor: clustered / uniform }))
    expect(clustered).toBeGreaterThan(uniform)
  })

  it.each(SWEEP_SEEDS)('is bounded, repeatable, and species-specific at seed %i', (seed) => {
    const oak = hashString(`stand:${seed}:oak`)
    const pine = hashString(`stand:${seed}:pine`)
    let different = 0
    for (let index = 0; index < 1000; index += 1) {
      const x = index % 40 - 20
      const z = Math.floor(index / 40) - 12
      const value = standMask(x, z, oak, tuning.layout.stands)
      expect(value).toBeGreaterThanOrEqual(tuning.layout.stands.floor)
      expect(value).toBeLessThanOrEqual(1)
      expect(value).toBe(standMask(x, z, oak, tuning.layout.stands))
      if (value !== standMask(x, z, pine, tuning.layout.stands)) different += 1
      expect(standMask(x, z, oak, { ...tuning.layout.stands, floor: 1 })).toBe(1)
    }
    expect(different).toBeGreaterThan(0)
  })
})
