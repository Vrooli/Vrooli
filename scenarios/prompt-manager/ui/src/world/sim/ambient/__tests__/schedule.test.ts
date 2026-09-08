import { describe, expect, it } from 'vitest'
import { meteorCandidate, nextSkyBoundary, skyEventsAt } from '../schedule'

const eligible = { night: true, clearSky: true, ambientEnabled: true, reducedMotion: false }

describe('absolute-time ambient sky schedule', () => {
  it('wakes at event starts and ends without periodic idle checks', () => {
    const event = meteorCandidate(42, 1000)
    expect(nextSkyBoundary(42, event.start - 0.1, eligible)).toBe(event.start)
    expect(nextSkyBoundary(42, event.start + 0.1, eligible)).toBe(event.end)
    expect(nextSkyBoundary(42, event.end, eligible)).toBeGreaterThan(event.end)
    expect(nextSkyBoundary(42, event.start, { ...eligible, night: false })).toBeNull()
    expect(nextSkyBoundary(42, event.start, { ...eligible, ambientEnabled: false }, event)).toBeNull()
    expect(nextSkyBoundary(42, event.start, { ...eligible, night: false }, event)).toBe(event.end)
    expect(nextSkyBoundary(42, event.end, { ...eligible, night: false }, event)).toBeNull()
    // Reduced motion needs only a comet window/opportunity, not meteor cadence.
    const reduced = nextSkyBoundary(42, 0, { ...eligible, reducedMotion: true })
    expect(reduced).toBeGreaterThan(180)
  })
  it('has the same identities at 15, 30, 60 and 144 Hz and after direct seek', () => {
    const first = meteorCandidate(42, 1000)
    const identities = (fps: number) => {
      const ids = new Set<string>()
      for (let frame = 0; frame < 5 * fps; frame++) {
        for (const event of skyEventsAt(42, first.start - 1 + frame / fps, eligible)) ids.add(event.id)
      }
      return [...ids].sort()
    }
    for (const fps of [30, 60, 144]) expect(identities(fps)).toEqual(identities(15))
    expect(skyEventsAt(42, first.start + 0.1, eligible)).toContainEqual(first)
    expect(skyEventsAt(42, first.end, eligible).some(event => event.id === first.id)).toBe(false)
    const resume = first.end + 1000000
    expect(skyEventsAt(42, resume, eligible).every(event => event.start <= resume && event.end > resume)).toBe(true)
  })

  it('matches predeclared broad variant tolerances and mean rate on 100,000 fixed opportunities', () => {
    const counts: Record<string, number> = {}
    let previous = meteorCandidate(918273, 0)
    const firstStart = previous.start
    const acceptedFireballs: number[] = []
    for (let bucket = 0; bucket < 100000; bucket++) {
      const event = meteorCandidate(918273, bucket)
      counts[event.variant] = (counts[event.variant] ?? 0) + 1
      expect(event.end - event.start).toBeGreaterThanOrEqual(event.variant === 'great-fireball' ? 2.5 : 0.599999)
      expect(event.end - event.start).toBeLessThanOrEqual(event.variant === 'great-fireball' ? 4 : 1.800001)
      if (event.variant === 'great-fireball' && skyEventsAt(918273, event.start + 0.1, eligible).some(active => active.id === event.id)) acceptedFireballs.push(event.start)
      previous = event
    }
    const ranges = { white: [84000, 86000], blue: [9400, 10600], green: [3150, 3850], violet: [1150, 1650], 'great-fireball': [60, 140] } as const
    for (const [variant, [min, max]] of Object.entries(ranges)) {
      expect(counts[variant]).toBeGreaterThanOrEqual(min)
      expect(counts[variant]).toBeLessThanOrEqual(max)
    }
    expect((previous.start - firstStart) / 99999).toBeCloseTo(180, 1)
    expect(acceptedFireballs.length).toBeGreaterThan(20)
    let previousFireball = -Infinity
    for (const start of acceptedFireballs) {
      expect(start - previousFireball).toBeGreaterThanOrEqual(86400)
      previousFireball = start
    }
  })

  it('retains multi-night comet identity and permits static comets in reduced motion', () => {
    const comets = new Map<string, { start: number; end: number }>()
    // Independent fixed daily samples across 1,000 opportunities. Broad 8% tolerance.
    for (let day = 0; day < 30000; day++) {
      const time = day * 86400 + 43200
      const events = skyEventsAt(8765, time, { ...eligible, reducedMotion: true })
      expect(events.length).toBeLessThanOrEqual(1)
      for (const event of events) {
        expect(event.family).toBe('comet')
        expect((event.end - event.start) / 86400).toBeCloseTo(Math.round((event.end - event.start) / 86400), 6)
        expect(event.end - event.start).toBeGreaterThanOrEqual(3 * 86400 - 0.001)
        expect(event.end - event.start).toBeLessThanOrEqual(5 * 86400 + 0.001)
        expect(skyEventsAt(8765, event.start + 1, eligible)).toContainEqual(event)
        comets.set(event.id, event)
      }
    }
    expect(comets.size).toBeGreaterThanOrEqual(50)
    expect(comets.size).toBeLessThanOrEqual(110)
  })

  it('gates visibility without consuming identities, and rejects invalid clock inputs', () => {
    const candidate = meteorCandidate(7, -10)
    const time = candidate.start + 0.1
    for (const disabled of [{ ambientEnabled: false }, { night: false }, { clearSky: false }]) expect(skyEventsAt(7, time, { ...eligible, ...disabled })).toEqual([])
    expect(skyEventsAt(7, time, { ...eligible, reducedMotion: true }).every(event => event.family === 'comet')).toBe(true)
    expect(skyEventsAt(7, time, eligible)).toContainEqual(candidate)
    for (const time of [NaN, Infinity, 1e13]) expect(() => skyEventsAt(7, time, eligible)).toThrow('Invalid')
    expect(() => meteorCandidate(7, 0.5)).toThrow('bucket')
  })
})
