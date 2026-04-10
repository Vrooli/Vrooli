/**
 * BloomEffect - Standalone bloom effect component.
 * Use when you need more control than RenderPipeline provides.
 */

import { Bloom } from '@react-three/postprocessing'
import type { BloomConfig } from '@/types/graphics'
import { DEFAULT_BLOOM_CONFIG } from '@/config/graphics'

interface BloomEffectProps extends Partial<BloomConfig> {
  /** Enable/disable the effect */
  enabled?: boolean
}

/**
 * Configurable bloom effect for post-processing.
 * Merges provided props with defaults.
 */
export function BloomEffect({
  enabled = true,
  luminanceThreshold = DEFAULT_BLOOM_CONFIG.luminanceThreshold,
  luminanceSmoothing = DEFAULT_BLOOM_CONFIG.luminanceSmoothing,
  intensity = DEFAULT_BLOOM_CONFIG.intensity,
}: BloomEffectProps) {
  if (!enabled) return null

  return (
    <Bloom
      luminanceThreshold={luminanceThreshold}
      luminanceSmoothing={luminanceSmoothing}
      intensity={intensity}
      mipmapBlur
    />
  )
}
