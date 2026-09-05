import type { BiomeSet, LayoutTuning, TerrainResolver } from '../../config'
import type { DecorSpot, Place, Vec2, WorldBounds } from '../model'
import { Rng, hashString } from '../rng'
import type { TerrainField } from '../terrain'
import { isWater, shoreDistance } from '../terrain/water'
import { standMask } from '../terrain/stands'

function distanceSq(a: Vec2, b: Vec2): number {
  return (a[0] - b[0]) ** 2 + (a[1] - b[1]) ** 2
}

/** Cell width bounds every query radius, so nine buckets cover the whole disc. */
export class SpacingIndex {
  private readonly buckets = new Map<string, Vec2[]>()

  constructor(private readonly cellSize: number) {}

  overlaps(point: Vec2, spacing: number): boolean {
    const col = Math.floor(point[0] / this.cellSize)
    const row = Math.floor(point[1] / this.cellSize)
    for (let dz = -1; dz <= 1; dz += 1) for (let dx = -1; dx <= 1; dx += 1) {
      const bucket = this.buckets.get(`${col + dx}:${row + dz}`)
      if (bucket?.some((other) => distanceSq(point, other) < spacing ** 2)) return true
    }
    return false
  }

  insert(point: Vec2): void {
    const key = `${Math.floor(point[0] / this.cellSize)}:${Math.floor(point[1] / this.cellSize)}`
    const bucket = this.buckets.get(key)
    if (bucket) bucket.push(point)
    else this.buckets.set(key, [point])
  }
}

export interface ScatterInput {
  field: TerrainField
  tuning: TerrainResolver
  biomes: Uint8Array
  biomeSet: BiomeSet
  places: Place[]
  bounds: WorldBounds
  layout: LayoutTuning
  seed: number
  clearPoints: Vec2[]
  exclude?: (x: number, z: number) => boolean
}

/** Deterministic per-biome scatter with per-kind spacing and place clearance. */
export function scatterDecor(input: ScatterInput): DecorSpot[] {
  const { field, tuning, biomes, biomeSet, places, bounds, layout, seed, clearPoints } = input
  const rng = new Rng(hashString(`decor:${seed}`))
  const blockers = places.filter((place) => !place.parentId).map((place) => ({ point: place.position, radius: Math.hypot(place.size[0], place.size[1]) / 2 + layout.clearingRadius }))
  blockers.push(...clearPoints.map((point) => ({ point, radius: layout.clearingRadius })))
  const spots: DecorSpot[] = []
  const spacingFor = (density: number, kind: DecorSpot['kind']) => Math.max(field.cellSize, 1 / Math.sqrt(Math.max(density, Number.EPSILON))) * (kind === 'tree' ? 1 : layout.decorSpacingFactor)
  const maximumSpacing = new Map<string, number>()
  for (const biome of biomeSet.biomes) for (const [id, entry] of [...Object.entries(biome.vegetation), ...Object.entries(biome.decor)]) {
    if (entry.density > 0) maximumSpacing.set(id, Math.max(maximumSpacing.get(id) ?? 0, spacingFor(entry.density, entry.class)))
  }
  const spacingByProp = new Map([...maximumSpacing].map(([id, spacing]) => [id, new SpacingIndex(spacing)]))
  const halfWidth = bounds.width / 2
  const halfDepth = bounds.depth / 2
  for (let row = 0; row < field.rows; row += 1) {
    const z = field.originZ + row * field.cellSize
    if (Math.abs(z - bounds.center[1]) > halfDepth) continue
    for (let col = 0; col < field.cols; col += 1) {
      const x = field.originX + col * field.cellSize
      if (Math.abs(x - bounds.center[0]) > halfWidth) continue
      const biome = biomeSet.biomes[biomes[row * field.cols + col] ?? biomeSet.biomes.length - 1]
      if (!biome) continue
      const entries = [...Object.entries(biome.vegetation), ...Object.entries(biome.decor)]
      for (const [variant, [propId, entry]] of entries.entries()) {
        const { density } = entry
        const mask = standMask(x, z, hashString(`stand:${seed}:${propId}`), layout.stands)
        if (rng.next() >= density * mask * field.cellSize * field.cellSize) continue
        const candidate: Vec2 = [x + rng.range(-layout.scatterJitter, layout.scatterJitter) * field.cellSize, z + rng.range(-layout.scatterJitter, layout.scatterJitter) * field.cellSize]
        if (Math.hypot(candidate[0], candidate[1]) > field.radius) continue
        if (input.exclude?.(candidate[0], candidate[1])) continue
        if (isWater(field, tuning, candidate[0], candidate[1]) || shoreDistance(field, tuning, candidate[0], candidate[1]) < layout.shoreClearance) continue
        if (blockers.some((blocker) => distanceSq(candidate, blocker.point) < blocker.radius ** 2)) continue
        const spacing = spacingFor(density, entry.class)
        const index = spacingByProp.get(propId)
        if (!index || index.overlaps(candidate, spacing)) continue
        index.insert(candidate)
        spots.push({
          id: `${entry.class}:${propId}:${spots.length}`,
          kind: entry.class,
          scaleRef: entry.scaleRef,
          propId,
          variant,
          position: candidate,
          rotation: rng.range(0, Math.PI * 2),
          scale: rng.range(layout.decorScale.min, layout.decorScale.max),
          tint: [
            1 + rng.range(-layout.decorColorJitter, layout.decorColorJitter),
            1 + rng.range(-layout.decorColorJitter, layout.decorColorJitter),
            1 + rng.range(-layout.decorColorJitter, layout.decorColorJitter),
          ],
        })
      }
    }
  }
  return spots
}
