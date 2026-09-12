import type { LayoutTuning } from '../../config'
import { fbm } from './noise'

/** Pure position/seed field. Never consume the scatter RNG: its stream is shared. */
export function standMask(x: number, z: number, seed: number, tuning: LayoutTuning['stands']): number {
  const raw = fbm(x * tuning.frequency, z * tuning.frequency, seed, tuning.octaves, tuning.lacunarity, tuning.gain) * 0.5 + 0.5
  const t = Math.max(0, Math.min(1, (raw - tuning.threshold) / tuning.softness))
  const shaped = t * t * (3 - 2 * t)
  return tuning.floor + (1 - tuning.floor) * shaped ** tuning.contrast
}
