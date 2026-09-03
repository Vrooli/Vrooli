/**
 * step(state, dt, signals): the one function that advances the world.
 * Pure with respect to its inputs: it returns a new state and never mutates
 * the one it was given. Actors that change are copied; the rest are shared.
 */
import type { ActorTuning, LayoutTuning, SimTuning } from '../config'
import { advanceTimers, applySignal, arrive, type StepContext } from './actors/machine'
import { faceSocialPartner, rollIdle } from './idle/behaviors'
import type { Actor, Signal, WorldState } from './model'
import { moveAlongPath, updateAnimation } from './motion/move'
import { PathCache } from './nav/astar'
import { Rng } from './rng'

export interface StepTuning {
  sim: SimTuning
  layout: LayoutTuning
  actor: ActorTuning
}

const NEAREST_RINGS = 6
const pathCaches = new WeakMap<Uint8Array, PathCache>()

function cacheFor(state: WorldState, size: number): PathCache {
  let cache = pathCaches.get(state.nav.walkable)
  if (!cache) {
    cache = new PathCache(size)
    pathCaches.set(state.nav.walkable, cache)
  }
  return cache
}

function cloneActor(actor: Actor): Actor {
  return {
    ...actor,
    path: [...actor.path],
    idle: { ...actor.idle },
    anim: { ...actor.anim, emote: actor.anim.emote ? { ...actor.anim.emote } : undefined },
    lastRun: actor.lastRun ? { ...actor.lastRun } : undefined,
  }
}

export function step(state: WorldState, dt: number, signals: readonly Signal[], tuning: StepTuning): WorldState {
  const next: WorldState = {
    ...state,
    tick: state.tick + 1,
    time: state.time + dt,
    actors: { ...state.actors },
    occupancy: { ...state.occupancy },
    gatherings: { ...state.gatherings },
    events: [...state.events],
  }
  for (const id of next.actorOrder) {
    const actor = next.actors[id]
    if (actor) next.actors[id] = cloneActor(actor)
  }
  const rng = new Rng(state.rngState)
  const ctx: StepContext = {
    sim: tuning.sim,
    paths: cacheFor(next, tuning.sim.pathCacheSize),
    replansLeft: tuning.sim.maxReplansPerTick,
    emoteSeconds: tuning.actor.emoteSeconds,
    nearestRings: NEAREST_RINGS,
  }

  for (const signal of signals) applySignal(next, signal, ctx)

  for (const id of next.actorOrder) {
    const actor = next.actors[id]
    if (!actor) continue
    advanceTimers(next, actor, ctx)
    if (actor.state === 'idle' && actor.path.length === 0 && next.time >= actor.idle.until) {
      rollIdle(next, actor, rng, tuning.sim, tuning.layout, ctx)
    }
    const moving = actor.path.length > 0
    const arrived = moveAlongPath(actor, dt, tuning.sim)
    if (arrived) arrive(next, actor, ctx)
    faceSocialPartner(next, actor)
    updateAnimation(actor, dt, moving, tuning.actor, () => rng.next())
  }
  next.rngState = rng.state
  return next
}
