import { useEffect, useMemo } from 'react'
import { MeshStandardMaterial, type Material } from 'three'
import type { LightingPeriod, Scene } from '../config'

export interface Emissive {
  color: string
  intensity: number
}

export function slotEmissive(scene: Scene, period: LightingPeriod, slot: keyof NonNullable<Scene['emissive']>): Emissive | undefined {
  const color = scene.emissive?.[slot]
  return color && period.lampEmissive > 0 ? { color, intensity: period.lampEmissive } : undefined
}

const BLOOM_THRESHOLD = 1

/** Scalar memo keys avoid cloning on parent renders with fresh descriptor objects. */
export function usePropMaterials(parts: readonly { material: Material }[], emissive?: Emissive): Material[] {
  const color = emissive?.color
  const intensity = emissive?.intensity ?? 0
  const materials = useMemo(() => parts.map(({ material }) => {
    if (!color || intensity <= 0 || !(material instanceof MeshStandardMaterial)) return material
    const copy = material.clone()
    copy.emissive.set(color)
    copy.emissiveIntensity = intensity
    copy.toneMapped = intensity < BLOOM_THRESHOLD
    return copy
  }), [parts, color, intensity])
  useEffect(() => () => {
    materials.forEach((material, index) => {
      if (material !== parts[index]?.material) material.dispose()
    })
  }, [materials, parts])
  return materials
}
