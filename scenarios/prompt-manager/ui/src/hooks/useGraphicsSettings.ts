/**
 * useGraphicsSettings - Hook for accessing and modifying graphics settings.
 * Provides convenient access to the graphics store.
 */

import { useCallback, useEffect } from 'react'
import { useGraphicsStore, TIER_CONFIGS } from '@/stores/graphicsStore'
import { detectRecommendedTier } from '@/config/graphics'
import type { PerformanceTier, GraphicsConfig } from '@/types/graphics'

interface GraphicsSettingsResult {
  /** Current performance tier */
  tier: PerformanceTier
  /** Current graphics configuration */
  config: GraphicsConfig
  /** Whether auto-detection is enabled */
  autoDetect: boolean
  /** Set performance tier */
  setTier: (tier: PerformanceTier) => void
  /** Toggle auto-detection */
  setAutoDetect: (enabled: boolean) => void
  /** Override a specific setting */
  setOverride: <K extends keyof GraphicsConfig>(key: K, value: GraphicsConfig[K]) => void
  /** Clear all overrides */
  clearOverrides: () => void
  /** Detect and set recommended tier */
  detectAndSetTier: () => PerformanceTier
}

/**
 * Hook for managing graphics settings.
 *
 * @example
 * ```tsx
 * function SettingsPanel() {
 *   const { tier, setTier, config } = useGraphicsSettings()
 *
 *   return (
 *     <select value={tier} onChange={(e) => setTier(e.target.value as PerformanceTier)}>
 *       <option value="low">Low</option>
 *       <option value="medium">Medium</option>
 *       <option value="high">High</option>
 *       <option value="ultra">Ultra</option>
 *     </select>
 *   )
 * }
 * ```
 */
export function useGraphicsSettings(): GraphicsSettingsResult {
  const tier = useGraphicsStore((state) => state.tier)
  const config = useGraphicsStore((state) => state.config)
  const autoDetect = useGraphicsStore((state) => state.autoDetect)
  const setTier = useGraphicsStore((state) => state.setTier)
  const setAutoDetect = useGraphicsStore((state) => state.setAutoDetect)
  const setOverride = useGraphicsStore((state) => state.setOverride)
  const clearOverrides = useGraphicsStore((state) => state.clearOverrides)

  const detectAndSetTier = useCallback(() => {
    const recommended = detectRecommendedTier()
    setTier(recommended)
    return recommended
  }, [setTier])

  return {
    tier,
    config,
    autoDetect,
    setTier,
    setAutoDetect,
    setOverride,
    clearOverrides,
    detectAndSetTier,
  }
}

/**
 * Hook that auto-detects and sets the recommended graphics tier on mount.
 * Only runs once when autoDetect is enabled.
 */
export function useAutoDetectGraphics(): void {
  const autoDetect = useGraphicsStore((state) => state.autoDetect)
  const setTier = useGraphicsStore((state) => state.setTier)

  useEffect(() => {
    if (autoDetect) {
      const recommended = detectRecommendedTier()
      setTier(recommended)
    }
  }, []) // Run only on mount
}

/**
 * Hook to get tier configuration directly.
 */
export function useTierConfig(tier?: PerformanceTier): GraphicsConfig {
  const currentTier = useGraphicsStore((state) => state.tier)
  return TIER_CONFIGS[tier ?? currentTier]
}

/**
 * Hook to check if a specific feature is enabled.
 */
export function useGraphicsFeature(feature: keyof GraphicsConfig): boolean {
  const config = useGraphicsStore((state) => state.config)
  const value = config[feature]
  return typeof value === 'boolean' ? value : !!value
}

/**
 * Available tier names for UI rendering
 */
export const TIER_OPTIONS: { value: PerformanceTier; label: string }[] = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
  { value: 'ultra', label: 'Ultra' },
]
