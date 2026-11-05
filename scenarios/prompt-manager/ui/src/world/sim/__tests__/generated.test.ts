import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { createLiveWorld, createWorld, generateWorld } from '../world'
import { createWorldStore } from '../store'
import { hashState } from '../hash'
import { makeWorldInput } from './fixtures'
import { checkSeats, checkRestingInPlace, checkSeparation } from '../invariants'

describe('generated and live world ownership', () => {
  it.each(['park', 'office'] as const)('keeps resting occupants and new arrivals in valid seats after %s roster growth', scene => {
    const input = makeWorldInput({ scene, teams: 4, agents: 25 })
    const store = createWorldStore(input, tuning)
    const next = makeWorldInput({ scene, teams: 4, agents: 26 })
    store.adoptGenerated(generateWorld(next, tuning), next)
    const state = store.getState()
    expect(checkSeats(state)).toEqual([])
    expect(checkRestingInPlace(state, tuning.layout)).toEqual([])
    expect(checkSeparation(state, tuning)).toEqual([])
    expect(state.time).toBe(input.now)
  })
  it('removes vanished actors, seat reservations and team gatherings when adopting a roster', () => {
    const input = makeWorldInput({ teams: 2, agents: 4 })
    const store = createWorldStore(input, tuning)
    store.getState().gatherings['team-1'] = { teamId: 'team-1', scheduledAt: input.now, until: input.now + 100 }
    const nextInput = makeWorldInput({ teams: 1, agents: 2 })
    store.adoptGenerated(generateWorld(nextInput, tuning), nextInput)
    expect(store.getState().actorOrder).toEqual(nextInput.agents.map(agent => agent.id))
    expect(store.getState().actors['agent-1-0']).toBeUndefined()
    expect(Object.values(store.getState().occupancy)).not.toContain('agent-1-0')
    expect(store.getState().gatherings['team-1']).toBeUndefined()
  })

  it('adopts regenerated geometry without losing live runs, pending signals or the simulation clock', () => {
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const store = createWorldStore(input, tuning)
    const first = input.agents[0]?.id
    const second = input.agents[1]?.id
    if (!first || !second) throw new Error('Missing fixture actors')
    store.dispatch([{ kind: 'run.started', agentId: first, runId: 'running', at: input.now }])
    store.advance(tuning.sim.tickSeconds)
    const time = store.getState().time
    const tick = store.getState().tick
    store.dispatch([{ kind: 'run.started', agentId: second, runId: 'queued', at: time }])
    const nextInput = makeWorldInput({ teams: 1, agents: 3 })
    const generated = generateWorld(nextInput, tuning)
    store.adoptGenerated(generated, nextInput)
    expect(store.getState().terrain).toBe(generated.terrain)
    expect(store.getState().waterGeometry).toBe(generated.waterGeometry)
    expect(store.getState().actors[first]?.runId).toBe('running')
    expect(store.getState().actorOrder).toHaveLength(3)
    expect(store.getState().time).toBe(time)
    expect(store.getState().tick).toBe(tick)
    store.advance(tuning.sim.tickSeconds)
    expect(store.getState().actors[second]?.runId).toBe('queued')
  })

  it.each(['park', 'office'] as const)('preserves synchronous behavior for %s', scene => {
    const input = makeWorldInput({ scene, teams: 1, agents: 2 })
    const generated = generateWorld(input, tuning)
    const live = createLiveWorld(generated, input, tuning)
    expect(hashState(live)).toBe(hashState(createWorld(input, tuning)))
    expect(live.terrain).toBe(generated.terrain)
    expect(live.nav).toBe(generated.nav)
    expect('actors' in generated).toBe(false)
    expect('weather' in generated).toBe(false)
    expect('time' in generated).toBe(false)
  })
  it('validates topology and overlays current labels when reusing generated data', () => {
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const generated = generateWorld(input, tuning)
    const renamed = { ...input, teams: input.teams.map(team => ({ ...team, name: 'Renamed team' })), agents: input.agents.map(agent => ({ ...agent, name: 'Renamed agent' })) }
    const before = JSON.stringify(generated)
    const live = createLiveWorld(generated, renamed, tuning)
    expect(Object.values(live.places).filter(place => place.kind === 'room').every(place => place.label === 'Renamed team')).toBe(true)
    expect(JSON.stringify(generated)).toBe(before)
    expect(() => createLiveWorld(generated, { ...input, teams: [] }, tuning)).toThrow(/topology/)
    expect(() => createLiveWorld(generated, { ...input, seed: input.seed + 1 }, tuning)).toThrow(/seed/)
  })

  it('shares spatial output while independent live stores leave it unchanged', () => {
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const generated = generateWorld(input, tuning)
    const before = JSON.stringify(generated)
    const one = createWorldStore(input, tuning, 0, generated)
    const two = createWorldStore({ ...input, now: input.now + 100 }, tuning, 0, generated)
    expect(one.getState().actors).not.toBe(two.getState().actors)
    expect(one.getState().occupancy).not.toBe(two.getState().occupancy)
    expect(one.getState().weather).not.toBe(two.getState().weather)
    const agent = input.agents[0]
    if (!agent) throw new Error('Missing fixture agent')
    one.dispatch([{ kind: 'run.started', agentId: agent.id, runId: 'only-one', at: input.now }])
    for (let i = 0; i < 60; i++) one.advance(tuning.sim.tickSeconds)
    one.updatePresentation(input.teams, input.agents.map(value => ({ ...value, name: 'Renamed' })))
    expect(two.getState().actors[agent.id]?.runId).toBeUndefined()
    expect(JSON.stringify(generated)).toBe(before)
  })
})
