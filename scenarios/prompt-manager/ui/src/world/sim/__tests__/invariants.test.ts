import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { checkWorldInvariants, checkSeats, checkSeparation } from '../invariants'
import { roomId } from '../layout/generate'
import type { Signal } from '../model'
import { NOW, makeInput, run, world } from './fixtures'
import { createWorld } from '../world'

const MINUTES = 60 / tuning.sim.tickSeconds

function violations(state: ReturnType<typeof world>) {
  return checkWorldInvariants(state, tuning)
}

describe('world invariants', () => {
  it('hold on a fresh world of every size, including an empty team graph', () => {
    for (const [teams, members] of [[0, 0], [1, 1], [3, 4], [6, 5], [9, 8]] as const) {
      expect(violations(world(teams, members))).toEqual([])
    }
  })

  it('hold after five simulated minutes of idle life', () => {
    const s = run(world(4, 5), MINUTES * 5)
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
    let s = world(3, 4)
    for (let tick = 0; tick < MINUTES * 4; tick += 1) {
      s = run(s, 1, { 0: script[tick] ?? [] })
      // Seats and bounds must hold every tick; placement rules settle once walks end.
      expect(checkSeats(s)).toEqual([])
    }
    expect(violations(s)).toEqual([])
  })

  it('hold after a room is removed and its members are sent home', () => {
    const state = createWorld(makeInput(3, 3, { overrides: [{ placeId: roomId('team-0'), removed: true }] }), tuning, 0)
    expect(violations(state)).toEqual([])
    expect(violations(run(state, MINUTES * 3))).toEqual([])
  })

  it('name the offending actors when two stand on top of each other', () => {
    const s = world(1, 2)
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
    const s = world(1, 2)
    const a = s.actors['agent-0-0']
    const b = s.actors['agent-0-1']
    if (!a || !b || !a.seatId) throw new Error('missing actors')
    b.seatId = a.seatId
    expect(checkSeats(s).map((v) => v.rule)).toContain('seat-occupancy')
  })
})
