import { deepNightAmount } from './celestial'
import { describe, expect, it } from 'vitest'
import { celestialDirections, celestialKey, lunarPhase, stylizedSunDirection } from './celestial'
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

describe('cyclic quiet hours', () => {
  it('fades across midnight, holds through late night, and restores daylight', () => {
    expect(deepNightAmount(22 * 60)).toBe(0)
    expect(deepNightAmount(0)).toBeCloseTo(.5)
    for (const hour of [1, 2, 3, 4]) expect(deepNightAmount(hour * 60)).toBe(1)
    expect(deepNightAmount(270)).toBeCloseTo(.5)
    for (const hour of [5, 12, 18, 23]) expect(deepNightAmount(hour * 60)).toBe(0)
    for (const boundary of [0, 60, 240, 300, 1380, 1440]) {
      expect(Math.abs(deepNightAmount(boundary - .001) - deepNightAmount(boundary + .001))).toBeLessThan(.0001)
    }
    expect(deepNightAmount(-1440)).toBe(deepNightAmount(1440))
    expect(() => deepNightAmount(NaN)).toThrow()
  })
})

describe('shared visible-body and world-light directions', () => {
  it('keeps full and new moons opposite and beside the sun, across clock and preset views', () => {
    for (const preset of [undefined, { elevationDegrees: -20, setting: false }]) {
      const full = celestialDirections(0, { cycle: .5, latitudeDegrees: 0 }, preset)
      const empty = celestialDirections(0, { cycle: 0, latitudeDegrees: 0 }, preset)
      for (let axis = 0; axis < 3; axis++) {
        expect(full.moon[axis]).toBeCloseTo(-(full.sun[axis] ?? 0), 12)
        expect(empty.moon[axis]).toBeCloseTo(empty.sun[axis] ?? 0, 12)
      }
      for (const cycle of [.1, .25, .5, .75, .9]) expect(Math.hypot(...celestialDirections(250, { cycle, latitudeDegrees: 5 }, preset).moon)).toBeCloseTo(1, 12)
    }
  })
  it('casts full-moon light from the visible moon, dims with phase/clouds, and contributes nothing below the horizon', () => {
    const directions = celestialDirections(0, { cycle: .5, latitudeDegrees: 0 })
    const full = celestialKey(directions, 1, 0, 4)
    expect(full.intensity).toBeGreaterThan(.5)
    expect(full.direction).toEqual(directions.moon)
    const quarter = celestialKey(directions, .5, 0, 4)
    expect(quarter.intensity).toBeLessThan(full.intensity / 3)
    expect(celestialKey(directions, 0, 0, 4).intensity).toBe(0)
    expect(celestialKey(directions, 1, 1, 4).intensity).toBe(0)
    expect(celestialKey({ ...directions, moon: [0, -1, 0] }, 1, 0, 4).intensity).toBe(0)
    expect(celestialKey(directions, 1, .6, 4).intensity).toBeLessThan(full.intensity / 4)
  })
  it('keeps daytime illumination and a continuous horizon transition', () => {
    const day = celestialDirections(720, { cycle: 0, latitudeDegrees: 0 })
    expect(celestialKey(day, 1, 0, 4).intensity).toBe(4)
    expect(celestialKey(day, 1, 0, 4).moonIntensity).toBe(0)
    const a = celestialKey(celestialDirections(360 - .001, { cycle: .5, latitudeDegrees: 0 }), 1, 0, 4)
    const b = celestialKey(celestialDirections(360 + .001, { cycle: .5, latitudeDegrees: 0 }), 1, 0, 4)
    expect(Math.abs(a.intensity - b.intensity)).toBeLessThan(.0001)
  })
})
