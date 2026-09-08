import { describe, expect, it, vi } from 'vitest'
import { PathCache } from '../nav/astar'
import { tuning } from '../../config'
import { buildView, createViewSelector, equipmentTier } from '../view/select'
import { NOW, run, makeWorld, makeWorldStore } from './fixtures'

describe('view', () => {
  it('summarises states and exposes team rooms', () => {
    const s = run(makeWorld({ teams: 2, agents: 4, treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: 'agent-0-0', runId: 'r', at: NOW }] })
    const v = buildView(s, tuning.actor)
    expect(v.summary.total).toBe(4)
    expect(v.summary.running).toBe(1)
    expect(v.summary.idle).toBe(3)
    expect(v.teams.map((t) => t.id)).toEqual(['team-0', 'team-1'])
    expect(v.teams[0]?.states.working).toBe(1)
    expect(v.events[0]?.kind).toBe('actor.state')
  })

  it('reports the next heartbeat and equipment tiers', () => {
    const s = run(makeWorld({ teams: 1, agents: 1, treeVariants: 3 }), 1, { 0: [{ kind: 'heartbeat.upcoming', teamId: 'team-0', scheduledAt: NOW + 500, at: NOW }] })
    expect(buildView(s, tuning.actor).summary.nextHeartbeat).toEqual({ teamId: 'team-0', scheduledAt: NOW + 500 })
    const tiers = tuning.actor.equipmentTiers
    expect(equipmentTier(0, tiers)).toBe(0)
    expect(equipmentTier(tiers[4] ?? 0, tiers)).toBe(4)
    expect(equipmentTier((tiers[2] ?? 0) + 1, tiers)).toBe(2)
  })

  it('the selector returns the same object until the revision changes', () => {
    const select = createViewSelector(tuning.actor)
    const s = makeWorld({ teams: 1, agents: 1, treeVariants: 3 })
    expect(select(s)).toBe(select(s))
    const moved = run(s, 1)
    expect(select({ ...moved, revision: s.revision })).toBe(select(s))
    expect(select(run(s, 1, { 0: [{ kind: 'agent.message', agentId: 'agent-0-0', message: 'x', at: NOW }] }))).not.toBe(select(s))
  })
})

describe('store', () => {
  it('advances in fixed ticks with carry-over and notifies only on discrete change', () => {
    const store = makeWorldStore({ teams: 1, agents: 2, treeVariants: 3 })
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

  it('applies cache capacity on commit without advancing or replacing the world', () => {
    const store = makeWorldStore({ teams: 1, agents: 1, treeVariants: 3 })
    store.advance(tuning.sim.tickSeconds)
    const before = store.getState()
    const resize = vi.spyOn(PathCache.prototype, 'resize')
    try {
      const next = { ...tuning, sim: { ...tuning.sim, pathCacheSize: 16 } }
      store.setTuning(next)
      expect(resize).toHaveBeenCalledTimes(1)
      expect(resize).toHaveBeenCalledWith(16)
      expect(store.getState()).toBe(before)
      store.setTuning(structuredClone(next))
      expect(resize).toHaveBeenCalledTimes(1)
    } finally { resize.mockRestore() }
  })

  it('trims event history at commit and publishes combined view edits once', () => {
    const store = makeWorldStore({ teams: 1, agents: 1, treeVariants: 3 })
    store.dispatch(Array.from({ length: 20 }, (_, i) => ({ kind: 'agent.message' as const, agentId: 'agent-0-0', message: `event-${i}`, at: NOW + i })))
    store.advance(tuning.sim.tickSeconds)
    const before = store.getState()
    expect(before.events.length).toBeGreaterThan(8)
    let notices = 0
    store.subscribe(() => { notices++ })
    store.dispatch([{ kind: 'run.started', agentId: 'agent-0-0', runId: 'queued', at: NOW }])
    store.advance(tuning.sim.tickSeconds / 2)
    const next = { ...tuning, sim: { ...tuning.sim, eventsRing: 8 }, actor: { ...tuning.actor, equipmentTiers: [0, 0, 0, 0, 0] } }
    store.setTuning(next)
    expect(store.getState().events).toEqual(before.events.slice(-8))
    expect(before.events.length).toBeGreaterThan(8)
    expect(store.getState().actors).toBe(before.actors)
    expect(store.getState().nav).toBe(before.nav)
    expect(store.getState().time).toBe(before.time)
    expect(notices).toBe(1)
    store.setTuning(structuredClone(next))
    expect(notices).toBe(1)
    store.advance(tuning.sim.tickSeconds / 2)
    expect(store.getState().actors['agent-0-0']?.runId).toBe('queued')
    expect(store.getState().events.length).toBeLessThanOrEqual(8)
  })

  it('applies live tuning without invalidating unchanged HUD data', () => {
    const store = makeWorldStore({ teams: 1, agents: 1, treeVariants: 3 })
    const state = store.getState(), view = store.getView()
    let notifications = 0
    store.subscribe(() => { notifications++ })
    const next = structuredClone(tuning)
    next.sim.walkSpeed = 9
    next.camera.fov = 65
    next.layout.surfaces.floorRoughness = .37
    store.setTuning(next)
    expect(store.tuning()).toBe(next)
    expect(store.getState()).toBe(state)
    expect(store.getView()).toBe(view)
    expect(notifications).toBe(0)
  })

  it('publishes changed equipment tiers exactly once without moving actors', () => {
    const store = makeWorldStore({ teams: 1, agents: 1, treeVariants: 3 })
    const state = store.getState(), view = store.getView()
    let notifications = 0
    store.subscribe(() => { notifications++ })
    const next = { ...tuning, actor: { ...tuning.actor, equipmentTiers: tuning.actor.equipmentTiers.map(() => 0) } }
    store.setTuning(next)
    expect(store.getView()).not.toBe(view)
    expect(store.getView().actors[0]?.equipmentTier).toBe(next.actor.equipmentTiers.length - 1)
    expect(store.getState().actors).toBe(state.actors)
    expect(store.getState().nav).toBe(state.nav)
    expect(notifications).toBe(1)
    const updated = store.getView()
    store.setTuning(structuredClone(next))
    expect(store.getView()).toBe(updated)
    expect(notifications).toBe(1)
  })

  it('preserves queued signals and accumulated time across a tick-duration edit', () => {
    const store = makeWorldStore({ teams: 1, agents: 1, treeVariants: 3 })
    store.dispatch([{ kind: 'run.started', agentId: 'agent-0-0', runId: 'queued', at: NOW }])
    store.advance(tuning.sim.tickSeconds / 2)
    expect(store.getState().tick).toBe(0)
    store.setTuning({ ...tuning, sim: { ...tuning.sim, tickSeconds: tuning.sim.tickSeconds / 2 } })
    store.advance(0)
    expect(store.getState().tick).toBe(1)
    expect(store.getState().actors['agent-0-0']?.runId).toBe('queued')
    expect(store.getView().summary.running).toBe(1)
  })

})

describe('applyOverrides', () => {
  it('moves resting occupants with a room while preserving active workers and their runs', () => {
    const store = makeWorldStore({ teams: 2, agents: 4, treeVariants: 3 })
    store.dispatch([{ kind: 'run.started', agentId: 'agent-0-0', runId: 'r', at: NOW }])
    store.dispatch([{ kind: 'run.started', agentId: 'agent-1-1', runId: 'moving-room-run', at: NOW }])
    store.advance(tuning.sim.tickSeconds * 5)
    const before = store.getState()
    const positions = Object.fromEntries(before.actorOrder.map((id) => [id, before.actors[id]?.position]))
    store.applyOverrides([{ placeId: 'room:team-1', position: [30, -30] }])
    const after = store.getState()
    expect(after.places['room:team-1']?.position).toEqual([30, -30])
    expect(after.places['desk:agent-1-0']?.position[0]).toBeGreaterThan(20)
    for (const id of ['agent-0-0', 'agent-1-1']) expect(after.actors[id]?.position).toEqual(positions[id])
    // Resting occupants retain a meaningful home; active workers keep their live position.
    expect(after.actors['agent-0-0']?.state).toBe('working')
    expect(after.actors['agent-0-0']?.seatId).toBe(after.actors['agent-0-0']?.deskSeatId)
    const moved = after.actors['agent-1-0']
    expect(moved?.seatId).toBe(moved?.deskSeatId)
    expect(moved?.position).toEqual(moved?.seatId ? after.seats[moved.seatId]?.position : null)
    expect(after.actors['agent-1-1']?.seatId).toBeUndefined()
    expect(after.actors['agent-1-1']?.runId).toBe('moving-room-run')
    expect(after.actors['agent-0-0']?.path).toEqual([])
    expect(after.revision).toBe(before.revision + 1)
    expect(store.overrides()).toHaveLength(1)
    // The world keeps running afterwards: the active worker re-paths to its new desk.
    store.advance(tuning.sim.tickSeconds)
    expect(store.getState().actors['agent-1-1']?.path.length).toBeGreaterThan(0)
  })
})
