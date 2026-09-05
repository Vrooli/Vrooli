import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { hashState } from '../hash'
import type { Signal, WorldState } from '../model'
import { NOW, awayFromHome, quietTuning, run, makeWorld } from './fixtures'

const T = quietTuning()
const A = 'agent-0-0'

function until(state: WorldState, predicate: (s: WorldState) => boolean, maxTicks = 3000, signals: Record<number, Signal[]> = {}): WorldState {
  let s = state
  for (let i = 0; i < maxTicks; i += 1) {
    if (predicate(s)) return s
    s = run(s, 1, { 0: signals[i] ?? [] }, T)
  }
  return s
}

const actor = (s: WorldState) => {
  const a = s.actors[A]
  if (!a) throw new Error('missing actor')
  return a
}

describe('actor state machine', () => {
  it('Idle → WalkingToDesk on run.started when away from the desk', () => {
    const s = run(awayFromHome(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), A), 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T)
    expect(actor(s).state).toBe('walkingToDesk')
    expect(actor(s).path.length).toBeGreaterThan(0)
    expect(actor(s).hurrying).toBe(true)
  })

  it('Idle → Working at once on run.started when already at the desk', () => {
    const s = run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T)
    expect(actor(s).state).toBe('working')
    expect(actor(s).path).toEqual([])
    expect(actor(s).seatId).toBe(actor(s).deskSeatId)
  })

  it('WalkingToDesk → Working on arrival at the desk seat', () => {
    let s = run(awayFromHome(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), A), 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T)
    s = until(s, (x) => actor(x).state === 'working')
    expect(actor(s).state).toBe('working')
    const seat = s.seats[actor(s).seatId ?? '']
    expect(seat).toBeDefined()
    expect(actor(s).position[0]).toBeCloseTo(seat?.position[0] ?? NaN, 3)
    expect(actor(s).facing).toBeCloseTo(seat?.facing ?? NaN, 6)
  })

  it('Working → Idle on run.finished', () => {
    let s = until(run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T), (x) => actor(x).state === 'working')
    s = run(s, 1, { 0: [{ kind: 'run.finished', agentId: A, runId: 'r', at: NOW }] }, T)
    expect(actor(s).state).toBe('idle')
    expect(actor(s).lastRun?.status).toBe('completed')
  })

  it('Working → Failed on run.failed with the error kept', () => {
    let s = until(run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T), (x) => actor(x).state === 'working')
    s = run(s, 1, { 0: [{ kind: 'run.failed', agentId: A, runId: 'r', error: 'exit 1', at: NOW }] }, T)
    expect(actor(s).state).toBe('failed')
    expect(actor(s).failedError).toBe('exit 1')
    expect(actor(s).lastRun?.status).toBe('failed')
  })

  it('Failed → Idle on failed.acknowledged', () => {
    let s = until(run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T), (x) => actor(x).state === 'working')
    s = run(s, 1, { 0: [{ kind: 'run.failed', agentId: A, runId: 'r', error: 'x', at: NOW }] }, T)
    s = run(s, 1, { 0: [{ kind: 'failed.acknowledged', agentId: A, at: NOW }] }, T)
    expect(actor(s).state).toBe('idle')
    expect(actor(s).failedError).toBeUndefined()
  })

  it('Failed → Idle after failedAckSeconds on its own', () => {
    let s = until(run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T), (x) => actor(x).state === 'working')
    s = run(s, 1, { 0: [{ kind: 'run.failed', agentId: A, runId: 'r', error: 'x', at: NOW }] }, T)
    const ticks = Math.ceil(T.sim.failedAckSeconds / T.sim.tickSeconds) + 2
    s = run(s, ticks, {}, T)
    expect(actor(s).state).toBe('idle')
  })

  it('Failed → WalkingToDesk on the next run.started', () => {
    let s = until(run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T), (x) => actor(x).state === 'working')
    s = run(s, 1, { 0: [{ kind: 'run.failed', agentId: A, runId: 'r', error: 'x', at: NOW }] }, T)
    s = run(s, 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r2', at: NOW }] }, T)
    expect(['walkingToDesk', 'working']).toContain(actor(s).state)
    expect(actor(s).runId).toBe('r2')
  })

  it('Idle → WalkingToTable when a heartbeat is within the gather lead', () => {
    const at = NOW + T.sim.gatherLeadSeconds - 1
    const s = run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 2, { 0: [{ kind: 'heartbeat.upcoming', teamId: 'team-0', scheduledAt: at, at: NOW }] }, T)
    expect(actor(s).state).toBe('walkingToTable')
  })

  it('Idle stays Idle when the heartbeat is beyond the gather lead', () => {
    const at = NOW + T.sim.gatherLeadSeconds + 100
    const s = run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 2, { 0: [{ kind: 'heartbeat.upcoming', teamId: 'team-0', scheduledAt: at, at: NOW }] }, T)
    expect(actor(s).state).toBe('idle')
  })

  it('WalkingToTable → Gathered on arrival, seated at the team table', () => {
    let s = run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'heartbeat.upcoming', teamId: 'team-0', scheduledAt: NOW + 10, at: NOW }] }, T)
    s = until(s, (x) => actor(x).state === 'gathered')
    expect(actor(s).state).toBe('gathered')
    expect(actor(s).seatId?.startsWith('seat:table:team-0')).toBe(true)
    expect(actor(s).anim.seated).toBe(true)
  })

  it('Gathered → WalkingToDesk on run.started', () => {
    let s = run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'heartbeat.upcoming', teamId: 'team-0', scheduledAt: NOW + 10, at: NOW }] }, T)
    s = until(s, (x) => actor(x).state === 'gathered')
    s = run(s, 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T)
    expect(actor(s).state).toBe('walkingToDesk')
    expect(actor(s).anim.seated).toBe(false)
  })

  it('Gathered → Idle on heartbeat.cancelled', () => {
    let s = run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'heartbeat.upcoming', teamId: 'team-0', scheduledAt: NOW + 10, at: NOW }] }, T)
    s = until(s, (x) => actor(x).state === 'gathered')
    s = run(s, 1, { 0: [{ kind: 'heartbeat.cancelled', teamId: 'team-0', at: NOW }] }, T)
    expect(actor(s).state).toBe('idle')
    expect(Object.keys(s.occupancy).filter((seat) => seat.includes('table'))).toHaveLength(0)
  })

  it('Gathered → Idle once the gather window passes', () => {
    let s = run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'heartbeat.upcoming', teamId: 'team-0', scheduledAt: NOW + 1, at: NOW }] }, T)
    s = until(s, (x) => actor(x).state === 'gathered')
    s = run(s, Math.ceil(T.sim.gatherWindowSeconds / T.sim.tickSeconds) + 5, {}, T)
    expect(actor(s).state).toBe('idle')
  })

  it('Idle → Socializing → Idle through the idle roll and duration', () => {
    const social = { ...tuning, sim: { ...tuning.sim, idle: { ...tuning.sim.idle, weights: { rest: 0, wander: 0, socialize: 100, sit: 0 }, maxMoversRatio: 1 } } }
    let s = makeWorld({ teams: 1, agents: 4, treeVariants: 3 })
    let sawSocial = false
    for (let i = 0; i < 4000 && !sawSocial; i += 1) {
      s = run(s, 1, {}, social)
      sawSocial = Object.values(s.actors).some((a) => a.state === 'socializing')
    }
    expect(sawSocial).toBe(true)
    let backToIdle = false
    for (let i = 0; i < 4000 && !backToIdle; i += 1) {
      s = run(s, 1, {}, { ...social, sim: { ...social.sim, idle: { ...social.sim.idle, weights: { rest: 100, wander: 0, socialize: 0, sit: 0 } } } })
      backToIdle = Object.values(s.actors).every((a) => a.state === 'idle')
    }
    expect(backToIdle).toBe(true)
  })

  it('agent.message stores the bubble and emote; unknown agents are ignored', () => {
    const s = run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [
      { kind: 'agent.message', agentId: A, message: 'hello', at: NOW },
      { kind: 'agent.message', agentId: 'nobody', message: 'x', at: NOW },
    ] }, T)
    expect(actor(s).message?.text).toBe('hello')
    expect(actor(s).anim.emote?.kind).toBe('message')
    expect(s.events.filter((e) => e.kind === 'agent.message')).toHaveLength(1)
  })

  it('an unassigned agent works in place with no desk', () => {
    const s = run(makeWorld({ teams: [], agents: [{ id: 'solo', name: 'Solo' }], treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: 'solo', runId: 'r', at: NOW }] }, T)
    expect(s.actors.solo?.state).toBe('working')
  })

  it('every discrete change bumps the revision and lands in the ring', () => {
    const start = makeWorld({ teams: 1, agents: 2, treeVariants: 3 })
    const s = run(start, 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T)
    expect(s.revision).toBeGreaterThan(start.revision)
    expect(s.events.map((e) => e.kind)).toEqual(['run.started', 'actor.state'])
    const big = run(s, 1, { 0: Array.from({ length: T.sim.eventsRing + 5 }, (_, i) => ({ kind: 'agent.message' as const, agentId: A, message: `m${i}`, at: NOW })) }, T)
    expect(big.events.length).toBe(T.sim.eventsRing)
  })

  it('a repeated run.started for the same run while working is a no-op', () => {
    let s = until(run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T), (x) => actor(x).state === 'working')
    const before = hashState(s)
    s = run(s, 1, { 0: [{ kind: 'run.started', agentId: A, runId: 'r', at: NOW }] }, T)
    expect(actor(s).state).toBe('working')
    expect(hashState(s)).not.toBe(before)
    expect(actor(s).path).toEqual([])
  })
})
