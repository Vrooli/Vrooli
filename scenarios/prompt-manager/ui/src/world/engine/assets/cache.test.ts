import { describe, expect, it, vi } from 'vitest'
import { BufferGeometry, Float32BufferAttribute, MeshStandardMaterial } from 'three'
import type { PropPart } from './geometry'
import { geometryBytes, PreparedAssetCache } from './cache'

function asset(material = new MeshStandardMaterial()): [PropPart] {
  const geometry = new BufferGeometry()
  geometry.setAttribute('position', new Float32BufferAttribute([0, 0, 0, 1, 0, 0, 0, 1, 0], 3))
  return [{ geometry, material }]
}

describe('prepared asset ownership', () => {
  it('shares preparation by content across independent leases and warm returns', () => {
    const cache = new PreparedAssetCache(1000)
    const prepare = vi.fn(() => asset())
    const first = cache.acquire('same-content', prepare)
    const second = cache.acquire('same-content', prepare)
    const retained = first.retain()
    expect(second.parts).toBe(first.parts)
    first.release()
    first.release()
    expect(cache.stats().references).toBe(2)
    second.release()
    retained.release()
    const returned = cache.acquire('same-content', prepare)
    expect(prepare).toHaveBeenCalledTimes(1)
    expect(returned.parts).toBe(second.parts)
    returned.release()
    cache.trim(0)
    expect(cache.stats()).toMatchObject({ residentAssets: 0, references: 0, geometryDisposals: 1 })
    expect(() => first.retain()).toThrow(/released/)
  })

  it('protects active geometry under memory pressure and never disposes borrowed materials', () => {
    const cache = new PreparedAssetCache(100)
    const material = new MeshStandardMaterial()
    const materialDispose = vi.spyOn(material, 'dispose')
    const one = asset(material)
    const two = asset(material)
    const disposeOne = vi.spyOn(one[0].geometry, 'dispose')
    const disposeTwo = vi.spyOn(two[0].geometry, 'dispose')
    const a = cache.acquire('a', () => one)
    const b = cache.acquire('b', () => two)
    cache.setBudget(0)
    expect(cache.stats().overBudgetBytes).toBe(72)
    expect(disposeOne).not.toHaveBeenCalled()
    a.release()
    expect(disposeOne).toHaveBeenCalledTimes(1)
    expect(disposeTwo).not.toHaveBeenCalled()
    b.release()
    cache.trim(0)
    expect(disposeTwo).toHaveBeenCalledTimes(1)
    expect(materialDispose).not.toHaveBeenCalled()
    expect(cache.stats().cpuBytes).toBe(0)
  })

  it('evicts least recently used idle entries and retries failed preparation', () => {
    const cache = new PreparedAssetCache(72)
    expect(() => cache.acquire('retry', () => { throw new Error('decode failed') })).toThrow('decode failed')
    expect(cache.stats().residentAssets).toBe(0)
    cache.acquire('a', () => asset()).release()
    cache.acquire('b', () => asset()).release()
    cache.acquire('a', () => { throw new Error('warm asset was rebuilt') }).release()
    cache.acquire('retry', () => asset()).release()
    const rebuildB = vi.fn(() => asset())
    cache.acquire('b', rebuildB).release()
    expect(rebuildB).toHaveBeenCalledOnce()
    expect(cache.stats().residentAssets).toBe(2)
    cache.trim(0)
  })

  it('stabilizes across thirty alternating scene consumers', () => {
    const cache = new PreparedAssetCache(72)
    const prepare = vi.fn(() => asset())
    for (let switchIndex = 0; switchIndex < 30; switchIndex++) {
      const handle = cache.acquire(switchIndex % 2 === 0 ? 'park' : 'office', prepare)
      const mountedAgain = handle.retain()
      handle.release()
      mountedAgain.release()
      expect(cache.stats().references).toBe(0)
      expect(cache.stats().cpuBytes).toBeLessThanOrEqual(72)
    }
    expect(prepare).toHaveBeenCalledTimes(2)
    expect(cache.stats()).toMatchObject({ residentAssets: 2, geometryDisposals: 0 })
    cache.trim(0)
    expect(cache.stats().geometryDisposals).toBe(2)
  })

  it('counts backing storage once for shared attributes', () => {
    const geometry = new BufferGeometry()
    const attribute = new Float32BufferAttribute([0, 0, 1], 3)
    geometry.setAttribute('position', attribute)
    geometry.setAttribute('normal', attribute)
    expect(geometryBytes([geometry, geometry])).toBe(12)
  })
})
