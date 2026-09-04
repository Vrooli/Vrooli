/**
 * Layout generation: place is state.
 *
 * From the team graph: one room per team, one desk per member along the
 * room's back wall, one table per team, a commons with a campfire for
 * unassigned and idle actors, and a runs board. Everything is keyed by ids
 * (team id, agent id) so the world is stable across reloads and renames.
 * Operator overrides are applied on top by place id.
 */
import type { BiomeSet, LayoutTuning, TerrainTuning } from '../../config'
import type { AgentInput, DecorSpot, LayoutOverride, Place, Seat, TeamInput, Vec2, WorldBounds } from '../model'
import type { TerrainField } from '../terrain'
import { selectSites } from './sites'
import { terraceSite } from './terrace'
import { scatterDecor } from './scatter'
import { interiorDesks, interiorFor, interiorTablePosition } from './interior'

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
  terrainTuning: TerrainTuning
  biomes?: Uint8Array
  biomeSet?: BiomeSet
  treePropIds?: readonly string[]
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

function rotate(center: Vec2, localX: number, localZ: number, rotation: number): Vec2 {
  const cos = Math.cos(rotation)
  const sin = Math.sin(rotation)
  return [center[0] + localX * cos + localZ * sin, center[1] - localX * sin + localZ * cos]
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
export function generateLayout(teams: TeamInput[], agents: AgentInput[], layout: LayoutTuning, options: GenerateOptions): GeneratedLayout {
  const agentIds = new Set(agents.map((a) => a.id))
  const orderedTeams = [...teams].sort((a, b) => a.id.localeCompare(b.id))
  const places: Place[] = []
  const deskSeatByAgent: Record<string, string> = {}

  // The seed chooses buildable ground first; the org graph only assigns teams
  // to those stable sites in team-id order.
  const teamSizes = orderedTeams.map((team) => {
    const desks = team.memberIds.filter((id) => agentIds.has(id)).length
    const columns = Math.max(1, Math.ceil(Math.sqrt(desks)))
    const rows = Math.max(1, Math.ceil(desks / columns))
    const gridSpan = Math.max(columns, rows) * layout.deskPitch + layout.deskInset * 2
    const width = Math.max(layout.roomWidth, gridSpan + layout.tableSeatRadius * 2)
    const meetingDepth = layout.tableSeatRadius * 2 + layout.deskInset
    const depth = Math.max(layout.roomDepth, gridSpan + meetingDepth)
    return { width, depth, columns }
  })
  const selected = selectSites(options.terrain, { layout, terrain: options.terrainTuning }, teamSizes.map(({ width, depth }) => [width, depth]), options.seed)
  const commonsSite = terraceSite(options.terrain, options.terrainTuning, selected.commons)
  const teamSites = selected.sites.map((site) => terraceSite(options.terrain, options.terrainTuning, site))

  orderedTeams.forEach((team, index) => {
    const site = teamSites[index]
    if (!site) throw new Error(`site-selection: missing site for ${team.id}`)
    const center = site.position
    const teamSize = teamSizes[index] ?? { width: layout.roomWidth, depth: layout.roomDepth, columns: 1 }
    const roomWidth = teamSize.width
    const roomDepth = teamSize.depth
    const members = team.memberIds.filter((id) => agentIds.has(id))
    const room: Place = {
      id: roomId(team.id),
      kind: 'room',
      teamId: team.id,
      position: center,
      rotation: site.rotation,
      size: [roomWidth, roomDepth],
      seats: [],
      label: team.name,
    }
    places.push(room)

    const interior = interiorFor(options.seed, team.id, members.length, room.size, layout, options.fillerIds?.length)
    const deskLayout = interiorDesks(interior, members.length, room.size, layout)

    members.forEach((agentId, m) => {
      const desk = deskLayout[m]
      if (!desk) return
      const deskPosition = rotate(center, desk.position[0], desk.position[1], site.rotation)
      const seatPosition = rotate(center, desk.seat[0], desk.seat[1], site.rotation)
      const seat: Seat = {
        id: deskSeatId(agentId),
        placeId: deskId(agentId),
        position: seatPosition,
        facing: FACING_BACK + site.rotation + desk.rotation,
        sitting: false,
      }
      places.push({
        id: deskId(agentId),
        kind: 'desk',
        teamId: team.id,
        ownerAgentId: agentId,
        parentId: room.id,
        position: deskPosition,
        rotation: FACING_FRONT + site.rotation + desk.rotation,
        size: [layout.deskPitch * HALF, layout.deskInset],
        seats: [seat],
        label: agents.find((a) => a.id === agentId)?.name ?? agentId,
      })
      deskSeatByAgent[agentId] = seat.id
    })

    const tableLocal = interiorTablePosition(interior, room.size, layout)
    if (tableLocal) {
      const tableCenter = rotate(center, tableLocal[0], tableLocal[1], site.rotation)
      places.push({
      id: tableId(team.id),
      kind: 'table',
      teamId: team.id,
      parentId: room.id,
      position: tableCenter,
      rotation: site.rotation,
      size: [layout.tableRadius * 2, layout.tableRadius * 2],
      seats: ringSeats(tableId(team.id), tableCenter, layout.tableSeatRadius, layout.tableSeats, true, `seat:${tableId(team.id)}`),
      label: `${team.name} table`,
      })
    }
  })

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

  applyOverrides(places, options.overrides ?? [])
  // A removed room takes its desks with it; members fall back to the commons.
  const seatIds = new Set(places.flatMap((p) => p.seats.map((seat) => seat.id)))
  const survivingDesks = Object.fromEntries(Object.entries(deskSeatByAgent).filter(([, seatId]) => seatIds.has(seatId)))

  // Bounds from what was actually placed (after overrides), never smaller than the minimum slab.
  let minX = -layout.commonsRadius
  let maxX = layout.commonsRadius
  let minZ = -layout.commonsRadius
  let maxZ = layout.commonsRadius
  for (const place of places) {
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
  const bounds: WorldBounds = { width, depth, center, footprint: { width: maxX - minX, depth: maxZ - minZ, center: footprintCenter }, outline: outlinePoints(places, layout) }

  const decor = options.scatterDecor && options.biomes && options.biomeSet
    ? scatterDecor({ field: options.terrain, biomes: options.biomes, biomeSet: options.biomeSet, places, bounds, layout, seed: options.seed, clearPoints: options.clearPoints ?? [], treePropIds: options.treePropIds ?? [] })
    : []
  for (const room of places.filter((place) => place.kind === 'room')) {
    if (!room.teamId || !options.fillerIds?.length) continue
    const members = orderedTeams.find((team) => team.id === room.teamId)?.memberIds.length ?? 0
    const interior = interiorFor(options.seed, room.teamId, members, room.size, layout, options.fillerIds.length)
    for (const filler of interior.fillers) {
      const propId = options.fillerIds[filler.propIndex % options.fillerIds.length]
      if (!propId) continue
      const position = rotate(room.position, filler.local[0], filler.local[1], room.rotation)
      decor.push({ id: `filler:${room.teamId}:${filler.index}`, kind: 'decor', propId, variant: filler.index, position, rotation: room.rotation + filler.rotation, scale: 1, roomId: room.id })
    }
  }
  return { places, bounds, decor, deskSeatByAgent: survivingDesks }
}

/** Corners of every top-level place plus points around the commons rim; children (desks, tables) sit inside their room. */
export function outlinePoints(places: Place[], layout: LayoutTuning): Vec2[] {
  const points: Vec2[] = []
  for (const place of places) {
    if (place.parentId) continue
    if (place.kind === 'gathering') {
      for (let i = 0; i < layout.outlineRimSamples; i += 1) {
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
  for (const override of overrides) {
    const index = places.findIndex((p) => p.id === override.placeId)
    if (index === -1) continue
    const place = places[index]
    if (!place) continue
    if (override.removed) {
      const removed = new Set([place.id])
      for (const other of places) if (other.parentId === place.id) removed.add(other.id)
      for (let i = places.length - 1; i >= 0; i -= 1) {
        const candidate = places[i]
        if (candidate && removed.has(candidate.id)) places.splice(i, 1)
      }
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
    const transform = (p: Place) => {
      p.position = p === place ? [target[0], target[1]] : transformPoint(p.position)
      p.rotation += rotationDelta
      for (const seat of p.seats) {
        seat.position = transformPoint(seat.position)
        seat.facing += rotationDelta
      }
    }
    transform(place)
    for (const child of places) if (child.parentId === place.id) transform(child)
  }
}
