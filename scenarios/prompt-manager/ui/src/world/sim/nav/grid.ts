import type { DecorSpot, NavGrid, Place, Vec2, WorldBounds } from '../model'

const HALF = 0.5

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

function blockDisc(grid: NavGrid, center: Vec2, radius: number): void {
  const [c0, r0] = worldToCell(grid, [center[0] - radius, center[1] - radius])
  const [c1, r1] = worldToCell(grid, [center[0] + radius, center[1] + radius])
  for (let r = r0; r <= r1; r += 1) {
    for (let c = c0; c <= c1; c += 1) {
      if (!inGrid(grid, c, r)) continue
      const [x, z] = cellToWorld(grid, c, r)
      const dx = x - center[0]
      const dz = z - center[1]
      if (dx * dx + dz * dz <= radius * radius) grid.walkable[cellIndex(grid, c, r)] = 0
    }
  }
}

function blockRect(grid: NavGrid, center: Vec2, size: Vec2, rotation: number): void {
  const cos = Math.cos(rotation)
  const sin = Math.sin(rotation)
  const reach = Math.hypot(size[0], size[1]) * HALF
  const [c0, r0] = worldToCell(grid, [center[0] - reach, center[1] - reach])
  const [c1, r1] = worldToCell(grid, [center[0] + reach, center[1] + reach])
  for (let r = r0; r <= r1; r += 1) {
    for (let c = c0; c <= c1; c += 1) {
      if (!inGrid(grid, c, r)) continue
      const [x, z] = cellToWorld(grid, c, r)
      const dx = x - center[0]
      const dz = z - center[1]
      const lx = dx * cos - dz * sin
      const lz = dx * sin + dz * cos
      if (Math.abs(lx) <= size[0] * HALF && Math.abs(lz) <= size[1] * HALF) grid.walkable[cellIndex(grid, c, r)] = 0
    }
  }
}

/**
 * Walkable grid over the slab. Blocked: desks, tables, the campfire, the
 * board, tree trunks and the three walls of every room (the front is open).
 * Actors themselves never block; they path around static props only.
 */
export function buildNavGrid(bounds: WorldBounds, places: Place[], decor: DecorSpot[], cellSize: number, wallThickness: number, trunkRadius: number): NavGrid {
  const cols = Math.max(1, Math.ceil(bounds.width / cellSize))
  const rows = Math.max(1, Math.ceil(bounds.depth / cellSize))
  const grid: NavGrid = {
    cellSize,
    cols,
    rows,
    originX: bounds.center[0] - bounds.width * HALF,
    originZ: bounds.center[1] - bounds.depth * HALF,
    walkable: new Uint8Array(cols * rows).fill(1),
  }
  for (const place of places) {
    switch (place.kind) {
      case 'desk':
      case 'board':
        blockRect(grid, place.position, place.size, place.rotation)
        break
      case 'table':
      case 'campfire':
        blockDisc(grid, place.position, place.size[0] * HALF)
        break
      case 'room': {
        const [w, d] = place.size
        const [x, z] = place.position
        // back wall and two side walls; the front (+z) stays open
        blockRect(grid, [x, z - d * HALF], [w, wallThickness], place.rotation)
        blockRect(grid, [x - w * HALF, z], [wallThickness, d], place.rotation)
        blockRect(grid, [x + w * HALF, z], [wallThickness, d], place.rotation)
        break
      }
      default:
        break
    }
  }
  for (const spot of decor) if (spot.kind === 'tree') blockDisc(grid, spot.position, trunkRadius * spot.scale)
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
