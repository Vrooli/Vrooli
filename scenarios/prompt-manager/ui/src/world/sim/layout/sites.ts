import type { LayoutTuning, TerrainTuning } from '../../config'
import type { NavGrid, Vec2 } from '../model'
import { buildNavGrid, cellIndex, isCellWalkable, isWalkable, worldToCell } from '../nav/grid'
import { Rng, hashString } from '../rng'
import { shoreDistance, slopeAt, type TerrainField } from '../terrain'

export interface Site {
  position: Vec2
  rotation: number
  size: Vec2
  height: number
}

export interface SiteTuning {
  layout: LayoutTuning
  terrain: TerrainTuning
}

function clamp01(value: number): number {
  return Math.max(0, Math.min(1, value))
}

export function buildability(field: TerrainField, tuning: SiteTuning, nav: NavGrid | undefined, x: number, z: number, commons: Vec2 = [0, 0], selected: readonly Site[] = []): number {
  const { layout, terrain } = tuning
  const flat = 1 - clamp01(slopeAt(field, x, z) / terrain.maxSiteSlope)
  const dry = shoreDistance(field, terrain, x, z) >= terrain.shoreMargin ? 1 : 0
  const near = 1 - clamp01(Math.hypot(x - commons[0], z - commons[1]) / layout.siteRadiusMax)
  const nearest = selected.length === 0 ? layout.siteSpacing : Math.min(...selected.map((site) => Math.hypot(x - site.position[0], z - site.position[1])))
  const apart = clamp01(nearest / layout.siteSpacing)
  const walkable = nav ? (isWalkable(nav, [x, z]) ? 1 : 0) : 1
  return walkable * (layout.siteWeightFlat * flat + layout.siteWeightDry * dry + layout.siteWeightNear * near + layout.siteWeightApart * apart)
}

function snappedRotation(from: Vec2, toward: Vec2, step: number): number {
  const heading = Math.atan2(toward[0] - from[0], toward[1] - from[1])
  return Math.round(heading / step) * step
}

const CANDIDATE_EXPANSION_BATCHES = 4
const BACKTRACK_BRANCH_LIMIT = 8

function clearOfCommons(position: Vec2, rotation: number, size: Vec2, commons: Vec2, radius: number): boolean {
  const dx = commons[0] - position[0]
  const dz = commons[1] - position[1]
  const cos = Math.cos(rotation)
  const sin = Math.sin(rotation)
  const localX = dx * cos - dz * sin
  const localZ = dx * sin + dz * cos
  const outsideX = Math.max(Math.abs(localX) - size[0] / 2, 0)
  const outsideZ = Math.max(Math.abs(localZ) - size[1] / 2, 0)
  return Math.hypot(outsideX, outsideZ) >= radius
}

function rectanglesSeparated(a: Site, b: Site, clearance: number): boolean {
  const axes: Vec2[] = [
    [Math.cos(a.rotation), -Math.sin(a.rotation)],
    [Math.sin(a.rotation), Math.cos(a.rotation)],
    [Math.cos(b.rotation), -Math.sin(b.rotation)],
    [Math.sin(b.rotation), Math.cos(b.rotation)],
  ]
  const delta: Vec2 = [b.position[0] - a.position[0], b.position[1] - a.position[1]]
  return axes.some((axis) => {
    const projectedDistance = Math.abs(delta[0] * axis[0] + delta[1] * axis[1])
    const radius = (Math.abs(axis[0] * Math.cos(a.rotation) - axis[1] * Math.sin(a.rotation)) * a.size[0]
      + Math.abs(axis[0] * Math.sin(a.rotation) + axis[1] * Math.cos(a.rotation)) * a.size[1]
      + Math.abs(axis[0] * Math.cos(b.rotation) - axis[1] * Math.sin(b.rotation)) * b.size[0]
      + Math.abs(axis[0] * Math.sin(b.rotation) + axis[1] * Math.cos(b.rotation)) * b.size[1]) / 2
    return projectedDistance >= radius + clearance
  })
}

function reachableFrom(grid: NavGrid, start: Vec2): Set<number> {
  const [startCol, startRow] = worldToCell(grid, start)
  const found = new Set<number>()
  if (!isCellWalkable(grid, startCol, startRow)) return found
  const queue: Array<[number, number]> = [[startCol, startRow]]
  found.add(cellIndex(grid, startCol, startRow))
  for (let head = 0; head < queue.length; head += 1) {
    const current = queue[head]
    if (!current) continue
    for (let dz = -1; dz <= 1; dz += 1) for (let dx = -1; dx <= 1; dx += 1) {
      if (dx === 0 && dz === 0) continue
      const col = current[0] + dx
      const row = current[1] + dz
      const index = cellIndex(grid, col, row)
      if (found.has(index) || !isCellWalkable(grid, col, row)) continue
      if (dx !== 0 && dz !== 0 && (!isCellWalkable(grid, current[0] + dx, current[1]) || !isCellWalkable(grid, current[0], current[1] + dz))) continue
      found.add(index)
      queue.push([col, row])
    }
  }
  return found
}

/** Commons plus stable, ordered team sites selected from terrain, never API order. */
export function selectSites(field: TerrainField, tuning: SiteTuning, sizes: readonly Vec2[], seed: number): { commons: Site; sites: Site[] } {
  const rng = new Rng(hashString(`sites:${seed}`))
  const maxSiteReach = sizes.reduce((largest, size) => Math.max(largest, Math.hypot(size[0], size[1]) / 2), 0)
  const capacityRadius = maxSiteReach * 2 + tuning.layout.siteSpacing * 2
  const insideTerrainRadius = field.radius - maxSiteReach - tuning.terrain.kerbWidth - field.cellSize
  const candidateRadius = Math.max(0, Math.min(insideTerrainRadius, Math.max(tuning.layout.siteRadiusMax, capacityRadius)))
  const candidates: Vec2[] = []
  const appendCandidateBatch = () => {
    for (let index = 0; index < tuning.layout.siteCandidates; index += 1) {
      const angle = rng.range(0, Math.PI * 2)
      const radius = Math.sqrt(rng.next()) * candidateRadius
      candidates.push([Math.sin(angle) * radius, Math.cos(angle) * radius])
    }
  }
  appendCandidateBatch()
  const buildableCandidates = candidates.filter((point) => shoreDistance(field, tuning.terrain, point[0], point[1]) >= tuning.terrain.shoreMargin && slopeAt(field, point[0], point[1]) <= tuning.terrain.maxWalkSlope)
  const commonsPoint = [...buildableCandidates].sort((a, b) => {
    const score = (point: Vec2) => buildability(field, tuning, undefined, point[0], point[1]) - Math.hypot(point[0], point[1]) / candidateRadius
    return score(b) - score(a)
  })[0] ?? [0, 0]
  const terrainNav = buildNavGrid(
    { width: field.radius * 2, depth: field.radius * 2, center: [0, 0], footprint: { width: 0, depth: 0, center: [0, 0] }, outline: [] },
    [],
    [],
    tuning.layout.cellSize,
    tuning.layout.cellSize,
    0,
    field,
    tuning.terrain,
  )
  const reachable = reachableFrom(terrainNav, commonsPoint)
  const commons: Site = { position: commonsPoint, rotation: 0, size: [tuning.layout.commonsRadius * 2, tuning.layout.commonsRadius * 2], height: 0 }
  // A terrace modifies samples one field cell beyond its footprint and then
  // blends across kerbWidth. Keep every later kerb outside earlier pad cores.
  const terraceClearance = tuning.terrain.kerbWidth + field.cellSize
  const viableFor = (size: Vec2, placed: readonly Site[]) => candidates.filter((point) => {
      if (shoreDistance(field, tuning.terrain, point[0], point[1]) < tuning.terrain.shoreMargin) return false
      if (slopeAt(field, point[0], point[1]) > tuning.terrain.maxSiteSlope) return false
      const rotation = snappedRotation(point, commonsPoint, tuning.layout.siteRotationSnapRad)
      const candidate: Site = { position: point, rotation, size, height: 0 }
      const exitDistance = size[1] / 2 + tuning.layout.cellSize
      const exit: Vec2 = [point[0] + Math.sin(rotation) * exitDistance, point[1] + Math.cos(rotation) * exitDistance]
      const [col, row] = worldToCell(terrainNav, exit)
      if (!reachable.has(cellIndex(terrainNav, col, row))) return false
      if (!clearOfCommons(point, rotation, size, commonsPoint, tuning.layout.commonsRadius + terraceClearance)) return false
      return placed.every((site) => rectanglesSeparated(candidate, site, terraceClearance))
    })
  const rankedFor = (size: Vec2, placed: readonly Site[]) => viableFor(size, placed).sort(
    (a, b) => buildability(field, tuning, undefined, b[0], b[1], commonsPoint, placed) - buildability(field, tuning, undefined, a[0], a[1], commonsPoint, placed),
  )
  const makeSite = (point: Vec2, size: Vec2): Site => ({ position: point, rotation: snappedRotation(point, commonsPoint, tuning.layout.siteRotationSnapRad), size, height: 0 })

  // Preserve the stable greedy prefix for ordinary worlds. Only invoke bounded
  // backtracking when a later large footprint proves that prefix is a dead end.
  const sites: Site[] = []
  for (const size of sizes) {
    const best = rankedFor(size, sites)[0]
    if (!best) {
      const search = (index: number, placed: readonly Site[]): Site[] | null => {
        const nextSize = sizes[index]
        if (!nextSize) return [...placed]
        for (const point of rankedFor(nextSize, placed).slice(0, BACKTRACK_BRANCH_LIMIT)) {
          const result = search(index + 1, [...placed, makeSite(point, nextSize)])
          if (result) return result
        }
        return null
      }
      for (let batch = 0; batch <= CANDIDATE_EXPANSION_BATCHES; batch += 1) {
        const recovered = search(0, [])
        if (recovered) return { commons, sites: recovered }
        appendCandidateBatch()
      }
      throw new Error(`site-selection: no buildable site for index ${sites.length} at seed ${seed}`)
    }
    sites.push(makeSite(best, size))
  }
  return { commons, sites }
}
