/**
 * MaterialProvider - Context provider for shared material instances.
 * Enables material reuse across components to reduce memory usage.
 */

import React, { useMemo } from 'react'
import * as THREE from 'three'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { MATERIAL_PRESETS, type StandardPresetName } from './presets'
import { MaterialContext, type MaterialCache } from './MaterialContext'

interface MaterialProviderProps {
  children: React.ReactNode
}

/**
 * Provider component for shared material instances.
 * Caches materials by preset+color combination.
 */
export function MaterialProvider({ children }: MaterialProviderProps) {
  const quality = useGraphicsStore((state) => state.config.materialQuality)

  const cache: MaterialCache = useMemo(() => {
    const materials = new Map<string, THREE.Material>()

    const getMaterial = (preset: StandardPresetName, color: string): THREE.Material => {
      const key = `${preset}:${color}:${quality}`

      const cached = materials.get(key)
      if (cached) {
        return cached
      }

      const presetData = MATERIAL_PRESETS[preset]
      let material: THREE.Material

      if (quality === 'basic') {
        material = new THREE.MeshBasicMaterial({ color })
      } else {
        material = new THREE.MeshStandardMaterial({
          color,
          metalness: presetData.metalness,
          roughness: presetData.roughness,
          envMapIntensity: 'envMapIntensity' in presetData ? presetData.envMapIntensity : 1.0,
        })
      }

      materials.set(key, material)
      return material
    }

    const clear = () => {
      materials.forEach((material) => material.dispose())
      materials.clear()
    }

    const stats = () => ({
      count: materials.size,
      presets: Array.from(materials.keys()),
    })

    return { getMaterial, clear, stats }
  }, [quality])

  return (
    <MaterialContext.Provider value={cache}>
      {children}
    </MaterialContext.Provider>
  )
}
