import { describe, expect, it, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { MeshBasicMaterial, MeshStandardMaterial } from 'three'
import { PERIOD_IDS, scenes, resolvePeriod } from '../config'
import { slotEmissive, usePropMaterials, type Emissive } from './propMaterials'

describe('role-driven prop emission', () => {
  it('keeps both scenes dark at day and lights declared lamps at night', () => {
    for (const scene of Object.values(scenes)) {
      const day = resolvePeriod(scene, 'day')
      for (const slot of ['lamp', 'hearth'] as const) expect(slotEmissive(scene, day, slot)).toBeUndefined()
      expect(slotEmissive(scene, resolvePeriod(scene, 'night'), 'lamp')?.intensity).toBeGreaterThan(0)
    }
  })
  it('never lights an office coffee table in any period', () => {
    for (const period of PERIOD_IDS) expect(slotEmissive(scenes.office, resolvePeriod(scenes.office, period), 'hearth')).toBeUndefined()
    expect(slotEmissive(scenes.park, resolvePeriod(scenes.park, 'night'), 'hearth')?.color).toBe(scenes.park.emissive?.hearth)
  })
  it('clones once for identical scalar values across parent rerenders and disposes only owned copies', () => {
    const source = new MeshStandardMaterial()
    const basic = new MeshBasicMaterial()
    const clone = vi.spyOn(source, 'clone')
    const sourceDispose = vi.spyOn(source, 'dispose')
    const basicDispose = vi.spyOn(basic, 'dispose')
    const parts = [{ material: source }, { material: basic }]
    const emissive = slotEmissive(scenes.park, resolvePeriod(scenes.park, 'night'), 'lamp')
    const { result, rerender, unmount } = renderHook(({ glow }: { glow: Emissive | undefined }) => usePropMaterials(parts, glow), { initialProps: { glow: emissive } })
    const materials = result.current
    const owned = materials[0]
    if (!owned) throw new Error('missing material')
    const ownedDispose = vi.spyOn(owned, 'dispose')
    for (let frame = 0; frame < 100; frame += 1) rerender({ glow: emissive ? { ...emissive } : undefined })
    expect(clone).toHaveBeenCalledTimes(1)
    expect(result.current).toBe(materials)
    expect(ownedDispose).not.toHaveBeenCalled()
    rerender({ glow: undefined })
    expect(result.current[0]).toBe(source)
    expect(ownedDispose).toHaveBeenCalledTimes(1)
    unmount()
    expect(sourceDispose).not.toHaveBeenCalled()
    expect(basicDispose).not.toHaveBeenCalled()
  })
  it('does not clone day materials and updates emission when intensity changes', () => {
    const source = new MeshStandardMaterial()
    const clone = vi.spyOn(source, 'clone')
    const parts = [{ material: source }]
    const { result, rerender, unmount } = renderHook(({ glow }: { glow: Emissive | undefined }) => usePropMaterials(parts, glow), { initialProps: { glow: undefined as Emissive | undefined } })
    expect(result.current[0]).toBe(source)
    expect(clone).not.toHaveBeenCalled()
    rerender({ glow: { color: '#ffe4bc', intensity: 2 } })
    expect((result.current[0] as MeshStandardMaterial).emissiveIntensity).toBe(2)
    rerender({ glow: { color: '#ffe4bc', intensity: 3 } })
    expect((result.current[0] as MeshStandardMaterial).emissiveIntensity).toBe(3)
    expect(clone).toHaveBeenCalledTimes(2)
    unmount()
  })
})
