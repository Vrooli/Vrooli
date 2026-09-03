/**
 * Layout generation: place is state.
 *
 * From the team graph: one room per team, one desk per member along the
 * room's back wall, one table per team, a commons with a campfire for
 * unassigned and idle actors, and a runs board. Everything is keyed by ids
 * (team id, agent id) so the world is stable across reloads and renames.
 * Operator overrides are applied on top by place id.
 */
import type { LayoutTuning } from '../../config'
import type { AgentInput, DecorSpot, LayoutOverride, Place, Seat, TeamInput, Vec2, WorldBounds } from '../model'
import { Rng, hashString } from '../rng'

export interface GeneratedLayout {
  places: Place[]
  bounds: WorldBounds
  decor: DecorSpot[]
  /** agentId -> desk seat id */
  deskSeatByAgent: Record<string, string>
}

export interface GenerateOptions {
  seed: number
  trees: boolean
  /** Number of tree prop variants the scene offers. */
  treeVariants: number
  clearPoints?: Vec2[]
  overrides?: LayoutOverride[]
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
export const COMMONS_ID = 'commons'
export const CAMPFIRE_ID = 'campfire'
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
export function generateLayout(teams: TeamInput[], agents: AgentInput[], layout: LayoutTuning, options: GenerateOptions): GeneratedLayout {
  const agentIds = new Set(agents.map((a) => a.id))
  const orderedTeams = [...teams].sort((a, b) => a.id.localeCompare(b.id))
  const places: Place[] = []
  const deskSeatByAgent: Record<string, string> = {}

  // Rooms: a grid behind the commons, widest room decides the column pitch.
  const teamWidths = orderedTeams.map((team) => {
    const desks = team.memberIds.filter((id) => agentIds.has(id)).length
    return Math.max(layout.roomWidth, desks * layout.deskPitch + layout.deskInset * 2)
  })
  const roomWidth = teamWidths.length > 0 ? Math.max(...teamWidths) : layout.roomWidth
  const cols = Math.max(1, Math.min(orderedTeams.length, layout.maxRoomsPerRow))
  const rows = orderedTeams.length === 0 ? 0 : Math.ceil(orderedTeams.length / cols)
  const pitchX = roomWidth + layout.roomGap
  const pitchZ = layout.roomDepth + layout.roomGap
  const firstRowZ = -(layout.commonsRadius + layout.commonsGap + layout.roomDepth * HALF)

  orderedTeams.forEach((team, index) => {
    const col = index % cols
    const row = Math.floor(index / cols)
    const rowCount = row === rows - 1 ? orderedTeams.length - row * cols : cols
    const x = (col - (rowCount - 1) * HALF) * pitchX
    const z = firstRowZ - row * pitchZ
    const center: Vec2 = [x, z]
    const members = team.memberIds.filter((id) => agentIds.has(id))
    const room: Place = {
      id: roomId(team.id),
      kind: 'room',
      teamId: team.id,
      position: center,
      rotation: 0,
      size: [roomWidth, layout.roomDepth],
      seats: [],
      label: team.name,
    }
    places.push(room)

    const deskZ = z - layout.roomDepth * HALF + layout.deskInset
    members.forEach((agentId, m) => {
      const deskX = x + (m - (members.length - 1) * HALF) * layout.deskPitch
      const seat: Seat = {
        id: deskSeatId(agentId),
        placeId: deskId(agentId),
        position: [deskX, deskZ + layout.deskSeatOffset],
        facing: FACING_BACK,
        sitting: false,
      }
      places.push({
        id: deskId(agentId),
        kind: 'desk',
        teamId: team.id,
        ownerAgentId: agentId,
        parentId: room.id,
        position: [deskX, deskZ],
        rotation: FACING_FRONT,
        size: [layout.deskPitch * HALF, layout.deskInset],
        seats: [seat],
        label: agents.find((a) => a.id === agentId)?.name ?? agentId,
      })
      deskSeatByAgent[agentId] = seat.id
    })

    const tableCenter: Vec2 = [x, z + layout.roomDepth * HALF - layout.tableSeatRadius - layout.deskInset]
    places.push({
      id: tableId(team.id),
      kind: 'table',
      teamId: team.id,
      parentId: room.id,
      position: tableCenter,
      rotation: 0,
      size: [layout.tableRadius * 2, layout.tableRadius * 2],
      seats: ringSeats(tableId(team.id), tableCenter, layout.tableSeatRadius, layout.tableSeats, true, `seat:${tableId(team.id)}`),
      label: `${team.name} table`,
    })
  })

  // Commons, campfire and board.
  const commonsCenter: Vec2 = [0, 0]
  places.push({
    id: COMMONS_ID,
    kind: 'commons',
    position: commonsCenter,
    rotation: 0,
    size: [layout.commonsRadius * 2, layout.commonsRadius * 2],
    seats: [],
    label: 'Commons',
  })
  places.push({
    id: CAMPFIRE_ID,
    kind: 'campfire',
    parentId: COMMONS_ID,
    position: commonsCenter,
    rotation: 0,
    size: [layout.commonsSeatRadius * HALF, layout.commonsSeatRadius * HALF],
    seats: ringSeats(CAMPFIRE_ID, commonsCenter, layout.commonsSeatRadius, layout.commonsSeats, true, 'seat:campfire'),
    label: 'Campfire',
  })
  const boardPosition: Vec2 = [layout.commonsRadius + layout.boardOffset, 0]
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
  let maxX = layout.commonsRadius + layout.boardOffset + layout.deskPitch
  let minZ = -layout.commonsRadius
  let maxZ = layout.commonsRadius
  for (const place of places) {
    const reach = Math.hypot(place.size[0], place.size[1]) * HALF
    minX = Math.min(minX, place.position[0] - reach)
    maxX = Math.max(maxX, place.position[0] + reach)
    minZ = Math.min(minZ, place.position[1] - reach)
    maxZ = Math.max(maxZ, place.position[1] + reach)
  }
  const width = Math.max(layout.minSlabWidth, maxX - minX + layout.slabMargin * 2)
  const depth = Math.max(layout.minSlabDepth, maxZ - minZ + layout.slabMargin * 2)
  const center: Vec2 = [(minX + maxX) * HALF, (minZ + maxZ) * HALF]
  const bounds: WorldBounds = { width, depth, center, footprint: { width: maxX - minX, depth: maxZ - minZ, center }, outline: outlinePoints(places, layout) }

  const decor = options.trees ? scatterTrees(places, bounds, layout, options.seed, options.clearPoints ?? [], options.treeVariants) : []
  return { places, bounds, decor, deskSeatByAgent: survivingDesks }
}

/** Corners of every top-level place plus points around the commons rim; children (desks, tables) sit inside their room. */
export function outlinePoints(places: Place[], layout: LayoutTuning): Vec2[] {
  const points: Vec2[] = []
  for (const place of places) {
    if (place.parentId) continue
    if (place.kind === 'commons') {
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

/** Apply operator overrides by place id. Removed places drop with their seats; moved places carry their seats along. */
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
    const dx = override.position ? override.position[0] - place.position[0] : 0
    const dz = override.position ? override.position[1] - place.position[1] : 0
    const move = (p: Place) => {
      p.position = [p.position[0] + dx, p.position[1] + dz]
      for (const seat of p.seats) seat.position = [seat.position[0] + dx, seat.position[1] + dz]
    }
    move(place)
    for (const child of places) if (child.parentId === place.id) move(child)
    if (override.rotation !== undefined) place.rotation = override.rotation
  }
}

function distanceSq(a: Vec2, b: Vec2): number {
  const dx = a[0] - b[0]
  const dz = a[1] - b[1]
  return dx * dx + dz * dz
}

/** Blocked discs for tree scatter: every place plus the clearing radius, and the clear points. */
function clearances(places: Place[], layout: LayoutTuning, clearPoints: Vec2[]): Array<{ center: Vec2; radius: number }> {
  const discs = places.map((p) => ({
    center: p.position,
    radius: Math.hypot(p.size[0], p.size[1]) * HALF + layout.clearingRadius,
  }))
  for (const point of clearPoints) discs.push({ center: point, radius: layout.clearingRadius })
  return discs
}

/**
 * Deterministic tree scatter over the free slab ground: rejection sampling
 * with a minimum spacing, seeded from the layout seed so it is stable.
 */
export function scatterTrees(places: Place[], bounds: WorldBounds, layout: LayoutTuning, seed: number, clearPoints: Vec2[], variants: number): DecorSpot[] {
  const rng = new Rng(hashString(`trees:${seed}`))
  const discs = clearances(places, layout, clearPoints)
  const halfW = bounds.width * HALF - layout.treeMargin
  const halfD = bounds.depth * HALF - layout.treeMargin
  const area = Math.max(0, halfW * 2 * halfD * 2)
  const wanted = Math.floor(area * layout.treeDensity)
  const spacingSq = layout.treeMargin * layout.treeMargin
  const spots: DecorSpot[] = []
  const attempts = wanted * layout.treeAttemptsPerTree
  for (let i = 0; i < attempts && spots.length < wanted; i += 1) {
    const candidate: Vec2 = [bounds.center[0] + rng.range(-halfW, halfW), bounds.center[1] + rng.range(-halfD, halfD)]
    let blocked = false
    for (const disc of discs) {
      if (distanceSq(candidate, disc.center) < disc.radius * disc.radius) {
        blocked = true
        break
      }
    }
    if (blocked) continue
    for (const spot of spots) {
      if (distanceSq(candidate, spot.position) < spacingSq) {
        blocked = true
        break
      }
    }
    if (blocked) continue
    spots.push({
      id: `tree:${spots.length}`,
      kind: 'tree',
      variant: variants > 0 ? rng.int(variants) : 0,
      position: candidate,
      rotation: rng.range(0, Math.PI * 2),
      scale: rng.range(HALF + HALF * HALF, 1 + HALF * HALF),
    })
  }
  return spots
}
