import type { LightingPeriod, WeatherId, WeatherTuning } from '../../config'
import { Color } from 'three'

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}

export function applyWeather(period: LightingPeriod, weather: WeatherId, tuning: WeatherTuning): LightingPeriod {
  const preset = tuning.states[weather]
  const limits = tuning.lightingLimits
  const tint = (source: string) => `#${new Color(source).lerp(new Color(preset.skyTint), preset.skyTintMix).getHexString()}`
  return {
    ...period,
    backgroundColor: tint(period.backgroundColor),
    fogColor: tint(period.fogColor),
    fogNear: clamp(period.fogNear * preset.fogNearScale, 0, limits.fogNearMax),
    fogFar: clamp(period.fogFar * preset.fogFarScale, limits.fogFarMin, limits.fogFarMax),
    exposure: clamp(period.exposure * preset.exposureScale, 0, limits.exposureMax),
    keyIntensity: clamp(period.keyIntensity * preset.keyIntensityScale, 0, limits.keyIntensityMax),
    ambientIntensity: clamp(period.ambientIntensity * preset.ambientScale, 0, limits.ambientIntensityMax),
    skyBlur: clamp(period.skyBlur + preset.skyBlurAdd, 0, 1),
  }
}
