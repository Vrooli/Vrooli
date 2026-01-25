/**
 * useMaterial hook for accessing material presets and creating materials.
 */

import { useMemo } from 'react'
import * as THREE from 'three'
import { useGraphicsStore } from '@/stores/graphicsStore'
import {
  MATERIAL_PRESETS,
  PHYSICAL_PRESETS,
  type StandardPresetName,
  type PhysicalPresetName,
  type MaterialPreset,
  type EmissiveMaterialPreset,
} from './presets'

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
    const baseParams: THREE.MeshStandardMaterialParameters = {
      color: color ?? '#ffffff',
      metalness: preset.metalness,
      roughness: preset.roughness,
      envMapIntensity: preset.envMapIntensity ?? 1.0,
      ...overrides,
    }

    // Add emissive properties if preset supports it
    if ('emissiveIntensity' in preset) {
      const emissivePreset = preset as EmissiveMaterialPreset
      baseParams.emissive = new THREE.Color(emissive ?? emissivePreset.emissive ?? color)
      baseParams.emissiveIntensity = emissivePreset.emissiveIntensity
      baseParams.toneMapped = emissivePreset.toneMapped ?? true
    }

    // Use appropriate material class based on quality
    if (quality === 'physical' && presetName in PHYSICAL_PRESETS) {
      // Cast to a flexible record type to access optional physical properties
      const physicalPreset = preset as unknown as Record<string, unknown>
      return new THREE.MeshPhysicalMaterial({
        ...baseParams,
        clearcoat: (physicalPreset.clearcoat as number) ?? 0,
        clearcoatRoughness: (physicalPreset.clearcoatRoughness as number) ?? 0,
        transmission: (physicalPreset.transmission as number) ?? 0,
        thickness: (physicalPreset.thickness as number) ?? 0,
        ior: (physicalPreset.ior as number) ?? 1.5,
        iridescence: (physicalPreset.iridescence as number) ?? 0,
        iridescenceIOR: (physicalPreset.iridescenceIOR as number) ?? 1.3,
        sheen: (physicalPreset.sheen as number) ?? 0,
        sheenRoughness: (physicalPreset.sheenRoughness as number) ?? 0,
        sheenColor: physicalPreset.sheenColor
          ? new THREE.Color(physicalPreset.sheenColor as string)
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

      const baseParams: THREE.MeshStandardMaterialParameters = {
        color: color ?? '#ffffff',
        metalness: preset.metalness,
        roughness: preset.roughness,
        envMapIntensity: preset.envMapIntensity ?? 1.0,
      }

      if ('emissiveIntensity' in preset) {
        const emissivePreset = preset as EmissiveMaterialPreset
        baseParams.emissive = new THREE.Color(emissivePreset.emissive ?? color)
        baseParams.emissiveIntensity = emissivePreset.emissiveIntensity
        baseParams.toneMapped = emissivePreset.toneMapped ?? true
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
