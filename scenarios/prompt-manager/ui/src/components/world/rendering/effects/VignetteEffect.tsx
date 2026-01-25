/**
 * VignetteEffect - Standalone vignette effect component.
 * Darkens the edges of the screen for cinematic framing.
 */

import { Vignette } from '@react-three/postprocessing'
import type { VignetteConfig } from '@/types/graphics'
import { DEFAULT_VIGNETTE_CONFIG } from '@/config/graphics'

interface VignetteEffectProps extends Partial<VignetteConfig> {
  /** Enable/disable the effect */
  enabled?: boolean
  /** Use Eskil's vignette technique */
  eskil?: boolean
}

/**
 * Configurable vignette effect for post-processing.
 */
export function VignetteEffect({
  enabled = true,
  offset = DEFAULT_VIGNETTE_CONFIG.offset,
  darkness = DEFAULT_VIGNETTE_CONFIG.darkness,
  eskil = false,
}: VignetteEffectProps) {
  if (!enabled) return null

  return <Vignette eskil={eskil} offset={offset} darkness={darkness} />
}
