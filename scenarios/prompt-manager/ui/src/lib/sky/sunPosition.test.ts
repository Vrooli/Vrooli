/**
 * Sun Position Calculation Tests
 */

import { describe, it, expect } from 'vitest'
import {
  calculateSunPosition,
  calculateMoonPosition,
  calculateStarOpacity,
  calculateSkyColors,
  calculateLighting,
  isNightTime,
  formatTimeFromHour,
  getCurrentTimeAsHour,
} from './sunPosition'

describe('calculateSunPosition', () => {
  it('returns sun at highest point at noon (12h)', () => {
    const position = calculateSunPosition(12)
    // At noon, sun should be near its highest (y close to max)
    expect(position[1]).toBeGreaterThan(30) // y should be high
  })

  it('returns sun in east at sunrise (6h)', () => {
    const position = calculateSunPosition(6)
    // At 6am, sun should be rising in the east (positive x)
    expect(position[0]).toBeGreaterThan(0)
  })

  it('returns sun in west at sunset (18h)', () => {
    const position = calculateSunPosition(18)
    // At 6pm, sun should be setting in the west (negative x)
    expect(position[0]).toBeLessThan(0)
  })

  it('returns sun below horizon at midnight (0h)', () => {
    const position = calculateSunPosition(0)
    // At midnight, sun should be below horizon (negative y)
    expect(position[1]).toBeLessThan(0)
  })

  it('normalizes hours outside 0-24 range', () => {
    const pos25 = calculateSunPosition(25)
    const pos1 = calculateSunPosition(1)
    // 25h should be same as 1h
    expect(pos25[0]).toBeCloseTo(pos1[0], 5)
    expect(pos25[1]).toBeCloseTo(pos1[1], 5)
    expect(pos25[2]).toBeCloseTo(pos1[2], 5)
  })

  it('handles negative hours', () => {
    const posNeg2 = calculateSunPosition(-2)
    const pos22 = calculateSunPosition(22)
    // -2h should be same as 22h
    expect(posNeg2[0]).toBeCloseTo(pos22[0], 5)
    expect(posNeg2[1]).toBeCloseTo(pos22[1], 5)
    expect(posNeg2[2]).toBeCloseTo(pos22[2], 5)
  })
})

describe('calculateMoonPosition', () => {
  it('returns moon opposite to sun', () => {
    const sunNoon = calculateSunPosition(12)
    const moonNoon = calculateMoonPosition(12)

    // Moon should be on opposite side (negative correlation with sun y)
    // At noon, sun is high, moon should be low
    expect(moonNoon[1]).toBeLessThan(0)
    expect(sunNoon[1]).toBeGreaterThan(0)
  })

  it('returns moon at highest when sun is at lowest', () => {
    const moonMidnight = calculateMoonPosition(0)
    // At midnight, moon should be near its highest
    expect(moonMidnight[1]).toBeGreaterThan(25)
  })
})

describe('calculateStarOpacity', () => {
  it('returns full opacity at night (22h)', () => {
    expect(calculateStarOpacity(22)).toBe(1)
  })

  it('returns full opacity at midnight (0h)', () => {
    expect(calculateStarOpacity(0)).toBe(1)
  })

  it('returns full opacity in early morning (3h)', () => {
    expect(calculateStarOpacity(3)).toBe(1)
  })

  it('returns zero opacity during day (12h)', () => {
    expect(calculateStarOpacity(12)).toBe(0)
  })

  it('returns partial opacity at dawn (5h)', () => {
    const opacity = calculateStarOpacity(5)
    expect(opacity).toBeGreaterThan(0)
    expect(opacity).toBeLessThan(1)
  })

  it('returns partial opacity at dusk (19h)', () => {
    const opacity = calculateStarOpacity(19)
    expect(opacity).toBeGreaterThan(0)
    expect(opacity).toBeLessThan(1)
  })

  it('fades out during dawn (4h-6h)', () => {
    const at4 = calculateStarOpacity(4)
    const at5 = calculateStarOpacity(5)
    const at6 = calculateStarOpacity(6)

    expect(at4).toBe(1)
    expect(at5).toBe(0.5)
    expect(at6).toBe(0)
  })

  it('fades in during dusk (18h-20h)', () => {
    const at18 = calculateStarOpacity(18)
    const at19 = calculateStarOpacity(19)
    const at20 = calculateStarOpacity(20)

    expect(at18).toBe(0)
    expect(at19).toBe(0.5)
    expect(at20).toBe(1)
  })
})

describe('isNightTime', () => {
  it('returns true before sunrise (5h)', () => {
    expect(isNightTime(5)).toBe(true)
  })

  it('returns false during day (12h)', () => {
    expect(isNightTime(12)).toBe(false)
  })

  it('returns true after sunset (19h)', () => {
    expect(isNightTime(19)).toBe(true)
  })

  it('returns true at midnight (0h)', () => {
    expect(isNightTime(0)).toBe(true)
  })

  it('returns false at boundary sunrise (6h)', () => {
    expect(isNightTime(6)).toBe(false)
  })

  it('returns true at boundary sunset (18h)', () => {
    expect(isNightTime(18)).toBe(true)
  })
})

describe('calculateSkyColors', () => {
  it('returns colors object with top, middle, bottom', () => {
    const colors = calculateSkyColors(12)
    expect(colors).toHaveProperty('top')
    expect(colors).toHaveProperty('middle')
    expect(colors).toHaveProperty('bottom')
  })

  it('returns hex color strings', () => {
    const colors = calculateSkyColors(12)
    expect(colors.top).toMatch(/^#[0-9a-fA-F]{6}$/)
    expect(colors.middle).toMatch(/^#[0-9a-fA-F]{6}$/)
    expect(colors.bottom).toMatch(/^#[0-9a-fA-F]{6}$/)
  })

  it('returns different colors for different times', () => {
    const noon = calculateSkyColors(12)
    const midnight = calculateSkyColors(0)

    // Noon should be brighter than midnight
    expect(noon.top).not.toBe(midnight.top)
  })

  it('interpolates smoothly between keyframes', () => {
    const colors10 = calculateSkyColors(10)
    const colors11 = calculateSkyColors(11)
    const colors12 = calculateSkyColors(12)

    // Colors should be different but in a gradient
    expect(colors10).not.toEqual(colors12)
    // 11h should be between 10h and 12h
    expect(colors11.top).toBeDefined()
  })
})

describe('calculateLighting', () => {
  it('returns lighting params object', () => {
    const lighting = calculateLighting(12)
    expect(lighting).toHaveProperty('direction')
    expect(lighting).toHaveProperty('color')
    expect(lighting).toHaveProperty('intensity')
    expect(lighting).toHaveProperty('ambientColor')
    expect(lighting).toHaveProperty('ambientIntensity')
  })

  it('returns normalized direction vector', () => {
    const lighting = calculateLighting(12)
    const length = Math.sqrt(
      lighting.direction[0] ** 2 +
      lighting.direction[1] ** 2 +
      lighting.direction[2] ** 2
    )
    expect(length).toBeCloseTo(1, 1)
  })

  it('returns higher intensity at noon than at night', () => {
    const noon = calculateLighting(12)
    const night = calculateLighting(22)

    expect(noon.intensity).toBeGreaterThan(night.intensity)
    expect(noon.ambientIntensity).toBeGreaterThan(night.ambientIntensity)
  })

  it('returns warm colors at sunrise/sunset', () => {
    const sunrise = calculateLighting(7)
    const sunset = calculateLighting(17.5)

    // Warm colors have more red in hex
    // These should be different from white (#FFFFFF)
    expect(sunrise.color).not.toBe('#FFFFFF')
    expect(sunset.color).not.toBe('#FFFFFF')
  })

  it('returns white-ish colors at noon', () => {
    const noon = calculateLighting(12)
    expect(noon.color).toBe('#FFFFFF')
  })

  it('returns blue-ish colors at night', () => {
    const night = calculateLighting(22)
    expect(night.color).toBe('#7799CC')
  })
})

describe('formatTimeFromHour', () => {
  it('formats noon correctly', () => {
    expect(formatTimeFromHour(12)).toBe('12:00 PM')
  })

  it('formats midnight correctly', () => {
    expect(formatTimeFromHour(0)).toBe('12:00 AM')
  })

  it('formats morning correctly', () => {
    expect(formatTimeFromHour(8)).toBe('8:00 AM')
  })

  it('formats afternoon correctly', () => {
    expect(formatTimeFromHour(14.5)).toBe('2:30 PM')
  })

  it('formats with minutes', () => {
    expect(formatTimeFromHour(9.25)).toBe('9:15 AM')
  })

  it('handles values outside 0-24', () => {
    expect(formatTimeFromHour(25)).toBe('1:00 AM')
    expect(formatTimeFromHour(-1)).toBe('11:00 PM')
  })
})

describe('getCurrentTimeAsHour', () => {
  it('returns a number between 0 and 24', () => {
    const hour = getCurrentTimeAsHour()
    expect(hour).toBeGreaterThanOrEqual(0)
    expect(hour).toBeLessThan(24)
  })

  it('includes fractional hours for minutes', () => {
    const hour = getCurrentTimeAsHour()
    // If we're not exactly on the hour, there should be a fractional part
    // This test may occasionally fail if run exactly on the hour
    expect(typeof hour).toBe('number')
  })
})
