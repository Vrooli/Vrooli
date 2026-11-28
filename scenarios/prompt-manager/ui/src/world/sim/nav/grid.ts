import { spaceStructures } from '../layout/spaces'
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

function pathStrengthAt(terrain: TerrainField | undefined, mask: Float32Array | undefined, point: Vec2): number {
  if (!terrain || !mask) return 0
  const col = Math.round((point[0] - terrain.originX) / terrain.cellSize)
  const row = Math.round((point[1] - terrain.originZ) / terrain.cellSize)
  if (col < 0 || row < 0 || col >= terrain.cols || row >= terrain.rows) return 0
  return mask[row * terrain.cols + col] ?? 0
}

/**
 * Walkable grid over the slab. Blocked: desks, tables, the campfire, the
 * board, tree trunks and the shared structural walls/furniture of every space.
 * Actors themselves never block; they path around static props only.
 */
export const MAX_NAV_CELLS = 1_048_576

export function buildNavGrid(...args: Parameters<typeof buildNavGridSteps>): NavGrid {
  const steps = buildNavGridSteps(...args)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* buildNavGridSteps(bounds: WorldBounds, places: Place[], decor: DecorSpot[], cellSize: number, trunkRadius: number, terrain?: TerrainField, terrainTuning?: TerrainResolver, pathMask?: Float32Array): Generator<{ completed: number; total: number }, NavGrid> {
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
      case 'filler':
        yield* blockRect(grid, place.position, place.size, place.rotation, { completed, total })
        break
      case 'table':
      case 'hearth':
        yield* blockDisc(grid, place.position, place.size[0] * HALF, { completed, total })
        break
      case 'room': {
        for (const box of spaceStructures(place)) {
          if (box.surface !== 'wall' && box.surface !== 'furniture') continue
          yield* blockRect(grid, [box.position[0], box.position[2]], [box.size[0] + cellSize, box.size[2] + cellSize], box.rotation, { completed, total })
        }
        break
      }
      default:
        break
    }
    yield { completed: ++completed, total }
  }
  // Seats remain intentional navigable destinations after furniture rasterization.
  for (const place of places) {
    for (const seat of place.seats) {
      const [col, row] = worldToCell(grid, seat.position)
      if (col >= 0 && col < grid.cols && row >= 0 && row < grid.rows) grid.walkable[cellIndex(grid, col, row)] = 1
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
