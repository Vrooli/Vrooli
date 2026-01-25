/**
 * MaterialProvider - Context provider for shared material instances.
 * Enables material reuse across components to reduce memory usage.
 */

import React, { createContext, useContext, useMemo } from 'react'
import * as THREE from 'three'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { MATERIAL_PRESETS, type StandardPresetName } from './presets'

/** Cached material instances */
interface MaterialCache {
  /** Get or create a material for the given preset and color */
  getMaterial: (preset: StandardPresetName, color: string) => THREE.Material
  /** Clear all cached materials */
  clear: () => void
  /** Get cache statistics */
  stats: () => { count: number; presets: string[] }
}

const MaterialContext = createContext<MaterialCache | null>(null)

interface MaterialProviderProps {
  children: React.ReactNode
}

/**
 * Provider component for shared material instances.
 * Caches materials by preset+color combination.
 */
export function MaterialProvider({ children }: MaterialProviderProps) {
  const quality = useGraphicsStore((state) => state.config.materialQuality)

  const cache = useMemo(() => {
    const materials = new Map<string, THREE.Material>()

    const getMaterial = (preset: StandardPresetName, color: string): THREE.Material => {
      const key = `${preset}:${color}:${quality}`

      if (materials.has(key)) {
        return materials.get(key)!
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

/**
 * Hook to access the material cache.
 */
export function useMaterialCache(): MaterialCache {
  const context = useContext(MaterialContext)

  if (!context) {
    throw new Error('useMaterialCache must be used within a MaterialProvider')
  }

  return context
}

/**
 * Hook to get a cached material.
 * Falls back to creating a new material if cache isn't available.
 */
export function useCachedMaterial(preset: StandardPresetName, color: string): THREE.Material {
  const context = useContext(MaterialContext)

  return useMemo(() => {
    if (context) {
      return context.getMaterial(preset, color)
    }

    // Fallback: create material without caching
    const presetData = MATERIAL_PRESETS[preset]
    return new THREE.MeshStandardMaterial({
      color,
      metalness: presetData.metalness,
      roughness: presetData.roughness,
      envMapIntensity: 'envMapIntensity' in presetData ? presetData.envMapIntensity : 1.0,
    })
  }, [context, preset, color])
}
