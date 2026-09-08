/**
 * Layout generation: place is state.
 *
 * From the team graph: one campsite per team, one sleeping station per member,
 * authored shelters and gathering furniture, shared commons and a runs board.
 * Existing room/desk identifiers remain stable for saved layouts and actors.
 * Everything is keyed by ids
 * (team id, agent id) so the world is stable across reloads and renames.
 * Operator overrides are applied on top by place id.
 */
import type { BiomeSet, LayoutTuning, TerrainResolver } from '../../config'
import type { AgentInput, DecorSpot, LayoutOverride, Place, Seat, TeamInput, Vec2, WorldBounds } from '../model'
import type { TerrainField } from '../terrain'
import { fitSavedSiteSteps, selectSitesSteps, snappedRotation, type Site } from './sites'
import { terraceSiteSteps } from './terrace'
import { sortSteps } from '../cooperative'
import { scatterDecorSteps } from './scatter'
import { campsiteFor, campsiteStation, campsiteSize, spacePoint } from './spaces'
import { architecture } from '../../config/architecture'

export interface GeneratedLayout {
  places: Place[]
  bounds: WorldBounds
  decor: DecorSpot[]
  /** agentId -> desk seat id */
  deskSeatByAgent: Record<string, string>
}

export interface GenerateOptions {
  seed: number
  scatterDecor: boolean
  /** Number of tree prop variants the scene offers. */
  treeVariants: number
  clearPoints?: Vec2[]
  overrides?: LayoutOverride[]
  terrain: TerrainField
  terrainTuning: TerrainResolver
  biomes?: Uint8Array
  biomeSet?: BiomeSet
  gatheringLabel?: string
  fillerIds?: readonly string[]
}

const HALF = 0.5
const FACING_BACK = Math.PI
const FACING_FRONT = 0

export function roomId(teamId: string): string {
  return `room:${teamId}`
}
export function deskId(agentId: string): string {
  return `desk:${agentId}`
}
export function deskSeatId(agentId: string): string {
  return `seat:desk:${agentId}`
}
export function tableId(teamId: string): string {
  return `table:${teamId}`
}
export const GATHERING_ID = 'gathering'
export const HEARTH_ID = 'hearth'
export const BOARD_ID = 'board'

function facingToward(from: Vec2, to: Vec2): number {
  return Math.atan2(to[0] - from[0], to[1] - from[1])
}

function ringSeats(placeId: string, center: Vec2, radius: number, count: number, sitting: boolean, prefix: string): Seat[] {
  const seats: Seat[] = []
  for (let i = 0; i < count; i += 1) {
    const angle = (i / count) * Math.PI * 2
    const position: Vec2 = [center[0] + Math.sin(angle) * radius, center[1] + Math.cos(angle) * radius]
    seats.push({ id: `${prefix}:${i}`, placeId, position, facing: facingToward(position, center), sitting })
  }
  return seats
}

/**
 * Generate the layout. Teams are ordered by id so the grid never depends on
 * API ordering; members keep their team order for desk placement.
 */
export function generateLayout(...args: Parameters<typeof generateLayoutSteps>): GeneratedLayout {
  const steps = generateLayoutSteps(...args)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* generateLayoutSteps(teams: TeamInput[], agents: AgentInput[], layout: LayoutTuning, options: GenerateOptions): Generator<{ completed: number; total: number; operation?: 'assembly' | 'ranking' | 'median' }, GeneratedLayout> {
  const agentById = new Map<string, AgentInput>()
  for (const [index, agent] of agents.entries()) {
    if (index % 128 === 0) yield { completed: index, total: agents.length }
    if (!agentById.has(agent.id)) agentById.set(agent.id, agent)
  }
  const orderedTeams = yield* sortSteps(teams, (a, b) => a.id.localeCompare(b.id))
  const membersByTeam = new Map<string, string[]>()
  const places: Place[] = []
  const deskSeatByAgent: Record<string, string> = {}

  // The seed chooses buildable ground first; the org graph only assigns teams
  // to those stable sites in team-id order.
  const teamSizes: Array<{ width: number; depth: number; columns: number }> = []
  const siteSizes: Vec2[] = []
  for (const [index, team] of orderedTeams.entries()) {
    if (index % 128 === 0) yield { completed: index, total: orderedTeams.length }
    const members: string[] = []
    for (const [memberIndex, id] of team.memberIds.entries()) {
      if (memberIndex % 128 === 0) yield { completed: memberIndex, total: team.memberIds.length }
      if (agentById.has(id)) members.push(id)
    }
    membersByTeam.set(team.id, members)
    const [width, depth] = campsiteSize(members.length)
    const columns = Math.ceil(Math.sqrt(Math.max(1, members.length)))
    teamSizes.push({ width, depth, columns })
    siteSizes.push([width, depth])
  }
  const selected = yield* selectSitesSteps(options.terrain, { layout, terrain: options.terrainTuning }, siteSizes, options.seed)
  const commonsSite = selected.commons
  const teamSites = selected.sites

  for (const [index, team] of orderedTeams.entries()) {
    yield { completed: index, total: orderedTeams.length, operation: 'assembly' }
    const site = teamSites[index]
    if (!site) throw new Error(`site-selection: missing site for ${team.id}`)
    const center = site.position
    const teamSize = teamSizes[index] ?? { width: layout.roomWidth, depth: layout.roomDepth, columns: 1 }
    const roomWidth = teamSize.width
    const roomDepth = teamSize.depth
    const members = membersByTeam.get(team.id) ?? []
    const room: Place = {
      id: roomId(team.id),
      kind: 'room',
      teamId: team.id,
      position: center,
      rotation: site.rotation,
      size: [roomWidth, roomDepth],
      seats: [],
      label: team.name,
      space: campsiteFor(team.id, members, [roomWidth, roomDepth]),
    }
    places.push(room)

    const space = room.space
    if (!space) throw new Error('Campsite template is missing')
    for (const [m, agentId] of members.entries()) {
      if (m % 128 === 0) yield { completed: m, total: members.length, operation: 'assembly' }
      const station = campsiteStation(space, m)
      const seatPosition = spacePoint(room, station.seat)
      const deskPosition = spacePoint(room, station.bed)
      const seat: Seat = {
        id: deskSeatId(agentId),
        placeId: deskId(agentId),
        position: seatPosition,
        facing: FACING_BACK + station.rotation + site.rotation,
        sitting: false,
      }
      places.push({
        id: deskId(agentId),
        kind: 'desk',
        teamId: team.id,
        ownerAgentId: agentId,
        parentId: room.id,
        position: deskPosition,
        rotation: FACING_FRONT + station.rotation + site.rotation,
        size: architecture.bedrollSize,
        seats: [seat],
        label: agentById.get(agentId)?.name ?? agentId,
      })
      deskSeatByAgent[agentId] = seat.id
    }

    const firePosition = spacePoint(room, space.gathering)
    places.push({ id: `hearth:${team.id}`, kind: 'hearth', teamId: team.id, parentId: room.id,
      position: firePosition, rotation: room.rotation, size: [architecture.fireRadius * 2, architecture.fireRadius * 2],
      seats: ringSeats(`hearth:${team.id}`, firePosition, architecture.fireSeatRadius, architecture.fireSeats, true, `seat:fire:${team.id}`), label: `${team.name} campfire` })
    const picnic = spacePoint(room, [-roomWidth / 2 + layout.tableSeatRadius + .4, roomDepth / 2 - architecture.gatheringDepth])
    places.push({ id: tableId(team.id), kind: 'table', teamId: team.id, parentId: room.id,
      position: picnic, rotation: room.rotation, size: [layout.tableRadius * 2, layout.tableRadius * 2],
      seats: ringSeats(tableId(team.id), picnic, layout.tableSeatRadius, layout.tableSeats, true, `seat:${tableId(team.id)}`), label: `${team.name} picnic table` })
  }

  // Commons, campfire and board.
  const commonsCenter = commonsSite.position
  places.push({
    id: GATHERING_ID,
    kind: 'gathering',
    position: commonsCenter,
    rotation: 0,
    size: [layout.commonsRadius * 2, layout.commonsRadius * 2],
    seats: [],
    label: options.gatheringLabel ?? 'Commons',
  })
  places.push({
    id: HEARTH_ID,
    kind: 'hearth',
    parentId: GATHERING_ID,
    position: commonsCenter,
    rotation: 0,
    size: [layout.commonsSeatRadius * HALF, layout.commonsSeatRadius * HALF],
    seats: ringSeats(HEARTH_ID, commonsCenter, layout.commonsSeatRadius, layout.commonsSeats, true, 'seat:hearth'),
    label: 'Campfire',
  })
  places.push({ id: 'landmark', kind: 'filler', parentId: GATHERING_ID,
    position: [commonsCenter[0] - 3, commonsCenter[1] - 3], rotation: 0,
    size: [1.6, 1.6], seats: [], label: 'Campground sculpture' })
  // boardOffset is defined from the commons centre. Keeping the board inside
  // the terraced commons also guarantees that it remains above water.
  const boardPosition: Vec2 = [commonsCenter[0] + layout.boardOffset, commonsCenter[1]]
  places.push({
    id: BOARD_ID,
    kind: 'board',
    position: boardPosition,
    rotation: -Math.PI * HALF,
    size: [layout.deskInset, layout.deskPitch],
    seats: [],
    label: 'Runs board',
  })

  // Older saved positions did not include orientation. Derive it from the saved
  // site, not a newly generated candidate that can change when the team grows.
  const savedCommons = options.overrides?.find(override => override.placeId === GATHERING_ID)?.position ?? commonsCenter
  const overrides = (options.overrides ?? []).map(override => override.position && override.rotation === undefined && places.some(place => place.id === override.placeId && place.kind === 'room')
    ? { ...override, rotation: snappedRotation(override.position, savedCommons, layout.siteRotationSnapRad) } : override)
  yield* applyOverridesSteps(places, overrides)
  if (options.overrides?.length) {
    const pinned = new Set(options.overrides.map(override => override.placeId))
    const rooms = places.filter(place => place.kind === 'room')
    const gathering = places.find(place => place.kind === 'gathering')
    const occupied: Site[] = []
    // Saved placements take precedence; generated neighbors adapt around them.
    for (const saved of [true, false]) for (const room of rooms) {
      if (pinned.has(room.id) !== saved) continue
      const site = yield* fitSavedSiteSteps({ ...room, height: 0 }, occupied, gathering ? { ...gathering, height: 0 } : undefined, options.terrain, { layout, terrain: options.terrainTuning })
      if (site.position !== room.position) yield* applyOverridesSteps(places, [{ placeId: room.id, position: site.position }])
      occupied.push(site)
    }
  }
  // Ground support belongs to the final layout. Applying saved transforms after
  // terracing left the pad at the old position and put shelters on uneven ground.
  // Retain commons-first ordering so untouched layouts keep the same terraces.
  for (const kind of ['gathering', 'room'] as const) for (const place of places) {
    if (place.kind === kind) yield* terraceSiteSteps(options.terrain, options.terrainTuning, {
      position: place.position, rotation: place.rotation, size: place.size, height: 0,
    })
  }
  // A removed room takes its desks with it; members fall back to the commons.
  const seatIds = new Set<string>()
  for (const [index, place] of places.entries()) {
    if (index % 128 === 0) yield { completed: index, total: places.length }
    for (const [seatIndex, seat] of place.seats.entries()) {
      if (seatIndex % 128 === 0) yield { completed: seatIndex, total: place.seats.length }
      seatIds.add(seat.id)
    }
  }
  const survivingDesks: Record<string, string> = {}
  let deskIndex = 0
  for (const agentId in deskSeatByAgent) {
    if (deskIndex++ % 128 === 0) yield { completed: 0, total: 1 }
    const seatId = deskSeatByAgent[agentId]
    if (seatId && seatIds.has(seatId)) survivingDesks[agentId] = seatId
  }

  // Bounds from what was actually placed (after overrides), never smaller than the minimum slab.
  let minX = -layout.commonsRadius
  let maxX = layout.commonsRadius
  let minZ = -layout.commonsRadius
  let maxZ = layout.commonsRadius
  for (const [index, place] of places.entries()) {
    if (index % 128 === 0) yield { completed: index, total: places.length }
    const reach = Math.hypot(place.size[0], place.size[1]) * HALF
    minX = Math.min(minX, place.position[0] - reach)
    maxX = Math.max(maxX, place.position[0] + reach)
    minZ = Math.min(minZ, place.position[1] - reach)
    maxZ = Math.max(maxZ, place.position[1] + reach)
  }
  const width = options.terrain.radius * 2
  const depth = options.terrain.radius * 2
  const footprintCenter: Vec2 = [(minX + maxX) * HALF, (minZ + maxZ) * HALF]
  const center: Vec2 = [0, 0]
  const bounds: WorldBounds = { width, depth, center, footprint: { width: maxX - minX, depth: maxZ - minZ, center: footprintCenter }, outline: yield* outlinePointsSteps(places, layout) }

  const decor = options.scatterDecor && options.biomes && options.biomeSet
    ? yield* scatterDecorSteps({ field: options.terrain, tuning: options.terrainTuning, biomes: options.biomes, biomeSet: options.biomeSet, places, bounds, layout, seed: options.seed, clearPoints: options.clearPoints ?? [] })
    : []
  return { places, bounds, decor, deskSeatByAgent: survivingDesks }
}

/** Corners of every top-level place plus points around the commons rim; children (desks, tables) sit inside their room. */
export function outlinePoints(places: Place[], layout: LayoutTuning): Vec2[] {
  const steps = outlinePointsSteps(places, layout)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* outlinePointsSteps(places: Place[], layout: LayoutTuning): Generator<{ completed: number; total: number }, Vec2[]> {
  const points: Vec2[] = []
  for (const [index, place] of places.entries()) {
    if (index % 128 === 0) yield { completed: index, total: places.length }
    if (place.parentId) continue
    if (place.kind === 'gathering') {
      for (let i = 0; i < layout.outlineRimSamples; i += 1) {
        if (i % 128 === 0) yield { completed: i, total: layout.outlineRimSamples }
        const angle = (i / layout.outlineRimSamples) * Math.PI * 2
        points.push([place.position[0] + Math.sin(angle) * layout.commonsRadius, place.position[1] + Math.cos(angle) * layout.commonsRadius])
      }
      continue
    }
    const cos = Math.cos(place.rotation)
    const sin = Math.sin(place.rotation)
    for (const sx of [-1, 1]) {
      for (const sz of [-1, 1]) {
        const x = sx * place.size[0] * HALF
        const z = sz * place.size[1] * HALF
        points.push([place.position[0] + x * cos + z * sin, place.position[1] - x * sin + z * cos])
      }
    }
  }
  return points
}

/** Apply operator overrides by place id. Removed places drop with their seats; transformed places carry their children. */
export function applyOverrides(places: Place[], overrides: LayoutOverride[]): void {
  const steps = applyOverridesSteps(places, overrides)
  while (!steps.next().done) { /* synchronous compatibility API */ }
}

/** Only generation-owned places may be mutated; cancellation discards partial results. */
export function* applyOverridesSteps(places: Place[], overrides: LayoutOverride[]): Generator<{ completed: number; total: number }, void> {
  for (const override of overrides) {
    yield { completed: 0, total: places.length }
    let index = -1
    for (const [candidateIndex, candidate] of places.entries()) {
      if (candidateIndex % 128 === 0) yield { completed: candidateIndex, total: places.length }
      if (candidate.id === override.placeId) { index = candidateIndex; break }
    }
    if (index === -1) continue
    const place = places[index]
    if (!place) continue
    if (override.removed) {
      const removed = new Set([place.id])
      for (const [otherIndex, other] of places.entries()) {
        if (otherIndex % 128 === 0) yield { completed: otherIndex, total: places.length }
        if (other.parentId === place.id) removed.add(other.id)
      }
      let retained = 0
      for (let i = 0; i < places.length; i++) {
        if (i % 128 === 0) yield { completed: i, total: places.length }
        const candidate = places[i]
        if (candidate && !removed.has(candidate.id)) places[retained++] = candidate
      }
      places.length = retained
      continue
    }
    const origin = place.position
    const target = override.position ?? origin
    const rotationDelta = override.rotation === undefined ? 0 : override.rotation - place.rotation
    const transformPoint = (point: Vec2): Vec2 => {
      const localX = point[0] - origin[0]
      const localZ = point[1] - origin[1]
      const cos = Math.cos(rotationDelta)
      const sin = Math.sin(rotationDelta)
      return [target[0] + localX * cos + localZ * sin, target[1] - localX * sin + localZ * cos]
    }
    function* transform(p: Place): Generator<{ completed: number; total: number }, void> {
      p.position = p === place ? [target[0], target[1]] : transformPoint(p.position)
      p.rotation += rotationDelta
      for (const [seatIndex, seat] of p.seats.entries()) {
        if (seatIndex % 128 === 0) yield { completed: seatIndex, total: p.seats.length }
        seat.position = transformPoint(seat.position)
        seat.facing += rotationDelta
      }
    }
    yield* transform(place)
    for (const [childIndex, child] of places.entries()) {
      if (childIndex % 128 === 0) yield { completed: childIndex, total: places.length }
      if (child.parentId === place.id) yield* transform(child)
    }
  }
}
