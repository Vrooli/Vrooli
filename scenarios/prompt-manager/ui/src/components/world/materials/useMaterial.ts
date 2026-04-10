/**
 * useMaterial hook for accessing material presets and creating materials.
 */

import { useMemo, useContext } from 'react'
import * as THREE from 'three'
import { useGraphicsStore } from '@/stores/graphicsStore'
import {
  MATERIAL_PRESETS,
  PHYSICAL_PRESETS,
  type StandardPresetName,
  type PhysicalPresetName,
  type MaterialPreset,
} from './presets'
import { MaterialContext, type MaterialCache } from './MaterialContext'

interface UseMaterialOptions {
  /** Override color */
  color?: string
  /** Override emissive color */
  emissive?: string
  /** Additional material properties */
  overrides?: Partial<THREE.MeshStandardMaterialParameters>
}

/**
 * Hook to get a material based on preset and current quality settings.
 * Returns a THREE.js material instance.
 */
export function useMaterial(
  presetName: StandardPresetName | PhysicalPresetName,
  options: UseMaterialOptions = {}
) {
  const quality = useGraphicsStore((state) => state.config.materialQuality)

  return useMemo(() => {
    const { color, emissive, overrides } = options

    // Get the appropriate preset
    let preset: MaterialPreset

    if (presetName in PHYSICAL_PRESETS && quality === 'physical') {
      preset = PHYSICAL_PRESETS[presetName as PhysicalPresetName]
    } else if (presetName in MATERIAL_PRESETS) {
      preset = MATERIAL_PRESETS[presetName as StandardPresetName]
    } else {
      preset = MATERIAL_PRESETS.matte
    }

    // Create material based on quality
    const envMapIntensity = 'envMapIntensity' in preset ? preset.envMapIntensity : 1.0
    const baseParams: THREE.MeshStandardMaterialParameters = {
      color: color ?? '#ffffff',
      metalness: preset.metalness,
      roughness: preset.roughness,
      envMapIntensity: envMapIntensity ?? 1.0,
      ...overrides,
    }

    // Add emissive properties if preset supports it
    if ('emissiveIntensity' in preset) {
      const emissiveColor = preset.emissive ?? color
      baseParams.emissive = new THREE.Color(emissive ?? emissiveColor)
      baseParams.emissiveIntensity = preset.emissiveIntensity
      const toneMapped = preset.toneMapped
      baseParams.toneMapped = toneMapped ?? true
    }

    // Use appropriate material class based on quality
    if (quality === 'physical' && presetName in PHYSICAL_PRESETS) {
      // Cast to a flexible record type to access optional physical properties
      const physicalPreset = preset as unknown as Record<string, unknown>
      const getNum = (key: string, defaultVal: number): number => {
        const val = physicalPreset[key]
        return typeof val === 'number' ? val : defaultVal
      }
      return new THREE.MeshPhysicalMaterial({
        ...baseParams,
        clearcoat: getNum('clearcoat', 0),
        clearcoatRoughness: getNum('clearcoatRoughness', 0),
        transmission: getNum('transmission', 0),
        thickness: getNum('thickness', 0),
        ior: getNum('ior', 1.5),
        iridescence: getNum('iridescence', 0),
        iridescenceIOR: getNum('iridescenceIOR', 1.3),
        sheen: getNum('sheen', 0),
        sheenRoughness: getNum('sheenRoughness', 0),
        sheenColor: typeof physicalPreset.sheenColor === 'string'
          ? new THREE.Color(physicalPreset.sheenColor)
          : undefined,
      } as THREE.MeshPhysicalMaterialParameters)
    }

    if (quality === 'basic') {
      return new THREE.MeshBasicMaterial({
        color: color ?? '#ffffff',
        ...overrides,
      })
    }

    return new THREE.MeshStandardMaterial(baseParams)
  }, [presetName, quality, options])
}

/**
 * Hook to get multiple materials as a map.
 * Useful for complex objects with multiple parts.
 */
export function useMaterialMap<T extends Record<string, StandardPresetName | PhysicalPresetName>>(
  presetMap: T,
  colorMap?: Partial<Record<keyof T, string>>
): Record<keyof T, THREE.Material> {
  const quality = useGraphicsStore((state) => state.config.materialQuality)

  return useMemo(() => {
    const materials = {} as Record<keyof T, THREE.Material>

    for (const [key, presetName] of Object.entries(presetMap)) {
      const color = colorMap?.[key as keyof T]

      let preset: MaterialPreset
      if (presetName in PHYSICAL_PRESETS && quality === 'physical') {
        preset = PHYSICAL_PRESETS[presetName as PhysicalPresetName]
      } else if (presetName in MATERIAL_PRESETS) {
        preset = MATERIAL_PRESETS[presetName as StandardPresetName]
      } else {
        preset = MATERIAL_PRESETS.matte
      }

      const envIntensity = 'envMapIntensity' in preset ? preset.envMapIntensity : 1.0
      const baseParams: THREE.MeshStandardMaterialParameters = {
        color: color ?? '#ffffff',
        metalness: preset.metalness,
        roughness: preset.roughness,
        envMapIntensity: envIntensity ?? 1.0,
      }

      if ('emissiveIntensity' in preset) {
        const emissiveColor = preset.emissive ?? color
        baseParams.emissive = new THREE.Color(emissiveColor)
        baseParams.emissiveIntensity = preset.emissiveIntensity
        const toneMapped = preset.toneMapped
        baseParams.toneMapped = toneMapped ?? true
      }

      if (quality === 'basic') {
        materials[key as keyof T] = new THREE.MeshBasicMaterial({ color: color ?? '#ffffff' })
      } else {
        materials[key as keyof T] = new THREE.MeshStandardMaterial(baseParams)
      }
    }

    return materials
  }, [presetMap, colorMap, quality])
}

/**
 * Hook to get material quality setting
 */
export function useMaterialQuality() {
  return useGraphicsStore((state) => state.config.materialQuality)
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
