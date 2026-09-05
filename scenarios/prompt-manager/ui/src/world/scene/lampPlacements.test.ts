import { describe, expect, it } from 'vitest'
import { scenes, tuning } from '../config'
import { heightAt, type Place } from '../sim'
import { makeWorld } from '../sim/__tests__/fixtures'
import { lampPlacements } from './lampPlacements'

describe('lamp placements', () => {
  it('places room and corridor lamps on the raised office ground', () => {
    const state = makeWorld({ teams: 5, agents: 25, ...{ scene: 'office', seed: 1 }, treeVariants: 3 })
    const places = state.placeOrder.map((id) => state.places[id]).filter((place): place is Place => Boolean(place))
    const lamps = lampPlacements(places, state.seed, state.terrain, tuning.layout, scenes.office.props.filler.length)
    expect(lamps.some((lamp) => lamp.key.startsWith('corridor:'))).toBe(true)
    for (const lamp of lamps) {
      expect(lamp.y).toBe(heightAt(state.terrain, ...lamp.position))
      expect(lamp.y).toBeGreaterThan(0)
    }
  })

  it('uses configured corridor spacing and scale without changing default distribution', () => {
    const state = makeWorld({ teams: 2, agents: 6, ...{ scene: 'office' }, treeVariants: 3 })
    const corridor: Place = { id: 'corridor:test', kind: 'corridor', position: [0, 0], rotation: 0, size: [40, 4], seats: [], label: 'Test' }
    const initial = lampPlacements([corridor], 1, state.terrain, tuning.layout, 0)
    expect(initial.map((lamp) => lamp.position)).toEqual([[-15, 0], [-5, 0], [5, 0], [15, 0]])
    expect(initial.every((lamp) => lamp.scale === 0.8)).toBe(true)
    const changed = lampPlacements([corridor], 1, state.terrain, { ...tuning.layout, corridorLampSpacing: 20, corridorLampScale: 0.6 }, 0)
    expect(changed.map((lamp) => lamp.position)).toEqual([[-10, 0], [10, 0]])
    expect(changed.every((lamp) => lamp.scale === 0.6)).toBe(true)
  })
})
