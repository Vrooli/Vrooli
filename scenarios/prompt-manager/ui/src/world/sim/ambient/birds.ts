import { ambientPolicy } from '../../config/ambient'
import { HABITAT_IDS } from '../../config/habitats'
import { hashString } from '../rng'
import { MAX_TERRAIN_CELLS, type TerrainField } from '../terrain/field'

export interface BirdRoute { id: string; sample: number; x: number; z: number; altitude: number; radius: number; phase: number; rank: number }

/** A summed habitat mask proves the complete bounding square of each orbit
 * contains only meadow/woodland. Selection is cooperative and memory-bounded.
 * Cruise height is relative to the highest generated terrain, not local hills.
 */
export function* birdRouteSteps(seed: number, field: TerrainField, habitats: Uint8Array) {
  const count = field.cols * field.rows
  if (!Number.isSafeInteger(count) || count < 0 || count > MAX_TERRAIN_CELLS || habitats.length !== count || field.height.length !== count) throw new Error('Invalid bird habitat inputs')
  const meadow = HABITAT_IDS.indexOf('meadow'), woodland = HABITAT_IDS.indexOf('woodland')
  const stride = field.cols + 1
  const sums = new Uint32Array(stride * (field.rows + 1))
  let highest = 0, work = 0
  for (let row = 0; row < field.rows; row++) {
    let rowSum = 0
    for (let col = 0; col < field.cols; col++) {
      if (work++ % 128 === 0) yield { completed: work, total: count * 2 }
      const sample = row * field.cols + col
      const code = habitats[sample]
      if (code !== meadow && code !== woodland) rowSum++
      sums[(row + 1) * stride + col + 1] = (sums[row * stride + col + 1] ?? 0) + rowSum
      highest = Math.max(highest, field.height[sample] ?? 0)
    }
  }
  const candidates: BirdRoute[] = []
  const r = ambientPolicy.birds.radiusCells
  for (let row = r; row < field.rows - r; row++) for (let col = r; col < field.cols - r; col++) {
    if (work++ % 128 === 0) yield { completed: work, total: count * 2 }
    const x0 = col - r, x1 = col + r + 1, z0 = row - r, z1 = row + r + 1
    const blocked = (sums[z1 * stride + x1] ?? 0) - (sums[z0 * stride + x1] ?? 0) - (sums[z1 * stride + x0] ?? 0) + (sums[z0 * stride + x0] ?? 0)
    if (blocked) continue
    const sample = row * field.cols + col
    const rank = hashString(`bird-route-v1:${seed}:${sample}`)
    const route: BirdRoute = { id: `bird:${seed}:${sample}`, sample, rank,
      x: field.originX + col * field.cellSize, z: field.originZ + row * field.cellSize,
      altitude: highest + ambientPolicy.birds.clearance,
      radius: (r - .5) * field.cellSize,
      phase: hashString(`bird-phase-v1:${seed}:${sample}`) / 4294967296 * Math.PI * 2,
    }
    const insertion = candidates.findIndex(other => other.rank > rank || other.rank === rank && other.sample > sample)
    if (insertion >= 0) candidates.splice(insertion, 0, route)
    else if (candidates.length < 64) candidates.push(route)
    if (candidates.length > 64) candidates.pop()
  }
  const routes: BirdRoute[] = []
  for (const candidate of candidates) {
    if (routes.every(other => Math.hypot(other.x - candidate.x, other.z - candidate.z) >= candidate.radius * 3)) routes.push(candidate)
    if (routes.length === ambientPolicy.birds.maximumVisible.ultra) break
  }
  return routes
}

export function birdPose(route: BirdRoute, seconds: number) {
  const angle = seconds % ambientPolicy.birds.periodSeconds / ambientPolicy.birds.periodSeconds * Math.PI * 2 + route.phase
  return {
    position: [route.x + Math.cos(angle) * route.radius, route.altitude + Math.sin(angle * 2) * .5, route.z + Math.sin(angle) * route.radius] as const,
    yaw: -angle,
    bank: -.18,
  }
}
