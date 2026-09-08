import { Color } from 'three'
import { PERIOD_IDS, resolvePeriod, type LightingPeriod, type Scene, type WorldTuning } from '../../config'

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
