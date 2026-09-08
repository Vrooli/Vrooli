import { describe, expect, it } from 'vitest'
import { tuning } from '../../../config'
import { checkWorldInvariants } from '../../invariants'
import { makeWorld } from '../../__tests__/fixtures'
import { runCooperatively } from '../../cooperative'
import { Rng } from '../../rng'
import { floorplateSteps } from './plate'
import { assignRoomsSteps } from './assign'

describe('floorplan strategy', () => {
  it('assigns largest demand first and breaks ties by identity', async () => {
    const teams = [
      { id: 'z', name: 'First name', memberIds: ['a'] },
      { id: 'b', name: 'Last name', memberIds: ['b', 'c'] },
      { id: 'a', name: 'Renamable', memberIds: ['d'] },
    ]
    const leaves = [30, 20, 10].map(width => ({ x: 0, z: 0, width, depth: 10, side: 'north' as const }))
    const result = await runCooperatively(assignRoomsSteps(teams, leaves))
    expect(result.map(({ team, leaf }) => [team.id, leaf.width])).toEqual([['b', 30], ['a', 20], ['z', 10]])
    expect(teams.map(team => team.id)).toEqual(['z', 'b', 'a'])
  })

  it('cancels floor sizing during its scan before consuming the aspect random draw', async () => {
    const sizes = Array.from({ length: 1024 }, () => [20, 15] as const)
    const rng = new Rng(7)
    const controller = new AbortController()
    const steps = floorplateSteps(sizes, 1000, tuning.layout, rng)
    await expect(runCooperatively(steps, {
      signal: controller.signal,
      onProgress: progress => { if (progress.completed === 128) controller.abort(new Error('cancel sizing')) },
    })).rejects.toThrow('cancel sizing')
    expect(rng.state).toBe(7)
    expect(steps.next().done).toBe(true)
  })

  it('is deterministic and produces connected invariant-safe office state', () => {
    const agents = Array.from({ length: 6 }, (_, index) => ({ id: `agent-${index}`, name: `Agent ${index}` }))
    const roster = {
      agents,
      teams: [
        { id: 'alpha', name: 'Alpha', memberIds: agents.slice(0, 3).map((agent) => agent.id) },
        { id: 'beta', name: 'Beta', memberIds: agents.slice(3).map((agent) => agent.id) },
      ],
    }
    const input = { ...roster, seed: 7, now: 1, scene: 'office' as const }
    const first = makeWorld(input)
    const again = makeWorld(input)
    expect(first.placeOrder.map((id) => first.places[id])).toEqual(again.placeOrder.map((id) => again.places[id]))
    const corridors = first.placeOrder.filter((id) => first.places[id]?.kind === 'corridor')
    expect(corridors.length).toBeGreaterThanOrEqual(1 + tuning.layout.floorplan.secondaryCorridors.min)
    expect(corridors.length).toBeLessThanOrEqual(1 + tuning.layout.floorplan.secondaryCorridors.max)
    expect(first.placeOrder.filter((id) => first.places[id]?.kind === 'door')).toHaveLength(roster.teams.length)
    expect(checkWorldInvariants(first, tuning)).toEqual([])
  })

  it('is stable across team rename and changes when team identity changes', () => {
    const agents = [{ id: 'a', name: 'A' }, { id: 'b', name: 'B' }]
    const team = { id: 'alpha', name: 'Alpha', memberIds: ['a', 'b'] }
    const original = { seed: 9 * 9 + 9 * 2, now: 1, scene: 'office' as const, agents, teams: [team] }
    const renamed = { ...original, teams: [{ ...team, name: 'Renamed' }] }
    const reidentified = { ...original, teams: [{ ...team, id: 'beta' }] }
    const geometry = (input: typeof original) => {
      const state = makeWorld(input)
      return state.placeOrder.map((id) => state.places[id]).map((place) => place ? { ...place, label: '' } : place)
    }
    expect(geometry(renamed)).toEqual(geometry(original))
    expect(geometry(reidentified)).not.toEqual(geometry(original))
  })

  it('keeps every seeded actor-count case invariant-safe with one door per room', () => {
    for (const seed of [1, 7, 9 * 9 + 9 * 2, 12_345]) {
      for (const actorCount of [5 * 5, (5 + 5) * (5 + 5), 4 * (5 + 5) * (5 + 5), (5 + 5) * (5 + 5) * (5 + 5)]) {
        const agents = Array.from({ length: actorCount }, (_, index) => ({ id: `agent-${index}`, name: `Agent ${index}` }))
        const teams = Array.from({ length: 5 }, (_, teamIndex) => ({
          id: `team-${teamIndex}`,
          name: `Team ${teamIndex}`,
          memberIds: agents.filter((_, index) => index % 5 === teamIndex).map((agent) => agent.id),
        }))
        const state = makeWorld({ agents, teams, seed, now: 1, scene: 'office' })
        const rooms = state.placeOrder.map((id) => state.places[id]).filter((place) => place?.kind === 'room')
        const doors = state.placeOrder.map((id) => state.places[id]).filter((place) => place?.kind === 'door')
        expect(doors).toHaveLength(rooms.length)
        expect(checkWorldInvariants(state, tuning)).toEqual([])
      }
    }
  }, 60_000)
})
