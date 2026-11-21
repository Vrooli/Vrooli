import { ambientPolicy } from '../../config/ambient'
import type { BiomeSet } from '../../config'
import type { DecorSpot, NavGrid, Vec2 } from '../model'
import { hashString } from '../rng'
import { heightAt, type TerrainField } from '../terrain/field'
import { clearGroundRoute } from './rabbits'

export interface SquirrelRoute {
  id: string; treeId: string; tree: Vec2; forage: Vec2; approach: Vec2; trunk: Vec2; height: number; phase: number; rank: number;
}

/** Routes attach to an actual tree. Only the final leap/climb enters that tree's
 * exclusion disc; foraging and darting use the existing ground/nav authority.
 */
export function* squirrelRouteSteps(seed: number, field: TerrainField, habitats: Uint8Array, nav: NavGrid, decor: readonly DecorSpot[], biome: BiomeSet) {
  const candidates: Array<{ tree: DecorSpot; rank: number }> = []
  for (const [index, tree] of decor.entries()) {
    if (index % 128 === 0) yield { completed: index, total: decor.length + 128 }
    if (tree.kind !== 'tree' || tree.roomId || biome.assetSet !== 'park' || !Object.prototype.hasOwnProperty.call(ambientPolicy.squirrels.canopyBottom, tree.propId ?? '')) continue
    const col = Math.round((tree.position[0] - field.originX) / field.cellSize), row = Math.round((tree.position[1] - field.originZ) / field.cellSize)
    if (col < 0 || row < 0 || col >= field.cols || row >= field.rows || ![1, 2].includes(habitats[row * field.cols + col] ?? 0)) continue
    const rank = hashString(`squirrel:${seed}:${tree.id}`)
    const at = candidates.findIndex(other => other.rank > rank)
    if (at >= 0) candidates.splice(at, 0, { tree, rank })
    else if (candidates.length < 128) candidates.push({ tree, rank })
    if (candidates.length > 128) candidates.pop()
  }
  const routes: SquirrelRoute[] = []
  for (const [index, { tree, rank }] of candidates.entries()) {
    yield { completed: decor.length + index, total: decor.length + 128 }
    for (let direction = 0; direction < 8; direction++) {
      const angle = ((rank % 8) + direction) * Math.PI / 4, x = Math.sin(angle), z = Math.cos(angle)
      const point = (distance: number): Vec2 => [tree.position[0] + x * distance, tree.position[1] + z * distance]
      const forage = point(3.5), approach = point(1.5)
      const treeKind = tree.propId as keyof typeof ambientPolicy.squirrels.canopyBottom
      const canopyBottom = ambientPolicy.squirrels.canopyBottom[treeKind]
      const scale = biome.propScale * (tree.scaleRef === 'tree' ? biome.treeScale : 1) * tree.scale
      const trunk = point(ambientPolicy.squirrels.trunkRadius[treeKind] * scale)
      // Excluding the small positive kit ground lift leaves extra headroom.
      // Sample each support point so sloping terrain cannot raise the animal into foliage.
      const height = Math.min(2.2, heightAt(field, ...tree.position) + canopyBottom * scale - heightAt(field, ...trunk) - ambientPolicy.squirrels.climbClearance)
      if (height < ambientPolicy.squirrels.minimumClimbHeight) continue
      if (!clearGroundRoute(field, habitats, nav, forage, approach, ambientPolicy.squirrels.radius)) continue
      // A second tree must not occupy the leap corridor to the chosen trunk.
      if (decor.some(other => other.id !== tree.id && other.kind === 'tree' && Math.hypot(other.position[0] - tree.position[0], other.position[1] - tree.position[1]) < 2.2)) continue
      routes.push({ id: `squirrel:${seed}:${tree.id}`, treeId: tree.id, tree: tree.position, forage, approach,
        trunk, height, rank,
        phase: hashString(`squirrel-phase:${seed}:${tree.id}`) / 4294967296 })
      break
    }
    if (routes.length === ambientPolicy.squirrels.routeBudget) break
  }
  return routes
}

export function squirrelPose(route: SquirrelRoute, field: TerrainField, seconds: number) {
  const period = ambientPolicy.squirrels.periodSeconds
  const t = ((seconds % period + route.phase * period) % period + period) % period
  let from = route.forage, to = route.forage, progress = 0, lift = 0, pitch = 0
  let state: 'forage' | 'dart' | 'leap' | 'climb' | 'perch' | 'rest' = 'forage'
  if (t >= 10 && t < 12) { state = 'dart'; from = route.forage; to = route.approach; progress = (t - 10) / 2 }
  else if (t >= 12 && t < 13) { state = 'leap'; from = route.approach; to = route.trunk; progress = t - 12; lift = progress * .65 + Math.sin(progress * Math.PI) * .3; pitch = -progress * Math.PI / 2 }
  else if (t >= 13 && t < 15) { state = 'climb'; from = route.trunk; lift = .65 + (route.height - .65) * (t - 13) / 2; pitch = -Math.PI / 2 }
  else if (t >= 15 && t < 20) { state = 'perch'; from = route.trunk; lift = route.height; pitch = -Math.PI / 2 }
  else if (t >= 20 && t < 22) { state = 'climb'; from = route.trunk; lift = route.height - (route.height - .65) * (t - 20) / 2; pitch = -Math.PI / 2 }
  else if (t >= 22 && t < 23) { state = 'leap'; from = route.trunk; to = route.approach; progress = t - 22; lift = (1 - progress) * .65 + Math.sin(progress * Math.PI) * .3; pitch = -(1 - progress) * Math.PI / 2 }
  else if (t >= 23 && t < 25) { state = 'dart'; from = route.approach; to = route.forage; progress = (t - 23) / 2 }
  else if (t >= 25) { state = 'rest' }
  if (state === 'climb' || state === 'perch') to = from
  const eased = progress * progress * (3 - 2 * progress)
  const x = from[0] + (to[0] - from[0]) * eased, z = from[1] + (to[1] - from[1]) * eased
  const yaw = Math.atan2(route.tree[0] - route.forage[0], route.tree[1] - route.forage[1])
  return { state, position: [x, heightAt(field, x, z) + lift, z] as const, yaw: yaw + (state === 'dart' && t >= 23 ? Math.PI : 0), pitch,
    nibble: state === 'forage' ? Math.sin(t * 8) * .025 : 0,
    stride: state === 'dart' || state === 'climb' ? Math.sin(t * 22) : 0,
    animating: state !== 'rest' && state !== 'perch' }
}

export function nextSquirrelBoundary(route: SquirrelRoute, seconds: number) {
  const period = ambientPolicy.squirrels.periodSeconds
  const t = ((seconds % period + route.phase * period) % period + period) % period
  return seconds + Math.max(.001, ([10, 12, 13, 15, 20, 22, 23, 25, 30].find(boundary => boundary > t) ?? period) - t)
}
