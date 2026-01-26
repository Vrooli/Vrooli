/**
 * Environment preset configurations.
 */

import type {
  EnvironmentConfig,
  TimeOfDay,
  SceneType,
  LightingPreset,
  FogConfig,
  SkyboxConfig,
  GroundConfig,
  DreiEnvironmentPreset,
} from '@/types/environment'

/**
 * Lighting presets for different times of day
 */
export const LIGHTING_PRESETS: Record<TimeOfDay, LightingPreset> = {
  morning: {
    ambient: { color: '#fff5e6', intensity: 0.5 },
    directional: [
      {
        position: [5, 10, 5],
        color: '#ffeedd',
        intensity: 1.2,
        castShadow: true,
        shadowMapSize: 2048,
      },
    ],
    point: [
      { position: [0, 3, 0], color: '#ffe4c4', intensity: 0.2, distance: 15 },
    ],
  },
  noon: {
    ambient: { color: '#ffffff', intensity: 0.6 },
    directional: [
      {
        position: [0, 15, 0],
        color: '#ffffff',
        intensity: 1.5,
        castShadow: true,
        shadowMapSize: 2048,
      },
    ],
  },
  sunset: {
    ambient: { color: '#ffddcc', intensity: 0.4 },
    directional: [
      {
        position: [-10, 5, 5],
        color: '#ff9966',
        intensity: 1.0,
        castShadow: true,
        shadowMapSize: 2048,
      },
    ],
    point: [
      { position: [10, 2, 0], color: '#ff6633', intensity: 0.5, distance: 20 },
      { position: [-5, 1, -5], color: '#ff4488', intensity: 0.3, distance: 15 },
    ],
  },
  night: {
    ambient: { color: '#334466', intensity: 0.3 },
    directional: [
      {
        position: [5, 10, -5],
        color: '#6688bb',
        intensity: 0.6,
        castShadow: true,
        shadowMapSize: 2048,
      },
    ],
    point: [
      { position: [0, 5, 0], color: '#6366f1', intensity: 0.4, distance: 15 },
      { position: [-10, 3, -10], color: '#22d3ee', intensity: 0.3, distance: 12 },
    ],
  },
}

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
 * Skybox presets
 */
export const SKYBOX_PRESETS: Record<TimeOfDay, SkyboxConfig> = {
  morning: {
    type: 'gradient',
    source: ['#87CEEB', '#FFF8DC', '#FFE4B5'],
    blur: 0.5,
  },
  noon: {
    type: 'gradient',
    source: ['#87CEEB', '#ADD8E6'],
    blur: 0.3,
  },
  sunset: {
    type: 'gradient',
    source: ['#2C1810', '#FF6B35', '#F7C59F'],
    blur: 0.4,
  },
  night: {
    type: 'solid',
    source: '#0f172a',
    blur: 0,
  },
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
  },
  'indoor-office': {
    visible: true,
    type: 'plane',
    color: '#808080',
    size: 50,
    position: 0,
  },
  'abstract-space': {
    visible: true,
    type: 'grid',
    color: '#1e293b',
    size: 30,
    divisions: 30,
    position: 0,
  },
  custom: {
    visible: false,
    type: 'none',
  },
}

/**
 * Theme to drei preset mapping
 */
export const THEME_TO_DREI_PRESET: Record<'light' | 'dark', DreiEnvironmentPreset> = {
  light: 'studio',
  dark: 'night',
}

/**
 * Time of day to drei preset mapping
 */
export const TIME_TO_DREI_PRESET: Record<TimeOfDay, DreiEnvironmentPreset> = {
  morning: 'dawn',
  noon: 'studio',
  sunset: 'sunset',
  night: 'night',
}

/**
 * Pre-built environment configurations
 */
export const ENVIRONMENT_PRESETS: Record<string, EnvironmentConfig> = {
  'default-dark': {
    id: 'default-dark',
    name: 'Dark Space',
    type: 'abstract-space',
    timeOfDay: 'night',
    lighting: LIGHTING_PRESETS.night,
    fog: FOG_PRESETS['abstract-space'],
    skybox: SKYBOX_PRESETS.night,
    ground: GROUND_PRESETS['abstract-space'],
  },
  'default-light': {
    id: 'default-light',
    name: 'Light Studio',
    type: 'indoor-office',
    timeOfDay: 'noon',
    lighting: LIGHTING_PRESETS.noon,
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
  },
  'outdoor-morning': {
    id: 'outdoor-morning',
    name: 'Morning Park',
    type: 'outdoor-park',
    timeOfDay: 'morning',
    lighting: LIGHTING_PRESETS.morning,
    fog: FOG_PRESETS['outdoor-park'],
    skybox: SKYBOX_PRESETS.morning,
    ground: GROUND_PRESETS['outdoor-park'],
  },
  'outdoor-sunset': {
    id: 'outdoor-sunset',
    name: 'Sunset Vista',
    type: 'outdoor-park',
    timeOfDay: 'sunset',
    lighting: LIGHTING_PRESETS.sunset,
    fog: { color: '#FFB347', near: 15, far: 60 },
    skybox: SKYBOX_PRESETS.sunset,
    ground: GROUND_PRESETS['outdoor-park'],
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
    timeOfDay?: TimeOfDay
    customLighting?: Partial<LightingPreset>
    customFog?: FogConfig
    customGround?: Partial<GroundConfig>
  }
): EnvironmentConfig {
  const { sceneType = 'abstract-space', timeOfDay = 'night' } = options

  return {
    id,
    name,
    type: sceneType,
    timeOfDay,
    lighting: {
      ...LIGHTING_PRESETS[timeOfDay],
      ...options.customLighting,
    },
    fog: options.customFog ?? FOG_PRESETS[sceneType],
    skybox: SKYBOX_PRESETS[timeOfDay],
    ground: {
      ...GROUND_PRESETS[sceneType],
      ...options.customGround,
    },
  }
}
