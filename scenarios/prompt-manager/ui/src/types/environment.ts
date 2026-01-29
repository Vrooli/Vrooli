/**
 * Environment types for scene backgrounds and lighting.
 * Supports different presets and times of day.
 */

/** Time of day variants */
export type TimeOfDay = 'morning' | 'noon' | 'sunset' | 'night'

/** Scene type categories */
export type SceneType = 'outdoor-park' | 'indoor-office' | 'abstract-space' | 'custom'

/** Skybox rendering method */
export type SkyboxType = 'hdri' | 'gradient' | 'solid' | 'procedural'

/**
 * Ambient light configuration
 */
export interface AmbientLightConfig {
  color: string
  intensity: number
}

/**
 * Directional light configuration
 */
export interface DirectionalLightConfig {
  position: [number, number, number]
  color: string
  intensity: number
  castShadow?: boolean
  shadowMapSize?: number
}

/**
 * Point light configuration
 */
export interface PointLightConfig {
  position: [number, number, number]
  color: string
  intensity: number
  distance?: number
  decay?: number
}

/**
 * Complete lighting preset
 */
export interface LightingPreset {
  ambient: AmbientLightConfig
  directional: DirectionalLightConfig[]
  point?: PointLightConfig[]
}

/**
 * Fog configuration
 */
export interface FogConfig {
  color: string
  near: number
  far: number
}

/**
 * Skybox configuration
 */
export interface SkyboxConfig {
  type: SkyboxType
  /** HDRI path or gradient colors */
  source?: string | string[]
  /** Rotation in radians */
  rotation?: number
  /** Background blur amount */
  blur?: number
}

/**
 * Ground/floor configuration
 */
export interface GroundConfig {
  visible: boolean
  type: 'grid' | 'plane' | 'none'
  color?: string
  size?: number
  divisions?: number
  /** Y position of ground */
  position?: number
  /** Material configuration for textured/solid ground surfaces */
  material?: GroundMaterialConfig
}

export type GroundMaterialType = 'solid' | 'texture'

export type GroundTextureId = 'grass' | 'concrete' | 'wood-plank' | 'stone' | 'metal-panel'

export type GroundProjection = 'uv' | 'triplanar'

export interface GroundMacroVariationConfig {
  enabled: boolean
  /** World units per macro texture tile */
  scale: number
  /** Intensity multiplier (0-1 recommended) */
  intensity: number
}

export interface GroundTextureConfig {
  /** Texture set identifier */
  id: GroundTextureId
  /** World units per texture tile */
  tileSize?: number
  /** UV rotation in radians */
  rotation?: number
  /** Projection mode for textures */
  projection?: GroundProjection
  /** Normal map intensity multiplier */
  normalScale?: number
  /** Roughness multiplier applied to the material */
  roughnessIntensity?: number
  /** Ambient occlusion intensity multiplier */
  aoIntensity?: number
  /** Macro variation overlay settings */
  macroVariation?: GroundMacroVariationConfig
  /** Enable stochastic tiling to eliminate visible repetition (default: true) */
  stochasticEnabled?: boolean
}

export interface GroundMaterialConfig {
  type: GroundMaterialType
  /** Solid color fallback or tint */
  color?: string
  /** Texture-based material settings */
  texture?: GroundTextureConfig
}

/**
 * Boundary outline configuration for the world.
 * Shapes are centered at the origin on the XZ plane.
 */
export interface BoundaryConfig {
  visible: boolean
  shape: 'square' | 'circle' | 'path'
  /** Size for square (side length) or circle (diameter) */
  size?: number
  /** Path points on the XZ plane (used when shape = "path") */
  points?: [number, number][]
  /** Y position of boundary line */
  position?: number
  color?: string
  opacity?: number
}

/**
 * Placement rules for dragging and dropping objects.
 */
export interface PlacementConfig {
  /** Snap placement to the grid */
  snapToGrid: boolean
  /** Grid size for snapping (world units) */
  snapSize: number
  /** Clamp placement to the boundary */
  clampToBoundary: boolean
}

/**
 * Complete environment configuration
 */
export interface EnvironmentConfig {
  id: string
  name: string
  type: SceneType
  timeOfDay: TimeOfDay
  lighting: LightingPreset
  fog?: FogConfig
  skybox: SkyboxConfig
  ground: GroundConfig
  boundary: BoundaryConfig
  placement: PlacementConfig
}

/**
 * Environment transition options
 */
export interface EnvironmentTransition {
  /** Duration in seconds */
  duration: number
  /** Easing function name */
  easing: 'linear' | 'easeIn' | 'easeOut' | 'easeInOut'
}

/**
 * drei Environment preset names (subset of available)
 */
export type DreiEnvironmentPreset =
  | 'apartment'
  | 'city'
  | 'dawn'
  | 'forest'
  | 'lobby'
  | 'night'
  | 'park'
  | 'studio'
  | 'sunset'
  | 'warehouse'

/**
 * Theme-to-environment mapping
 */
export const THEME_ENVIRONMENTS: Record<'dark' | 'light', DreiEnvironmentPreset> = {
  dark: 'night',
  light: 'studio',
}

/**
 * Default lighting presets for each time of day
 */
export const TIME_OF_DAY_LIGHTING: Record<TimeOfDay, LightingPreset> = {
  morning: {
    ambient: { color: '#fff5e6', intensity: 0.5 },
    directional: [
      { position: [5, 10, 5], color: '#ffeedd', intensity: 1.2, castShadow: true },
    ],
  },
  noon: {
    ambient: { color: '#ffffff', intensity: 0.6 },
    directional: [
      { position: [0, 15, 0], color: '#ffffff', intensity: 1.5, castShadow: true },
    ],
  },
  sunset: {
    ambient: { color: '#ffddcc', intensity: 0.4 },
    directional: [
      { position: [-10, 5, 5], color: '#ff9966', intensity: 1.0, castShadow: true },
    ],
    point: [
      { position: [10, 2, 0], color: '#ff6633', intensity: 0.5, distance: 20 },
    ],
  },
  night: {
    ambient: { color: '#334466', intensity: 0.3 },
    directional: [
      { position: [5, 10, -5], color: '#6688bb', intensity: 0.6, castShadow: true },
    ],
    point: [
      { position: [0, 5, 0], color: '#6366f1', intensity: 0.3, distance: 15 },
    ],
  },
}

/**
 * Default fog settings for each scene type
 */
export const SCENE_TYPE_FOG: Record<SceneType, FogConfig | undefined> = {
  'outdoor-park': { color: '#e8f4f8', near: 20, far: 80 },
  'indoor-office': { color: '#f5f5f5', near: 30, far: 60 },
  'abstract-space': { color: '#0f172a', near: 10, far: 50 },
  custom: undefined,
}
