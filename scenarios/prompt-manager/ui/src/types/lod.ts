/**
 * Level of Detail (LOD) types for optimizing 3D rendering.
 * Supports distance-based detail scaling and cursor tracking optimization.
 */

/** LOD levels from highest to lowest detail */
export type LODLevel = 'high' | 'medium' | 'low' | 'culled'

/**
 * Configuration for LOD distance thresholds.
 * Objects beyond each threshold switch to the next lower LOD level.
 */
export interface LODThresholds {
  /** Distance below which high detail is used */
  high: number
  /** Distance below which medium detail is used */
  medium: number
  /** Distance below which low detail is used */
  low: number
  /** Distance beyond which object is culled (not rendered) */
  culled: number
}

/**
 * LOD state for a single object
 */
export interface LODState {
  /** Current LOD level */
  level: LODLevel
  /** Distance from camera */
  distance: number
  /** Whether object is visible (not culled) */
  isVisible: boolean
  /** Whether cursor tracking should be active */
  shouldTrackCursor: boolean
  /** Whether animations should be simplified */
  useSimplifiedAnimations: boolean
  /** Whether object should receive hover events */
  canReceiveHover: boolean
}

/**
 * Configuration for LOD behavior
 */
export interface LODConfig {
  /** Distance thresholds for LOD switching */
  thresholds: LODThresholds
  /** Whether to enable cursor tracking LOD */
  enableCursorLOD: boolean
  /** Whether to enable animation LOD */
  enableAnimationLOD: boolean
  /** Whether to enable hover LOD */
  enableHoverLOD: boolean
  /** Hysteresis factor to prevent LOD flickering (0-1) */
  hysteresis: number
}

/** Default LOD thresholds optimized for member viewing */
export const DEFAULT_LOD_THRESHOLDS: LODThresholds = {
  high: 5,      // Full detail within 5 units
  medium: 12,   // Medium detail 5-12 units
  low: 25,      // Low detail 12-25 units
  culled: 50,   // Culled beyond 50 units
}

/** Default LOD configuration */
export const DEFAULT_LOD_CONFIG: LODConfig = {
  thresholds: DEFAULT_LOD_THRESHOLDS,
  enableCursorLOD: true,
  enableAnimationLOD: true,
  enableHoverLOD: true,
  hysteresis: 0.1, // 10% hysteresis to prevent flickering
}
