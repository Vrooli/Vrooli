/**
 * Material preset definitions for consistent visual styling.
 * These presets can be used with MeshStandardMaterial or MeshPhysicalMaterial.
 */

import type { MaterialQuality } from '@/types/graphics'

/**
 * Base material properties shared across presets
 */
export interface MaterialPresetBase {
  metalness: number
  roughness: number
  envMapIntensity?: number
}

/**
 * Extended properties for physical materials
 */
export interface PhysicalMaterialPreset extends MaterialPresetBase {
  clearcoat?: number
  clearcoatRoughness?: number
  transmission?: number
  thickness?: number
  ior?: number
  reflectivity?: number
  iridescence?: number
  iridescenceIOR?: number
  sheen?: number
  sheenRoughness?: number
  sheenColor?: string
}

/**
 * Emissive material preset
 */
export interface EmissiveMaterialPreset extends MaterialPresetBase {
  emissive?: string
  emissiveIntensity: number
  toneMapped?: boolean
}

/**
 * Combined material preset type
 */
export type MaterialPreset = MaterialPresetBase | PhysicalMaterialPreset | EmissiveMaterialPreset

/**
 * Standard material presets for common use cases
 */
export const MATERIAL_PRESETS = {
  /** Shiny metallic surface */
  metallic: {
    metalness: 0.8,
    roughness: 0.2,
    envMapIntensity: 1.5,
  } satisfies MaterialPresetBase,

  /** Soft matte finish */
  matte: {
    metalness: 0.0,
    roughness: 0.9,
    envMapIntensity: 0.3,
  } satisfies MaterialPresetBase,

  /** Semi-glossy plastic-like surface */
  plastic: {
    metalness: 0.0,
    roughness: 0.4,
    envMapIntensity: 0.8,
  } satisfies MaterialPresetBase,

  /** Ceramic/porcelain-like surface */
  ceramic: {
    metalness: 0.0,
    roughness: 0.3,
    envMapIntensity: 1.0,
  } satisfies MaterialPresetBase,

  /** Soft skin-like material */
  skin: {
    metalness: 0.0,
    roughness: 0.6,
    envMapIntensity: 0.4,
  } satisfies MaterialPresetBase,

  /** Glowing emissive material (for bloom) */
  emissive: {
    metalness: 0.0,
    roughness: 0.5,
    envMapIntensity: 0.3,
    emissiveIntensity: 0.5,
    toneMapped: false, // Allows bloom to work
  } satisfies EmissiveMaterialPreset,

  /** Strong glow for accents */
  glowing: {
    metalness: 0.0,
    roughness: 0.3,
    envMapIntensity: 0.2,
    emissiveIntensity: 1.0,
    toneMapped: false,
  } satisfies EmissiveMaterialPreset,
} as const

/**
 * Physical material presets (for high quality rendering)
 */
export const PHYSICAL_PRESETS = {
  /** Glass-like transparent material */
  glass: {
    metalness: 0.0,
    roughness: 0.1,
    transmission: 0.9,
    thickness: 0.5,
    ior: 1.5,
    envMapIntensity: 1.0,
  } satisfies PhysicalMaterialPreset,

  /** Frosted glass effect */
  frostedGlass: {
    metalness: 0.0,
    roughness: 0.3,
    transmission: 0.8,
    thickness: 0.5,
    ior: 1.3,
    envMapIntensity: 0.5,
  } satisfies PhysicalMaterialPreset,

  /** Car paint with clearcoat */
  carPaint: {
    metalness: 0.9,
    roughness: 0.3,
    clearcoat: 1.0,
    clearcoatRoughness: 0.1,
    envMapIntensity: 1.5,
  } satisfies PhysicalMaterialPreset,

  /** Iridescent holographic surface */
  holographic: {
    metalness: 0.5,
    roughness: 0.2,
    iridescence: 1.0,
    iridescenceIOR: 1.3,
    envMapIntensity: 1.2,
  } satisfies PhysicalMaterialPreset,

  /** Velvet/fabric with sheen */
  velvet: {
    metalness: 0.0,
    roughness: 0.8,
    sheen: 1.0,
    sheenRoughness: 0.5,
    sheenColor: '#ffffff',
    envMapIntensity: 0.3,
  } satisfies PhysicalMaterialPreset,
} as const

/**
 * Get appropriate preset based on quality setting
 */
export function getPresetForQuality(
  presetName: keyof typeof MATERIAL_PRESETS | keyof typeof PHYSICAL_PRESETS,
  quality: MaterialQuality
): MaterialPreset {
  // Physical presets only available at physical quality
  if (presetName in PHYSICAL_PRESETS) {
    if (quality === 'physical') {
      return PHYSICAL_PRESETS[presetName as keyof typeof PHYSICAL_PRESETS]
    }
    // Fallback to basic metallic for glass/special materials
    return MATERIAL_PRESETS.metallic
  }

  return MATERIAL_PRESETS[presetName as keyof typeof MATERIAL_PRESETS]
}

/**
 * Preset names for type-safe access
 */
export type StandardPresetName = keyof typeof MATERIAL_PRESETS
export type PhysicalPresetName = keyof typeof PHYSICAL_PRESETS
export type PresetName = StandardPresetName | PhysicalPresetName

/**
 * All available preset names
 */
export const PRESET_NAMES = [
  ...Object.keys(MATERIAL_PRESETS),
  ...Object.keys(PHYSICAL_PRESETS),
] as PresetName[]
