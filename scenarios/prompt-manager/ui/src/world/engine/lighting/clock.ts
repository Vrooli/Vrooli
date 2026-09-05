import { useEffect, useState } from 'react'
import { periodForHour, type LightingTuning, type PeriodId } from '../../config'

export type LightingMode = { kind: 'clock' } | { kind: 'fixed'; period: PeriodId }

/** Fixed captures create no timer. Re-entering clock mode reads the hour immediately. */
export function useLightingPeriod(mode: LightingMode, lighting: LightingTuning): PeriodId {
  const [hour, setHour] = useState(() => new Date().getHours())
  const clock = mode.kind === 'clock'
  useEffect(() => {
    if (!clock) return
    const readHour = () => setHour(new Date().getHours())
    readHour()
    const interval = window.setInterval(readHour, lighting.clockPollSeconds * 1000)
    return () => window.clearInterval(interval)
  }, [clock, lighting.clockPollSeconds])
  return mode.kind === 'fixed' ? mode.period : periodForHour(hour, lighting)
}
