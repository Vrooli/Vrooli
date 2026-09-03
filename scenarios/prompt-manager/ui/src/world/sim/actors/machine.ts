/**
 * The actor state machine. Every arrow in docs/concepts/WORLD-SIM.md is one
 * function here, and one test in __tests__/machine.test.ts.
 */
import type { SimTuning } from '../../config'
import type { Actor, ActorState, EmoteKind, Signal, WorldState } from '../model'
import { findPath, type PathCache } from '../nav/astar'
import { nearestWalkable } from '../nav/grid'
import { COMMONS_ID } from '../layout/generate'

/** Record without one key; the sim never deletes in place. */
function without<T>(record: Record<string, T>, key: string): Record<string, T> {
  const { [key]: _dropped, ...rest } = record
  return rest
}

export interface StepContext {
  sim: SimTuning
  paths: PathCache
  replansLeft: number
  emoteSeconds: number
  nearestRings: number
}

export function pushEvent(state: WorldState, event: Omit<WorldState['events'][number], 'seq'>, ring: number): void {
  state.events.push({ ...event, seq: state.nextSeq })
  state.nextSeq += 1
  if (state.events.length > ring) state.events.splice(0, state.events.length - ring)
  state.revision += 1
}

export function setState(state: WorldState, actor: Actor, next: ActorState, ring: number): void {
  if (actor.state === next) return
  actor.state = next
  actor.stateSince = state.time
  pushEvent(state, { at: state.time, kind: 'actor.state', agentId: actor.id, teamId: actor.teamId, state: next }, ring)
}

export function emote(actor: Actor, kind: EmoteKind, seconds: number): void {
  actor.anim.emote = { kind, remaining: seconds }
}

export function releaseSeat(state: WorldState, actor: Actor): void {
  if (actor.seatId && state.occupancy[actor.seatId] === actor.id) state.occupancy = without(state.occupancy, actor.seatId)
  actor.seatId = undefined
  actor.anim.seated = false
}

/** Route the actor to a point; falls back to the nearest walkable cell. Returns false when no path exists. */
export function routeTo(state: WorldState, actor: Actor, target: readonly [number, number], ctx: StepContext): boolean {
  if (ctx.replansLeft <= 0) return false
  ctx.replansLeft -= 1
  const goal = nearestWalkable(state.nav, target, ctx.nearestRings)
  const start = nearestWalkable(state.nav, actor.position, ctx.nearestRings) ?? actor.position
  if (!goal) return false
  const path = findPath(state.nav, start, goal, ctx.paths)
  if (!path) return false
  actor.path = start === actor.position ? path : [start, ...path]
  actor.destination = goal
  return true
}

/** Claim a seat and walk to it. */
export function routeToSeat(state: WorldState, actor: Actor, seatId: string, ctx: StepContext): boolean {
  const seat = state.seats[seatId]
  if (!seat) return false
  const holder = state.occupancy[seatId]
  if (holder && holder !== actor.id) return false
  releaseSeat(state, actor)
  if (!routeTo(state, actor, seat.position, ctx)) return false
  state.occupancy[seatId] = actor.id
  actor.seatId = seatId
  return true
}

function commonsSpot(state: WorldState, actor: Actor): readonly [number, number] {
  const commons = state.places[COMMONS_ID]
  return commons ? commons.position : actor.position
}

/** Idle → WalkingToDesk (run.started), also Failed/Gathered → WalkingToDesk; an actor already at its desk goes straight to Working. */
export function startRun(state: WorldState, actor: Actor, runId: string, ctx: StepContext): void {
  actor.runId = runId
  actor.failedError = undefined
  actor.lastRun = { runId, status: 'running', startedAt: state.time }
  actor.hurrying = true
  actor.idle = { activity: 'rest', until: state.time }
  emote(actor, 'start', ctx.emoteSeconds)
  if (actor.deskSeatId && actor.seatId === actor.deskSeatId && actor.path.length === 0) {
    // Already home: work where you stand.
    setState(state, actor, 'working', ctx.sim.eventsRing)
  } else if (actor.deskSeatId) {
    routeToSeat(state, actor, actor.deskSeatId, ctx)
    setState(state, actor, 'walkingToDesk', ctx.sim.eventsRing)
  } else {
    // No desk: work where you stand (unassigned agents live in the commons).
    releaseSeat(state, actor)
    actor.path = []
    setState(state, actor, 'working', ctx.sim.eventsRing)
  }
}

/** Working → Idle (run.finished). */
export function finishRun(state: WorldState, actor: Actor, runId: string, ctx: StepContext): void {
  if (actor.lastRun && actor.lastRun.runId === runId) actor.lastRun = { ...actor.lastRun, status: 'completed', endedAt: state.time }
  actor.runId = undefined
  actor.hurrying = false
  emote(actor, 'done', ctx.emoteSeconds)
  goIdle(state, actor, ctx)
}

/** Working → Failed (run.failed). */
export function failRun(state: WorldState, actor: Actor, runId: string, error: string, ctx: StepContext): void {
  if (actor.lastRun && actor.lastRun.runId === runId) actor.lastRun = { ...actor.lastRun, status: 'failed', endedAt: state.time, error }
  actor.runId = undefined
  actor.hurrying = false
  actor.failedError = error
  emote(actor, 'fail', ctx.emoteSeconds)
  setState(state, actor, 'failed', ctx.sim.eventsRing)
}

/** Any → Idle: the actor heads back to the commons on its next idle roll. */
export function goIdle(state: WorldState, actor: Actor, ctx: StepContext): void {
  actor.hurrying = false
  actor.idle = { activity: 'rest', until: state.time }
  setState(state, actor, 'idle', ctx.sim.eventsRing)
}

/** Idle → WalkingToTable (heartbeat.upcoming within the gather lead). */
export function gather(state: WorldState, actor: Actor, teamId: string, ctx: StepContext): void {
  const table = Object.values(state.places).find((p) => p.kind === 'table' && p.teamId === teamId)
  if (!table) return
  const free = table.seats.find((s) => !state.occupancy[s.id] || state.occupancy[s.id] === actor.id)
  if (!free) return
  actor.idle = { activity: 'rest', until: state.time }
  if (routeToSeat(state, actor, free.id, ctx)) {
    emote(actor, 'gather', ctx.emoteSeconds)
    setState(state, actor, 'walkingToTable', ctx.sim.eventsRing)
  }
}

/** Arrival transitions: WalkingToDesk → Working, WalkingToTable → Gathered, idle walks settle. */
export function arrive(state: WorldState, actor: Actor, ctx: StepContext): void {
  const seat = actor.seatId ? state.seats[actor.seatId] : undefined
  if (seat) {
    actor.facing = seat.facing
    actor.anim.seated = seat.sitting
  }
  actor.destination = undefined
  pushEvent(state, { at: state.time, kind: 'actor.arrived', agentId: actor.id, teamId: actor.teamId }, ctx.sim.eventsRing)
  if (actor.state === 'walkingToDesk') setState(state, actor, 'working', ctx.sim.eventsRing)
  else if (actor.state === 'walkingToTable') setState(state, actor, 'gathered', ctx.sim.eventsRing)
}

/** Apply one signal to the world. Unknown agents are ignored; every applied signal becomes an event. */
export function applySignal(state: WorldState, signal: Signal, ctx: StepContext): void {
  const ring = ctx.sim.eventsRing
  switch (signal.kind) {
    case 'run.started': {
      const actor = state.actors[signal.agentId]
      if (!actor) return
      pushEvent(state, { at: signal.at, kind: signal.kind, agentId: actor.id, teamId: actor.teamId, runId: signal.runId }, ring)
      if (actor.state === 'working' && actor.runId === signal.runId) return
      startRun(state, actor, signal.runId, ctx)
      return
    }
    case 'run.finished': {
      const actor = state.actors[signal.agentId]
      if (!actor) return
      pushEvent(state, { at: signal.at, kind: signal.kind, agentId: actor.id, teamId: actor.teamId, runId: signal.runId }, ring)
      if (actor.state === 'working' || actor.state === 'walkingToDesk') finishRun(state, actor, signal.runId, ctx)
      return
    }
    case 'run.failed': {
      const actor = state.actors[signal.agentId]
      if (!actor) return
      pushEvent(state, { at: signal.at, kind: signal.kind, agentId: actor.id, teamId: actor.teamId, runId: signal.runId, message: signal.error }, ring)
      if (actor.state === 'working' || actor.state === 'walkingToDesk') failRun(state, actor, signal.runId, signal.error, ctx)
      return
    }
    case 'heartbeat.upcoming': {
      pushEvent(state, { at: signal.at, kind: signal.kind, teamId: signal.teamId }, ring)
      state.gatherings[signal.teamId] = {
        teamId: signal.teamId,
        scheduledAt: signal.scheduledAt,
        until: signal.scheduledAt + ctx.sim.gatherWindowSeconds,
      }
      return
    }
    case 'heartbeat.cancelled': {
      pushEvent(state, { at: signal.at, kind: signal.kind, teamId: signal.teamId }, ring)
      state.gatherings = without(state.gatherings, signal.teamId)
      for (const id of state.actorOrder) {
        const actor = state.actors[id]
        if (!actor || actor.teamId !== signal.teamId) continue
        if (actor.state === 'gathered' || actor.state === 'walkingToTable') {
          releaseSeat(state, actor)
          actor.path = []
          goIdle(state, actor, ctx)
        }
      }
      return
    }
    case 'agent.message': {
      const actor = state.actors[signal.agentId]
      if (!actor) return
      actor.message = { text: signal.message, at: signal.at }
      emote(actor, 'message', ctx.emoteSeconds)
      pushEvent(state, { at: signal.at, kind: signal.kind, agentId: actor.id, teamId: actor.teamId, message: signal.message }, ring)
      return
    }
    case 'failed.acknowledged': {
      const actor = state.actors[signal.agentId]
      if (!actor) return
      pushEvent(state, { at: signal.at, kind: signal.kind, agentId: actor.id, teamId: actor.teamId }, ring)
      if (actor.state === 'failed') {
        actor.failedError = undefined
        goIdle(state, actor, ctx)
      }
      return
    }
  }
}

/** Time-driven transitions: gather lead/window, failed timeout, socialize end. Called once per actor per tick. */
export function advanceTimers(state: WorldState, actor: Actor, ctx: StepContext): void {
  const sim = ctx.sim
  if (actor.state === 'idle' && actor.teamId) {
    const gathering = state.gatherings[actor.teamId]
    if (gathering && state.time >= gathering.scheduledAt - sim.gatherLeadSeconds && state.time < gathering.until) {
      gather(state, actor, actor.teamId, ctx)
      return
    }
  }
  if ((actor.state === 'gathered' || actor.state === 'walkingToTable') && actor.teamId) {
    const gathering = state.gatherings[actor.teamId]
    if (!gathering || state.time >= gathering.until) {
      if (gathering) state.gatherings = without(state.gatherings, actor.teamId)
      releaseSeat(state, actor)
      actor.path = []
      goIdle(state, actor, ctx)
      return
    }
  }
  if (actor.state === 'failed' && state.time - actor.stateSince >= sim.failedAckSeconds) {
    actor.failedError = undefined
    goIdle(state, actor, ctx)
    return
  }
  if (actor.state === 'socializing' && state.time >= actor.idle.until) {
    const partner = actor.idle.partnerId ? state.actors[actor.idle.partnerId] : undefined
    goIdle(state, actor, ctx)
    if (partner && partner.state === 'socializing') goIdle(state, partner, ctx)
  }
}

export function commonsCenter(state: WorldState, actor: Actor): readonly [number, number] {
  return commonsSpot(state, actor)
}
