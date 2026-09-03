import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { hashState } from '../hash'
import type { Signal } from '../model'
import { NOW, run, world } from './fixtures'

function script(): Record<number, Signal[]> {
  return {
    5: [{ kind: 'run.started', agentId: 'agent-0-0', runId: 'r1', at: NOW }],
    400: [{ kind: 'heartbeat.upcoming', teamId: 'team-1', scheduledAt: NOW + 60, at: NOW + 40 }],
    900: [{ kind: 'run.failed', agentId: 'agent-0-0', runId: 'r1', error: 'boom', at: NOW + 90 }],
    1500: [{ kind: 'agent.message', agentId: 'agent-1-1', message: 'hi', at: NOW + 150 }],
    3000: [{ kind: 'heartbeat.cancelled', teamId: 'team-1', at: NOW + 300 }],
    5000: [{ kind: 'run.started', agentId: 'agent-0-0', runId: 'r2', at: NOW + 500 }],
    7000: [{ kind: 'run.finished', agentId: 'agent-0-0', runId: 'r2', at: NOW + 700 }],
  }
}

describe('determinism', () => {
  it('two runs from seed 7 and the same signal script hash identically after 10,000 ticks', () => {
    const a = run(world(3, 4), 10_000, script())
    const b = run(world(3, 4), 10_000, script())
    expect(hashState(a)).toBe(hashState(b))
    expect(a.tick).toBe(10_000)
  })

  it('a different seed diverges', () => {
    const a = run(world(3, 4), 2_000, script())
    const b = run(world(3, 4, { seed: 8 }), 2_000, script())
    expect(hashState(a)).not.toBe(hashState(b))
  })

  it('step never mutates its input', () => {
    const start = world(2, 2)
    const snapshot = JSON.stringify({ actors: start.actors, occupancy: start.occupancy, events: start.events, rng: start.rngState })
    run(start, 500, script())
    expect(JSON.stringify({ actors: start.actors, occupancy: start.occupancy, events: start.events, rng: start.rngState })).toBe(snapshot)
  })

  it('actors never leave the slab over 5,000 ticks with 50 actors', () => {
    let state = world(5, 10)
    const half = { w: state.bounds.width / 2, d: state.bounds.depth / 2 }
    const [cx, cz] = state.bounds.center
    for (let i = 0; i < 5_000; i += 1) {
      state = run(state, 1, i % 700 === 0 ? { 0: [{ kind: 'run.started', agentId: `agent-${(i / 700) % 5}-1`, runId: `r${i}`, at: NOW + i }] } : {})
      for (const id of state.actorOrder) {
        const a = state.actors[id]
        if (!a) continue
        expect(Math.abs(a.position[0] - cx)).toBeLessThanOrEqual(half.w)
        expect(Math.abs(a.position[1] - cz)).toBeLessThanOrEqual(half.d)
      }
    }
  })

  it('the tick count and time advance by the tuning tick', () => {
    const s = run(world(1, 1), 10)
    expect(s.time).toBeCloseTo(NOW + tuning.sim.tickSeconds * 10, 4)
  })
})
