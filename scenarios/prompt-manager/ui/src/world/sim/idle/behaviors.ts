/**
 * The idle layer: what an actor does when nothing is asked of it.
 *
 * Home is the desk. A member rests at its desk seat and takes outings to the
 * commons (wander, socialize, sit at the campfire); an unassigned agent has
 * no desk and lives in the commons. Every outing ends with a roll that, on
 * `rest`, walks the actor home. Weights and durations are levers.
 */
import type { LayoutTuning, SimTuning } from '../../config'
import type { Actor, Vec2, WorldState } from '../model'
import { Rng } from '../rng'
import { GATHERING_ID, HEARTH_ID } from '../layout/generate'
import { routeTo, routeToSeat, releaseSeat, setState, type StepContext } from '../actors/machine'
import { headingTo } from '../motion/move'
import { isWalkable } from '../nav/grid'

const HALF = 0.5

function idleActors(state: WorldState): Actor[] {
  const out: Actor[] = []
  for (const id of state.actorOrder) {
    const a = state.actors[id]
    if (a && a.state === 'idle') out.push(a)
  }
  return out
}

function movers(actors: Actor[]): number {
  return actors.filter((a) => a.path.length > 0).length
}

function distance(a: Vec2, b: Vec2): number {
  return Math.hypot(a[0] - b[0], a[1] - b[1])
}

function commonsCenter(state: WorldState): Vec2 {
  const commons = state.places[GATHERING_ID]
  return commons ? commons.position : [0, 0]
}

export function insideCommons(state: WorldState, point: Vec2, layout: LayoutTuning): boolean {
  return distance(point, commonsCenter(state)) <= layout.commonsRadius
}

/** The actor is standing at its desk seat (or has no desk and is in the commons). */
export function atHome(state: WorldState, actor: Actor, layout: LayoutTuning): boolean {
  if (actor.path.length > 0) return false
  if (actor.deskSeatId) return actor.seatId === actor.deskSeatId
  return insideCommons(state, actor.position, layout)
}

/** Where other actors are or are heading, so a new spot keeps its distance. */
function claimedSpots(state: WorldState, except: string): Vec2[] {
  const spots: Vec2[] = []
  for (const id of state.actorOrder) {
    const other = state.actors[id]
    if (!other || other.id === except) continue
    spots.push(other.destination ?? other.position)
  }
  return spots
}

/**
 * A random point on the commons ring (outside the campfire seats, inside the
 * clearing) at least `spacing` from every other actor's spot. Falls back to
 * the last candidate when the commons is crowded, so the roll never stalls.
 */
export function randomCommonsPoint(state: WorldState, rng: Rng, sim: SimTuning, layout: LayoutTuning, except: string): Vec2 {
  const center = commonsCenter(state)
  const inner = layout.commonsSeatRadius + layout.clearingRadius * HALF
  const outer = layout.commonsRadius
  const taken = claimedSpots(state, except)
  let candidate: Vec2 = center
  for (let attempt = 0; attempt < sim.idle.spacingAttempts; attempt += 1) {
    const radius = inner + (outer - inner) * Math.sqrt(rng.next())
    const angle = rng.next() * Math.PI * 2
    candidate = [center[0] + Math.sin(angle) * radius, center[1] + Math.cos(angle) * radius]
    if (isWalkable(state.nav, candidate) && taken.every((spot) => distance(spot, candidate) >= sim.idle.spacing)) return candidate
  }
  return candidate
}

/** Walk home: the desk seat for a member, a commons spot for an unassigned agent. Returns false when no route exists. */
export function routeHome(state: WorldState, actor: Actor, rng: Rng, sim: SimTuning, layout: LayoutTuning, ctx: StepContext): boolean {
  if (actor.deskSeatId) return routeToSeat(state, actor, actor.deskSeatId, ctx)
  releaseSeat(state, actor)
  return routeTo(state, actor, randomCommonsPoint(state, rng, sim, layout, actor.id), ctx)
}

function rest(state: WorldState, actor: Actor, rng: Rng, sim: SimTuning): void {
  actor.idle = { activity: 'rest', until: state.time + rng.range(sim.idle.restSeconds.min, sim.idle.restSeconds.max) }
}

/**
 * Roll a new idle activity for an actor whose current one has ended.
 * `rest` means "be at home"; the other three are outings to the commons.
 * A blocked roll (no route, no partner, no seat, mover budget spent) rests.
 */
export function rollIdle(state: WorldState, actor: Actor, rng: Rng, sim: SimTuning, layout: LayoutTuning, ctx: StepContext): void {
  const idle = idleActors(state)
  const moverBudget = Math.floor(idle.length * sim.idle.maxMoversRatio)
  const canMove = movers(idle) < Math.max(1, moverBudget)
  const w = sim.idle.weights
  const home = atHome(state, actor, layout)

  // Adrift (fresh from a table, a removed room, an ended outing): head home first.
  if (!home && !insideCommons(state, actor.position, layout)) {
    if (canMove && routeHome(state, actor, rng, sim, layout, ctx)) {
      actor.idle = { activity: 'rest', until: state.time + sim.idle.restSeconds.max }
      return
    }
    actor.idle = { activity: 'rest', until: state.time + sim.idle.rollIntervalSeconds }
    return
  }

  const pick = rng.weighted([w.rest, canMove ? w.wander : 0, canMove ? w.socialize : 0, canMove ? w.sit : 0])
  switch (pick) {
    case 0: {
      if (!home && canMove && routeHome(state, actor, rng, sim, layout, ctx)) {
        actor.idle = { activity: 'rest', until: state.time + sim.idle.restSeconds.max }
        return
      }
      break
    }
    case 1: {
      const spot = randomCommonsPoint(state, rng, sim, layout, actor.id)
      releaseSeat(state, actor)
      if (routeTo(state, actor, spot, ctx)) {
        actor.idle = { activity: 'wander', until: state.time + rng.range(sim.idle.restSeconds.min, sim.idle.restSeconds.max) }
        return
      }
      break
    }
    case 2: {
      const partner = idle.find((other) => other.id !== actor.id && other.idle.activity === 'rest' && other.path.length === 0)
      if (partner) {
        const writablePartner = ctx.touch?.(partner.id) ?? partner
        // Meet on the commons ring: the two stand `socializeGap` apart around a shared spot.
        const spot = randomCommonsPoint(state, rng, sim, layout, actor.id)
        const heading = headingTo(actor.position, spot)
        const gap = sim.idle.socializeGap * HALF
        const mine: Vec2 = [spot[0] - Math.sin(heading) * gap, spot[1] - Math.cos(heading) * gap]
        const theirs: Vec2 = [spot[0] + Math.sin(heading) * gap, spot[1] + Math.cos(heading) * gap]
        const until = state.time + rng.range(sim.idle.socializeSeconds.min, sim.idle.socializeSeconds.max)
        releaseSeat(state, actor)
        releaseSeat(state, writablePartner)
        if (routeTo(state, actor, mine, ctx) && routeTo(state, writablePartner, theirs, ctx)) {
          actor.idle = { activity: 'socialize', until, partnerId: partner.id }
          writablePartner.idle = { activity: 'socialize', until, partnerId: actor.id }
          setState(state, actor, 'socializing', sim.eventsRing)
          setState(state, writablePartner, 'socializing', sim.eventsRing)
          return
        }
      }
      break
    }
    case 3: {
      const campfire = state.places[HEARTH_ID]
      const free = campfire?.seats.find((s) => !state.occupancy[s.id])
      if (free && routeToSeat(state, actor, free.id, ctx)) {
        actor.idle = { activity: 'sit', until: state.time + rng.range(sim.idle.sitSeconds.min, sim.idle.sitSeconds.max), seatId: free.id }
        return
      }
      break
    }
    default:
      break
  }
  if (actor.idle.activity === 'sit' && actor.seatId) {
    // Keep sitting a little longer when the roll landed on rest.
    actor.idle = { ...actor.idle, until: state.time + sim.idle.rollIntervalSeconds }
    return
  }
  rest(state, actor, rng, sim)
}

/** Socializing pairs face each other once both have arrived. */
export function faceSocialPartner(state: WorldState, actor: Actor): void {
  if (actor.state !== 'socializing' || actor.path.length > 0 || !actor.idle.partnerId) return
  const partner = state.actors[actor.idle.partnerId]
  if (partner) actor.facing = headingTo(actor.position, partner.position)
}
