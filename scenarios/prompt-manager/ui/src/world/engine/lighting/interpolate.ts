import { Color } from 'three'
import { PERIOD_IDS, resolvePeriod, type LightingPeriod, type Scene, type WorldTuning } from '../../config'

/** Night sky and fog follow the visible sun, rather than retaining sunset pink
 * for hours while the sun is already far below the horizon. */
export function applySolarNight(period: LightingPeriod, night: LightingPeriod, sunHeight: number): LightingPeriod {
  const amount = Math.max(0, Math.min(1, -sunHeight / .25))
  const t = amount * amount * (3 - 2 * amount)
  const blend = (from: number, to: number) => from + (to - from) * t
  const color = (from: string, to: string) => `#${new Color(from).lerp(new Color(to), t).getHexString()}`
  return { ...period,
    sunElevationDeg: Math.asin(Math.max(-1, Math.min(1, sunHeight))) * 180 / Math.PI,
    backgroundColor: color(period.backgroundColor, night.backgroundColor),
    fogColor: color(period.fogColor, night.fogColor),
    exposure: blend(period.exposure, Math.max(.7, night.exposure)),
    ambientIntensity: blend(period.ambientIntensity, Math.max(.16, night.ambientIntensity)),
    envIntensity: blend(period.envIntensity, night.envIntensity),
    skyIntensity: blend(period.skyIntensity, night.skyIntensity),
  }
}

/** Apply after weather so quiet hours cannot accidentally restore local lamps.
 * Moonlit fill remains readable after the warm local lighting extinguishes.
 */
export function applyDeepNight(period: LightingPeriod, amount: number): LightingPeriod {
  const t = Math.max(0, Math.min(1, amount))
  if (t === 0) return period
  const blend = (from: number, to: number) => from + (to - from) * t
  const color = (from: string, to: string) => `#${new Color(from).lerp(new Color(to), t).getHexString()}`
  return { ...period,
    lampEmissive: period.lampEmissive * (1 - t),
    exposure: blend(period.exposure, .85),
    ambientIntensity: blend(period.ambientIntensity, .32),
    keyIntensity: blend(period.keyIntensity, .45),
    envIntensity: blend(period.envIntensity, .12),
    skyIntensity: blend(period.skyIntensity, .08),
    keyColor: color(period.keyColor, '#b6c9ef'),
    fogColor: color(period.fogColor, '#111c32'),
    backgroundColor: color(period.backgroundColor, '#080e20'),
  }
}

/** Named presets anchor the centre of each configured civil-time band. Blend
 * scene-resolved values cyclically, with zero slope at every anchor.
 */
export function continuousPeriod(scene: Scene, localMinutes: number, tuning: WorldTuning): LightingPeriod {
  if (!Number.isFinite(localMinutes)) throw new Error('Invalid lighting time')
  const hour = ((localMinutes / 60) % 24 + 24) % 24
  const anchors = PERIOD_IDS.map(id => {
    const band = tuning.lighting.periodHours[id]
    const span = ((band.to - band.from) % 24 + 24) % 24
    return { id, hour: (band.from + span / 2) % 24 }
  }).sort((a, b) => a.hour - b.hour)
  const first = anchors[0]
  if (!first) throw new Error('Missing lighting anchors')
  for (let index = 0; index < anchors.length; index++) {
    const a = anchors[index]
    const b = anchors[(index + 1) % anchors.length]
    if (!a || !b) continue
    const end = b.hour + (index === anchors.length - 1 ? 24 : 0)
    const time = hour < first.hour ? hour + 24 : hour
    if (time < a.hour || time >= end) continue
    const t = (time - a.hour) / (end - a.hour)
    const blend = t * t * (3 - 2 * t)
    const from = resolvePeriod(scene, a.id, tuning)
    const to = resolvePeriod(scene, b.id, tuning)
    const result = { ...from }
    const numberKeys = ['exposure', 'envIntensity', 'keyIntensity', 'ambientIntensity', 'fogNear', 'fogFar', 'skyIntensity', 'skyBlur', 'sunElevationDeg', 'lampEmissive'] as const
    for (const key of numberKeys) result[key] = from[key] + (to[key] - from[key]) * blend
    // Three decodes sRGB inputs before interpolation and encodes the output.
    for (const key of ['keyColor', 'fogColor', 'backgroundColor'] as const) result[key] = `#${new Color(from[key]).lerp(new Color(to[key]), blend).getHexString()}`
    return result
  }
  throw new Error('Invalid lighting anchors')
}
