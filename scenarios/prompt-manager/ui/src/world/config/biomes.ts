import raw from './biomes.json'
import { BiomeSetsSchema, type BiomeSet } from './biomes.schema'

const parsed = BiomeSetsSchema.parse(raw)
export const biomeSets: Record<'park' | 'office', BiomeSet> = parsed
