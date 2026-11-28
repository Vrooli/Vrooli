import { describe, expect, it } from 'vitest'
import { tuning } from '../../../config'
import { smoothPressure, weatherPressure } from '..'
import { makeWorld } from '../../__tests__/fixtures'

describe('weather pressure', () => {
  it('maps no failures to zero and all failures to one when all terms agree', () => {
    const state = makeWorld({ teams: 1, agents: 4, treeVariants: 3 })
    expect(weatherPressure(state, tuning.weather)).toBe(0)
    state.events = state.actorOrder.map((id, index) => ({ seq: index, at: state.time, kind: 'run.failed' as const, agentId: id }))
    state.actorOrder.forEach((id) => { const actor = state.actors[id]; if (actor) actor.state = 'failed' })
    state.gatherings = { 'team-0': { teamId: 'team-0', scheduledAt: state.time - 10, until: state.time - 1 } }
    expect(weatherPressure(state, tuning.weather)).toBe(1)
  })

  it('smooths a spike instead of applying it immediately', () => {
    const value = smoothPressure(0, 1, tuning.sim.tickSeconds, tuning.weather)
    expect(value).toBeGreaterThan(0)
    expect(value).toBeLessThan(1)
    expect(value).toBe(smoothPressure(0, 1, tuning.sim.tickSeconds, tuning.weather))
  })
})
