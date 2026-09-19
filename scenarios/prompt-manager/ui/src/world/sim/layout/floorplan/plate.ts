import type { LayoutTuning } from '../../../config'
import type { Place, Vec2 } from '../../model'
import type { WorkProgress } from '../../cooperative'
import { architecture } from '../../../config/architecture'
import { rectanglesSeparated } from '../sites'
import { spacePoint } from '../spaces'

export interface Rect {
  x: number
  z: number
  width: number
  depth: number
}

export interface OfficeWingPlan {
  plate: Rect
  rooms: Array<Rect & { side: 'north' | 'south' }>
  corridors: Rect[]
  lounge: Rect
  kitchen: Rect
}

/** Retain a saved room's orientation and connect its doorway to an existing hall.
 * The access floor is also rendered as a ceiling in walking modes. Rooms and
 * passages are reserved together so later rooms cannot obstruct that entrance. */
export function* fitOfficeRoomSteps(room: Place, occupied: readonly Place[], halls: readonly Place[], radius: number, tuning: LayoutTuning): Generator<WorkProgress, { position: Vec2; passage: Place }> {
  const width = architecture.doorWidth + architecture.wallThickness * 2
  const direction: Vec2 = [Math.sin(room.rotation), Math.cos(room.rotation)]
  const evaluate = (position: Vec2) => {
    const candidate = { ...room, position }
    for (const x of [-room.size[0] / 2, room.size[0] / 2]) for (const z of [-room.size[1] / 2, room.size[1] / 2]) {
      if (Math.hypot(...spacePoint(candidate, [x, z])) > radius - tuning.cellSize) return undefined
    }
    if (occupied.some(other => !rectanglesSeparated(candidate, other, architecture.office.roomGap))
      || halls.some(hall => !rectanglesSeparated(candidate, hall, -1e-6))) return undefined
    const entrance = spacePoint(candidate, [0, room.size[1] / 2])
    let best: Place | undefined
    for (const hall of halls) {
      // Slab intersection against an inset hall ensures the whole access width
      // joins the existing floor, including when the room was saved rotated.
      let near = 0, far = Infinity
      for (const axis of [0, 1] as const) {
        const half = hall.size[axis] / 2 - width / 2
        const delta = hall.position[axis] - entrance[axis]
        if (Math.abs(direction[axis]) < 1e-8) {
          if (Math.abs(delta) > half + 1e-6) { far = -1; break }
        } else {
          const a = (delta - half) / direction[axis], b = (delta + half) / direction[axis]
          near = Math.max(near, Math.min(a, b)); far = Math.min(far, Math.max(a, b))
        }
      }
      if (near > far || far < 0 || near < 1e-6) continue
      const passage: Place = { id: `corridor:access:${room.id}`, parentId: room.id, kind: 'corridor',
        position: [entrance[0] + direction[0] * near / 2, entrance[1] + direction[1] * near / 2],
        rotation: room.rotation, size: [width, near], seats: [], label: `${room.label} entrance hall` }
      if (occupied.some(other => other.kind === 'room' && !rectanglesSeparated(passage, other, architecture.wallThickness))) continue
      if (!best || near < best.size[1]) best = passage
    }
    return best
  }
  const exact = evaluate(room.position)
  if (exact) return { position: room.position, passage: exact }
  const step = tuning.cellSize, limit = Math.ceil(radius * 2 / step)
  const origin: Vec2 = [Math.max(-radius, Math.min(radius, room.position[0])), Math.max(-radius, Math.min(radius, room.position[1]))]
  let best: { position: Vec2; passage: Place } | undefined, distance = Infinity, completed = 0
  for (let ring = 0; ring <= limit && ring * step <= distance; ring++) {
    // Enumerate the perimeter only, keeping the bounded search quadratic.
    const offsets: Vec2[] = ring === 0 ? [[0, 0]] : []
    for (let n = -ring; n < ring; n++) offsets.push([n, -ring], [ring, n], [-n, ring], [-ring, -n])
    for (const [dx, dz] of offsets) {
      if (completed++ % 128 === 0) yield { completed, total: (limit * 2 + 1) ** 2 }
      const nextDistance = Math.hypot(dx, dz) * step
      if (nextDistance >= distance) continue
      const position: Vec2 = [origin[0] + dx * step, origin[1] + dz * step]
      const passage = evaluate(position)
      if (passage) { best = { position, passage }; distance = nextDistance }
    }
  }
  if (!best) throw new Error('Saved office layout has no clear connection to its halls. Reset the layout or reduce its footprint.')
  return best
}

/** Rooms occupy paired corridor wings. A shared spine connects every wing. */
export function officeWings(roomSizes: readonly Vec2[], tuning: LayoutTuning): OfficeWingPlan {
  const { sharedWidth, roomGap, margin } = architecture.office
  const columns = Math.max(1, Math.ceil(Math.sqrt(roomSizes.length / 2)))
  const wings = Math.max(1, Math.ceil(roomSizes.length / (columns * 2)))
  const corridor = tuning.floorplan.corridorWidth
  const widths = Array.from({ length: columns }, (_, column) => Math.max(tuning.roomWidth,
    ...Array.from({ length: wings * 2 }, (_, index) => roomSizes[Math.floor(index / 2) * columns * 2 + column * 2 + index % 2]?.[0] ?? 0)))
  const depths = Array.from({ length: wings }, (_, wing) => [0, 1].map(side => {
    let depth = 0
    for (let column = 0; column < columns; column++) {
      depth = Math.max(depth, roomSizes[wing * columns * 2 + column * 2 + side]?.[1] ?? 0)
    }
    return depth
  }))
  const width = sharedWidth + corridor + widths.reduce((sum, value) => sum + value + roomGap, 0) + margin * 2
  const depth = Math.max(sharedWidth * 2 + corridor, depths.reduce((sum, value) => sum + (value[0] ?? 0) + (value[1] ?? 0) + corridor + roomGap, 0)) + margin * 2
  const spineX = -width / 2 + margin + sharedWidth + corridor / 2
  const corridors: Rect[] = [{ x: spineX, z: 0, width: corridor, depth: depth - margin * 2 }]
  const rooms: OfficeWingPlan['rooms'] = []
  let bottom = -depth / 2 + margin
  for (let wing = 0; wing < wings; wing++) {
    const northDepth = depths[wing]?.[0] ?? 0
    const southDepth = depths[wing]?.[1] ?? 0
    const z = bottom + northDepth + corridor / 2
    const end = width / 2 - margin
    corridors.push({ x: (spineX + end) / 2, z, width: end - spineX, depth: corridor })
    let left = spineX + corridor / 2 + roomGap
    for (let column = 0; column < columns; column++) {
      for (let side = 0; side < 2; side++) {
        const index = wing * columns * 2 + column * 2 + side
        const size = roomSizes[index]
        if (!size) continue
        rooms[index] = { x: left + size[0] / 2, z: z + (side === 0 ? -1 : 1) * (corridor / 2 + size[1] / 2), width: size[0], depth: size[1], side: side === 0 ? 'south' : 'north' }
      }
      left += (widths[column] ?? tuning.roomWidth) + roomGap
    }
    bottom += northDepth + southDepth + corridor + roomGap
  }
  const sharedX = -width / 2 + margin + sharedWidth / 2
  return { plate: { x: 0, z: 0, width, depth }, rooms, corridors,
    lounge: { x: sharedX, z: -sharedWidth / 2 - corridor / 2, width: sharedWidth, depth: sharedWidth },
    kitchen: { x: sharedX, z: sharedWidth / 2 + corridor / 2, width: sharedWidth, depth: sharedWidth } }
}
