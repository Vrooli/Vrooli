import type { LightingPeriod, WeatherId, WeatherTuning } from '../../config'
import { Color } from 'three'

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}

export function applyWeather(period: LightingPeriod, weather: WeatherId, tuning: WeatherTuning): LightingPeriod {
  const preset = tuning.states[weather]
  const tint = (source: string) => `#${new Color(source).lerp(new Color(preset.skyTint), preset.skyTintMix).getHexString()}`
  return {
    ...period,
    backgroundColor: tint(period.backgroundColor),
    fogColor: tint(period.fogColor),
    fogNear: clamp(period.fogNear * preset.fogNearScale, 0, 10),
    fogFar: clamp(period.fogFar * preset.fogFarScale, 0.1, 20),
    exposure: clamp(period.exposure * preset.exposureScale, 0, 4),
    keyIntensity: clamp(period.keyIntensity * preset.keyIntensityScale, 0, 20),
    ambientIntensity: clamp(period.ambientIntensity * preset.ambientScale, 0, 4),
    skyBlur: clamp(period.skyBlur + preset.skyBlurAdd, 0, 1),
  }
}
