import type { TerrainResolver } from '../../config'
import { blendHeight, smoothstep } from '../../config/regions'
import type { NavGrid, Vec2 } from '../model'
import { findPathSteps } from '../nav/astar'
import { nearestWalkable } from '../nav/grid'
import { type TerrainField } from '../terrain'
import type { Site } from './sites'

const PATH_WORK_CHUNK = 128

type TerraceProgress = { completed: number; total: number; operation?: 'median' }

/** Bottom-up merge passes bound comparison/copy work between cancellation points. */
function* medianSteps(samples: number[]): Generator<TerraceProgress, number> {
  const count = samples.length
  if (!count) return 0
  let source = samples
  let target = new Array<number>(count)
  let completed = 0
  const total = count * Math.ceil(Math.log2(count))
  for (let width = 1; width < count; width *= 2) {
    for (let start = 0; start < count; start += width * 2) {
      const middle = Math.min(start + width, count)
      const end = Math.min(start + width * 2, count)
      let left = start
      let right = middle
      for (let index = start; index < end; index++) {
        if (completed % PATH_WORK_CHUNK === 0) yield { completed, total, operation: 'median' }
        target[index] = left < middle && (right >= end || (source[left] ?? 0) <= (source[right] ?? 0))
          ? (source[left++] ?? 0) : (source[right++] ?? 0)
        completed++
      }
    }
    ;[source, target] = [target, source]
  }
  return source[Math.floor(count / 2)] ?? 0
}

function local(site: Site, x: number, z: number): Vec2 {
  const dx = x - site.position[0]
  const dz = z - site.position[1]
  const cos = Math.cos(site.rotation)
  const sin = Math.sin(site.rotation)
  return [dx * cos - dz * sin, dx * sin + dz * cos]
}

export function terraceSite(...args: Parameters<typeof terraceSiteSteps>): Site {
  const steps = terraceSiteSteps(...args)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

/** Mutates only generation-owned terrain; cancelling discards that unpublished field. */
export function* terraceSiteSteps(field: TerrainField, resolver: TerrainResolver, site: Site): Generator<TerraceProgress, Site> {
  const tuning = resolver.at(site.position[0], site.position[1])
  const samples: number[] = []
  for (let row = 0; row < field.rows; row += 1) {
    yield { completed: row, total: field.rows }
    for (let col = 0; col < field.cols; col += 1) {
      if (col % PATH_WORK_CHUNK === 0) yield { completed: row, total: field.rows }
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      const [lx, lz] = local(site, x, z)
      if (Math.abs(lx) <= site.size[0] / 2 && Math.abs(lz) <= site.size[1] / 2) samples.push(field.height[row * field.cols + col] ?? 0)
    }
  }
  const median = yield* medianSteps(samples)
  // Clear the maximum moisture basin bias plus a navigable dry-side margin.
  const padHeight = Math.max(median, tuning.waterLevel + tuning.padClearance)
  for (let row = 0; row < field.rows; row += 1) {
    yield { completed: row, total: field.rows }
    for (let col = 0; col < field.cols; col += 1) {
      if (col % PATH_WORK_CHUNK === 0) yield { completed: row, total: field.rows }
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      const [lx, lz] = local(site, x, z)
      // Keep one sample beyond the footprint flat so bilinear sampling and
      // navigation at a rotated pad edge cannot mix in the kerb slope.
      const outsideX = Math.max(0, Math.abs(lx) - site.size[0] / 2 - field.cellSize)
      const outsideZ = Math.max(0, Math.abs(lz) - site.size[1] / 2 - field.cellSize)
      const distance = Math.hypot(outsideX, outsideZ)
      const kerbWidth = resolver.at(x, z).kerbWidth
      if (distance > kerbWidth) continue
      const index = row * field.cols + col
      const blend = distance === 0 ? 1 : 1 - distance / kerbWidth
      const smooth = smoothstep(0, 1, blend)
      field.height[index] = blendHeight(field.height[index] ?? 0, padHeight, smooth)
    }
  }
  return { ...site, height: padHeight }
}

export function pathMask(...args: Parameters<typeof pathMaskSteps>): Float32Array {
  const steps = pathMaskSteps(...args)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* pathMaskSteps(field: TerrainField, resolver: TerrainResolver, nav: NavGrid, sites: readonly Site[], commons: Vec2): Generator<{ completed: number; total: number }, Float32Array> {
  const tuning = resolver.at(commons[0], commons[1])
  const mask = new Float32Array(field.cols * field.rows)
  const searchRings = Math.ceil((tuning.kerbWidth + tuning.shoreMargin + tuning.pathWidth) / nav.cellSize)
  const goal = nearestWalkable(nav, commons, searchRings)
  if (!goal) return mask
  let completed = 0
  for (const site of sites) {
    const progress = { completed, total: sites.length }
    yield progress
    const start = nearestWalkable(nav, site.position, searchRings)
    const search = start ? findPathSteps(nav, start, goal) : null
    let path: Vec2[] | null = null
    if (search) {
      try {
        let result = search.next()
        while (!result.done) { yield progress; result = search.next() }
        path = result.value
      } finally { search.return(null) }
    }
    completed++
    if (!path) continue
    for (let segment = 1; segment < path.length; segment += 1) {
      const from = path[segment - 1]
      const to = path[segment]
      if (!from || !to) continue
      const steps = Math.max(1, Math.ceil(Math.hypot(to[0] - from[0], to[1] - from[1]) / (field.cellSize * 0.5)))
      for (let step = 0; step <= steps; step += 1) {
        if (step % PATH_WORK_CHUNK === 0) yield progress
        const t = step / steps
        const x = from[0] + (to[0] - from[0]) * t
        const z = from[1] + (to[1] - from[1]) * t
        const tuning = resolver.at(x, z)
        const radius = Math.ceil(tuning.pathWidth / field.cellSize)
        const centerCol = Math.round((x - field.originX) / field.cellSize)
        const centerRow = Math.round((z - field.originZ) / field.cellSize)
        for (let dz = Math.max(-radius, -centerRow); dz <= Math.min(radius, field.rows - 1 - centerRow); dz += 1) for (let dx = Math.max(-radius, -centerCol); dx <= Math.min(radius, field.cols - 1 - centerCol); dx += 1) {
          if (dx % PATH_WORK_CHUNK === 0) yield progress
          const col = centerCol + dx
          const row = centerRow + dz
          if (col < 0 || row < 0 || col >= field.cols || row >= field.rows) continue
          const distance = Math.hypot(dx, dz) * field.cellSize
          const strength = Math.max(0, 1 - distance / tuning.pathWidth)
          const index = row * field.cols + col
          mask[index] = Math.max(mask[index] ?? 0, strength)
        }
      }
    }
  }
  yield { completed, total: sites.length }
  return mask
}
