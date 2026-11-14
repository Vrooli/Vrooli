import { describe, expect, it } from 'vitest'
import { AmbientMeasurements } from './ambient'

describe('bounded ambient CPU accounting', () => {
  it('bounds history and reports recent timing instead of lifetime outliers', () => {
    const costs = new AmbientMeasurements()
    costs.record('birds', 100, 4, 4)
    for (let i = 0; i < 300; i++) costs.record('birds', 2, 4, 4)
    const snapshot = costs.snapshot(), birds = snapshot.families.find(row => row.family === 'birds')
    expect(snapshot.bufferBytes).toBe(14336)
    expect(birds).toMatchObject({ samples: 301, windowSamples: 256, meanMs: 2, p95Ms: 2, maxMs: 2, active: 4, capacity: 4 })
  })
  it('keeps independent families and removes unmounted work from current totals', () => {
    const costs = new AmbientMeasurements()
    costs.record('fireflies', .1, 12, 48, 12)
    costs.record('fish', .2, 1, 2, 1)
    expect(costs.snapshot().latestTotalMs).toBeCloseTo(.3)
    costs.clear('fish')
    expect(costs.snapshot().latestTotalMs).toBe(.1)
    expect(costs.snapshot().families.find(row => row.family === 'fish')).toMatchObject({ mounted: false, active: 0, capacity: 0, latestMs: 0 })
    costs.reset()
    expect(costs.snapshot().families.every(row => row.samples === 0 && !row.mounted)).toBe(true)
  })
  it('rejects invalid limits and timings without corrupting an existing sample', () => {
    const costs = new AmbientMeasurements()
    costs.record('rabbits', .1, 1, 2, 1)
    const before = costs.snapshot()
    expect(() => costs.record('rabbits', .1, 2, 2, 1)).toThrow('Invalid')
    expect(() => costs.record('rabbits', NaN, 0, 2)).toThrow('Invalid')
    expect(() => costs.record('rabbits', -1, 0, 2)).toThrow('Invalid')
    expect(() => costs.record('rabbits', 0, 0, 257)).toThrow('Invalid')
    expect(costs.snapshot()).toEqual(before)
  })
})
