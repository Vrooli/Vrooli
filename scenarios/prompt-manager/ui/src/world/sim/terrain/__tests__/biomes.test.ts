import { describe, expect, it } from 'vitest'
import { uniformTerrain, biomeSets, tuning, scenes } from '../../../config'
import { sceneBiomes } from '../../../config/biomes'
import { makeWorld } from '../../__tests__/fixtures'
import { centreWeight, regionForBounds } from '../../layout/centre'
import { biomeGrid, buildTerrain, classify, isWater } from '..'
import { SWEEP_SEEDS } from '../../__tests__/seeds'

describe('biomes', () => {
  it('classifies every cell deterministically and shows variety outdoors', () => {
    const field = buildTerrain({ seed: 1, tuning: uniformTerrain(tuning.terrain) })
    const first = biomeGrid(field, uniformTerrain(tuning.terrain), biomeSets.park)
    const again = biomeGrid(field, uniformTerrain(tuning.terrain), biomeSets.park)
    expect(first).toEqual(again)
    expect(new Set(first).size).toBeGreaterThanOrEqual(3)
    expect([...first].every((index) => index < biomeSets.park.biomes.length)).toBe(true)
  })

  it('uses office floor inside the centre and park biomes outside it', () => {
    const state = makeWorld({ teams: 2, agents: 6, ...{ scene: 'office', seed: 99 }, treeVariants: 3 })
    const set = sceneBiomes(scenes.office)
    const region = regionForBounds(scenes.office, state.bounds)
    if (!region) throw new Error('office centre is missing')
    const outside = new Set<string>()
    state.biomes.forEach((index, cell) => {
      const x = state.terrain.originX + (cell % state.terrain.cols) * state.terrain.cellSize
      const z = state.terrain.originZ + Math.floor(cell / state.terrain.cols) * state.terrain.cellSize
      const id = set.biomes[index]?.id
      if (!id) throw new Error('generated biome index has no palette entry')
      if (centreWeight(region, x, z) === 1) expect(id).toBe('floor')
      else { expect(id).not.toBe('floor'); outside.add(id) }
    })
    expect(outside.size).toBeGreaterThan(2)
  })

  it.each(SWEEP_SEEDS)('assigns every submerged cell to water for seed %i', (seed) => {
    const field = buildTerrain({ seed, tuning: uniformTerrain(tuning.terrain) })
    let wet = 0
    for (const biomeSet of Object.values(biomeSets)) {
      for (let row = 0; row < field.rows; row += 1) for (let col = 0; col < field.cols; col += 1) {
        const x = field.originX + col * field.cellSize
        const z = field.originZ + row * field.cellSize
        if (!isWater(field, uniformTerrain(tuning.terrain), x, z)) continue
        wet += 1
        expect(classify(field, uniformTerrain(tuning.terrain), biomeSet, x, z)).toBe('water')
      }
    }
    expect(wet).toBeGreaterThan(0)
  })

  it('keeps the water rule empty and follows a changed water level', () => {
    const field = buildTerrain({ seed: 1, tuning: uniformTerrain(tuning.terrain) })
    for (const biomeSet of Object.values(biomeSets)) {
      const water = biomeSet.biomes.find((biome) => biome.id === 'water')
      expect(water?.vegetation).toEqual({})
      expect(water?.decor).toEqual({})
      expect(classify(field, uniformTerrain({ ...tuning.terrain, waterLevel: 10 }), biomeSet, 0, 0)).toBe('water')
      expect(classify(field, uniformTerrain({ ...tuning.terrain, waterLevel: -10 }), biomeSet, 0, 0)).not.toBe('water')
    }
  })
})
