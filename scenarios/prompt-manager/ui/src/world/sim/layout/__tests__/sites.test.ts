import { describe, expect, it } from 'vitest'
import { uniformTerrain, tuning } from '../../../config'
import { isWater, buildTerrain, slopeAt } from '../../terrain'
import { selectSites } from '../sites'

const sizes = [[8, 6], [10, 6], [9, 6], [12, 6]] as const

describe('site selection', () => {
  it('is deterministic and seed-sensitive', () => {
    const field = buildTerrain({ seed: 1, tuning: uniformTerrain(tuning.terrain) })
    const first = selectSites(field, { layout: tuning.layout, terrain: uniformTerrain(tuning.terrain) }, sizes, 1)
    const again = selectSites(field, { layout: tuning.layout, terrain: uniformTerrain(tuning.terrain) }, sizes, 1)
    expect(first).toEqual(again)
    const otherField = buildTerrain({ seed: 99, tuning: uniformTerrain(tuning.terrain) })
    expect(selectSites(otherField, { layout: tuning.layout, terrain: uniformTerrain(tuning.terrain) }, sizes, 99).sites.map((site) => site.position)).not.toEqual(first.sites.map((site) => site.position))
  })

  it('selects dry, flat, disjoint sites with 15-degree rotations', () => {
    const field = buildTerrain({ seed: 7, tuning: uniformTerrain(tuning.terrain) })
    const result = selectSites(field, { layout: tuning.layout, terrain: uniformTerrain(tuning.terrain) }, sizes, 7)
    result.sites.forEach((site, index) => {
      expect(isWater(field, uniformTerrain(tuning.terrain), site.position[0], site.position[1])).toBe(false)
      expect(slopeAt(field, site.position[0], site.position[1])).toBeLessThanOrEqual(tuning.terrain.maxSiteSlope)
      expect(site.rotation / (Math.PI / 12)).toBeCloseTo(Math.round(site.rotation / (Math.PI / 12)), 8)
      result.sites.slice(index + 1).forEach((other) => {
        expect(Math.hypot(site.position[0] - other.position[0], site.position[1] - other.position[1])).toBeGreaterThanOrEqual(tuning.layout.siteSpacing)
      })
    })
    expect(new Set(result.sites.map((site) => site.rotation)).size).toBeGreaterThan(1)
  })
})
