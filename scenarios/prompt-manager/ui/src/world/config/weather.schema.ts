import { z } from 'zod'

const ratio = (what: string) => z.number().min(0).max(1).describe(`${what} (0..1)`)
const scale = (what: string) => z.number().min(0).max(3).describe(`${what} (multiplier)`)
const seconds = (what: string) => z.number().min(1).max(3600).describe(`${what} (seconds)`)

export const WeatherIdSchema = z.enum(['clear', 'cloudy', 'rain', 'snow'])
export const WeatherPresetSchema = z.object({
  fogNearScale: scale('Fog near-distance scale'),
  fogFarScale: scale('Fog far-distance scale'),
  exposureScale: scale('Exposure scale'),
  keyIntensityScale: scale('Directional light intensity scale'),
  ambientScale: scale('Ambient light intensity scale'),
  skyBlurAdd: ratio('Additional sky blur'),
  cloudCoverage: ratio('Cloud layer coverage'),
  particleRate: ratio('Weather particle rate'),
  wetness: ratio('Terrain wetness'),
  terrainTint: z.string().regex(/^#[0-9a-fA-F]{6}$/).describe('Terrain weather tint (hex colour)'),
  terrainTintMix: ratio('Share of the terrain colour replaced by the weather tint'),
  terrainShadowTint: z.string().regex(/^#[0-9a-fA-F]{6}$/).describe('Secondary terrain tint used for weather variation (hex colour)'),
  terrainTintVariation: ratio('Maximum deterministic blend from terrainTint toward terrainShadowTint'),
  skyTint: z.string().regex(/^#[0-9a-fA-F]{6}$/).describe('Sky and fog weather tint (hex colour)'),
  skyTintMix: ratio('Share of the period sky and fog colours replaced by the weather tint'),
  minSeconds: seconds('Minimum state duration'),
  maxSeconds: seconds('Maximum state duration'),
})

export const WeatherTuningSchema = z.object({
  states: z.object({ clear: WeatherPresetSchema, cloudy: WeatherPresetSchema, rain: WeatherPresetSchema, snow: WeatherPresetSchema }),
  pressure: z.object({
    recentFailureWeight: ratio('Weight of recent run failures'),
    failedActorWeight: ratio('Weight of actors currently failed'),
    expiredGatheringWeight: ratio('Weight of expired gatherings'),
    eventWindowSeconds: seconds('Age window for run outcome events'),
  }),
  pressureSmoothingSeconds: seconds('Time constant used to smooth weather pressure'),
  particleBaseCount: z.number().int().min(0).max(20000).describe('Particle count at rate and profile scale 1 (count)'),
  cloudAltitude: z.number().min(10).max(1000).describe('Cloud layer height above the world (metres)'),
})

export type WeatherId = z.infer<typeof WeatherIdSchema>
export type WeatherPreset = z.infer<typeof WeatherPresetSchema>
export type WeatherTuning = z.infer<typeof WeatherTuningSchema>
