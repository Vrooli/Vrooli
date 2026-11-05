import raw from './biomes.json'
import { BiomeSetsSchema, type BiomeSet } from './biomes.schema'
import type { Scene } from './scenes.schema'

const parsed = BiomeSetsSchema.parse(raw)
export const biomeSets: Record<'park' | 'office', BiomeSet> = parsed

/** One stable index palette for the base landscape and optional centre. */
export function sceneBiomes(scene: Scene): BiomeSet {
  const base = biomeSets[scene.biomeSet]
  const centre = scene.centre?.biomeSet
  if (!centre || centre === scene.biomeSet) return base
  const ids = new Set(base.biomes.map((biome) => biome.id))
  return { ...base, biomes: [...base.biomes, ...biomeSets[centre].biomes.filter((biome) => !ids.has(biome.id))] }
}
