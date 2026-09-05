import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { PERIOD_IDS, periodForHour, tuning } from '../../config'
import { useLightingPeriod, type LightingMode } from './clock'
import { applyWeather } from './weather'

afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks() })

describe('weather lighting limits', () => {
  it('preserves the former clamps for every period and weather preset', () => {
    for (const period of Object.values(tuning.lighting.periods)) {
      for (const weather of ['clear', 'cloudy', 'rain', 'snow'] as const) {
        const preset = tuning.weather.states[weather]
        const result = applyWeather(period, weather, tuning.weather)
        expect(result.fogNear).toBe(Math.max(0, Math.min(10, period.fogNear * preset.fogNearScale)))
        expect(result.fogFar).toBe(Math.max(0.1, Math.min(20, period.fogFar * preset.fogFarScale)))
        expect(result.exposure).toBe(Math.max(0, Math.min(4, period.exposure * preset.exposureScale)))
        expect(result.keyIntensity).toBe(Math.max(0, Math.min(20, period.keyIntensity * preset.keyIntensityScale)))
        expect(result.ambientIntensity).toBe(Math.max(0, Math.min(4, period.ambientIntensity * preset.ambientScale)))
      }
    }
  })

  it('uses configured ceilings and floor independently', () => {
    const weather = structuredClone(tuning.weather)
    weather.lightingLimits = { fogNearMax: 0.01, fogFarMin: 8, fogFarMax: 9, exposureMax: 0.01, keyIntensityMax: 0.01, ambientIntensityMax: 0.01 }
    const result = applyWeather(tuning.lighting.periods.day, 'clear', weather)
    expect(result).toMatchObject({ fogNear: 0.01, fogFar: 8, exposure: 0.01, keyIntensity: 0.01, ambientIntensity: 0.01 })
  })
})

describe('period clock', () => {
  it('resolves every configured band boundary and wraps midnight', () => {
    for (const id of PERIOD_IDS) {
      const band = tuning.lighting.periodHours[id]
      expect(periodForHour(band.from, tuning.lighting)).toBe(id)
      expect(periodForHour(band.to - 0.001, tuning.lighting)).toBe(id)
      expect(periodForHour(band.to, tuning.lighting)).not.toBe(id)
    }
    expect(periodForHour(0, tuning.lighting)).toBe('night')
    expect(periodForHour(24, tuning.lighting)).toBe('night')
    expect(periodForHour(-1, tuning.lighting)).toBe('night')
  })
  it('advances through the day without remounting and cleans up the interval', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 8, 4, 4, 59, 0))
    const interval = vi.spyOn(window, 'setInterval')
    const { result, unmount } = renderHook(() => useLightingPeriod({ kind: 'clock' }, tuning.lighting))
    expect(result.current).toBe('night')
    for (const id of PERIOD_IDS) {
      act(() => {
        vi.setSystemTime(new Date(2026, 8, 4, tuning.lighting.periodHours[id].from, 0, 0))
        vi.advanceTimersByTime(tuning.lighting.clockPollSeconds * 1000)
      })
      expect(result.current).toBe(id)
    }
    expect(interval).toHaveBeenCalledTimes(1)
    unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
  it('starts no interval in fixed mode and resumes with a fresh hour after switching back', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 8, 4, 12))
    const interval = vi.spyOn(window, 'setInterval')
    const { result, rerender, unmount } = renderHook(({ mode }: { mode: LightingMode }) => useLightingPeriod(mode, tuning.lighting), { initialProps: { mode: { kind: 'fixed', period: 'dusk' } as LightingMode } })
    expect(interval).not.toHaveBeenCalled()
    act(() => { vi.setSystemTime(new Date(2026, 8, 4, 23)); vi.advanceTimersByTime(3600000) })
    expect(result.current).toBe('dusk')
    rerender({ mode: { kind: 'clock' } })
    expect(result.current).toBe('night')
    expect(interval).toHaveBeenCalledTimes(1)
    rerender({ mode: { kind: 'fixed', period: 'day' } })
    expect(result.current).toBe('day')
    expect(vi.getTimerCount()).toBe(0)
    unmount()
  })
})
