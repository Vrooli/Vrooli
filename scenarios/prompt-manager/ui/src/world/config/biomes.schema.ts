import { z } from 'zod'

const hex = (what: string) => z.string().regex(/^#[0-9a-fA-F]{6}$/).describe(`${what} (hex colour)`)
const densityTable = z.record(z.string(), z.number().min(0).max(2).describe('Prop density (instances per square metre)'))
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
  biomes: z.array(BiomeSchema).min(1).describe('Ordered biome rules; first match wins and the final rule must be catch-all'),
})

export const BiomeSetsSchema = z.object({
  park: BiomeSetSchema,
  office: BiomeSetSchema,
})

export type Biome = z.infer<typeof BiomeSchema>
export type BiomeSet = z.infer<typeof BiomeSetSchema>
