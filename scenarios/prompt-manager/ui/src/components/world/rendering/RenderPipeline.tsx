/**
 * RenderPipeline - Post-processing effects wrapper for the 3D scene.
 * Manages EffectComposer and configurable visual effects.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#rendering-pipeline
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#performance-tiers

import { EffectComposer, Bloom, Vignette, SMAA } from '@react-three/postprocessing'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { BLOOM_CONFIGS, DEFAULT_VIGNETTE_CONFIG } from '@/config/graphics'
import { WorldErrorBoundary } from '../WorldErrorBoundary'

interface RenderPipelineProps {
  children: React.ReactNode
}

/**
 * Wraps scene content with post-processing effects based on graphics settings.
 * Bypasses effects entirely when postProcessing is disabled.
 */
export function RenderPipeline({ children }: RenderPipelineProps) {
  const config = useGraphicsStore((state) => state.config)
  const tier = useGraphicsStore((state) => state.tier)

  // Skip post-processing entirely if disabled
  if (!config.postProcessing) {
    return <>{children}</>
  }

  const bloomConfig = BLOOM_CONFIGS[tier]
  const vignetteConfig = DEFAULT_VIGNETTE_CONFIG

  // Build effects array based on config
  const effects: React.ReactElement[] = []

  if (config.bloom && bloomConfig) {
    effects.push(
      <Bloom
        key="bloom"
        luminanceThreshold={bloomConfig.luminanceThreshold}
        luminanceSmoothing={bloomConfig.luminanceSmoothing}
        intensity={bloomConfig.intensity}
        mipmapBlur
      />
    )
  }

  if (config.vignette) {
    effects.push(
      <Vignette
        key="vignette"
        eskil={false}
        offset={vignetteConfig.offset}
        darkness={vignetteConfig.darkness}
      />
    )
  }

  if (config.antialiasing === 'smaa') {
    effects.push(<SMAA key="smaa" />)
  }

  // If no effects, just render children
  if (effects.length === 0) {
    return <>{children}</>
  }

  return (
    <>
      {children}
      <WorldErrorBoundary componentName="PostProcessing" minimal>
        <EffectComposer>{effects}</EffectComposer>
      </WorldErrorBoundary>
    </>
  )
}
