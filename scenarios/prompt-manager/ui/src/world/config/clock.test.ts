import { describe, expect, it } from 'vitest'
import { isTimeZone, WorldClock } from './clock'

describe('shared presentation clock', () => {
  it('records an explicit civil zone and rejects unknown zones', () => {
    const now = () => Date.parse('2028-02-29T12:00:00Z')
    const utc = new WorldClock(now, 'UTC').snapshot()
    const newYork = new WorldClock(now, 'America/New_York').snapshot()
    expect(utc.utcMilliseconds).toBe(newYork.utcMilliseconds)
    expect(newYork.localMinutes).toBe(420)
    expect(newYork.timeZone).toBe('America/New_York')
    expect(isTimeZone('Unknown/Place')).toBe(false)
    expect(() => new WorldClock(now, 'Unknown/Place')).toThrow()
  })
  it('freezes, seeks, progresses at a controlled rate and resumes current civil time', () => {
    let now = Date.parse('2028-02-29T12:00:00Z')
    const clock = new WorldClock(() => now, 'UTC')
    expect(clock.snapshot().localMinutes).toBe(720)
    clock.fix(now)
    now += 60000
    expect(clock.snapshot().utcMilliseconds).toBe(Date.parse('2028-02-29T12:00:00Z'))
    clock.fix(now, 10)
    now += 1000
    expect(clock.snapshot().utcMilliseconds).toBe(now + 9000)
    expect(clock.snapshot().timeScale).toBe(10)
    clock.live()
    expect(clock.snapshot().utcMilliseconds).toBe(now)
    expect(clock.snapshot().mode).toBe('clock')
  })
  it('derives civil minutes across DST while UTC remains continuous', () => {
    let now = Date.parse('2026-03-08T06:59:00Z')
    const clock = new WorldClock(() => now, 'America/New_York')
    expect(clock.snapshot().localMinutes).toBe(119)
    now += 60000
    expect(clock.snapshot().localMinutes).toBe(180)
    expect(clock.snapshot().utcMilliseconds).toBe(Date.parse('2026-03-08T07:00:00Z'))
  })
  it('notifies changes and rejects invalid instants without mutating state', () => {
    const clock = new WorldClock(() => 1000, 'UTC')
    let changes = 0
    const off = clock.subscribe(() => changes++)
    clock.fix(2000)
    for (const value of [NaN, Infinity, 1e16]) expect(() => clock.fix(value)).toThrow('Invalid')
    expect(() => clock.fix(0, -1)).toThrow('Invalid')
    expect(clock.snapshot().utcMilliseconds).toBe(2000)
    expect(changes).toBe(1)
    off(); clock.live()
    expect(changes).toBe(1)
  })
})
