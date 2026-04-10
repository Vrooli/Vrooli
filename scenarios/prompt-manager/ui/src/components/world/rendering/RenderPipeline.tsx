/**
 * RenderPipeline - Post-processing effects wrapper for the 3D scene.
 * Manages EffectComposer and configurable visual effects.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#rendering-pipeline
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#performance-tiers

// NOTE: Imports commented out while post-processing is disabled
// import { EffectComposer, Bloom, Vignette, SMAA } from '@react-three/postprocessing'
// import { useGraphicsStore } from '@/stores/graphicsStore'
// import { BLOOM_CONFIGS, DEFAULT_VIGNETTE_CONFIG } from '@/config/graphics'
// import { WorldErrorBoundary } from '../WorldErrorBoundary'

interface RenderPipelineProps {
  children: React.ReactNode
}

/**
 * Wraps scene content with post-processing effects based on graphics settings.
 * Bypasses effects entirely when postProcessing is disabled.
 *
 * NOTE: Post-processing is temporarily disabled due to compatibility issues
 * with the current Three.js/R3F setup. Effects can be re-enabled once
 * the underlying issue is resolved.
 */
export function RenderPipeline({ children }: RenderPipelineProps) {
  // TEMPORARY: Disable all post-processing to isolate rendering issues
  // The EffectComposer was causing "Cannot read properties of undefined (reading 'length')"
  // errors from the postprocessing library internals
  return <>{children}</>

  // Original implementation commented out for reference:
  /*
  const config = useGraphicsStore((state) => state.config)
  const tier = useGraphicsStore((state) => state.tier)

  // Defensive: ensure config exists
  if (!config || typeof config.postProcessing !== 'boolean') {
    return <>{children}</>
  }

  // Skip post-processing entirely if disabled
  if (!config.postProcessing) {
    return <>{children}</>
  }

  // Defensive: ensure tier is valid
  const validTier = tier && tier in BLOOM_CONFIGS ? tier : 'medium'
  const bloomConfig = BLOOM_CONFIGS[validTier]
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
  */
}
