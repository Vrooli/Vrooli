import type { TerrainTuning } from '../../config'
import type { NavGrid, Vec2 } from '../model'
import { findPath } from '../nav/astar'
import { nearestWalkable } from '../nav/grid'
import { type TerrainField } from '../terrain'
import type { Site } from './sites'

function local(site: Site, x: number, z: number): Vec2 {
  const dx = x - site.position[0]
  const dz = z - site.position[1]
  const cos = Math.cos(site.rotation)
  const sin = Math.sin(site.rotation)
  return [dx * cos - dz * sin, dx * sin + dz * cos]
}

export function terraceSite(field: TerrainField, tuning: TerrainTuning, site: Site): Site {
  const samples: number[] = []
  for (let row = 0; row < field.rows; row += 1) {
    for (let col = 0; col < field.cols; col += 1) {
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      const [lx, lz] = local(site, x, z)
      if (Math.abs(lx) <= site.size[0] / 2 && Math.abs(lz) <= site.size[1] / 2) samples.push(field.height[row * field.cols + col] ?? 0)
    }
  }
  samples.sort((a, b) => a - b)
  const median = samples[Math.floor(samples.length / 2)] ?? 0
  // Clear the maximum moisture basin bias plus a navigable dry-side margin.
  const padHeight = Math.max(median, tuning.waterLevel + tuning.padClearance)
  for (let row = 0; row < field.rows; row += 1) {
    for (let col = 0; col < field.cols; col += 1) {
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      const [lx, lz] = local(site, x, z)
      // Keep one sample beyond the footprint flat so bilinear sampling and
      // navigation at a rotated pad edge cannot mix in the kerb slope.
      const outsideX = Math.max(0, Math.abs(lx) - site.size[0] / 2 - field.cellSize)
      const outsideZ = Math.max(0, Math.abs(lz) - site.size[1] / 2 - field.cellSize)
      const distance = Math.hypot(outsideX, outsideZ)
      if (distance > tuning.kerbWidth) continue
      const index = row * field.cols + col
      const blend = distance === 0 ? 1 : 1 - distance / tuning.kerbWidth
      const smooth = blend * blend * (3 - 2 * blend)
      field.height[index] = (field.height[index] ?? 0) * (1 - smooth) + padHeight * smooth
    }
  }
  return { ...site, height: padHeight }
}

export function pathMask(field: TerrainField, tuning: TerrainTuning, nav: NavGrid, sites: readonly Site[], commons: Vec2): Float32Array {
  const mask = new Float32Array(field.cols * field.rows)
  const searchRings = Math.ceil((tuning.kerbWidth + tuning.shoreMargin + tuning.pathWidth) / nav.cellSize)
  const goal = nearestWalkable(nav, commons, searchRings)
  if (!goal) return mask
  for (const site of sites) {
    const start = nearestWalkable(nav, site.position, searchRings)
    const path = start ? findPath(nav, start, goal) : null
    if (!path) continue
    for (let segment = 1; segment < path.length; segment += 1) {
      const from = path[segment - 1]
      const to = path[segment]
      if (!from || !to) continue
      const steps = Math.max(1, Math.ceil(Math.hypot(to[0] - from[0], to[1] - from[1]) / (field.cellSize * 0.5)))
      for (let step = 0; step <= steps; step += 1) {
        const t = step / steps
        const x = from[0] + (to[0] - from[0]) * t
        const z = from[1] + (to[1] - from[1]) * t
        const radius = Math.ceil(tuning.pathWidth / field.cellSize)
        const centerCol = Math.round((x - field.originX) / field.cellSize)
        const centerRow = Math.round((z - field.originZ) / field.cellSize)
        for (let dz = -radius; dz <= radius; dz += 1) for (let dx = -radius; dx <= radius; dx += 1) {
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
  return mask
}
