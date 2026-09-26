import { describe, expect, it } from 'vitest'
import { PresentationPool, type PoolEvent } from './pool'

const event = (id: string, start = 0, end = 10): PoolEvent => ({ id, start, end })
describe('bounded ambient presentation slots', () => {
  it('admits deterministic priority, retains slots, and reuses expired capacity', () => {
    const pool = new PresentationPool(2)
    pool.sync([event('c'), event('b'), event('a')], 1)
    expect(pool.slots.map(slot => slot.event?.id)).toEqual(['a', 'b'])
    expect(pool.stats()).toEqual({ capacity: 2, active: 2, acquired: 2, released: 0, rejected: 1 })
    pool.sync([event('b'), event('a'), event('c')], 2)
    expect(pool.stats().acquired).toBe(2)
    pool.sync([event('b'), event('d', 3, 20)], 4)
    expect(pool.slots.map(slot => slot.event?.id)).toEqual(['d', 'b'])
    expect(pool.stats().released).toBe(1)
  })
  it('releases on disable, expires at the boundary, and skips missed events after resume', () => {
    const pool = new PresentationPool(2)
    const events = [event('past'), event('future', 100, 110)]
    pool.sync(events, 1)
    pool.sync(events, 10)
    expect(pool.stats().active).toBe(0)
    pool.sync(events, 105)
    expect(pool.slots.find(slot => slot.event)?.event?.id).toBe('future')
    pool.sync(events, 105, 0)
    expect(pool.stats().active).toBe(0)
    pool.sync(events, 1000)
    expect(pool.stats().active).toBe(0)
    pool.clear(); pool.clear()
    expect(pool.stats().released).toBe(pool.stats().acquired)
  })
  it('bounds input and rejects malformed updates atomically', () => {
    const pool = new PresentationPool(1)
    pool.sync([event('a')], 1)
    for (const invalid of [[event('b'), event('b')], [event('b', 2, 1)], Array.from({ length: 1025 }, (_, i) => event(String(i)))]) {
      expect(() => pool.sync(invalid, 1)).toThrow('Invalid')
      expect(pool.slots[0]?.event?.id).toBe('a')
    }
    expect(() => pool.sync([], NaN)).toThrow('Invalid')
    expect(() => new PresentationPool(257)).toThrow('capacity')
  })
})
