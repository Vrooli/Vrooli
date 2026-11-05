import { ambientPolicy } from '../../config/ambient'
import { hashString, Rng } from '../rng'
import type { WaterGeometryData } from '../terrain/waterSurface'
import type { TerrainField } from '../terrain/field'

export interface FishSite { position: readonly [number, number, number]; radius: number; triangle: number }
export interface FishPond { component: number; area: number; sites: FishSite[] }
export interface FishEvent { id: string; start: number; end: number; seed: number; pond: number; site: FishSite }

/** Projected triangle area is the rendered connected pond's authority. Each
 * site's in-circle is wholly within a water triangle, including clipped banks.
 * Keep at most eight sites per pond and 64 largest eligible ponds.
 */
export function* fishPondSteps(surfaces: readonly WaterGeometryData[], field: TerrainField, habitats: Uint8Array) {
  const ponds: FishPond[] = []
  let work = 0
  for (const surface of surfaces) {
    let area = 0
    const sites: FishSite[] = []
    const point = (vertex: number) => [surface.positions[vertex * 3] ?? 0, surface.positions[vertex * 3 + 1] ?? 0, surface.positions[vertex * 3 + 2] ?? 0] as const
    for (let offset = 0; offset < surface.indices.length; offset += 3) {
      if (work++ % 128 === 0) yield { completed: work, total: work + 1 }
      const a = point(surface.indices[offset] ?? 0), b = point(surface.indices[offset + 1] ?? 0), c = point(surface.indices[offset + 2] ?? 0)
      const twiceArea = Math.abs((b[0] - a[0]) * (c[2] - a[2]) - (b[2] - a[2]) * (c[0] - a[0]))
      area += twiceArea / 2
      const wa = Math.hypot(b[0] - c[0], b[2] - c[2]), wb = Math.hypot(a[0] - c[0], a[2] - c[2]), wc = Math.hypot(a[0] - b[0], a[2] - b[2])
      const perimeter = wa + wb + wc
      if (perimeter <= 0 || twiceArea <= 0) continue
      const radius = twiceArea / perimeter * .9
      const site: FishSite = { triangle: offset / 3, radius,
        position: [(a[0] * wa + b[0] * wb + c[0] * wc) / perimeter, (a[1] * wa + b[1] * wb + c[1] * wc) / perimeter, (a[2] * wa + b[2] * wb + c[2] * wc) / perimeter] }
      let eligible = true
      for (let row = Math.floor((site.position[2] - radius - field.originZ) / field.cellSize); row <= Math.ceil((site.position[2] + radius - field.originZ) / field.cellSize); row++) {
        for (let col = Math.floor((site.position[0] - radius - field.originX) / field.cellSize); col <= Math.ceil((site.position[0] + radius - field.originX) / field.cellSize); col++) {
          if (row < 0 || col < 0 || row >= field.rows || col >= field.cols || habitats[row * field.cols + col] !== 5) eligible = false
        }
      }
      if (!eligible) continue
      const insertion = sites.findIndex(other => other.radius < radius)
      if (insertion >= 0) sites.splice(insertion, 0, site)
      else if (sites.length < 8) sites.push(site)
      if (sites.length > 8) sites.pop()
    }
    if (area < ambientPolicy.fish.minimumPondArea || sites.length === 0) continue
    const pond = { component: surface.component, area, sites }
    const insertion = ponds.findIndex(other => other.area < area || other.area === area && other.component > surface.component)
    if (insertion >= 0) ponds.splice(insertion, 0, pond)
    else if (ponds.length < 64) ponds.push(pond)
    if (ponds.length > 64) ponds.pop()
  }
  return ponds.sort((a, b) => a.component - b.component)
}

export function fishCandidate(seed: number, bucket: number, ponds: readonly FishPond[]): FishEvent | null {
  if (!ponds.length) return null
  const id = `ambient-v1:${seed}:fish:${bucket}`
  const eventSeed = hashString(id), rng = new Rng(eventSeed)
  const start = (bucket + rng.next()) * ambientPolicy.fish.meanEligibleSeconds
  const pond = ponds[rng.int(ponds.length)]
  const site = pond?.sites[rng.int(pond.sites.length)]
  return pond && site ? { id, seed: eventSeed, start, end: start + ambientPolicy.fish.jumpSeconds + ambientPolicy.fish.effectSeconds, pond: pond.component, site } : null
}

export function fishEventsAt(seed: number, now: number, ponds: readonly FishPond[]) {
  const bucket = Math.floor(now / ambientPolicy.fish.meanEligibleSeconds)
  return [fishCandidate(seed, bucket - 1, ponds), fishCandidate(seed, bucket, ponds)]
    .filter((event): event is FishEvent => event !== null && event.start <= now && now < event.end)
}

export function nextFishBoundary(seed: number, now: number, ponds: readonly FishPond[]) {
  if (!ponds.length) return null
  const bucket = Math.floor(now / ambientPolicy.fish.meanEligibleSeconds)
  let next = (bucket + 1) * ambientPolicy.fish.meanEligibleSeconds
  for (const index of [bucket - 1, bucket]) {
    const event = fishCandidate(seed, index, ponds)
    if (!event) continue
    if (event.start > now) next = Math.min(next, event.start)
    if (event.end > now) next = Math.min(next, event.end)
  }
  return next
}

export function fishPose(event: FishEvent, now: number) {
  const elapsed = now - event.start
  const jump = elapsed / ambientPolicy.fish.jumpSeconds
  const ripple = Math.max(0, (elapsed - ambientPolicy.fish.jumpSeconds) / ambientPolicy.fish.effectSeconds)
  return { jumping: jump >= 0 && jump < 1, height: Math.sin(Math.min(1, Math.max(0, jump)) * Math.PI) * .65,
    pitch: Math.PI * (Math.min(1, Math.max(0, jump)) - .5), ripple,
    radius: event.site.radius * Math.min(1, ripple), opacity: Math.max(0, 1 - ripple) }
}
