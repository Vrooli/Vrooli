/**
 * Environment preset configurations.
 *
 * The sky system uses continuous time (0-24 hours) for smooth day/night
 * transitions. Lighting and sky colors are calculated dynamically from time
 * using calculateLightingPreset() and calculateSkyColors() from '@/lib/sky/sunPosition'.
 */

import type {
  EnvironmentConfig,
  SceneType,
  LightingPreset,
  FogConfig,
  GroundConfig,
  BoundaryConfig,
  PlacementConfig,
  DreiEnvironmentPreset,
} from '@/types/environment'
import { calculateLightingPreset, calculateSkyColors } from '@/lib/sky/sunPosition'

/**
 * Fog presets for different scene types
 */
export const FOG_PRESETS: Record<SceneType, FogConfig | undefined> = {
  'outdoor-park': { color: '#e8f4f8', near: 20, far: 80 },
  'indoor-office': { color: '#f5f5f5', near: 30, far: 60 },
  'abstract-space': { color: '#0f172a', near: 10, far: 50 },
  custom: undefined,
}

/**
 * Create a skybox config from continuous time value.
 * Uses calculateSkyColors() for dynamic gradient generation.
 */
export function createSkyboxFromTime(timeValue: number): { type: 'gradient' | 'solid'; source: string[] } {
  const colors = calculateSkyColors(timeValue)
  return {
    type: 'gradient',
    source: [colors.top, colors.middle, colors.bottom],
  }
}

/**
 * Ground presets
 */
export const GROUND_PRESETS: Record<SceneType, GroundConfig> = {
  'outdoor-park': {
    visible: true,
    type: 'plane',
    color: '#228B22',
    size: 100,
    position: 0,
    material: {
      type: 'texture',
      color: '#2f6b3a',
      texture: {
        id: 'grass',
        tileSize: 4,
        projection: 'triplanar',
        normalScale: 0.7,
        roughnessIntensity: 0.9,
        aoIntensity: 0.6,
        macroVariation: {
          enabled: true,
          scale: 18,
          intensity: 0.35,
        },
      },
    },
  },
  'indoor-office': {
    visible: true,
    type: 'plane',
    color: '#808080',
    size: 50,
    position: 0,
    material: {
      type: 'texture',
      color: '#7a7a7a',
      texture: {
        id: 'concrete',
        tileSize: 3,
        projection: 'uv',
        normalScale: 0.45,
        roughnessIntensity: 0.75,
        aoIntensity: 0.5,
        macroVariation: {
          enabled: true,
          scale: 12,
          intensity: 0.2,
        },
      },
    },
  },
  'abstract-space': {
    visible: true,
    type: 'grid',
    color: '#1e293b',
    size: 30,
    divisions: 30,
    position: 0,
    material: {
      type: 'solid',
      color: '#1e293b',
    },
  },
  custom: {
    visible: false,
    type: 'none',
    material: {
      type: 'solid',
      color: '#2f2f2f',
    },
  },
}

/**
 * Default boundary configuration per scene type.
 */
export const BOUNDARY_PRESETS: Record<SceneType, BoundaryConfig> = {
  'outdoor-park': {
    visible: true,
    shape: 'square',
    size: 200,
    position: 0.01,
    color: '#94a3b8',
    opacity: 0.4,
  },
  'indoor-office': {
    visible: true,
    shape: 'square',
    size: 100,
    position: 0.01,
    color: '#94a3b8',
    opacity: 0.4,
  },
  'abstract-space': {
    visible: true,
    shape: 'square',
    size: 60,
    position: 0.01,
    color: '#94a3b8',
    opacity: 0.4,
  },
  custom: {
    visible: false,
    shape: 'square',
  },
}

/**
 * Default placement configuration
 */
export const PLACEMENT_DEFAULTS: PlacementConfig = {
  snapToGrid: true,
  snapSize: 1,
  clampToBoundary: true,
}

/**
 * Theme to drei preset mapping
 */
export const THEME_TO_DREI_PRESET: Record<'light' | 'dark', DreiEnvironmentPreset> = {
  light: 'studio',
  dark: 'night',
}

/**
 * Map continuous time (0-24) to drei preset.
 * Uses time ranges to select appropriate preset.
 */
export function getPresetFromTime(timeValue: number): DreiEnvironmentPreset {
  const hour = ((timeValue % 24) + 24) % 24 // Normalize to 0-24
  if (hour >= 5 && hour < 9) return 'dawn'
  if (hour >= 9 && hour < 17) return 'studio'
  if (hour >= 17 && hour < 20) return 'sunset'
  return 'night'
}

/** Time constants for presets (hours in 24h format) */
const TIME_NIGHT = 22
const TIME_NOON = 12
const TIME_MORNING = 8
const TIME_SUNSET = 18.5

/**
 * Pre-built environment configurations.
 * Lighting and skybox are calculated dynamically from timeValue.
 */
export const ENVIRONMENT_PRESETS: Record<string, EnvironmentConfig> = {
  'default-dark': {
    id: 'default-dark',
    name: 'Dark Space',
    type: 'abstract-space',
    timeValue: TIME_NIGHT,
    lighting: calculateLightingPreset(TIME_NIGHT),
    fog: FOG_PRESETS['abstract-space'],
    skybox: createSkyboxFromTime(TIME_NIGHT),
    ground: GROUND_PRESETS['abstract-space'],
    boundary: BOUNDARY_PRESETS['abstract-space'],
    placement: PLACEMENT_DEFAULTS,
  },
  'default-light': {
    id: 'default-light',
    name: 'Light Studio',
    type: 'indoor-office',
    timeValue: TIME_NOON,
    lighting: calculateLightingPreset(TIME_NOON),
    fog: FOG_PRESETS['indoor-office'],
    skybox: { type: 'solid', source: '#f8fafc' },
    ground: {
      visible: true,
      type: 'grid',
      color: '#e2e8f0',
      size: 30,
      divisions: 30,
      position: 0,
    },
    boundary: {
      ...BOUNDARY_PRESETS['indoor-office'],
      size: 60,
    },
    placement: PLACEMENT_DEFAULTS,
  },
  'outdoor-morning': {
    id: 'outdoor-morning',
    name: 'Morning Park',
    type: 'outdoor-park',
    timeValue: TIME_MORNING,
    lighting: calculateLightingPreset(TIME_MORNING),
    fog: FOG_PRESETS['outdoor-park'],
    skybox: createSkyboxFromTime(TIME_MORNING),
    ground: GROUND_PRESETS['outdoor-park'],
    boundary: BOUNDARY_PRESETS['outdoor-park'],
    placement: PLACEMENT_DEFAULTS,
  },
  'outdoor-sunset': {
    id: 'outdoor-sunset',
    name: 'Sunset Vista',
    type: 'outdoor-park',
    timeValue: TIME_SUNSET,
    lighting: calculateLightingPreset(TIME_SUNSET),
    fog: { color: '#FFB347', near: 15, far: 60 },
    skybox: createSkyboxFromTime(TIME_SUNSET),
    ground: GROUND_PRESETS['outdoor-park'],
    boundary: BOUNDARY_PRESETS['outdoor-park'],
    placement: PLACEMENT_DEFAULTS,
  },
}

/**
 * Get environment config for current theme
 */
export function getEnvironmentForTheme(theme: 'light' | 'dark'): EnvironmentConfig {
  const presetKey = theme === 'dark' ? 'default-dark' : 'default-light'
  const preset = ENVIRONMENT_PRESETS[presetKey]
  // These presets are guaranteed to exist in ENVIRONMENT_PRESETS
  if (!preset) {
    throw new Error(`Environment preset '${presetKey}' not found`)
  }
  return preset
}

/**
 * Create custom environment config
 */
export function createEnvironmentConfig(
  id: string,
  name: string,
  options: {
    sceneType?: SceneType
    /** Continuous time value (0-24 hours). Default: 8 (morning) */
    timeValue?: number
    customLighting?: Partial<LightingPreset>
    customFog?: FogConfig
    customGround?: Partial<GroundConfig>
    customBoundary?: Partial<BoundaryConfig>
    customPlacement?: Partial<PlacementConfig>
  }
): EnvironmentConfig {
  const { sceneType = 'outdoor-park', timeValue = TIME_MORNING } = options

  const ground = {
    ...GROUND_PRESETS[sceneType],
    ...options.customGround,
  }

  const boundary: BoundaryConfig = {
    ...BOUNDARY_PRESETS[sceneType],
    ...options.customBoundary,
  }

  if (options.customBoundary?.size === undefined && options.customGround?.size) {
    boundary.size = options.customGround.size * 2
  }

  const snapFromGrid = ground.type === 'grid' && ground.size && ground.divisions
    ? ground.size / ground.divisions
    : undefined

  const placement: PlacementConfig = {
    ...PLACEMENT_DEFAULTS,
    ...(snapFromGrid ? { snapSize: snapFromGrid } : null),
    ...options.customPlacement,
  }

  return {
    id,
    name,
    type: sceneType,
    timeValue,
    lighting: {
      ...calculateLightingPreset(timeValue),
      ...options.customLighting,
    },
    fog: options.customFog ?? FOG_PRESETS[sceneType],
    skybox: createSkyboxFromTime(timeValue),
    ground,
    boundary,
    placement,
  }
}
