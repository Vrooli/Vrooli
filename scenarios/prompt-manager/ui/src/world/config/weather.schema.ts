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
  particleFallSpeed: z.number().min(0).max(100).describe('Particle downward speed (metres per second)'),
  particleSize: z.number().min(0.001).max(5).describe('Particle point size before perspective scaling (metres)'),
  particleColor: z.string().regex(/^#[0-9a-fA-F]{6}$/).describe('Particle colour (hex colour)'),
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
  lightingLimits: z.object({
    fogNearMax: z.number().min(0).max(100).describe('Maximum weather-adjusted fog near distance (framing-distance multiplier)'),
    fogFarMin: z.number().min(0.001).max(100).describe('Minimum weather-adjusted fog far distance (framing-distance multiplier)'),
    fogFarMax: z.number().min(0.001).max(100).describe('Maximum weather-adjusted fog far distance (framing-distance multiplier)'),
    exposureMax: z.number().min(0).max(20).describe('Maximum weather-adjusted exposure (multiplier)'),
    keyIntensityMax: z.number().min(0).max(100).describe('Maximum weather-adjusted key intensity (light intensity units)'),
    ambientIntensityMax: z.number().min(0).max(20).describe('Maximum weather-adjusted ambient intensity (light intensity units)'),
  }).refine((limits) => limits.fogFarMin <= limits.fogFarMax, { message: 'Fog far minimum must not exceed maximum', path: ['fogFarMin'] }),
  states: z.object({ clear: WeatherPresetSchema, cloudy: WeatherPresetSchema, rain: WeatherPresetSchema, snow: WeatherPresetSchema }),
  pressure: z.object({
    recentFailureWeight: ratio('Weight of recent run failures'),
    failedActorWeight: ratio('Weight of actors currently failed'),
    expiredGatheringWeight: ratio('Weight of expired gatherings'),
    eventWindowSeconds: seconds('Age window for run outcome events'),
  }),
  pressureSmoothingSeconds: seconds('Time constant used to smooth weather pressure'),
  particleBaseCount: z.number().int().min(0).max(20000).describe('Particle count at rate and profile scale 1 (count)'),
  particles: z.object({
    spiralAngleStep: z.number().min(0.001).max(Math.PI * 2).describe('Angle between consecutive particles (radians)'),
    columnRadius: z.number().min(0).max(200).describe('Particle column radius around the camera target (metres)'),
    columnHeight: z.number().min(0.1).max(200).describe('Particle column vertical wrap height (metres)'),
    verticalStride: z.number().min(0).max(200).describe('Initial height increment between particles (metres)'),
    pointSizeScale: z.number().min(1).max(2000).describe('Shader point-size perspective scale (pixels)'),
    opacity: ratio('Particle opacity'),
  }),
  cloudPlaneSpan: z.number().min(0.1).max(10).describe('Cloud plane span relative to world bounds (multiplier)'),
  cloudOpacityScale: ratio('Cloud opacity per unit coverage'),
  cloudAltitude: z.number().min(10).max(1000).describe('Cloud layer height above the world (metres)'),
})

export type WeatherId = z.infer<typeof WeatherIdSchema>
export type WeatherPreset = z.infer<typeof WeatherPresetSchema>
export type WeatherTuning = z.infer<typeof WeatherTuningSchema>
