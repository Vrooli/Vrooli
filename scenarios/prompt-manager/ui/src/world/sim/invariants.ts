/**
 * World invariants: properties every settled world state must satisfy,
 * checked without a renderer. Tests run them after simulated minutes; the
 * smoke tool runs them on the live page. A violation names the rule and the
 * offending ids so a failure reads as a sentence, not a diff.
 *
 * Rules:
 *   seat-occupancy      occupancy is a bijection and matches actor.seatId
 *   seat-exists         every seat an actor references exists
 *   inside-world        every actor stands inside the generated world
 *   desk-in-room        every desk and table sits inside its parent room
 *   sites-disjoint      site pads do not overlap each other or the commons
 *   separation          two standing actors are never closer than a body width
 *   resting-in-place    an idle actor that is not walking stands somewhere meaningful
 */
import type { ActorTuning, LayoutTuning, TerrainTuning } from '../config'
import type { Actor, Place, Vec2, WorldState } from './model'
import { COMMONS_ID } from './layout/generate'
import { heightAt } from './terrain'
import { findPath } from './nav/astar'
import { isWalkable, nearestWalkable } from './nav/grid'

export type InvariantRule =
  | 'seat-occupancy'
  | 'seat-exists'
  | 'inside-world'
  | 'desk-in-room'
  | 'sites-disjoint'
  | 'separation'
  | 'resting-in-place'
  | 'above-water'
  | 'site-level'
  | 'every-team-sited'
  | 'commons-reachable'
  | 'weather-defined'

export interface Violation {
  rule: InvariantRule
  /** Ids involved, most specific first. */
  ids: string[]
  detail: string
}

export interface InvariantTuning {
  layout: LayoutTuning
  actor: Pick<ActorTuning, 'bodyRadius'>
  terrain?: TerrainTuning
}

const HALF = 0.5

function distance(a: Vec2, b: Vec2): number {
  return Math.hypot(a[0] - b[0], a[1] - b[1])
}

function insideRect(point: Vec2, place: Place): boolean {
  const dx = point[0] - place.position[0]
  const dz = point[1] - place.position[1]
  const cos = Math.cos(place.rotation)
  const sin = Math.sin(place.rotation)
  const localX = dx * cos - dz * sin
  const localZ = dx * sin + dz * cos
  return Math.abs(localX) <= place.size[0] * HALF && Math.abs(localZ) <= place.size[1] * HALF
}

/** Whether a disc touches a place footprint: nearest point of the rectangle to the centre lies inside the radius. */
function rectDiscOverlap(rect: Place, center: Vec2, radius: number): boolean {
  const worldX = center[0] - rect.position[0]
  const worldZ = center[1] - rect.position[1]
  const cos = Math.cos(rect.rotation)
  const sin = Math.sin(rect.rotation)
  const dx = Math.max(Math.abs(worldX * cos - worldZ * sin) - rect.size[0] * HALF, 0)
  const dz = Math.max(Math.abs(worldX * sin + worldZ * cos) - rect.size[1] * HALF, 0)
  return dx * dx + dz * dz < radius * radius
}

function rectsOverlap(a: Place, b: Place): boolean {
  const axes: Vec2[] = [
    [Math.cos(a.rotation), -Math.sin(a.rotation)],
    [Math.sin(a.rotation), Math.cos(a.rotation)],
    [Math.cos(b.rotation), -Math.sin(b.rotation)],
    [Math.sin(b.rotation), Math.cos(b.rotation)],
  ]
  const delta: Vec2 = [b.position[0] - a.position[0], b.position[1] - a.position[1]]
  return axes.every((axis) => {
    const distance = Math.abs(delta[0] * axis[0] + delta[1] * axis[1])
    const radius = (Math.abs(axis[0] * Math.cos(a.rotation) - axis[1] * Math.sin(a.rotation)) * a.size[0] + Math.abs(axis[0] * Math.sin(a.rotation) + axis[1] * Math.cos(a.rotation)) * a.size[1] + Math.abs(axis[0] * Math.cos(b.rotation) - axis[1] * Math.sin(b.rotation)) * b.size[0] + Math.abs(axis[0] * Math.sin(b.rotation) + axis[1] * Math.cos(b.rotation)) * b.size[1]) * HALF
    return distance < radius
  })
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
  for (const actor of actors(state)) {
    if (Math.hypot(actor.position[0], actor.position[1]) >= state.terrain.radius || !isWalkable(state.nav, actor.position)) {
      out.push({ rule: 'inside-world', ids: [actor.id], detail: `${actor.id} at (${actor.position[0].toFixed(1)}, ${actor.position[1].toFixed(1)}) is outside the generated world` })
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
      if (b && rectsOverlap(a, b)) out.push({ rule: 'sites-disjoint', ids: [a.id, b.id], detail: `${a.id} overlaps ${b.id}` })
    }
    if (commons && rectDiscOverlap(a, commons.position, layout.commonsRadius)) {
      out.push({ rule: 'sites-disjoint', ids: [a.id, COMMONS_ID], detail: `${a.id} overlaps the commons` })
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

export function checkAboveWater(state: WorldState, terrain: TerrainTuning): Violation[] {
  const out: Violation[] = []
  const places = state.placeOrder.map((id) => state.places[id]).filter((value): value is Place => Boolean(value))
  for (const place of places) {
    const points: Array<{ id: string; point: Vec2 }> = [{ id: place.id, point: place.position }]
    for (const seat of place.seats) points.push({ id: seat.id, point: seat.position })
    for (const { id, point } of points) {
      if (heightAt(state.terrain, point[0], point[1]) < terrain.waterLevel) out.push({ rule: 'above-water', ids: [id], detail: `${id} lies below the water surface` })
    }
  }
  for (const actor of actors(state)) {
    actor.path.forEach((point, index) => {
      if (heightAt(state.terrain, point[0], point[1]) < terrain.waterLevel) out.push({ rule: 'above-water', ids: [actor.id, String(index)], detail: `${actor.id} path waypoint ${index} lies below the water surface` })
    })
  }
  return out
}

export function checkSites(state: WorldState, terrain: TerrainTuning): Violation[] {
  const out: Violation[] = []
  const rooms = state.placeOrder.map((id) => state.places[id]).filter((place): place is Place => place?.kind === 'room')
  const roomTeams = new Map<string, number>()
  for (const room of rooms) if (room.teamId) roomTeams.set(room.teamId, (roomTeams.get(room.teamId) ?? 0) + 1)
  const actorTeams = new Set(actors(state).filter((actor) => Boolean(actor.deskSeatId)).map((actor) => actor.teamId).filter((id): id is string => Boolean(id)))
  for (const teamId of actorTeams) {
    if (roomTeams.get(teamId) !== 1) out.push({ rule: 'every-team-sited', ids: [teamId], detail: `${teamId} has ${roomTeams.get(teamId) ?? 0} sites` })
  }
  for (const room of rooms) {
    let minimum = Infinity
    let maximum = -Infinity
    const spacing = state.terrain.cellSize
    const stepsX = Math.ceil(room.size[0] / spacing)
    const stepsZ = Math.ceil(room.size[1] / spacing)
    for (let iz = 0; iz <= stepsZ; iz += 1) for (let ix = 0; ix <= stepsX; ix += 1) {
      const localX = -room.size[0] / 2 + (ix / Math.max(1, stepsX)) * room.size[0]
      const localZ = -room.size[1] / 2 + (iz / Math.max(1, stepsZ)) * room.size[1]
      const cos = Math.cos(room.rotation)
      const sin = Math.sin(room.rotation)
      const x = room.position[0] + localX * cos + localZ * sin
      const z = room.position[1] - localX * sin + localZ * cos
      const height = heightAt(state.terrain, x, z)
      minimum = Math.min(minimum, height)
      maximum = Math.max(maximum, height)
    }
    if (maximum - minimum > terrain.siteLevelTolerance) out.push({ rule: 'site-level', ids: [room.id], detail: `${room.id} varies ${(maximum - minimum).toFixed(3)} m across its pad` })
  }
  const commons = state.places[COMMONS_ID]
  if (commons) {
    for (const seat of Object.values(state.seats).filter((seat) => seat.id.startsWith('seat:desk:'))) {
      const start = nearestWalkable(state.nav, seat.position, 6)
      const goal = nearestWalkable(state.nav, commons.position, 6)
      if (!start || !goal || !findPath(state.nav, start, goal)) out.push({ rule: 'commons-reachable', ids: [seat.id, COMMONS_ID], detail: `${seat.id} cannot reach the commons` })
    }
  }
  return out
}

export function checkWeather(state: WorldState): Violation[] {
  const knownStates = new Set<unknown>(['clear', 'cloudy', 'rain', 'snow'])
  return knownStates.has(state.weather.state)
    ? []
    : [{ rule: 'weather-defined', ids: [state.weather.state], detail: `unknown weather state ${state.weather.state}` }]
}

/** Every rule at once. Empty means the state is well formed. */
export function checkWorldInvariants(state: WorldState, tuning: InvariantTuning): Violation[] {
  return [
    ...checkSeats(state),
    ...checkBounds(state),
    ...checkPlaces(state, tuning.layout),
    ...checkSeparation(state, tuning),
    ...checkRestingInPlace(state, tuning.layout),
    ...(tuning.terrain ? checkAboveWater(state, tuning.terrain) : []),
    ...(tuning.terrain ? checkSites(state, tuning.terrain) : []),
    ...checkWeather(state),
  ]
}
