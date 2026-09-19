import { describe, expect, it } from 'vitest'
import { uniformTerrain, tuning, SCENE_IDS } from '../../config'
import { checkWorldInvariants, checkSeats, checkSeparation, checkVegetationDry } from '../invariants'
import { SWEEP_SEEDS } from './seeds'
import { isWater } from '../terrain/water'
import { roomId } from '../layout/generate'
import type { Signal } from '../model'
import { NOW, makeWorld, run } from './fixtures'

const MINUTES = 60 / tuning.sim.tickSeconds

function violations(state: ReturnType<typeof makeWorld>) {
  return checkWorldInvariants(state, tuning)
}

describe('world invariants', () => {
  it.each(SCENE_IDS.flatMap((scene) => SWEEP_SEEDS.map((seed) => [scene, seed] as const)))('satisfies every invariant in %s for seed %i', (scene, seed) => {
    const state = makeWorld({ teams: 5, agents: 25, scene, seed })
    expect(checkWorldInvariants(state, tuning)).toEqual([])
  })

  it('detects submerged decor and an enlarged shore exclusion band', () => {
    const state = makeWorld({ teams: 5, agents: 25, ...{ seed: 1 }, treeVariants: 3 })
    expect(state.decor.length).toBeGreaterThan(0)
    expect(checkVegetationDry(state, uniformTerrain(tuning.terrain), { ...tuning.layout, shoreClearance: 40 }).length).toBeGreaterThan(0)
    const spot = state.decor[0]
    if (!spot) throw new Error('fixture has no decor to submerge')
    let found = false
    for (let row = 0; row < state.terrain.rows && !found; row += 1) for (let col = 0; col < state.terrain.cols; col += 1) {
      const x = state.terrain.originX + col * state.terrain.cellSize
      const z = state.terrain.originZ + row * state.terrain.cellSize
      if (!isWater(state.terrain, uniformTerrain(tuning.terrain), x, z)) continue
      spot.position = [x, z]
      found = true
      break
    }
    expect(found).toBe(true)
    expect(checkVegetationDry(state, uniformTerrain(tuning.terrain), tuning.layout).map((v) => v.ids)).toContainEqual([spot.id])
    expect(checkWorldInvariants(state, tuning).map((v) => v.rule)).toContain('vegetation-dry')
  })

  it('hold on a fresh world of every size, including an empty team graph', () => {
    for (const [teams, members] of [[0, 0], [1, 1], [3, 4], [6, 5], [9, 8]] as const) {
      expect(violations(makeWorld({ teams: teams, agents: (teams) * (members), treeVariants: 3 }))).toEqual([])
    }
  })

  it('hold for the 25-actor smoke roster at seed 1', () => {
    const state = makeWorld({ teams: 5, agents: 25, seed: 1 })
    expect(violations(state)).toEqual([])
    expect(violations(run(state, MINUTES))).toEqual([])
  })

  it('hold after five simulated minutes of idle life', () => {
    const s = run(makeWorld({ teams: 4, agents: 20, treeVariants: 3 }), MINUTES * 5)
    expect(violations(s)).toEqual([])
  })

  it('hold through a run, a failure, a gathering and their returns', () => {
    const script: Record<number, Signal[]> = {
      0: [{ kind: 'run.started', agentId: 'agent-0-1', runId: 'r1', at: NOW }],
      300: [{ kind: 'run.failed', agentId: 'agent-0-1', runId: 'r1', error: 'boom', at: NOW + 30 }],
      400: [{ kind: 'heartbeat.upcoming', teamId: 'team-1', scheduledAt: NOW + 60, at: NOW + 40 }],
      900: [{ kind: 'failed.acknowledged', agentId: 'agent-0-1', at: NOW + 90 }],
      1200: [{ kind: 'heartbeat.cancelled', teamId: 'team-1', at: NOW + 120 }],
    }
    let s = makeWorld({ teams: 3, agents: 12, treeVariants: 3 })
    for (let tick = 0; tick < MINUTES * 4; tick += 1) {
      s = run(s, 1, { 0: script[tick] ?? [] })
      // Seats and bounds must hold every tick; placement rules settle once walks end.
      expect(checkSeats(s)).toEqual([])
    }
    expect(violations(s)).toEqual([])
  })

  it('hold after a room is removed and its members are sent home', () => {
    const state = makeWorld({ teams: 3, agents: 9, overrides: [{ placeId: roomId('team-0'), removed: true }] })
    expect(violations(state)).toEqual([])
    expect(violations(run(state, MINUTES * 3))).toEqual([])
  })

  it('name the offending actors when two stand on top of each other', () => {
    const s = makeWorld({ teams: 1, agents: 2, treeVariants: 3 })
    const a = s.actors['agent-0-0']
    const b = s.actors['agent-0-1']
    if (!a || !b) throw new Error('missing actors')
    b.position = [a.position[0] + 0.1, a.position[1]]
    const found = checkSeparation(s, tuning)
    expect(found).toHaveLength(1)
    expect(found[0]?.rule).toBe('separation')
    expect(found[0]?.ids).toEqual(['agent-0-0', 'agent-0-1'])
  })

  it('report a seat held by one actor but pointed at by another', () => {
    const s = makeWorld({ teams: 1, agents: 2, treeVariants: 3 })
    const a = s.actors['agent-0-0']
    const b = s.actors['agent-0-1']
    if (!a || !b || !a.seatId) throw new Error('missing actors')
    b.seatId = a.seatId
    expect(checkSeats(s).map((v) => v.rule)).toContain('seat-occupancy')
  })
})
