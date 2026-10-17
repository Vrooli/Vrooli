import type { NavGrid } from '../sim'
import { cellToWorld, MAX_NAV_CELLS } from '../sim/nav/grid'

/** Exact cell coverage; no downsampling that could hide a blocked cell. */
export function* navigationOverlaySteps(nav: NavGrid, height: (x: number, z: number) => number) {
  const count = nav.cols * nav.rows
  if (!Number.isSafeInteger(count) || count < 0 || count > MAX_NAV_CELLS || nav.walkable.length !== count) throw new Error('Invalid navigation overlay grid')
  const positions = new Float32Array(count * 3)
  const colors = new Float32Array(count * 3)
  let walkable = 0
  for (let index = 0; index < count; index++) {
    if (index % 128 === 0) yield { completed: index, total: count }
    const [x, z] = cellToWorld(nav, index % nav.cols, Math.floor(index / nav.cols))
    positions.set([x, height(x, z) + 0.05, z], index * 3)
    const open = nav.walkable[index] === 1
    if (open) walkable++
    colors.set(open ? [0.1, 1, 0.3] : [1, 0.15, 0.1], index * 3)
  }
  return { positions, colors, count, walkable, blocked: count - walkable, bytes: positions.byteLength + colors.byteLength,
    summary: `${walkable.toLocaleString()} walkable · ${(count - walkable).toLocaleString()} blocked` }
}
