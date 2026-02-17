/**
 * Graphics settings store for managing render quality.
 * Supports performance tiers and individual setting overrides.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#performance-tiers
// DOC: docs/internal/SEAMS.md#4-graphics-tier-system

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { PerformanceTier, GraphicsConfig } from '@/types/graphics'

/**
 * Preset configurations for each performance tier
 */
const TIER_CONFIGS: Record<PerformanceTier, GraphicsConfig> = {
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

function applyTierPolicy(tier: PerformanceTier, config: GraphicsConfig): GraphicsConfig {
  if (tier !== 'low') return config

  // Enforce safe low-tier defaults regardless of manual overrides.
  return {
    ...config,
    shadows: false,
    shadowMapSize: 512,
    contactShadows: false,
    materialQuality: 'basic',
    antialiasing: 'none',
    postProcessing: false,
    bloom: false,
    ssao: false,
    vignette: false,
    envMap: false,
  }
}

interface GraphicsState {
  /** Current performance tier */
  tier: PerformanceTier
  /** Resolved graphics configuration */
  config: GraphicsConfig
  /** Whether auto-detection is enabled */
  autoDetect: boolean
  /** Individual setting overrides */
  overrides: Partial<GraphicsConfig>
}

interface GraphicsActions {
  /** Set performance tier and update config */
  setTier: (tier: PerformanceTier) => void
  /** Toggle auto-detection */
  setAutoDetect: (enabled: boolean) => void
  /** Override a specific setting */
  setOverride: <K extends keyof GraphicsConfig>(key: K, value: GraphicsConfig[K]) => void
  /** Clear all overrides */
  clearOverrides: () => void
  /** Get effective config (tier + overrides) */
  getEffectiveConfig: () => GraphicsConfig
}

type GraphicsStore = GraphicsState & GraphicsActions

/**
 * Zustand store for graphics settings with persistence
 */
export const useGraphicsStore = create<GraphicsStore>()(
  persist(
    (set, get) => ({
      // Initial state
      tier: 'medium',
      config: TIER_CONFIGS.medium,
      autoDetect: true,
      overrides: {},

      // Actions
      setTier: (tier) =>
        set({
          tier,
          config: applyTierPolicy(tier, { ...TIER_CONFIGS[tier], ...get().overrides }),
        }),

      setAutoDetect: (enabled) =>
        set({ autoDetect: enabled }),

      setOverride: (key, value) => {
        const newOverrides = { ...get().overrides, [key]: value }
        const tier = get().tier
        set({
          overrides: newOverrides,
          config: applyTierPolicy(tier, { ...TIER_CONFIGS[tier], ...newOverrides }),
        })
      },

      clearOverrides: () =>
        {
          const tier = get().tier
          return set({
            overrides: {},
            config: applyTierPolicy(tier, TIER_CONFIGS[tier]),
          })
        },

      getEffectiveConfig: () => ({
        ...applyTierPolicy(get().tier, {
          ...TIER_CONFIGS[get().tier],
          ...get().overrides,
        }),
      }),
    }),
    {
      name: 'graphics-settings',
      partialize: (state) => ({
        tier: state.tier,
        autoDetect: state.autoDetect,
        overrides: state.overrides,
      }),
    }
  )
)

/**
 * Export tier configs for external use
 */
export { TIER_CONFIGS }

/**
 * Hook to get shadow configuration for lights.
 */
export function useShadowConfig() {
  const config = useGraphicsStore((state) => state.config)

  return {
    enabled: config.shadows,
    mapSize: config.shadowMapSize,
    bias: -0.0001,
    normalBias: 0.02,
  }
}
