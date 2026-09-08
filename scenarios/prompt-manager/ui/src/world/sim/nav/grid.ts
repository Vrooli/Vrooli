import type { TerrainResolver } from '../../config'
import type { DecorSpot, NavGrid, Place, Vec2, WorldBounds } from '../model'
import { shoreDistance, slopeAt, type TerrainField } from '../terrain'

const HALF = 0.5
const NAV_WORK_CHUNK = 128
interface NavProgress { completed: number; total: number }

export function worldToCell(grid: NavGrid, point: Vec2): [col: number, row: number] {
  return [Math.floor((point[0] - grid.originX) / grid.cellSize), Math.floor((point[1] - grid.originZ) / grid.cellSize)]
}

export function cellToWorld(grid: NavGrid, col: number, row: number): Vec2 {
  return [grid.originX + (col + HALF) * grid.cellSize, grid.originZ + (row + HALF) * grid.cellSize]
}

export function cellIndex(grid: NavGrid, col: number, row: number): number {
  return row * grid.cols + col
}

export function inGrid(grid: NavGrid, col: number, row: number): boolean {
  return col >= 0 && row >= 0 && col < grid.cols && row < grid.rows
}

export function isCellWalkable(grid: NavGrid, col: number, row: number): boolean {
  return inGrid(grid, col, row) && grid.walkable[cellIndex(grid, col, row)] === 1
}

export function isWalkable(grid: NavGrid, point: Vec2): boolean {
  const [c, r] = worldToCell(grid, point)
  return isCellWalkable(grid, c, r)
}

function* blockDisc(grid: NavGrid, center: Vec2, radius: number, progress: NavProgress): Generator<NavProgress> {
  let work = 0
  const [c0, r0] = worldToCell(grid, [center[0] - radius, center[1] - radius])
  const [c1, r1] = worldToCell(grid, [center[0] + radius, center[1] + radius])
  for (let r = Math.max(0, r0); r <= Math.min(grid.rows - 1, r1); r += 1) {
    for (let c = Math.max(0, c0); c <= Math.min(grid.cols - 1, c1); c += 1) {
      if (++work % NAV_WORK_CHUNK === 0) yield progress
      const [x, z] = cellToWorld(grid, c, r)
      const dx = x - center[0]
      const dz = z - center[1]
      if (dx * dx + dz * dz <= radius * radius) grid.walkable[cellIndex(grid, c, r)] = 0
    }
  }
}

function* blockRect(grid: NavGrid, center: Vec2, size: Vec2, rotation: number, progress: NavProgress): Generator<NavProgress> {
  let work = 0
  const cos = Math.cos(rotation)
  const sin = Math.sin(rotation)
  const reach = Math.hypot(size[0], size[1]) * HALF
  const [c0, r0] = worldToCell(grid, [center[0] - reach, center[1] - reach])
  const [c1, r1] = worldToCell(grid, [center[0] + reach, center[1] + reach])
  for (let r = Math.max(0, r0); r <= Math.min(grid.rows - 1, r1); r += 1) {
    for (let c = Math.max(0, c0); c <= Math.min(grid.cols - 1, c1); c += 1) {
      if (++work % NAV_WORK_CHUNK === 0) yield progress
      const [x, z] = cellToWorld(grid, c, r)
      const dx = x - center[0]
      const dz = z - center[1]
      const lx = dx * cos - dz * sin
      const lz = dx * sin + dz * cos
      if (Math.abs(lx) <= size[0] * HALF && Math.abs(lz) <= size[1] * HALF) grid.walkable[cellIndex(grid, c, r)] = 0
    }
  }
}

function* carveLine(grid: NavGrid, from: Vec2, to: Vec2, progress: NavProgress): Generator<NavProgress> {
  const distance = Math.hypot(to[0] - from[0], to[1] - from[1])
  const steps = Math.max(1, Math.ceil(distance / (grid.cellSize * HALF)))
  if (!Number.isSafeInteger(steps)) throw new Error('Navigation aisle length exceeds supported coordinates')
  // Clip the parameter interval, preserving the original sampling lattice.
  // Distant off-grid endpoints must not create billions of discarded samples.
  let enter = 0
  let leave = 1
  const dx = to[0] - from[0]
  const dz = to[1] - from[1]
  const edges = [[-dx, from[0] - grid.originX], [dx, grid.originX + grid.cols * grid.cellSize - from[0]], [-dz, from[1] - grid.originZ], [dz, grid.originZ + grid.rows * grid.cellSize - from[1]]]
  for (const [direction = 0, distance = 0] of edges) {
    if (direction === 0) { if (distance < 0) return; continue }
    const t = distance / direction
    if (direction < 0) enter = Math.max(enter, t)
    else leave = Math.min(leave, t)
    if (enter > leave) return
  }
  for (let step = Math.max(0, Math.ceil(enter * steps)); step <= Math.min(steps, Math.floor(leave * steps)); step += 1) {
    if (step % NAV_WORK_CHUNK === 0) yield progress
    const t = step / steps
    const [col, row] = worldToCell(grid, [from[0] + (to[0] - from[0]) * t, from[1] + (to[1] - from[1]) * t])
    if (inGrid(grid, col, row)) grid.walkable[cellIndex(grid, col, row)] = 1
  }
}

function offset(center: Vec2, localX: number, localZ: number, rotation: number): Vec2 {
  const cos = Math.cos(rotation)
  const sin = Math.sin(rotation)
  return [center[0] + localX * cos + localZ * sin, center[1] - localX * sin + localZ * cos]
}

function pathStrengthAt(terrain: TerrainField | undefined, mask: Float32Array | undefined, point: Vec2): number {
  if (!terrain || !mask) return 0
  const col = Math.round((point[0] - terrain.originX) / terrain.cellSize)
  const row = Math.round((point[1] - terrain.originZ) / terrain.cellSize)
  if (col < 0 || row < 0 || col >= terrain.cols || row >= terrain.rows) return 0
  return mask[row * terrain.cols + col] ?? 0
}

/**
 * Walkable grid over the slab. Blocked: desks, tables, the campfire, the
 * board, tree trunks and the three walls of every room (the front is open).
 * Actors themselves never block; they path around static props only.
 */
export const MAX_NAV_CELLS = 1_048_576

export function buildNavGrid(...args: Parameters<typeof buildNavGridSteps>): NavGrid {
  const steps = buildNavGridSteps(...args)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* buildNavGridSteps(bounds: WorldBounds, places: Place[], decor: DecorSpot[], cellSize: number, wallThickness: number, trunkRadius: number, terrain?: TerrainField, terrainTuning?: TerrainResolver, pathMask?: Float32Array): Generator<{ completed: number; total: number }, NavGrid> {
  if (![bounds.width, bounds.depth, cellSize].every(value => Number.isFinite(value) && value > 0) || !bounds.center.every(Number.isFinite)) throw new Error('Navigation bounds and cell size must be finite and positive')
  const cols = Math.max(1, Math.ceil(bounds.width / cellSize))
  const rows = Math.max(1, Math.ceil(bounds.depth / cellSize))
  if (!Number.isSafeInteger(cols * rows) || cols * rows > MAX_NAV_CELLS) throw new Error('Navigation grid exceeds the allocation budget')
  const total = (terrain && terrainTuning ? rows : 0) + places.length * 2 + decor.length
  let completed = 0
  const grid: NavGrid = {
    cellSize,
    cols,
    rows,
    originX: bounds.center[0] - bounds.width * HALF,
    originZ: bounds.center[1] - bounds.depth * HALF,
    walkable: new Uint8Array(cols * rows).fill(1),
  }
  if (terrain && terrainTuning) {
    for (let row = 0; row < grid.rows; row += 1) {
      for (let col = 0; col < grid.cols; col += 1) {
        if (col > 0 && col % NAV_WORK_CHUNK === 0) yield { completed, total }
        const [x, z] = cellToWorld(grid, col, row)
        const index = cellIndex(grid, col, row)
        const onPath = pathStrengthAt(terrain, pathMask, [x, z]) > 0
        const local = terrainTuning.at(x, z)
        if (shoreDistance(terrain, terrainTuning, x, z) < local.shoreMargin || (!onPath && slopeAt(terrain, x, z) > local.maxWalkSlope)) grid.walkable[index] = 0
      }
      yield { completed: ++completed, total }
    }
  }
  for (const place of places) {
    switch (place.kind) {
      case 'desk':
      case 'board':
        yield* blockRect(grid, place.position, place.size, place.rotation, { completed, total })
        break
      case 'table':
      case 'hearth':
        yield* blockDisc(grid, place.position, place.size[0] * HALF, { completed, total })
        break
      case 'room': {
        const [w, d] = place.size
        // Outdoor rooms keep an open front. Indoor rooms carry a door record;
        // split that fourth wall around the doorway gap.
        yield* blockRect(grid, offset(place.position, 0, -d * HALF, place.rotation), [w, wallThickness], place.rotation, { completed, total })
        yield* blockRect(grid, offset(place.position, -w * HALF, 0, place.rotation), [wallThickness, d], place.rotation, { completed, total })
        yield* blockRect(grid, offset(place.position, w * HALF, 0, place.rotation), [wallThickness, d], place.rotation, { completed, total })
        const door = places.find((candidate) => candidate.kind === 'door' && candidate.parentId === place.id)
        if (door) {
          const segmentWidth = Math.max(0, (w - door.size[0]) * HALF)
          const centerOffset = door.size[0] * HALF + segmentWidth * HALF
          yield* blockRect(grid, offset(place.position, -centerOffset, d * HALF, place.rotation), [segmentWidth, wallThickness], place.rotation, { completed, total })
          yield* blockRect(grid, offset(place.position, centerOffset, d * HALF, place.rotation), [segmentWidth, wallThickness], place.rotation, { completed, total })
        }
        break
      }
      default:
        break
    }
    yield { completed: ++completed, total }
  }
  // Seats are intentional navigable destinations. Restore narrow room aisles
  // after conservative furniture rasterisation: across each desk row, then
  // down the room centre to its open front.
  const rooms = new Map(places.filter((place) => place.kind === 'room').map((place) => [place.id, place]))
  for (const place of places) {
    for (const seat of place.seats) {
      const [col, row] = worldToCell(grid, seat.position)
      if (col >= 0 && col < grid.cols && row >= 0 && row < grid.rows) grid.walkable[cellIndex(grid, col, row)] = 1
      const room = place.parentId ? rooms.get(place.parentId) : undefined
      if (!room || place.kind !== 'desk') continue
      const dx = seat.position[0] - room.position[0]
      const dz = seat.position[1] - room.position[1]
      const localZ = dx * Math.sin(room.rotation) + dz * Math.cos(room.rotation)
      const aisle = offset(room.position, 0, localZ, room.rotation)
      const exit = offset(room.position, 0, room.size[1] * HALF + grid.cellSize, room.rotation)
      yield* carveLine(grid, seat.position, aisle, { completed, total })
      yield* carveLine(grid, aisle, exit, { completed, total })
    }
    yield { completed: ++completed, total }
  }
  for (const spot of decor) {
    if (spot.kind === 'tree' && pathStrengthAt(terrain, pathMask, spot.position) <= 0) yield* blockDisc(grid, spot.position, trunkRadius * spot.scale, { completed, total })
    yield { completed: ++completed, total }
  }
  return grid
}

/** Nearest walkable cell centre to a point, searching outward in rings. */
export function nearestWalkable(grid: NavGrid, point: Vec2, maxRings: number): Vec2 | null {
  const [c, r] = worldToCell(grid, point)
  if (isCellWalkable(grid, c, r)) return point
  for (let ring = 1; ring <= maxRings; ring += 1) {
    for (let dr = -ring; dr <= ring; dr += 1) {
      for (let dc = -ring; dc <= ring; dc += 1) {
        if (Math.max(Math.abs(dr), Math.abs(dc)) !== ring) continue
        if (isCellWalkable(grid, c + dc, r + dr)) return cellToWorld(grid, c + dc, r + dr)
      }
    }
  }
  return null
}
