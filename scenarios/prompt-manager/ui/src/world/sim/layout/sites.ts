import type { LayoutTuning, TerrainResolver } from '../../config'
import type { NavGrid, Vec2 } from '../model'
import { buildNavGridSteps, cellIndex, isCellWalkable, isWalkable, worldToCell } from '../nav/grid'
import { Rng, hashString } from '../rng'
import { sortSteps } from '../cooperative'
import { shoreDistance, slopeAt, type TerrainField } from '../terrain'

export interface Site {
  position: Vec2
  rotation: number
  size: Vec2
  height: number
}

export interface SiteProgress { completed: number; total: number; operation?: 'ranking' }

export interface SiteTuning {
  layout: LayoutTuning
  terrain: TerrainResolver
}

function clamp01(value: number): number {
  return Math.max(0, Math.min(1, value))
}

export function buildability(field: TerrainField, tuning: SiteTuning, nav: NavGrid | undefined, x: number, z: number, commons: Vec2 = [0, 0], selected: readonly Site[] = []): number {
  const { layout } = tuning
  const terrain = tuning.terrain.at(x, z)
  const flat = 1 - clamp01(slopeAt(field, x, z) / terrain.maxSiteSlope)
  const dry = shoreDistance(field, tuning.terrain, x, z) >= terrain.shoreMargin ? 1 : 0
  const near = 1 - clamp01(Math.hypot(x - commons[0], z - commons[1]) / layout.siteRadiusMax)
  const nearest = selected.length === 0 ? layout.siteSpacing : Math.min(...selected.map((site) => Math.hypot(x - site.position[0], z - site.position[1])))
  const apart = clamp01(nearest / layout.siteSpacing)
  const walkable = nav ? (isWalkable(nav, [x, z]) ? 1 : 0) : 1
  return walkable * (layout.siteWeightFlat * flat + layout.siteWeightDry * dry + layout.siteWeightNear * near + layout.siteWeightApart * apart)
}

export function snappedRotation(from: Vec2, toward: Vec2, step: number): number {
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

export function rectanglesSeparated(a: Pick<Site, 'position' | 'rotation' | 'size'>, b: Pick<Site, 'position' | 'rotation' | 'size'>, clearance: number): boolean {
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

/** Preserve a saved site when possible; otherwise find the closest clear grid
 * position without changing its orientation or splitting its furnishings. */
export function* fitSavedSiteSteps(site: Site, occupied: readonly Site[], commons: Site | undefined, field: TerrainField, tuning: SiteTuning): Generator<SiteProgress, Site> {
  const clearance = tuning.terrain.base().kerbWidth + field.cellSize
  const valid = (candidate: Site) => {
    const [x, z] = candidate.position, local = tuning.terrain.at(x, z)
    if (shoreDistance(field, tuning.terrain, x, z) < local.shoreMargin || slopeAt(field, x, z) > local.maxSiteSlope) return false
    for (const dx of [-candidate.size[0] / 2, candidate.size[0] / 2]) for (const dz of [-candidate.size[1] / 2, candidate.size[1] / 2]) {
      if (Math.hypot(x + dx * Math.cos(candidate.rotation) + dz * Math.sin(candidate.rotation), z - dx * Math.sin(candidate.rotation) + dz * Math.cos(candidate.rotation)) > field.radius - field.cellSize) return false
    }
    return (!commons || clearOfCommons(candidate.position, candidate.rotation, candidate.size, commons.position, tuning.layout.commonsRadius + clearance))
      && occupied.every(other => rectanglesSeparated(candidate, other, clearance))
  }
  if (valid(site)) return site
  const step = Math.max(field.cellSize, tuning.layout.cellSize)
  const limit = Math.ceil(field.radius * 2 / step)
  const origin: Vec2 = [Math.max(-field.radius, Math.min(field.radius, site.position[0])), Math.max(-field.radius, Math.min(field.radius, site.position[1]))]
  let best: Site | undefined, bestDistance = Infinity, completed = 0
  for (let ring = 0; ring <= limit && ring * step <= bestDistance; ring++) {
    const offsets: Vec2[] = ring === 0 ? [[0, 0]] : []
    for (let n = -ring; n < ring; n++) offsets.push([n, -ring], [ring, n], [-n, ring], [-ring, -n])
    for (const [dx, dz] of offsets) {
      if (completed++ % 128 === 0) yield { completed, total: (limit * 2 + 1) ** 2, operation: 'ranking' }
      const distance = Math.hypot(dx, dz) * step
      if (distance >= bestDistance) continue
      const candidate = { ...site, position: [origin[0] + dx * step, origin[1] + dz * step] as Vec2 }
      if (valid(candidate)) { best = candidate; bestDistance = distance }
    }
  }
  if (!best) throw new Error('Saved campsite layout has no clear ground available. Reset the layout or reduce its footprint.')
  return best
}

function* reachableFrom(grid: NavGrid, start: Vec2): Generator<{ completed: number; total: number }, Set<number>> {
  const [startCol, startRow] = worldToCell(grid, start)
  const found = new Set<number>()
  if (!isCellWalkable(grid, startCol, startRow)) return found
  const queue: Array<[number, number]> = [[startCol, startRow]]
  found.add(cellIndex(grid, startCol, startRow))
  for (let head = 0; head < queue.length; head += 1) {
    if (head % 128 === 0) yield { completed: head, total: grid.cols * grid.rows }
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
export function selectSites(...args: Parameters<typeof selectSitesSteps>): { commons: Site; sites: Site[] } {
  const steps = selectSitesSteps(...args)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* selectSitesSteps(field: TerrainField, tuning: SiteTuning, sizes: readonly Vec2[], seed: number): Generator<SiteProgress, { commons: Site; sites: Site[] }> {
  const rng = new Rng(hashString(`sites:${seed}`))
  const maxSiteReach = sizes.reduce((largest, size) => Math.max(largest, Math.hypot(size[0], size[1]) / 2), 0)
  const capacityRadius = maxSiteReach * 2 + tuning.layout.siteSpacing * 2
  const insideTerrainRadius = field.radius - maxSiteReach - tuning.terrain.base().kerbWidth - field.cellSize
  const candidateRadius = Math.max(0, Math.min(insideTerrainRadius, Math.max(tuning.layout.siteRadiusMax, capacityRadius)))
  const candidates: Vec2[] = []
  function* appendCandidateBatch(radiusLimit = candidateRadius) {
    for (let index = 0; index < tuning.layout.siteCandidates; index += 1) {
      if (index % 128 === 0) yield { completed: index, total: tuning.layout.siteCandidates }
      const angle = rng.range(0, Math.PI * 2)
      const radius = Math.sqrt(rng.next()) * radiusLimit
      candidates.push([Math.sin(angle) * radius, Math.cos(angle) * radius])
    }
  }
  yield* appendCandidateBatch()
  const terrainNav = yield* buildNavGridSteps(
    { width: field.radius * 2, depth: field.radius * 2, center: [0, 0], footprint: { width: 0, depth: 0, center: [0, 0] }, outline: [] },
    [],
    [],
    tuning.layout.cellSize,
    0,
    field,
    tuning.terrain,
  )
  // A dry continuous point can map to a blocked navigation cell. Select the
  // commons from walkable candidates so its connectivity search has a root.
  let commonsPoint: Vec2 | undefined
  let commonsScore = -Infinity
  for (const [index, point] of candidates.entries()) {
    if (index % 128 === 0) yield { completed: index, total: candidates.length }
    const local = tuning.terrain.at(point[0], point[1])
    if (shoreDistance(field, tuning.terrain, point[0], point[1]) < local.shoreMargin || slopeAt(field, point[0], point[1]) > local.maxWalkSlope || !isWalkable(terrainNav, point)) continue
    const score = buildability(field, tuning, undefined, point[0], point[1]) - Math.hypot(point[0], point[1]) / candidateRadius
    if (!commonsPoint || score > commonsScore) { commonsPoint = point; commonsScore = score }
  }
  if (!commonsPoint) throw new Error(`site-selection: no walkable commons at seed ${seed}`)
  const reachable = yield* reachableFrom(terrainNav, commonsPoint)
  const commons: Site = { position: commonsPoint, rotation: 0, size: [tuning.layout.commonsRadius * 2, tuning.layout.commonsRadius * 2], height: 0 }
  // A terrace modifies samples one field cell beyond its footprint and then
  // blends across kerbWidth. Keep every later kerb outside earlier pad cores.
  const terraceClearance = tuning.terrain.base().kerbWidth + field.cellSize
  const commonsPosition = commonsPoint
  function* rankedFor(size: Vec2, placed: readonly Site[], diverse = false, quarterTurns = 0): Generator<SiteProgress, Vec2[]> {
    const ranked: Array<{ point: Vec2; score: number }> = []
    for (const [index, point] of candidates.entries()) {
      const progress: SiteProgress = { completed: index, total: candidates.length, operation: 'ranking' }
      if (index % 128 === 0) yield progress
      const local = tuning.terrain.at(point[0], point[1])
      if (shoreDistance(field, tuning.terrain, point[0], point[1]) < local.shoreMargin || slopeAt(field, point[0], point[1]) > local.maxSiteSlope) continue
      const rotation = snappedRotation(point, commonsPosition, tuning.layout.siteRotationSnapRad) + quarterTurns * Math.PI / 2
      const candidate: Site = { position: point, rotation, size, height: 0 }
      const c = Math.cos(rotation), s = Math.sin(rotation)
      if ([-1, 1].some(x => [-1, 1].some(z => Math.hypot(point[0] + x * size[0] / 2 * c + z * size[1] / 2 * s,
        point[1] - x * size[0] / 2 * s + z * size[1] / 2 * c) > field.radius - terraceClearance))) continue
      const exitDistance = size[1] / 2 + tuning.layout.cellSize
      const exit: Vec2 = [point[0] + Math.sin(rotation) * exitDistance, point[1] + Math.cos(rotation) * exitDistance]
      const [col, row] = worldToCell(terrainNav, exit)
      if (!reachable.has(cellIndex(terrainNav, col, row)) || !clearOfCommons(point, rotation, size, commonsPosition, tuning.layout.commonsRadius + terraceClearance)) continue
      let separated = true
      let nearest: Site | undefined
      let nearestDistance = Infinity
      for (const [placedIndex, site] of placed.entries()) {
        if (placedIndex % 128 === 0) yield progress
        if (!rectanglesSeparated(candidate, site, terraceClearance)) { separated = false; break }
        const distance = Math.hypot(point[0] - site.position[0], point[1] - site.position[1])
        if (distance < nearestDistance) { nearest = site; nearestDistance = distance }
      }
      if (!separated) continue
      const score = buildability(field, tuning, undefined, point[0], point[1], commonsPosition, nearest ? [nearest] : [])
      if (diverse) { ranked.push({ point, score }); continue }
      // Stable insertion preserves original candidate order on equal scores.
      const at = ranked.findIndex(entry => score > entry.score)
      ranked.splice(at < 0 ? ranked.length : at, 0, { point, score })
      if (ranked.length > BACKTRACK_BRANCH_LIMIT) ranked.pop()
    }
    if (!diverse) return ranked.map(entry => entry.point)
    // Nearby high-scoring samples describe the same placement decision. The
    // recovery search needs alternatives in different parts of the campground,
    // not eight almost identical centres that all block the next large site.
    const ordered = yield* sortSteps(ranked, (a, b) => b.score - a.score)
    const alternatives: Vec2[] = []
    const separation = Math.min(...size) / 2
    for (const [index, { point }] of ordered.entries()) {
      if (index % 128 === 0) yield { completed: index, total: ordered.length, operation: 'ranking' }
      if (alternatives.some(other => Math.hypot(point[0] - other[0], point[1] - other[1]) < separation)) continue
      alternatives.push(point)
      if (alternatives.length === BACKTRACK_BRANCH_LIMIT) break
    }
    return alternatives
  }
  const makeSite = (point: Vec2, size: Vec2, quarterTurns = 0): Site => ({ position: point, rotation: snappedRotation(point, commonsPoint, tuning.layout.siteRotationSnapRad) + quarterTurns * Math.PI / 2, size, height: 0 })

  // Preserve the stable greedy prefix for ordinary worlds. Only invoke bounded
  // backtracking when a later large footprint proves that prefix is a dead end.
  const sites: Site[] = []
  for (const size of sizes) {
    yield { completed: sites.length, total: sizes.length }
    const best = (yield* rankedFor(size, sites))[0]
    if (!best) {
      let searched = 0
      function* search(index: number, placed: readonly Site[], quarterTurns: number): Generator<{ completed: number; total: number }, Site[] | null> {
        if (++searched > 512) return null
        yield { completed: index, total: sizes.length }
        const nextSize = sizes[index]
        if (!nextSize) return [...placed]
        for (const point of yield* rankedFor(nextSize, placed, true, quarterTurns)) {
          const result = yield* search(index + 1, [...placed, makeSite(point, nextSize, quarterTurns)], quarterTurns)
          if (result) return result
        }
        return null
      }
      for (let batch = 0; batch <= CANDIDATE_EXPANSION_BATCHES; batch += 1) {
        // Large rectangular plots may fit the available land with their long
        // edge across it. Facing every doorway directly at the commons is a
        // preference; a clear, reachable entrance remains the requirement.
        for (const turns of [0, 1, 2, 3]) {
          searched = 0
          const recovered = yield* search(0, [], turns)
          if (recovered) return { commons, sites: recovered }
        }
        if (batch < CANDIDATE_EXPANSION_BATCHES) yield* appendCandidateBatch(field.radius - terraceClearance)
      }
      throw new Error(`site-selection: no buildable site for index ${sites.length} at seed ${seed}`)
    }
    sites.push(makeSite(best, size))
  }
  return { commons, sites }
}
