import { describe, expect, it, vi } from 'vitest'
import { createMilkyWay, createStarField } from './starField'
import { starVisibility } from '../config/celestial'

describe('bounded static sky stars', () => {
  it('generates the same normalized directions independently of profile and other random streams', () => {
    const a = createStarField(42), b = createStarField(42), other = createStarField(43)
    const positions = a.geometry.getAttribute('position')
    expect(positions.count).toBe(2048)
    expect(positions.array).toEqual(b.geometry.getAttribute('position').array)
    expect(positions.array).not.toEqual(other.geometry.getAttribute('position').array)
    for (let i = 0; i < positions.count; i++) expect(Math.hypot(positions.getX(i), positions.getY(i), positions.getZ(i))).toBeCloseTo(100, 4)
    const buffer = positions.array
    a.geometry.setDrawRange(0, 384)
    expect(a.geometry.getAttribute('position').array).toBe(buffer)
    expect(buffer.byteLength + a.geometry.getAttribute('brightness').array.byteLength).toBe(32768)
    a.dispose(); b.dispose(); other.dispose()
  })
  it('fades for daylight, clouds and an above-horizon bright moon', () => {
    expect(starVisibility(.5, 0, 0, 0)).toBe(0)
    expect(starVisibility(-.3, 1, 0, 0)).toBe(0)
    expect(starVisibility(-.3, 0, 0, 0)).toBe(1)
    expect(starVisibility(-.3, 0, 1, .5)).toBeCloseTo(.35)
    expect(starVisibility(-.3, 0, 1, -.5)).toBe(1)
    expect(starVisibility(-.1, 0, 0, 0)).toBeGreaterThan(0)
    expect(starVisibility(-.1, 0, 0, 0)).toBeLessThan(1)
  })
  it('releases its geometry and material together', () => {
    const field = createStarField(1)
    const geometry = vi.spyOn(field.geometry, 'dispose'), material = vi.spyOn(field.material, 'dispose')
    field.dispose()
    expect(geometry).toHaveBeenCalledTimes(1)
    expect(material).toHaveBeenCalledTimes(1)
  })
})

it('keeps the galactic band in one bounded draw with no picking or retained GPU resources', () => {
  const galaxy = createMilkyWay(42)
  expect(galaxy.geometry.getAttribute('position').count).toBeLessThan(1300)
  expect(galaxy.mesh.name).toBe('celestial-milky-way')
  expect(galaxy.material.depthWrite).toBe(false)
  expect(galaxy.uniforms.visibility.value).toBe(0)
  const geometry = vi.spyOn(galaxy.geometry, 'dispose'), material = vi.spyOn(galaxy.material, 'dispose')
  galaxy.dispose()
  expect(geometry).toHaveBeenCalledOnce()
  expect(material).toHaveBeenCalledOnce()
})
