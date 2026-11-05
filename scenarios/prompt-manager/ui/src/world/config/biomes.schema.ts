import { z } from 'zod'

const hex = (what: string) => z.string().regex(/^#[0-9a-fA-F]{6}$/).describe(`${what} (hex colour)`)
export const VegetationEntrySchema = z.object({
  density: z.number().min(0).max(1).describe('Prop density (instances per square metre)'),
  class: z.enum(['tree', 'shrub', 'ground']).describe('Vegetation class controlling spacing and navigation'),
  scaleRef: z.enum(['tree', 'prop']).describe('Scene scale multiplier applied to this prop'),
})
const densityTable = z.record(z.string(), VegetationEntrySchema)
const window = (what: string, min: number, max: number) => z.tuple([
  z.number().min(min).max(max).describe(`${what} lower bound`),
  z.number().min(min).max(max).describe(`${what} upper bound`),
])

export const BiomeSchema = z.object({
  id: z.string().min(1).describe('Stable biome identifier'),
  bounds: z.object({
    height: window('Elevation (metres)', -20, 20),
    moisture: window('Moisture (0..1)', 0, 1),
    slope: window('Slope (radians)', 0, Math.PI / 2),
  }),
  ramp: z.array(hex('Ground colour stop')).min(2).max(3).describe('Ground colour ramp'),
  rock: hex('Rock colour'),
  path: hex('Path colour'),
  aoStrength: z.number().min(0).max(1).describe('Baked height-field occlusion strength (0..1)'),
  vegetation: densityTable.describe('Vegetation prop densities'),
  decor: densityTable.describe('Decor prop densities'),
})

export const BiomeSetSchema = z.object({
  id: z.string().min(1).describe('Stable biome-set identifier'),
  assetSet: z.string().min(1).describe('Baked asset set for landscape vegetation'),
  propScale: z.number().min(0.1).max(10).describe('Landscape prop scale (world metres per asset unit)'),
  treeScale: z.number().min(0.1).max(10).describe('Additional scale for landscape trees (multiplier)'),
  biomes: z.array(BiomeSchema).min(1).describe('Ordered biome rules; first match wins and the final rule must be catch-all'),
}).superRefine((set, context) => {
  const water = set.biomes.find((biome) => biome.id === 'water')
  if (!water) context.addIssue({ code: 'custom', path: ['biomes'], message: 'Every biome set must declare water' })
  else if (Object.keys(water.vegetation).length || Object.keys(water.decor).length) {
    context.addIssue({ code: 'custom', path: ['biomes'], message: 'Water must have empty vegetation and decor tables' })
  }
})

export const BiomeSetsSchema = z.object({
  park: BiomeSetSchema,
  office: BiomeSetSchema,
})

export type Biome = z.infer<typeof BiomeSchema>
export type BiomeSet = z.infer<typeof BiomeSetSchema>
