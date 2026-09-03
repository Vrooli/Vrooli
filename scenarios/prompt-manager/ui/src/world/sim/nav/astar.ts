import type { NavGrid, Vec2 } from '../model'
import { cellIndex, cellToWorld, inGrid, isCellWalkable, worldToCell } from './grid'

const HALF = 0.5
const DIAGONAL = Math.SQRT2

/** Bounded LRU cache of paths keyed by start and goal cell. */
export class PathCache {
  private readonly map = new Map<string, Vec2[]>()

  constructor(private readonly capacity: number) {}

  get(key: string): Vec2[] | undefined {
    const hit = this.map.get(key)
    if (hit) {
      this.map.delete(key)
      this.map.set(key, hit)
    }
    return hit
  }

  set(key: string, value: Vec2[]): void {
    if (this.map.has(key)) this.map.delete(key)
    this.map.set(key, value)
    if (this.map.size > this.capacity) {
      const oldest = this.map.keys().next().value
      if (oldest !== undefined) this.map.delete(oldest)
    }
  }

  get size(): number {
    return this.map.size
  }

  clear(): void {
    this.map.clear()
  }
}

function heuristic(c0: number, r0: number, c1: number, r1: number): number {
  const dc = Math.abs(c0 - c1)
  const dr = Math.abs(r0 - r1)
  return Math.max(dc, dr) + (DIAGONAL - 1) * Math.min(dc, dr)
}

/** True when the straight segment between two cell centres crosses only walkable cells. */
export function lineOfSight(grid: NavGrid, a: Vec2, b: Vec2): boolean {
  const steps = Math.ceil(Math.hypot(b[0] - a[0], b[1] - a[1]) / (grid.cellSize * HALF))
  for (let i = 0; i <= steps; i += 1) {
    const t = steps === 0 ? 0 : i / steps
    const [c, r] = worldToCell(grid, [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t])
    if (!isCellWalkable(grid, c, r)) return false
  }
  return true
}

/** Drop waypoints that a straight line can skip; keeps paths natural. */
export function smoothPath(grid: NavGrid, path: Vec2[]): Vec2[] {
  if (path.length <= 2) return path
  const first = path[0]
  if (!first) return path
  const out: Vec2[] = [first]
  let anchor: Vec2 = first
  let i = 1
  while (i < path.length) {
    let j = path.length - 1
    while (j > i) {
      const candidate = path[j]
      if (candidate && lineOfSight(grid, anchor, candidate)) break
      j -= 1
    }
    const next: Vec2 = path[j] ?? anchor
    out.push(next)
    anchor = next
    i = j + 1
  }
  return out
}

/**
 * A* on the 8-neighbour grid without corner cutting. Returns world-space
 * waypoints ending at the exact goal, or null when the goal is unreachable.
 */
export function findPath(grid: NavGrid, from: Vec2, to: Vec2, cache?: PathCache): Vec2[] | null {
  const [sc, sr] = worldToCell(grid, from)
  const [gc, gr] = worldToCell(grid, to)
  if (!inGrid(grid, gc, gr) || !isCellWalkable(grid, gc, gr)) return null
  if (!inGrid(grid, sc, sr)) return null
  const key = `${sc},${sr}>${gc},${gr}`
  const cached = cache?.get(key)
  if (cached) return [...cached.slice(0, -1), to]
  if (sc === gc && sr === gr) return [to]

  const size = grid.cols * grid.rows
  const gScore = new Float64Array(size).fill(Infinity)
  const cameFrom = new Int32Array(size).fill(-1)
  const closed = new Uint8Array(size)
  const start = cellIndex(grid, sc, sr)
  const goal = cellIndex(grid, gc, gr)
  gScore[start] = 0
  // Binary heap of [f, index]
  const heap: Array<[number, number]> = [[heuristic(sc, sr, gc, gr), start]]
  const push = (item: [number, number]) => {
    heap.push(item)
    let i = heap.length - 1
    while (i > 0) {
      const p = (i - 1) >> 1
      const parent = heap[p]
      const child = heap[i]
      if (!parent || !child || parent[0] <= child[0]) break
      heap[p] = child
      heap[i] = parent
      i = p
    }
  }
  const pop = (): [number, number] | undefined => {
    const top = heap[0]
    const last = heap.pop()
    if (heap.length > 0 && last) {
      heap[0] = last
      let i = 0
      for (;;) {
        const l = i * 2 + 1
        const r = l + 1
        let m = i
        const mi = heap[m]
        const li = heap[l]
        const ri = heap[r]
        if (li && mi && li[0] < mi[0]) m = l
        const mm = heap[m]
        if (ri && mm && ri[0] < mm[0]) m = r
        if (m === i) break
        const a = heap[i]
        const b = heap[m]
        if (!a || !b) break
        heap[i] = b
        heap[m] = a
        i = m
      }
    }
    return top
  }

  while (heap.length > 0) {
    const current = pop()
    if (!current) break
    const index = current[1]
    if (closed[index]) continue
    if (index === goal) break
    closed[index] = 1
    const c = index % grid.cols
    const r = (index - c) / grid.cols
    for (let dr = -1; dr <= 1; dr += 1) {
      for (let dc = -1; dc <= 1; dc += 1) {
        if (dr === 0 && dc === 0) continue
        const nc = c + dc
        const nr = r + dr
        if (!isCellWalkable(grid, nc, nr)) continue
        if (dr !== 0 && dc !== 0 && (!isCellWalkable(grid, c + dc, r) || !isCellWalkable(grid, c, r + dr))) continue
        const ni = cellIndex(grid, nc, nr)
        if (closed[ni]) continue
        const tentative = (gScore[index] ?? Infinity) + (dr !== 0 && dc !== 0 ? DIAGONAL : 1)
        if (tentative < (gScore[ni] ?? Infinity)) {
          gScore[ni] = tentative
          cameFrom[ni] = index
          push([tentative + heuristic(nc, nr, gc, gr), ni])
        }
      }
    }
  }
  if (cameFrom[goal] === -1 && goal !== start) return null
  const cells: Vec2[] = []
  let cursor = goal
  while (cursor !== -1 && cursor !== start) {
    const c = cursor % grid.cols
    cells.push(cellToWorld(grid, c, (cursor - c) / grid.cols))
    cursor = cameFrom[cursor] ?? -1
  }
  cells.push(from)
  cells.reverse()
  const smoothed = smoothPath(grid, cells)
  smoothed[smoothed.length - 1] = to
  cache?.set(key, smoothed)
  return smoothed
}
