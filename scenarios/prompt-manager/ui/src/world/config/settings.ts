import type { SettingImpact } from './settingImpact'
import { z } from 'zod'
import { QualityProfileSchema, PERIOD_IDS, QUALITY_PROFILE_IDS, SCENE_IDS } from './tuning.schema'
import { scenes } from './scenes'
import { tuning } from './tuning'
import { WeatherIdSchema } from './weather.schema'

/** Shared public/development setting contracts. Runtime effects remain owned by composition. */
export interface IntegerSetting {
  id: string
  label: string
  description: string
  defaultValue: number
  minimum: number
  maximum: number
  impact: SettingImpact
  persistence: 'operator' | 'session' | 'development'
}
export const integerSettings = {
  seed: { id: 'seed', label: 'World seed', description: 'Changes the generated world. Saved in this page’s URL; default is 1.', defaultValue: 1, minimum: 0, maximum: 4294967295, impact: 'world', persistence: 'operator' },
  actors: { id: 'actors', label: 'Synthetic actors', description: 'Creates a deterministic demonstration roster. Zero uses the live roster.', defaultValue: 0, minimum: 0, maximum: 100000, impact: 'world', persistence: 'development' },
} as const satisfies Record<string, IntegerSetting>

export function parseIntegerSetting(descriptor: IntegerSetting, raw: string | null): { value: number; error: string | null } {
  if (raw === null) return { value: descriptor.defaultValue, error: null }
  const value = /^\d+$/.test(raw) ? Number(raw) : NaN
  if (Number.isSafeInteger(value) && value >= descriptor.minimum && value <= descriptor.maximum) return { value, error: null }
  return { value: descriptor.defaultValue, error: `Enter a whole number from ${descriptor.minimum} to ${descriptor.maximum}.` }
}

export interface ChoiceSetting<T extends string> {
  id: string
  label: string
  description: string
  defaultValue: T
  choices: ReadonlyArray<{ id: T; label: string }>
  impact: IntegerSetting['impact']
  persistence: IntegerSetting['persistence']
}
const title = (id: string) => id.charAt(0).toUpperCase() + id.slice(1)
export const choiceSettings = {
  scene: { id: 'scene', label: 'Scene', description: 'Selects the generated environment.', defaultValue: 'park', choices: SCENE_IDS.map(id => ({ id, label: scenes[id].title })), impact: 'world', persistence: 'operator' } satisfies ChoiceSetting<(typeof SCENE_IDS)[number]>,
  profile: { id: 'profile', label: 'Quality', description: 'Selects rendering cost and detail without changing world identity.', defaultValue: tuning.quality.defaultProfile, choices: QUALITY_PROFILE_IDS.map(id => ({ id, label: title(id) })), impact: 'geometry', persistence: 'operator' } satisfies ChoiceSetting<(typeof QUALITY_PROFILE_IDS)[number]>,
  period: { id: 'period', label: 'Time of day', description: 'Uses the shared clock or a named lighting preset.', defaultValue: 'clock', choices: [{ id: 'clock', label: 'Clock' }, ...PERIOD_IDS.map(id => ({ id, label: title(id) }))], impact: 'live', persistence: 'operator' } satisfies ChoiceSetting<'clock' | (typeof PERIOD_IDS)[number]>,
  weather: { id: 'weather', label: 'Weather', description: 'Automatic follows world activity. Your selection is saved in this page’s URL.', defaultValue: 'auto', choices: [{ id: 'auto', label: 'Automatic' }, ...WeatherIdSchema.options.map(id => ({ id, label: title(id) }))], impact: 'geometry', persistence: 'session' } satisfies ChoiceSetting<'auto' | (typeof WeatherIdSchema.options)[number]>,
} as const

export function parseChoiceSetting<T extends string>(descriptor: ChoiceSetting<T>, raw: string | null): { value: T; error: string | null } {
  if (raw === null) return { value: descriptor.defaultValue, error: null }
  const choice = descriptor.choices.find(choice => choice.id === raw)
  return choice ? { value: choice.id, error: null } : { value: descriptor.defaultValue, error: `Choose ${descriptor.choices.map(choice => choice.id).join(', ')}.` }
}

export const numericOverrides = {
  dpr: { id: 'dpr', label: 'Pixel ratio', schema: QualityProfileSchema.shape.dpr, defaultSource: 'Selected quality profile', persistence: 'development', unit: 'multiplier', impact: 'geometry', description: 'Overrides the rendering pixel ratio cap.' },
  msaa: { id: 'msaa', label: 'Multisample count', schema: QualityProfileSchema.shape.msaa, defaultSource: 'Selected quality profile', persistence: 'development', unit: 'samples', impact: 'geometry', description: 'Overrides composer multisampling; zero disables it.' },
  lampLights: { id: 'lampLights', label: 'Lamp light limit', schema: QualityProfileSchema.shape.lampLights, defaultSource: 'Selected quality profile', persistence: 'development', unit: 'lights', impact: 'live', description: 'Limits simultaneously rendered lamp point lights.' },
  pressure: { id: 'pressure', label: 'Weather pressure', schema: z.number().min(0).max(1), defaultSource: 'Automatic', persistence: 'development', unit: 'fraction', impact: 'live', description: 'Overrides the weather activity pressure for diagnostics.' },
} as const

/** Diagnostic overrides use schema bounds and retain the active default on invalid input. */
export function parseNumericOverride<T extends number | null>(descriptor: { schema: z.ZodNumber }, raw: string | null, fallback: T): { value: number | T; error: string | null } {
  if (raw === null) return { value: fallback, error: null }
  const syntax = (descriptor.schema.format === 'safeint') ? /^\d+$/ : /^(?:\d+(?:\.\d*)?|\.\d+)$/
  const parsed = descriptor.schema.safeParse(syntax.test(raw) ? Number(raw) : NaN)
  if (parsed.success) return { value: parsed.data, error: null }
  return { value: fallback, error: `Enter ${(descriptor.schema.format === 'safeint') ? 'a whole number' : 'a decimal number'} from ${descriptor.schema.minValue} to ${descriptor.schema.maxValue}.` }
}
