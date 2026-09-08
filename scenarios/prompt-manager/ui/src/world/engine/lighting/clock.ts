import { useEffect, useMemo, useState } from 'react'
import { WorldClock } from '../../config/clock'
import { periodForHour, type LightingTuning, type PeriodId } from '../../config'

export type LightingMode = { kind: 'clock' } | { kind: 'fixed'; period: PeriodId }

/** Fixed captures create no timer. Re-entering clock mode reads the hour immediately. */
export function useLightingSample(mode: LightingMode, lighting: LightingTuning, suppliedClock?: WorldClock) {
  const worldClock = useMemo(() => suppliedClock ?? new WorldClock(), [suppliedClock])
  const [hour, setHour] = useState(() => worldClock.snapshot().localMinutes / 60)
  const clock = mode.kind === 'clock'
  useEffect(() => {
    if (!clock) return
    let interval: number | undefined
    const readHour = () => { worldClock.refreshTimeZone(); setHour(worldClock.snapshot().localMinutes / 60) }
    const reset = () => {
      window.clearInterval(interval)
      readHour()
      if (worldClock.snapshot().timeScale > 0) interval = window.setInterval(readHour, lighting.clockPollSeconds * 1000)
    }
    reset()
    const off = worldClock.subscribe(reset)
    window.addEventListener('focus', reset)
    return () => { off(); window.clearInterval(interval); window.removeEventListener('focus', reset) }
  }, [clock, lighting.clockPollSeconds, worldClock])
  return { periodId: mode.kind === 'fixed' ? mode.period : periodForHour(hour, lighting), localMinutes: hour * 60 }
}

export function useLightingPeriod(mode: LightingMode, lighting: LightingTuning, suppliedClock?: WorldClock): PeriodId {
  return useLightingSample(mode, lighting, suppliedClock).periodId
}
