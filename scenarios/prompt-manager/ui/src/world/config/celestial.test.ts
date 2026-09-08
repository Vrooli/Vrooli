import { describe, expect, it } from 'vitest'
import { lunarPhase, stylizedSunDirection } from './celestial'
import reference from './lunar-reference-2026.json'

describe('stylized civil sun path', () => {
  it('rises and sets on opposite horizons and stays below the world at midnight', () => {
    const rise = stylizedSunDirection(360), noon = stylizedSunDirection(720), set = stylizedSunDirection(1080), night = stylizedSunDirection(0)
    expect(rise[0]).toBeCloseTo(1)
    expect(rise[1]).toBeCloseTo(0)
    expect(set[0]).toBeCloseTo(-1)
    expect(set[1]).toBeCloseTo(0)
    expect(noon[1]).toBeGreaterThan(.8)
    expect(night[1]).toBeLessThan(-.8)
  })
  it('is normalized, continuous and cyclic independently of camera or quality', () => {
    for (let minute = 0; minute < 1440; minute++) {
      expect(Math.hypot(...stylizedSunDirection(minute))).toBeCloseTo(1, 12)
      const a = stylizedSunDirection(minute), b = stylizedSunDirection(minute + .001)
      expect(Math.hypot(...a.map((value, index) => value - (b[index] ?? 0)))).toBeLessThan(.00001)
    }
    expect(stylizedSunDirection(0)).toEqual(stylizedSunDirection(1440))
    expect(() => stylizedSunDirection(NaN)).toThrow('Invalid')
  })
})

describe('UTC geometric lunar phase', () => {
  it('matches all 50 USNO 2026 phase instants within a predeclared 0.2-degree longitude tolerance', () => {
    // https://aa.usno.navy.mil/api/moon/phases/year?year=2026 (retrieved 2026-09-05)
    const cycles: Record<string, number> = { 'New Moon': 0, 'First Quarter': .25, 'Full Moon': .5, 'Last Quarter': .75 }
    expect(reference.phasedata).toHaveLength(50)
    for (const event of reference.phasedata) {
      const utc = Date.parse(`${event.year}-${String(event.month).padStart(2, '0')}-${String(event.day).padStart(2, '0')}T${event.time}:00Z`)
      const phase = lunarPhase(utc)
      const error = Math.abs(phase.cycle - (cycles[event.phase] ?? NaN))
      expect(Math.min(error, 1 - error) * 360, JSON.stringify(event)).toBeLessThan(.2)
      expect(phase.illumination).toBeGreaterThanOrEqual(0)
      expect(phase.illumination).toBeLessThanOrEqual(1)
    }
  })
  it('provides continuous waxing/waning illumination without timezone or frame inputs', () => {
    const firstQuarter = lunarPhase(Date.parse('2026-01-26T04:47:00Z'))
    const lastQuarter = lunarPhase(Date.parse('2026-01-10T15:48:00Z'))
    expect(firstQuarter.waxing).toBe(true)
    expect(lastQuarter.waxing).toBe(false)
    expect(firstQuarter.illumination).toBeCloseTo(.5, 2)
    expect(lastQuarter.illumination).toBeCloseTo(.5, 2)
    const instant = Date.parse('2026-09-05T12:00:00Z')
    expect(Math.abs(lunarPhase(instant + 1000).illumination - lunarPhase(instant).illumination)).toBeLessThan(.00001)
    expect(() => lunarPhase(NaN)).toThrow('Invalid')
  })
})
