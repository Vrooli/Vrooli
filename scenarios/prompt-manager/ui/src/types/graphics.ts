/**
 * Graphics configuration types for the 3D world rendering system.
 * Supports multiple quality tiers for different device capabilities.
 */

/** Performance tier for automatic quality adjustment */
export type PerformanceTier = 'low' | 'medium' | 'high' | 'ultra'

/** Antialiasing method */
export type AntialiasingMethod = 'none' | 'fxaa' | 'smaa'

/** Material quality level */
export type MaterialQuality = 'basic' | 'standard' | 'physical'

/**
 * Graphics configuration for the render pipeline.
 * Each setting can be adjusted independently.
 */
export interface GraphicsConfig {
  /** Device pixel ratio - can be fixed or adaptive range */
  dpr: number | [number, number]
  /** Enable shadow rendering */
  shadows: boolean
  /** Shadow map resolution (512, 1024, 2048, 4096) */
  shadowMapSize: number
  /** Enable post-processing effects */
  postProcessing: boolean
  /** Material quality level */
  materialQuality: MaterialQuality
  /** Enable environment map reflections */
  envMap: boolean
  /** Enable bloom effect */
  bloom: boolean
  /** Enable SSAO (Screen Space Ambient Occlusion) */
  ssao: boolean
  /** Antialiasing method */
  antialiasing: AntialiasingMethod
  /** Enable vignette effect */
  vignette: boolean
  /** Enable contact shadows */
  contactShadows: boolean
  /** Enable agent vertex wobble animation */
  agentWobble: boolean
}

/**
 * Bloom effect configuration
 */
export interface BloomConfig {
  /** Brightness threshold for bloom (0-1) */
  luminanceThreshold: number
  /** Smoothness of threshold transition */
  luminanceSmoothing: number
  /** Bloom intensity (0-2) */
  intensity: number
  /** Blur kernel size */
  kernelSize: number
}

/**
 * SSAO effect configuration
 */
export interface SSAOConfig {
  /** Occlusion sampling radius */
  radius: number
  /** Occlusion intensity */
  intensity: number
  /** Number of samples */
  samples: number
}

/**
 * Vignette effect configuration
 */
export interface VignetteConfig {
  /** Vignette offset from edges */
  offset: number
  /** Vignette darkness (0-1.5) */
  darkness: number
}

/**
 * Complete post-processing configuration
 */
export interface PostProcessingConfig {
  bloom: BloomConfig
  ssao: SSAOConfig
  vignette: VignetteConfig
}

/**
 * Device capability information for auto-detection
 */
export interface DeviceCapability {
  /** Detected GPU tier (0-3) */
  gpuTier: number
  /** Whether the device is mobile */
  isMobile: boolean
  /** Recommended performance tier */
  recommendedTier: PerformanceTier
}
