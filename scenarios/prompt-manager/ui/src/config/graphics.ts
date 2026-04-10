/**
 * Graphics configuration presets and performance tier definitions.
 */

import type {
  PerformanceTier,
  GraphicsConfig,
  BloomConfig,
  SSAOConfig,
  VignetteConfig,
  PostProcessingConfig,
} from '@/types/graphics'

/**
 * Performance tier configurations
 */
export const PERFORMANCE_TIERS: Record<PerformanceTier, GraphicsConfig> = {
  low: {
    dpr: 1,
    shadows: false,
    shadowMapSize: 512,
    postProcessing: false,
    materialQuality: 'basic',
    envMap: false,
    bloom: false,
    ssao: false,
    antialiasing: 'none',
    vignette: false,
    contactShadows: false,
    agentWobble: false,
  },
  medium: {
    dpr: 1,
    shadows: true,
    shadowMapSize: 1024,
    postProcessing: true,
    materialQuality: 'standard',
    envMap: true,
    bloom: true,
    ssao: false,
    antialiasing: 'fxaa',
    vignette: true,
    contactShadows: true,
    agentWobble: true,
  },
  high: {
    dpr: [1, 1.5],
    shadows: true,
    shadowMapSize: 2048,
    postProcessing: true,
    materialQuality: 'physical',
    envMap: true,
    bloom: true,
    ssao: true,
    antialiasing: 'smaa',
    vignette: true,
    contactShadows: true,
    agentWobble: true,
  },
  ultra: {
    dpr: 2,
    shadows: true,
    shadowMapSize: 4096,
    postProcessing: true,
    materialQuality: 'physical',
    envMap: true,
    bloom: true,
    ssao: true,
    antialiasing: 'smaa',
    vignette: true,
    contactShadows: true,
    agentWobble: true,
  },
}

/**
 * Default bloom effect configuration
 */
export const DEFAULT_BLOOM_CONFIG: BloomConfig = {
  luminanceThreshold: 0.9,
  luminanceSmoothing: 0.025,
  intensity: 0.4,
  kernelSize: 3,
}

/**
 * Bloom configs by performance tier
 */
export const BLOOM_CONFIGS: Record<PerformanceTier, BloomConfig | null> = {
  low: null,
  medium: {
    luminanceThreshold: 0.9,
    luminanceSmoothing: 0.025,
    intensity: 0.3,
    kernelSize: 2,
  },
  high: {
    luminanceThreshold: 0.85,
    luminanceSmoothing: 0.025,
    intensity: 0.4,
    kernelSize: 3,
  },
  ultra: {
    luminanceThreshold: 0.8,
    luminanceSmoothing: 0.02,
    intensity: 0.5,
    kernelSize: 4,
  },
}

/**
 * Default SSAO configuration
 */
export const DEFAULT_SSAO_CONFIG: SSAOConfig = {
  radius: 0.5,
  intensity: 1.5,
  samples: 16,
}

/**
 * SSAO configs by performance tier
 */
export const SSAO_CONFIGS: Record<PerformanceTier, SSAOConfig | null> = {
  low: null,
  medium: null,
  high: {
    radius: 0.4,
    intensity: 1.2,
    samples: 16,
  },
  ultra: {
    radius: 0.5,
    intensity: 1.5,
    samples: 32,
  },
}

/**
 * Default vignette configuration
 */
export const DEFAULT_VIGNETTE_CONFIG: VignetteConfig = {
  offset: 0.1,
  darkness: 1.1,
}

/**
 * Get complete post-processing config for a performance tier
 */
export function getPostProcessingConfig(tier: PerformanceTier): PostProcessingConfig | null {
  const tierConfig = PERFORMANCE_TIERS[tier]
  if (!tierConfig.postProcessing) return null

  return {
    bloom: BLOOM_CONFIGS[tier] ?? DEFAULT_BLOOM_CONFIG,
    ssao: SSAO_CONFIGS[tier] ?? DEFAULT_SSAO_CONFIG,
    vignette: DEFAULT_VIGNETTE_CONFIG,
  }
}

/**
 * GPU tier thresholds for auto-detection
 */
export const GPU_TIER_THRESHOLDS = {
  mobile: { fps: 30, recommendation: 'low' as PerformanceTier },
  lowEnd: { fps: 45, recommendation: 'medium' as PerformanceTier },
  midRange: { fps: 60, recommendation: 'high' as PerformanceTier },
  highEnd: { fps: 120, recommendation: 'ultra' as PerformanceTier },
}

/**
 * Detect whether the device has an integrated (low-power) GPU
 * by checking the WebGL renderer string.
 */
function detectIntegratedGPU(): boolean {
  try {
    const canvas = document.createElement('canvas')
    const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl')
    if (!gl) return true // No WebGL = assume weak
    const ext = (gl as WebGLRenderingContext).getExtension('WEBGL_debug_renderer_info')
    if (!ext) return false // Extension unavailable — can't tell
    const renderer = String((gl as WebGLRenderingContext).getParameter(ext.UNMASKED_RENDERER_WEBGL))
    canvas.remove()
    return /Intel|Integrated|UHD|HD Graphics|Mali|Adreno|Apple GPU/i.test(renderer)
  } catch {
    return false
  }
}

/**
 * Detect recommended performance tier based on device capabilities.
 * Checks mobile UA, GPU type, core count, and DPR.
 */
export function detectRecommendedTier(): PerformanceTier {
  if (typeof window === 'undefined') return 'medium'

  // Check if mobile
  const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
    navigator.userAgent
  )
  if (isMobile) return 'low'

  // Integrated GPUs and low-core machines should stay conservative
  const isIntegrated = detectIntegratedGPU()
  const cores = navigator.hardwareConcurrency || 4
  const dpr = window.devicePixelRatio

  if (isIntegrated || cores <= 4) {
    return dpr > 1.5 ? 'low' : 'medium'
  }

  if (dpr < 1.5) return 'medium'
  if (dpr < 2) return 'high'
  return 'ultra'
}

/**
 * Canvas configuration for each tier
 */
export const CANVAS_CONFIGS: Record<PerformanceTier, {
  dpr: number | [number, number]
  antialias: boolean
  alpha: boolean
  stencil: boolean
  depth: boolean
}> = {
  low: {
    dpr: 1,
    antialias: false,
    alpha: false,
    stencil: false,
    depth: true,
  },
  medium: {
    dpr: 1,
    antialias: true,
    alpha: false,
    stencil: false,
    depth: true,
  },
  high: {
    dpr: [1, 1.5],
    antialias: true,
    alpha: false,
    stencil: true,
    depth: true,
  },
  ultra: {
    dpr: 2,
    antialias: true,
    alpha: false,
    stencil: true,
    depth: true,
  },
}
