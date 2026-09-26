import { ambientPolicy } from '../../config/ambient'
import type { NavGrid, Vec2 } from '../model'
import { cellToWorld, isCellWalkable } from '../nav/grid'
import { hashString } from '../rng'
import { heightAt, type TerrainField } from '../terrain/field'

export interface RabbitRoute { id: string; start: Vec2; end: Vec2; rank: number; phase: number }

/** Validate the full swept rectangle in both authorities. Habitat admits only
 * meadow/woodland; navigation rejects water, steep ground, trunks and walls.
 */
export function clearGroundRoute(field: TerrainField, habitats: Uint8Array, nav: NavGrid, start: Vec2, end: Vec2, radius: number = ambientPolicy.rabbits.radius) {
  const minX = Math.min(start[0], end[0]) - radius, maxX = Math.max(start[0], end[0]) + radius
  const minZ = Math.min(start[1], end[1]) - radius, maxZ = Math.max(start[1], end[1]) + radius
  for (let row = Math.floor((minZ - nav.originZ) / nav.cellSize); row <= Math.floor((maxZ - nav.originZ) / nav.cellSize); row++) {
    for (let col = Math.floor((minX - nav.originX) / nav.cellSize); col <= Math.floor((maxX - nav.originX) / nav.cellSize); col++) if (!isCellWalkable(nav, col, row)) return false
  }
  for (let row = Math.floor((minZ - field.originZ) / field.cellSize); row <= Math.ceil((maxZ - field.originZ) / field.cellSize); row++) {
    for (let col = Math.floor((minX - field.originX) / field.cellSize); col <= Math.ceil((maxX - field.originX) / field.cellSize); col++) {
      if (col < 0 || row < 0 || col >= field.cols || row >= field.rows) return false
      const code = habitats[row * field.cols + col]
      if (code !== 1 && code !== 2) return false
    }
  }
  return true
}

export function* rabbitRouteSteps(seed: number, field: TerrainField, habitats: Uint8Array, nav: NavGrid) {
  const candidates: Array<{ sample: number; rank: number }> = []
  for (let sample = 0; sample < nav.walkable.length; sample++) {
    if (sample % 128 === 0) yield { completed: sample, total: nav.walkable.length + 128 }
    if (nav.walkable[sample] !== 1) continue
    const rank = hashString(`rabbit-route-v1:${seed}:${sample}`)
    const insertion = candidates.findIndex(other => other.rank > rank || other.rank === rank && other.sample > sample)
    if (insertion >= 0) candidates.splice(insertion, 0, { sample, rank })
    else if (candidates.length < 128) candidates.push({ sample, rank })
    if (candidates.length > 128) candidates.pop()
  }
  const routes: RabbitRoute[] = []
  for (const [offset, candidate] of candidates.entries()) {
    yield { completed: nav.walkable.length + offset, total: nav.walkable.length + 128 }
    const col = candidate.sample % nav.cols, row = Math.floor(candidate.sample / nav.cols)
    const start = cellToWorld(nav, col, row)
    const vertical = candidate.rank % 2 === 0
    const end = cellToWorld(nav, col + (vertical ? 0 : 2), row + (vertical ? 2 : 0))
    if (!clearGroundRoute(field, habitats, nav, start, end)) continue
    if (routes.some(route => Math.hypot(route.start[0] - start[0], route.start[1] - start[1]) < nav.cellSize * 6)) continue
    routes.push({ id: `rabbit:${seed}:${candidate.sample}`, start, end, rank: candidate.rank,
      phase: hashString(`rabbit-phase-v1:${seed}:${candidate.sample}`) / 4294967296 })
    if (routes.length === ambientPolicy.rabbits.maximumVisible.ultra) break
  }
  return routes
}

export function rabbitPose(route: RabbitRoute, field: TerrainField, seconds: number) {
  const { idleSeconds: idle, travelSeconds: travel } = ambientPolicy.rabbits
  const half = idle + travel, period = half * 2
  const phase = ((seconds % period + route.phase * period) % period + period) % period
  const returning = phase >= half, local = phase % half
  const moving = local >= idle
  const t = Math.max(0, (local - idle) / travel)
  const eased = t * t * (3 - 2 * t)
  const progress = returning ? 1 - eased : eased
  const x = route.start[0] + (route.end[0] - route.start[0]) * progress
  const z = route.start[1] + (route.end[1] - route.start[1]) * progress
  const grazePhase = Math.max(0, Math.min(1, (local - 3) / 2, (12 - local) / 2))
  const graze = grazePhase * grazePhase * (3 - 2 * grazePhase)
  return {
    position: [x, heightAt(field, x, z), z] as const,
    yaw: Math.atan2(route.end[0] - route.start[0], route.end[1] - route.start[1]) + (returning ? Math.PI : 0),
    moving, hop: moving ? Math.abs(Math.sin(t * Math.PI * 4)) * .08 : 0,
    graze, nibble: graze * Math.sin(local * 12) * .015,
    animating: moving || local >= 3 && local < 12,
  }
}

export function nextRabbitBoundary(route: RabbitRoute, seconds: number) {
  const { idleSeconds: idle, travelSeconds: travel } = ambientPolicy.rabbits
  const half = idle + travel, period = half * 2
  const local = ((seconds % period + route.phase * period) % period + period) % half
  // The shared clock has millisecond precision. Avoid rounding a tiny residual
  // back onto the current instant at large UTC values.
  const boundary = local < 3 ? 3 : local < 12 ? 12 : local < idle ? idle : half
  return seconds + Math.max(.001, boundary - local)
}
