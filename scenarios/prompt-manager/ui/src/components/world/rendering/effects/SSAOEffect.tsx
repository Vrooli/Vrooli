/**
 * SSAOEffect - Screen Space Ambient Occlusion effect.
 * Adds depth and realism by darkening crevices and contact areas.
 *
 * Note: SSAO is computationally expensive and should only be enabled
 * on high-end devices (high/ultra tiers).
 */

import { SSAO } from '@react-three/postprocessing'
import type { SSAOConfig } from '@/types/graphics'
import { DEFAULT_SSAO_CONFIG } from '@/config/graphics'

// BlendFunction values from postprocessing library
const BlendFunction = {
  MULTIPLY: 4,
} as const

interface SSAOEffectProps extends Partial<SSAOConfig> {
  /** Enable/disable the effect */
  enabled?: boolean
  /** Blend mode for the effect */
  blendFunction?: number
}

/**
 * Configurable SSAO effect for post-processing.
 * Adds subtle ambient occlusion to the scene.
 */
export function SSAOEffect({
  enabled = true,
  radius = DEFAULT_SSAO_CONFIG.radius,
  intensity = DEFAULT_SSAO_CONFIG.intensity,
  samples = DEFAULT_SSAO_CONFIG.samples,
  blendFunction = BlendFunction.MULTIPLY,
}: SSAOEffectProps) {
  if (!enabled) return null

  return (
    <SSAO
      blendFunction={blendFunction}
      samples={samples}
      radius={radius}
      intensity={intensity}
    />
  )
}
