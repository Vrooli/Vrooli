import { describe, expect, it } from 'vitest'
import { syntheticRoster } from '../synthetic'
import { tuning } from '../../config'
import { checkWorldInvariants, createWorld } from '../../sim'

describe('syntheticRoster', () => {
  it('builds stable teams and agents for a count', () => {
    const a = syntheticRoster(12, 5)
    const b = syntheticRoster(12, 5)
    expect(a).toEqual(b)
    expect(a.agents).toHaveLength(12)
    expect(a.teams).toHaveLength(3)
    expect(a.teams[2]?.memberIds).toHaveLength(2)
    expect(a.agents.every((x) => x.colors?.body)).toBe(true)
  })

  it('builds an invariant-safe 1000-actor performance world with four sites', () => {
    const roster = syntheticRoster(1000, 250, 1)
    const state = createWorld({ ...roster, seed: 1, now: 1_700_000_000, scene: 'park' }, tuning, 0)
    expect(Object.values(state.places).filter((place) => place.kind === 'room')).toHaveLength(4)
    expect(checkWorldInvariants(state, tuning)).toEqual([])
  })

  it.each([
    { actors: 100, perTeam: 25, seed: 99 },
    { actors: 400, perTeam: 100, seed: 1 },
    { actors: 1000, perTeam: 250, seed: 7 },
    { actors: 1000, perTeam: 250, seed: 99 },
  ])('keeps every large rotated site level for $actors actors at seed $seed', ({ actors, perTeam, seed }) => {
    const roster = syntheticRoster(actors, perTeam, seed)
    const state = createWorld({ ...roster, seed, now: 1_700_000_000, scene: 'park' }, tuning, 0)
    expect(checkWorldInvariants(state, tuning).filter((violation) => violation.rule === 'site-level')).toEqual([])
  })

  it('keeps the runs board on the dry commons terrace', () => {
    const roster = syntheticRoster(25, 5, 12345)
    const state = createWorld({ ...roster, seed: 12345, now: 1_700_000_000, scene: 'park' }, tuning, 0)
    expect(checkWorldInvariants(state, tuning).filter((violation) => violation.ids.includes('board'))).toEqual([])
  })
})
