import type { BiomeSet, LayoutTuning } from '../../config'
import type { DecorSpot, Place, Vec2, WorldBounds } from '../model'
import { Rng, hashString } from '../rng'
import type { TerrainField } from '../terrain'

function distanceSq(a: Vec2, b: Vec2): number {
  return (a[0] - b[0]) ** 2 + (a[1] - b[1]) ** 2
}

export interface ScatterInput {
  field: TerrainField
  biomes: Uint8Array
  biomeSet: BiomeSet
  places: Place[]
  bounds: WorldBounds
  layout: LayoutTuning
  seed: number
  clearPoints: Vec2[]
  treePropIds: readonly string[]
}

/** Deterministic per-biome scatter with per-kind spacing and place clearance. */
export function scatterDecor(input: ScatterInput): DecorSpot[] {
  const { field, biomes, biomeSet, places, bounds, layout, seed, clearPoints, treePropIds } = input
  const rng = new Rng(hashString(`decor:${seed}`))
  const blockers = places.filter((place) => !place.parentId).map((place) => ({ point: place.position, radius: Math.hypot(place.size[0], place.size[1]) / 2 + layout.clearingRadius }))
  blockers.push(...clearPoints.map((point) => ({ point, radius: layout.clearingRadius })))
  const spots: DecorSpot[] = []
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
      const tables: Array<{ kind: DecorSpot['kind']; entries: Array<[string, number]> }> = [
        { kind: 'tree', entries: Object.entries(biome.vegetation) },
        { kind: 'decor', entries: Object.entries(biome.decor) },
      ]
      for (const table of tables) for (const [propId, density] of table.entries) {
        if (rng.next() >= density * field.cellSize * field.cellSize) continue
        const candidate: Vec2 = [x + rng.range(-layout.scatterJitter, layout.scatterJitter) * field.cellSize, z + rng.range(-layout.scatterJitter, layout.scatterJitter) * field.cellSize]
        if (Math.hypot(candidate[0], candidate[1]) > field.radius) continue
        if (blockers.some((blocker) => distanceSq(candidate, blocker.point) < blocker.radius ** 2)) continue
        const spacing = Math.max(field.cellSize, 1 / Math.sqrt(Math.max(density, Number.EPSILON))) * (table.kind === 'tree' ? 1 : layout.decorSpacingFactor)
        if (spots.some((spot) => spot.propId === propId && distanceSq(candidate, spot.position) < spacing ** 2)) continue
        spots.push({
          id: `${table.kind}:${propId}:${spots.length}`,
          kind: table.kind,
          propId,
          variant: Math.max(0, treePropIds.indexOf(propId)),
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
