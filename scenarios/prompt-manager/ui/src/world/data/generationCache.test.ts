import { beforeAll, describe, expect, it } from 'vitest'
import { tuning } from '../config'
import { generateWorld } from '../sim/world'
import { makeWorldInput } from '../sim/__tests__/fixtures'
import type { GeneratedWorld } from '../sim/model'
import { GeneratedWorldCache, generatedWorldBytes } from './generationCache'

let world: GeneratedWorld
beforeAll(() => { world = generateWorld(makeWorldInput({ teams: 1, agents: 2 }), tuning) })

describe('generated world cache', () => {
  it('evicts least recently used entries without detaching visible buffers', () => {
    const cache = new GeneratedWorldCache(64 * 1024 * 1024, 2)
    const visible = world.terrain.height
    cache.put('a', world)
    cache.put('b', world)
    expect(cache.get('a')).toBe(world)
    cache.put('c', world)
    expect(cache.get('b')).toBeUndefined()
    expect(cache.get('a')).toBe(world)
    expect(visible.byteLength).toBeGreaterThan(0)
    expect(cache.stats()).toMatchObject({ entries: 2, evictions: 1, hits: 2, misses: 1 })
  })
  it('enforces the estimated byte budget and rejects oversized entries', () => {
    const bytes = generatedWorldBytes(world) + 2
    const cache = new GeneratedWorldCache(bytes, 4)
    expect(cache.put('a', world)).toBe(true)
    expect(cache.put('b', world)).toBe(true)
    expect(cache.get('a')).toBeUndefined()
    expect(cache.stats().estimatedBytes).toBe(bytes)
    const tooSmall = new GeneratedWorldCache(bytes - 1)
    expect(tooSmall.put('a', world)).toBe(false)
    expect(tooSmall.stats()).toMatchObject({ entries: 0, estimatedBytes: 0, rejected: 1 })
  })
  it('counts shared backing buffers once within each world', () => {
    const alias = { ...world, pathMask: world.terrain.height }
    expect(generatedWorldBytes(alias)).toBeLessThan(generatedWorldBytes(world))
  })
  it('rejects invalid budgets', () => {
    expect(() => new GeneratedWorldCache(-1)).toThrow('budget')
    expect(() => new GeneratedWorldCache(10, 1.5)).toThrow('budget')
  })
})
