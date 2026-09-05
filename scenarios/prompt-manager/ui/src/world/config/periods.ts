import type { PeriodId, WorldTuning } from './tuning.schema'
import { PERIOD_IDS } from './tuning.schema'

/**
 * Map an hour of day (0..24, fractional allowed) to a lighting period using
 * the bands in tuning.lighting.periodHours. A band whose `to` is smaller than
 * its `from` wraps past midnight.
 */
export function periodForHour(hour: number, lighting: WorldTuning['lighting']): PeriodId {
  const h = ((hour % 24) + 24) % 24
  for (const id of PERIOD_IDS) {
    const band = lighting.periodHours[id]
    const inBand = band.from <= band.to ? h >= band.from && h < band.to : h >= band.from || h < band.to
    if (inBand) return id
  }
  return 'day'
}

export function isPeriodId(value: string | null | undefined): value is PeriodId {
  return value === 'dawn' || value === 'day' || value === 'dusk' || value === 'night'
}
