import { readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { captureAxes, captureMatrix, expectedGoldenKeys, goldenAliases, goldenCoverage, goldenKey } from './goldenKey'

describe('capture matrix and golden filenames', () => {
  it('requires complete coverage in the actual published golden directory', () => {
    const names = readdirSync(resolve(import.meta.dirname, '../../__goldens__'))
    expect(goldenCoverage(expectedGoldenKeys(tuning), names)).toEqual({ missing: [], orphaned: [] })
  })
  it('covers every legacy alias exactly once from the canonical matrix', () => {
    const aliases = captureMatrix(tuning).flatMap(goldenAliases)
    expect([...aliases].sort()).toEqual(expectedGoldenKeys(tuning))
    expect(new Set(aliases).size).toBe(aliases.length)
    expect(goldenAliases({ scene: 'park', profile: 'high', period: 'night', weather: 'rain' })).toEqual(['park-high-night-weather-rain'])
    expect(goldenAliases({ scene: 'park', profile: 'high' })).toContain('park-high')
  })
  it('derives every axis directly from configuration keys', () => {
    const axes = captureAxes(tuning)
    expect(axes).toEqual({ scenes: Object.keys(tuning.budgets.scenes), profiles: Object.keys(tuning.quality.profiles), periods: Object.keys(tuning.lighting.periods), weather: Object.keys(tuning.weather.states) })
    expect(captureMatrix(tuning)).toHaveLength(axes.scenes.length * axes.profiles.length * axes.periods.length * axes.weather.length)
  })

  it('automatically includes a scratch fifth profile and newly added periods and weather', () => {
    const scratch = structuredClone(tuning)
    const config = { ...scratch, quality: { ...scratch.quality, profiles: { ...scratch.quality.profiles, fifth: scratch.quality.profiles.low } }, lighting: { ...scratch.lighting, periods: { ...scratch.lighting.periods, eclipse: scratch.lighting.periods.night } }, weather: { ...scratch.weather, states: { ...scratch.weather.states, mist: scratch.weather.states.cloudy } } }
    expect(captureMatrix(config)).toContainEqual({ scene: 'park', profile: 'fifth', period: 'eclipse', weather: 'mist' })
    expect(captureAxes(tuning).profiles).not.toContain('fifth')
  })

  it('reproduces every existing golden name, including legacy day and weather names', () => {
    const axes = captureAxes(tuning)
    const keys = new Set(axes.scenes.flatMap((scene) => axes.profiles.flatMap((profile) =>
      [undefined, ...axes.periods].flatMap((period) => [undefined, ...axes.weather].map((weather) => `${goldenKey({ scene, profile, period, weather })}.png`)),
    )))
    const names = readdirSync(resolve(import.meta.dirname, '../../__goldens__')).filter((name) => name.endsWith('.png'))
    expect(names.length).toBeGreaterThan(0)
    expect(names.filter((name) => !keys.has(name))).toEqual([])
    expect(goldenKey({ scene: 'park', profile: 'high' })).toBe('park-high')
    expect(goldenKey({ scene: 'park', profile: 'high', period: 'night', weather: 'snow' })).toBe('park-high-night-weather-snow')
  })

  it('detects missing, orphaned and newly required fifth-profile goldens', () => {
    const expected = expectedGoldenKeys(tuning)
    const files = expected.map((key) => `${key}.png`)
    expect(goldenCoverage(expected, files)).toEqual({ missing: [], orphaned: [] })
    expect(goldenCoverage(expected, [...files.slice(1), 'retired.png'])).toEqual({ missing: [files[0]], orphaned: ['retired.png'] })
    const scratch = { ...tuning, quality: { ...tuning.quality, profiles: { ...tuning.quality.profiles, fifth: tuning.quality.profiles.low } } }
    const missing = goldenCoverage(expectedGoldenKeys(scratch), files).missing
    expect(missing.length).toBeGreaterThan(0)
    expect(missing.every((name) => name.includes('-fifth'))).toBe(true)
  })
})
