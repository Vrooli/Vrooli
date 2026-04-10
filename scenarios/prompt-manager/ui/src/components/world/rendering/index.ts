/**
 * Rendering module exports
 */

export { RenderPipeline } from './RenderPipeline'
export { EnvironmentSetup } from './EnvironmentSetup'
export { ShadowSystem } from './ShadowSystem'
export { useShadowConfig } from '@/stores/graphicsStore'
export { DynamicLighting } from './DynamicLighting'
export { DynamicFog } from './DynamicFog'
export { useFogColor } from '@/stores/environmentStore'
export { DynamicSky, CelestialBody } from './DynamicSky'
export { Moon } from './Moon'
export { ProceduralClouds } from './ProceduralClouds'
export { GroundSurface } from './GroundSurface'

// Individual effects for custom compositions
export { BloomEffect } from './effects/BloomEffect'
export { VignetteEffect } from './effects/VignetteEffect'
export { SSAOEffect } from './effects/SSAOEffect'
