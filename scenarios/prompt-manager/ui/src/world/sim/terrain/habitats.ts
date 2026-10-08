import { HABITAT_BY_BIOME, HABITAT_IDS, type HabitatId } from '../../config/habitats'
import type { BiomeSet } from '../../config'
import type { Place } from '../model'
import { MAX_TERRAIN_CELLS, type TerrainField } from './field'

export const MAX_HABITAT_REGIONS = 32768
export interface HabitatRegion {
  /** First member's terrain index plus one; zero is reserved for excluded samples. */
  id: number
  habitat: HabitatId
  offset: number
  samples: number
  bounds: { minX: number; minZ: number; maxX: number; maxZ: number }
}
export interface HabitatRegionIndex {
  labels: Uint32Array
  /** Region members occupy [offset, offset + samples) in this packed array. */
  members: Uint32Array
  regions: HabitatRegion[]
}

/** Exact four-connected components, with bounded buffers and cooperative work. */
export function* habitatRegionSteps(field: TerrainField, habitats: Uint8Array, maxRegions = MAX_HABITAT_REGIONS) {
  const count = field.cols * field.rows
  if (!Number.isSafeInteger(count) || count < 0 || count > MAX_TERRAIN_CELLS || habitats.length !== count ||
      !Number.isInteger(maxRegions) || maxRegions < 1 || maxRegions > MAX_HABITAT_REGIONS) throw new Error('Invalid habitat region inputs')
  const labels = new Uint32Array(count)
  const queue = new Uint32Array(count)
  const regions: HabitatRegion[] = []
  let tail = 0
  let work = 0
  for (let root = 0; root < count; root++) {
    if (work++ % 128 === 0) yield { completed: work, total: count * 2 }
    const code = habitats[root] ?? 0
    if (code === 0 || labels[root] !== 0) continue
    const habitat = HABITAT_IDS[code]
    if (!habitat) throw new Error('Unknown habitat in region field')
    if (regions.length >= maxRegions) throw new Error('Habitat fragmentation exceeds the region budget')
    const id = root + 1
    const offset = tail
    let head = tail
    queue[tail++] = root
    labels[root] = id
    let minCol = field.cols, minRow = field.rows, maxCol = 0, maxRow = 0
    const add = (candidate: number) => {
      if (habitats[candidate] === code && labels[candidate] === 0) {
        labels[candidate] = id
        queue[tail++] = candidate
      }
    }
    while (head < tail) {
      if (work++ % 128 === 0) yield { completed: work, total: count * 2 }
      const index = queue[head++] ?? 0
      const col = index % field.cols
      const row = Math.floor(index / field.cols)
      minCol = Math.min(minCol, col); maxCol = Math.max(maxCol, col)
      minRow = Math.min(minRow, row); maxRow = Math.max(maxRow, row)
      if (col > 0) add(index - 1)
      if (col + 1 < field.cols) add(index + 1)
      if (row > 0) add(index - field.cols)
      if (row + 1 < field.rows) add(index + field.cols)
    }
    regions.push({ id, habitat, offset, samples: tail - offset, bounds: {
      minX: field.originX + minCol * field.cellSize, maxX: field.originX + maxCol * field.cellSize,
      minZ: field.originZ + minRow * field.cellSize, maxZ: field.originZ + maxRow * field.cellSize,
    } })
  }
  yield { completed: count * 2, total: count * 2 }
  return { labels, members: queue.slice(0, tail), regions } satisfies HabitatRegionIndex
}

/** Habitat is an immutable environmental mask, independent of actor occupancy.
 * Paths, built footprints and samples outside the terrain disc are ineligible.
 * Movement feasibility remains the navigation grid's responsibility.
 */
export function* habitatFieldSteps(field: TerrainField, biomes: Uint8Array, set: BiomeSet, paths: Float32Array, places: readonly Place[]) {
  const count = field.cols * field.rows
  if (!Number.isSafeInteger(count) || count < 0 || count > MAX_TERRAIN_CELLS || biomes.length !== count || paths.length !== count) throw new Error('Invalid habitat field inputs')
  const mapping = set.biomes.map(biome => HABITAT_IDS.indexOf(HABITAT_BY_BIOME[biome.id] ?? 'none'))
  const habitats = new Uint8Array(count)
  for (let index = 0; index < count; index++) {
    if (index % 128 === 0) yield { completed: index, total: count + places.length }
    const x = field.originX + index % field.cols * field.cellSize
    const z = field.originZ + Math.floor(index / field.cols) * field.cellSize
    if (x * x + z * z > field.radius * field.radius || (paths[index] ?? 0) > 0) continue
    habitats[index] = mapping[biomes[index] ?? -1] ?? 0
  }
  let work = 0
  for (const [placeIndex, place] of places.entries()) {
    const [width, depth] = place.size
    const reach = Math.hypot(width, depth) / 2
    const c0 = Math.max(0, Math.ceil((place.position[0] - reach - field.originX) / field.cellSize))
    const c1 = Math.min(field.cols - 1, Math.floor((place.position[0] + reach - field.originX) / field.cellSize))
    const r0 = Math.max(0, Math.ceil((place.position[1] - reach - field.originZ) / field.cellSize))
    const r1 = Math.min(field.rows - 1, Math.floor((place.position[1] + reach - field.originZ) / field.cellSize))
    const cos = Math.cos(place.rotation)
    const sin = Math.sin(place.rotation)
    for (let row = r0; row <= r1; row++) {
      for (let col = c0; col <= c1; col++) {
        if (work++ % 128 === 0) yield { completed: count + placeIndex, total: count + places.length }
        const dx = field.originX + col * field.cellSize - place.position[0]
        const dz = field.originZ + row * field.cellSize - place.position[1]
        if (Math.abs(dx * cos - dz * sin) <= width / 2 && Math.abs(dx * sin + dz * cos) <= depth / 2) habitats[row * field.cols + col] = 0
      }
    }
  }
  yield { completed: count + places.length, total: count + places.length }
  return habitats
}
