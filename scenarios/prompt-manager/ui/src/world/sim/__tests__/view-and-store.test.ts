import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { createWorldStore } from '../store'
import { buildView, createViewSelector, equipmentTier } from '../view/select'
import { NOW, makeInput, run, world } from './fixtures'

describe('view', () => {
  it('summarises states and exposes team rooms', () => {
    const s = run(world(2, 2), 1, { 0: [{ kind: 'run.started', agentId: 'agent-0-0', runId: 'r', at: NOW }] })
    const v = buildView(s, tuning.actor)
    expect(v.summary.total).toBe(4)
    expect(v.summary.running).toBe(1)
    expect(v.summary.idle).toBe(3)
    expect(v.teams.map((t) => t.id)).toEqual(['team-0', 'team-1'])
    expect(v.teams[0]?.states.working).toBe(1)
    expect(v.events[0]?.kind).toBe('actor.state')
  })

  it('reports the next heartbeat and equipment tiers', () => {
    const s = run(world(1, 1), 1, { 0: [{ kind: 'heartbeat.upcoming', teamId: 'team-0', scheduledAt: NOW + 500, at: NOW }] })
    expect(buildView(s, tuning.actor).summary.nextHeartbeat).toEqual({ teamId: 'team-0', scheduledAt: NOW + 500 })
    const tiers = tuning.actor.equipmentTiers
    expect(equipmentTier(0, tiers)).toBe(0)
    expect(equipmentTier(tiers[4] ?? 0, tiers)).toBe(4)
    expect(equipmentTier((tiers[2] ?? 0) + 1, tiers)).toBe(2)
  })

  it('the selector returns the same object until the revision changes', () => {
    const select = createViewSelector(tuning.actor)
    const s = world(1, 1)
    expect(select(s)).toBe(select(s))
    const moved = run(s, 1)
    expect(select({ ...moved, revision: s.revision })).toBe(select(s))
    expect(select(run(s, 1, { 0: [{ kind: 'agent.message', agentId: 'agent-0-0', message: 'x', at: NOW }] }))).not.toBe(select(s))
  })
})

describe('store', () => {
  it('advances in fixed ticks with carry-over and notifies only on discrete change', () => {
    const store = createWorldStore(makeInput(1, 2), tuning, 3)
    let notified = 0
    store.subscribe(() => { notified += 1 })
    store.advance(tuning.sim.tickSeconds * 2.5)
    expect(store.getState().tick).toBe(2)
    store.advance(tuning.sim.tickSeconds * 0.5)
    expect(store.getState().tick).toBe(3)
    const before = notified
    store.dispatch([{ kind: 'run.started', agentId: 'agent-0-0', runId: 'r', at: NOW }])
    store.advance(tuning.sim.tickSeconds)
    expect(notified).toBe(before + 1)
    expect(store.getView().summary.running).toBe(1)
  })

  it('setTuning swaps levers live and bumps the view', () => {
    const store = createWorldStore(makeInput(1, 1), tuning, 3)
    const v1 = store.getView()
    store.setTuning({ ...tuning, sim: { ...tuning.sim, walkSpeed: 9 } })
    expect(store.tuning().sim.walkSpeed).toBe(9)
    expect(store.getView()).not.toBe(v1)
  })
})

describe('applyOverrides', () => {
  it('moves a room live while every actor keeps its position and state', () => {
    const store = createWorldStore(makeInput(2, 2), tuning, 3)
    store.dispatch([{ kind: 'run.started', agentId: 'agent-0-0', runId: 'r', at: NOW }])
    store.advance(tuning.sim.tickSeconds * 5)
    const before = store.getState()
    const positions = Object.fromEntries(before.actorOrder.map((id) => [id, before.actors[id]?.position]))
    store.applyOverrides([{ placeId: 'room:team-1', position: [30, -30] }])
    const after = store.getState()
    expect(after.places['room:team-1']?.position).toEqual([30, -30])
    expect(after.places['desk:agent-1-0']?.position[0]).toBeGreaterThan(20)
    for (const id of after.actorOrder) expect(after.actors[id]?.position).toEqual(positions[id])
    // The unmoved room keeps its worker seated and working; the moved room's member lost its seat.
    expect(after.actors['agent-0-0']?.state).toBe('working')
    expect(after.actors['agent-0-0']?.seatId).toBe(after.actors['agent-0-0']?.deskSeatId)
    expect(after.actors['agent-1-0']?.seatId).toBeUndefined()
    expect(after.actors['agent-0-0']?.path).toEqual([])
    expect(after.revision).toBe(before.revision + 1)
    expect(store.overrides()).toHaveLength(1)
    // The world keeps running afterwards: the moved member re-paths to the new desk.
    store.dispatch([{ kind: 'run.started', agentId: 'agent-1-0', runId: 'r2', at: NOW }])
    store.advance(tuning.sim.tickSeconds)
    expect(store.getState().actors['agent-1-0']?.path.length).toBeGreaterThan(0)
  })
})
