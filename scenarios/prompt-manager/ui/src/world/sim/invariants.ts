/**
 * World invariants: properties every settled world state must satisfy,
 * checked without a renderer. Tests run them after simulated minutes; the
 * smoke tool runs them on the live page. A violation names the rule and the
 * offending ids so a failure reads as a sentence, not a diff.
 *
 * Rules:
 *   seat-occupancy      occupancy is a bijection and matches actor.seatId
 *   seat-exists         every seat an actor references exists
 *   inside-bounds       every actor stands on the slab
 *   desk-in-room        every desk and table sits inside its parent room
 *   rooms-disjoint      room footprints do not overlap each other or the commons
 *   separation          two standing actors are never closer than a body width
 *   resting-in-place    an idle actor that is not walking stands somewhere meaningful
 */
import type { ActorTuning, LayoutTuning } from '../config'
import type { Actor, Place, Vec2, WorldState } from './model'
import { COMMONS_ID } from './layout/generate'

export type InvariantRule =
  | 'seat-occupancy'
  | 'seat-exists'
  | 'inside-bounds'
  | 'desk-in-room'
  | 'rooms-disjoint'
  | 'separation'
  | 'resting-in-place'

export interface Violation {
  rule: InvariantRule
  /** Ids involved, most specific first. */
  ids: string[]
  detail: string
}

export interface InvariantTuning {
  layout: LayoutTuning
  actor: Pick<ActorTuning, 'bodyRadius'>
}

const HALF = 0.5

function distance(a: Vec2, b: Vec2): number {
  return Math.hypot(a[0] - b[0], a[1] - b[1])
}

function insideRect(point: Vec2, place: Place): boolean {
  return Math.abs(point[0] - place.position[0]) <= place.size[0] * HALF && Math.abs(point[1] - place.position[1]) <= place.size[1] * HALF
}

/** Whether a disc touches a place footprint: nearest point of the rectangle to the centre lies inside the radius. */
function rectDiscOverlap(rect: Place, center: Vec2, radius: number): boolean {
  const dx = Math.max(Math.abs(center[0] - rect.position[0]) - rect.size[0] * HALF, 0)
  const dz = Math.max(Math.abs(center[1] - rect.position[1]) - rect.size[1] * HALF, 0)
  return dx * dx + dz * dz < radius * radius
}

function rectsOverlap(a: Place, b: Place): boolean {
  return (
    Math.abs(a.position[0] - b.position[0]) < (a.size[0] + b.size[0]) * HALF &&
    Math.abs(a.position[1] - b.position[1]) < (a.size[1] + b.size[1]) * HALF
  )
}

function actors(state: WorldState): Actor[] {
  const out: Actor[] = []
  for (const id of state.actorOrder) {
    const a = state.actors[id]
    if (a) out.push(a)
  }
  return out
}

export function checkSeats(state: WorldState): Violation[] {
  const out: Violation[] = []
  const holders = new Map<string, string>()
  for (const [seatId, actorId] of Object.entries(state.occupancy)) {
    if (!state.seats[seatId]) out.push({ rule: 'seat-exists', ids: [seatId, actorId], detail: `occupancy names unknown seat ${seatId}` })
    const previous = holders.get(actorId)
    if (previous) out.push({ rule: 'seat-occupancy', ids: [actorId, previous, seatId], detail: `${actorId} holds two seats` })
    holders.set(actorId, seatId)
    const actor = state.actors[actorId]
    if (!actor) out.push({ rule: 'seat-occupancy', ids: [actorId, seatId], detail: `seat ${seatId} held by unknown actor` })
    else if (actor.seatId !== seatId) out.push({ rule: 'seat-occupancy', ids: [actorId, seatId], detail: `${actorId} holds ${seatId} but points at ${actor.seatId ?? 'nothing'}` })
  }
  for (const actor of actors(state)) {
    if (!actor.seatId) continue
    if (!state.seats[actor.seatId]) out.push({ rule: 'seat-exists', ids: [actor.id, actor.seatId], detail: `${actor.id} references unknown seat` })
    else if (state.occupancy[actor.seatId] !== actor.id) out.push({ rule: 'seat-occupancy', ids: [actor.id, actor.seatId], detail: `${actor.id} points at ${actor.seatId} without holding it` })
  }
  return out
}

export function checkBounds(state: WorldState): Violation[] {
  const out: Violation[] = []
  const { width, depth, center } = state.bounds
  for (const actor of actors(state)) {
    if (Math.abs(actor.position[0] - center[0]) > width * HALF || Math.abs(actor.position[1] - center[1]) > depth * HALF) {
      out.push({ rule: 'inside-bounds', ids: [actor.id], detail: `${actor.id} at (${actor.position[0].toFixed(1)}, ${actor.position[1].toFixed(1)}) is off the slab` })
    }
  }
  return out
}

export function checkPlaces(state: WorldState, layout: LayoutTuning): Violation[] {
  const out: Violation[] = []
  const places = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => Boolean(p))
  const rooms = places.filter((p) => p.kind === 'room')
  const commons = state.places[COMMONS_ID]
  for (const place of places) {
    if (!place.parentId || (place.kind !== 'desk' && place.kind !== 'table')) continue
    const room = state.places[place.parentId]
    if (!room) continue
    const corners: Vec2[] = [
      [place.position[0] - place.size[0] * HALF, place.position[1] - place.size[1] * HALF],
      [place.position[0] + place.size[0] * HALF, place.position[1] + place.size[1] * HALF],
    ]
    if (!corners.every((corner) => insideRect(corner, room))) {
      out.push({ rule: 'desk-in-room', ids: [place.id, room.id], detail: `${place.id} extends outside ${room.id}` })
    }
    for (const seat of place.seats) {
      if (!insideRect(seat.position, room)) out.push({ rule: 'desk-in-room', ids: [seat.id, room.id], detail: `${seat.id} lies outside ${room.id}` })
    }
  }
  for (let i = 0; i < rooms.length; i += 1) {
    const a = rooms[i]
    if (!a) continue
    for (let j = i + 1; j < rooms.length; j += 1) {
      const b = rooms[j]
      if (b && rectsOverlap(a, b)) out.push({ rule: 'rooms-disjoint', ids: [a.id, b.id], detail: `${a.id} overlaps ${b.id}` })
    }
    if (commons && rectDiscOverlap(a, commons.position, layout.commonsRadius)) {
      out.push({ rule: 'rooms-disjoint', ids: [a.id, COMMONS_ID], detail: `${a.id} overlaps the commons` })
    }
  }
  return out
}

/** Standing actors keep a body width apart; walkers and seated socializers are exempt. */
export function checkSeparation(state: WorldState, tuning: InvariantTuning): Violation[] {
  const out: Violation[] = []
  const standing = actors(state).filter((a) => a.path.length === 0)
  const minimum = tuning.actor.bodyRadius * 2
  for (let i = 0; i < standing.length; i += 1) {
    const a = standing[i]
    if (!a) continue
    for (let j = i + 1; j < standing.length; j += 1) {
      const b = standing[j]
      if (!b) continue
      const d = distance(a.position, b.position)
      if (d < minimum) out.push({ rule: 'separation', ids: [a.id, b.id], detail: `${a.id} and ${b.id} are ${d.toFixed(2)} m apart (minimum ${minimum.toFixed(2)})` })
    }
  }
  return out
}

/** A resting idle actor is at its desk seat, on the commons, or on a seat it holds. */
export function checkRestingInPlace(state: WorldState, layout: LayoutTuning): Violation[] {
  const out: Violation[] = []
  const commons = state.places[COMMONS_ID]
  for (const actor of actors(state)) {
    if (actor.state !== 'idle' || actor.path.length > 0) continue
    const seat = actor.seatId ? state.seats[actor.seatId] : undefined
    if (seat && distance(actor.position, seat.position) <= layout.cellSize) continue
    if (commons && distance(actor.position, commons.position) <= layout.commonsRadius + layout.cellSize) continue
    out.push({ rule: 'resting-in-place', ids: [actor.id], detail: `${actor.id} rests at (${actor.position[0].toFixed(1)}, ${actor.position[1].toFixed(1)}) with no seat and outside the commons` })
  }
  return out
}

/** Every rule at once. Empty means the state is well formed. */
export function checkWorldInvariants(state: WorldState, tuning: InvariantTuning): Violation[] {
  return [
    ...checkSeats(state),
    ...checkBounds(state),
    ...checkPlaces(state, tuning.layout),
    ...checkSeparation(state, tuning),
    ...checkRestingInPlace(state, tuning.layout),
  ]
}
