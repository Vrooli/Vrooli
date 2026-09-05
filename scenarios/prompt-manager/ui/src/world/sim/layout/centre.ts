import { resolveTerrain, type Scene, type TerrainTuning } from '../../config'
import { blendHeight, centreWeight, type CentreRegion } from '../../config/regions'
import type { WorldBounds } from '../model'
import type { TerrainField } from '../terrain/field'
import type { Rect } from './floorplan/plate'

export { centreWeight } from '../../config/regions'

export function centreRegion(scene: Scene, plate: Rect): CentreRegion | undefined {
  const centre = scene.centre
  return centre ? { ...plate, width: plate.width + centre.margin * 2, depth: plate.depth + centre.margin * 2, blend: centre.blend } : undefined
}

export function regionForBounds(scene: Scene, bounds: WorldBounds): CentreRegion | undefined {
  const plate = bounds.footprint
  return centreRegion(scene, { x: plate.center[0], z: plate.center[1], width: plate.width, depth: plate.depth })
}

export function terrainForBounds(scene: Scene, terrain: TerrainTuning, bounds: WorldBounds) {
  return resolveTerrain(scene, { terrain }, regionForBounds(scene, bounds))
}

/** Mean comes from untouched landscape, before centre amplitudes are applied. */
export function centreLevel(field: TerrainField, scene: Scene, bounds: WorldBounds, terrain: TerrainTuning): number | undefined {
  if (scene.centre?.levelTo !== 'plateMean') return undefined
  const plate = bounds.footprint
  let sum = 0
  let count = 0
  for (let row = 0; row < field.rows; row += 1) for (let col = 0; col < field.cols; col += 1) {
    const x = field.originX + col * field.cellSize
    const z = field.originZ + row * field.cellSize
    if (Math.abs(x - plate.center[0]) > plate.width / 2 || Math.abs(z - plate.center[1]) > plate.depth / 2) continue
    sum += field.height[row * field.cols + col] ?? 0
    count += 1
  }
  const local = terrainForBounds(scene, terrain, bounds).at(plate.center[0], plate.center[1])
  return Math.max(count ? sum / count : 0, local.waterLevel + local.moistureBasinDepth + local.padClearance)
}

export function levelCentre(field: TerrainField, region: CentreRegion, height: number): void {
  // Extend only as far as the first supporting samples. A whole extra cell
  // would needlessly compress the transition and steepen its slope.
  const supportFor = (center: number, extent: number, origin: number) => {
    const lo = (center - extent / 2 - origin) / field.cellSize
    const hi = (center + extent / 2 - origin) / field.cellSize
    return Math.max(lo - Math.floor(lo), Math.ceil(hi) - hi) * field.cellSize
  }
  const support = Math.min(region.blend, Math.max(supportFor(region.x, region.width, field.originX), supportFor(region.z, region.depth, field.originZ)))
  const levelRegion = { ...region, width: region.width + support * 2, depth: region.depth + support * 2, blend: region.blend - support }
  for (let row = 0; row < field.rows; row += 1) for (let col = 0; col < field.cols; col += 1) {
    const x = field.originX + col * field.cellSize
    const z = field.originZ + row * field.cellSize
    const weight = centreWeight(levelRegion, x, z)
    if (weight === 0) continue
    const index = row * field.cols + col
    field.height[index] = blendHeight(field.height[index] ?? 0, height, weight)
  }
}
