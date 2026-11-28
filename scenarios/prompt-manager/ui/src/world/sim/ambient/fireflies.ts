import { ambientPolicy } from '../../config/ambient'
import { HABITAT_IDS } from '../../config/habitats'
import { hashString } from '../rng'
import { MAX_TERRAIN_CELLS, type TerrainField } from '../terrain/field'
import type { HabitatRegionIndex } from '../terrain/habitats'

export interface FireflySite {
  id: string
  sample: number
  region: number
  position: readonly [number, number, number]
  phase: number
  rank: number
}

/** Select a stable bounded prefix from genuine wetland interfaces. Excluded
 * samples never create edges, so paths and buildings cannot attract insects.
 * This independent cosmetic stream never consumes the agent RNG or nav grid.
 */
export function* fireflySiteSteps(seed: number, field: TerrainField, habitats: Uint8Array, index: HabitatRegionIndex) {
  const count = field.cols * field.rows
  if (!Number.isSafeInteger(count) || count < 0 || count > MAX_TERRAIN_CELLS ||
      habitats.length !== count || index.labels.length !== count) throw new Error('Invalid firefly habitat inputs')
  const wetland = HABITAT_IDS.indexOf('wetland')
  const sites: FireflySite[] = []
  const maximum = ambientPolicy.fireflies.maximumVisible.ultra
  let work = 0
  for (const region of index.regions) {
    if (region.habitat !== 'wetland') continue
    for (let offset = region.offset; offset < region.offset + region.samples; offset++) {
      if (work++ % 128 === 0) yield { completed: work, total: index.members.length }
      const sample = index.members[offset]
      if (sample === undefined || sample >= count || habitats[sample] !== wetland || index.labels[sample] !== region.id) throw new Error('Invalid wetland region member')
      const col = sample % field.cols, row = Math.floor(sample / field.cols)
      const isInterface = (neighbor: number) => (habitats[neighbor] ?? 0) > 0 && habitats[neighbor] !== wetland
      if (!(col > 0 && isInterface(sample - 1) || col + 1 < field.cols && isInterface(sample + 1) ||
            row > 0 && isInterface(sample - field.cols) || row + 1 < field.rows && isInterface(sample + field.cols))) continue
      const rank = hashString(`firefly-site-v1:${seed}:${sample}`)
      const site: FireflySite = {
        id: `firefly:${seed}:${sample}`, sample, region: region.id, rank,
        position: [field.originX + col * field.cellSize, field.height[sample] ?? 0, field.originZ + row * field.cellSize],
        phase: hashString(`firefly-phase-v1:${seed}:${sample}`) / 4294967296 * Math.PI * 2,
      }
      const insertion = sites.findIndex(other => other.rank > rank || other.rank === rank && other.sample > sample)
      if (insertion >= 0) sites.splice(insertion, 0, site)
      else if (sites.length < maximum) sites.push(site)
      if (sites.length > maximum) sites.pop()
    }
  }
  return sites
}

/** Absolute time gives identical poses after pauses and across frame rates.
 * Vertical travel retains the habitat sample exactly, including near buildings.
 */
export function fireflyPose(site: FireflySite, seconds: number) {
  const cycle = seconds % 60 * Math.PI / 30 + site.phase
  const [low, high] = ambientPolicy.fireflies.height
  return {
    position: [site.position[0], site.position[1] + low + (high - low) * (0.5 + 0.5 * Math.sin(cycle)), site.position[2]] as const,
    glow: 0.15 + 0.85 * Math.pow(0.5 + 0.5 * Math.sin(cycle * 3), 4),
  }
}
