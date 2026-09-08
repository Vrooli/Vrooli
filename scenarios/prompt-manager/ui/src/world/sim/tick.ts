/**
 * step(state, dt, signals): the one function that advances the world.
 * Pure with respect to its inputs: it returns a new state and never mutates
 * the one it was given. Actors that change are copied; the rest are shared.
 */
import type { ActorTuning, LayoutTuning, SimTuning, WeatherTuning } from '../config'
import { advanceTimers, applySignal, arrive, type StepContext } from './actors/machine'
import { faceSocialPartner, rollIdle } from './idle/behaviors'
import type { Actor, Signal, WorldState } from './model'
import { headingTo, turnToward, moveAlongPath, updateAnimation } from './motion/move'
import { findPath, PathCache } from './nav/astar'
import { nearestWalkable } from './nav/grid'
import { Rng } from './rng'
import { smoothPressure, stepWeather, weatherPressure } from './weather'

export interface StepTuning {
  sim: SimTuning
  layout: LayoutTuning
  actor: ActorTuning
  weather: WeatherTuning
}

const NEAREST_RINGS = 6
const pathCaches = new WeakMap<Uint8Array, PathCache>()

/** Apply a live capacity edit without allocating a cache for an unused world. */
export function resizePathCache(state: WorldState, size: number): void {
  pathCaches.get(state.nav.walkable)?.resize(size)
}

function cacheFor(state: WorldState, size: number): PathCache {
  let cache = pathCaches.get(state.nav.walkable)
  if (!cache) {
    cache = new PathCache(size)
    pathCaches.set(state.nav.walkable, cache)
  }
  cache.resize(size)
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
  const touched = new Set<string>()
  const touch = (id: string): Actor | undefined => {
    const actor = next.actors[id]
    if (!actor) return undefined
    if (touched.has(id)) return actor
    const copy = cloneActor(actor)
    next.actors[id] = copy
    touched.add(id)
    return copy
  }
  const rng = new Rng(state.rngState)
  const ctx: StepContext = {
    sim: tuning.sim,
    paths: cacheFor(next, tuning.sim.pathCacheSize),
    replansLeft: tuning.sim.maxReplansPerTick,
    emoteSeconds: tuning.actor.emoteSeconds,
    nearestRings: NEAREST_RINGS,
    touch,
  }

  for (const signal of signals) applySignal(next, signal, ctx)
  // Sample health before timer cleanup removes expired gatherings.
  const pressureTarget = weatherPressure(next, tuning.weather)

  for (const id of next.actorOrder) {
    const actor = touch(id)
    if (!actor) continue
    const visitor = next.visitorConversation
    if (visitor?.agentId === id) {
      const desired: [number, number] = [visitor.position[0] + Math.sin(visitor.yaw) * 2, visitor.position[1] - Math.cos(visitor.yaw) * 2]
      let path = visitor.path ?? []
      let goal = visitor.goal
      if (!goal || Math.hypot(goal[0] - desired[0], goal[1] - desired[1]) > 1) {
        const target = nearestWalkable(next.nav, desired, ctx.nearestRings)
        if (target && ctx.replansLeft > 0) {
          ctx.replansLeft--
          const route = findPath(next.nav, actor.position, target, ctx.paths)
          if (route) { path = route; goal = desired }
        }
      }
      // Presentation motion is separate from the actor's real run state and route.
      const body = { ...actor, path: [...path], hurrying: false, anim: { ...actor.anim, seated: false } }
      moveAlongPath(body, dt, tuning.sim)
      if (!body.path.length) body.facing = turnToward(body.facing, headingTo(body.position, visitor.position), tuning.sim.turnRateRadPerSec, dt)
      actor.position = body.position; actor.facing = body.facing; actor.speed = body.speed; actor.anim = body.anim
      updateAnimation(actor, dt, body.path.length > 0, tuning.actor, () => rng.next())
      next.visitorConversation = { ...visitor, goal, path: body.path }
      continue
    }
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
  const pressure = smoothPressure(state.weather.pressure, pressureTarget, dt, tuning.weather)
  next.weather = stepWeather(state.weather, next.time, pressure, rng, tuning.weather, 'day')
  next.rngState = rng.state
  return next
}
