/**
 * useFPSMonitor - Hook for runtime FPS monitoring and dynamic tier adjustment.
 * Integrates with useFrame to track actual rendering performance.
 *
 * CRITICAL: Uses refs and getState() to minimize overhead in the render loop.
 * Only triggers React updates periodically for UI display.
 */
// AI_CHECK: FPS_TRACE_MONITOR_OVERHEAD=1 | LAST: 2026-02-17
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#performance-monitoring

import { useRef, useEffect, useCallback } from 'react'
import { useFrame } from '@react-three/fiber'
import { usePerformanceStore } from '@/stores/performanceStore'
import { useGraphicsStore } from '@/stores/graphicsStore'
import type { PerformanceMetrics, TierAdjustment } from '@/types/performance'
import type { PerformanceTier } from '@/types/graphics'

interface UseFPSMonitorOptions {
  /** Whether monitoring is enabled */
  enabled?: boolean
  /** Whether to automatically adjust graphics tier */
  autoAdjust?: boolean
  /** Callback when tier is adjusted */
  onTierAdjust?: (tier: PerformanceTier, reason: string) => void
}

interface UseFPSMonitorResult {
  /** Current metrics (reactive) */
  metrics: PerformanceMetrics
  /** Whether performance is degraded */
  isDegraded: boolean
  /** Whether FPS overlay is shown */
  showOverlay: boolean
  /** Toggle FPS overlay */
  toggleOverlay: () => void
  /** Toggle auto-adjust */
  toggleAutoAdjust: () => void
  /** Force a specific tier */
  setTier: (tier: PerformanceTier) => void
  /** Get current adjustment recommendation */
  getAdjustmentRecommendation: () => TierAdjustment
}

/**
 * Hook for monitoring FPS and automatically adjusting graphics tier.
 *
 * Must be used inside an R3F Canvas context.
 *
 * @example
 * ```tsx
 * function Scene() {
 *   const { metrics, isDegraded, toggleOverlay } = useFPSMonitor({
 *     enabled: true,
 *     autoAdjust: true,
 *     onTierAdjust: (tier, reason) => {
 *       console.log(`Adjusted to ${tier}: ${reason}`)
 *     }
 *   })
 *
 *   return (
 *     <>
 *       {isDegraded && <PerformanceWarning />}
 *       <SceneContent />
 *     </>
 *   )
 * }
 * ```
 */
export function useFPSMonitor(
  options: UseFPSMonitorOptions = {}
): UseFPSMonitorResult {
  const {
    enabled = true,
    autoAdjust = true,
    onTierAdjust,
  } = options

  // Store selectors (granular to avoid unnecessary re-renders)
  const metrics = usePerformanceStore((state) => state.metrics)
  const showOverlay = usePerformanceStore((state) => state.config.showOverlay)

  // Store actions via getState to avoid re-renders
  const lastFrameTimeRef = useRef<number>(performance.now())
  const adjustCheckCounterRef = useRef<number>(0)
  const isTabVisibleRef = useRef<boolean>(typeof document === 'undefined' ? true : !document.hidden)

  // Start/stop monitoring based on enabled prop
  useEffect(() => {
    if (enabled) {
      usePerformanceStore.getState().startMonitoring()
    } else {
      usePerformanceStore.getState().stopMonitoring()
    }

    return () => {
      usePerformanceStore.getState().stopMonitoring()
    }
  }, [enabled])

  // Sync auto-adjust preference
  useEffect(() => {
    usePerformanceStore.getState().setConfig({ autoAdjust })
  }, [autoAdjust])

  // Track tab visibility to avoid skewed traces when hidden
  useEffect(() => {
    if (typeof document === 'undefined') return

    const store = usePerformanceStore.getState()
    const initialVisible = !document.hidden
    isTabVisibleRef.current = initialVisible
    store.setTabVisibility(initialVisible)

    const handleVisibilityChange = () => {
      const isVisible = !document.hidden
      isTabVisibleRef.current = isVisible
      usePerformanceStore.getState().setTabVisibility(isVisible)
      if (isVisible) {
        lastFrameTimeRef.current = performance.now()
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [])

  // Sync current tier with performance store
  useEffect(() => {
    const tier = useGraphicsStore.getState().tier
    usePerformanceStore.getState().setCurrentTier(tier)

    // Subscribe to tier changes
    const unsubscribe = useGraphicsStore.subscribe((state) => {
      usePerformanceStore.getState().setCurrentTier(state.tier)
    })

    return unsubscribe
  }, [])

  // Record frame times in animation loop
  useFrame((state) => {
    if (!enabled) return
    if (!isTabVisibleRef.current) return

    const now = performance.now()
    const delta = now - lastFrameTimeRef.current
    lastFrameTimeRef.current = now

    // Record frame
    usePerformanceStore.getState().recordFrame(delta, {
      drawCalls: state.gl.info.render.calls,
      triangles: state.gl.info.render.triangles,
    })

    // Check for tier adjustment periodically (every 60 frames)
    adjustCheckCounterRef.current++
    if (adjustCheckCounterRef.current >= 60) {
      adjustCheckCounterRef.current = 0

      const perfState = usePerformanceStore.getState()
      if (perfState.config.autoAdjust) {
        const adjustment = perfState.getTierAdjustment()

        if (adjustment.shouldAdjust) {
          // Apply the adjustment
          useGraphicsStore.getState().setTier(adjustment.recommendedTier)
          perfState.recordAdjustment()

          // Notify callback
          onTierAdjust?.(adjustment.recommendedTier, adjustment.reason)
        }
      }
    }
  })

  const toggleOverlay = useCallback(() => {
    usePerformanceStore.getState().toggleOverlay()
  }, [])

  const toggleAutoAdjust = useCallback(() => {
    usePerformanceStore.getState().toggleAutoAdjust()
  }, [])

  const setTier = useCallback((tier: PerformanceTier) => {
    useGraphicsStore.getState().setTier(tier)
    usePerformanceStore.getState().setCurrentTier(tier)
  }, [])

  const getAdjustmentRecommendation = useCallback(() => {
    return usePerformanceStore.getState().getTierAdjustment()
  }, [])

  return {
    metrics,
    isDegraded: metrics.isDegraded,
    showOverlay,
    toggleOverlay,
    toggleAutoAdjust,
    setTier,
    getAdjustmentRecommendation,
  }
}

/**
 * Hook for just reading FPS metrics (no side effects).
 * Lighter weight than useFPSMonitor.
 */
export function useFPSMetrics(): PerformanceMetrics {
  return usePerformanceStore((state) => state.metrics)
}

/**
 * Hook for checking if performance is degraded.
 */
export function useIsDegraded(): boolean {
  return usePerformanceStore((state) => state.metrics.isDegraded)
}

/**
 * Hook for checking if performance is degraded.
 * Alias for useIsDegraded to keep external API stable.
 */
export function useIsPerformanceDegraded(): boolean {
  return useIsDegraded()
}

/**
 * Hook for getting current FPS.
 */
export function useCurrentFPS(): number {
  return usePerformanceStore((state) => state.metrics.currentFps)
}

/**
 * Hook for getting average FPS.
 */
export function useAverageFPS(): number {
  return usePerformanceStore((state) => state.metrics.averageFps)
}
