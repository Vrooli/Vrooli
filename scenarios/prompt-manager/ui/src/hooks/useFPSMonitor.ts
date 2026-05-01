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
import { useSelectionStore } from '@/stores/selectionStore'
import { useCameraStore } from '@/stores/cameraStore'
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
  const bookkeepingSampleCounterRef = useRef<number>(0)
  const sceneStatsCounterRef = useRef<number>(0)
  const glInfoInitializedRef = useRef<boolean>(false)

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
    const initialFocused = typeof document.hasFocus === 'function' ? document.hasFocus() : true
    isTabVisibleRef.current = initialVisible
    store.setTabVisibility(initialVisible)
    store.setWindowFocus(initialFocused)
    store.setSceneSnapshot({
      documentHidden: !initialVisible,
      windowFocused: initialFocused,
      lastVisibilityEvent: initialVisible ? 'visible' : 'hidden',
      lastFocusEvent: initialFocused ? 'focus' : 'blur',
    })

    const handleVisibilityChange = () => {
      const isVisible = !document.hidden
      isTabVisibleRef.current = isVisible
      usePerformanceStore.getState().setTabVisibility(isVisible)
      if (isVisible) {
        lastFrameTimeRef.current = performance.now()
      }
    }

    const handleFocus = () => {
      usePerformanceStore.getState().setWindowFocus(true)
    }

    const handleBlur = () => {
      usePerformanceStore.getState().setWindowFocus(false)
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    window.addEventListener('focus', handleFocus)
    window.addEventListener('blur', handleBlur)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      window.removeEventListener('focus', handleFocus)
      window.removeEventListener('blur', handleBlur)
    }
  }, [])

  // Track semantic selection changes for trace correlation.
  useEffect(() => {
    let previousSignature = (() => {
      const s = useSelectionStore.getState()
      return JSON.stringify({
        skills: s.selectedSkillIds,
      })
    })()

    const unsubscribe = useSelectionStore.subscribe((state) => {
      const nextSignature = JSON.stringify({
        skills: state.selectedSkillIds,
      })
      if (nextSignature !== previousSignature) {
        previousSignature = nextSignature
        const summary = `Selection: skills=${state.selectedSkillIds.length}`
        usePerformanceStore.getState().recordTraceMarker('selection-change', summary)
      }
    })

    return unsubscribe
  }, [])

  // Track camera mode transitions for trace correlation.
  useEffect(() => {
    let previousMode = useCameraStore.getState().mode
    const unsubscribe = useCameraStore.subscribe((state) => {
      if (state.mode === previousMode) return
      previousMode = state.mode
      usePerformanceStore
        .getState()
        .recordTraceMarker('camera-mode-change', `Camera mode: ${state.mode}`)
    })

    return unsubscribe
  }, [])

  // Event-loop lag sampler (captures timer drift that can indicate scheduler throttling).
  useEffect(() => {
    if (typeof window === 'undefined') return

    let active = true
    const intervalMs = 1000
    let expected = performance.now() + intervalMs
    const timer = window.setInterval(() => {
      if (!active) return
      const now = performance.now()
      const drift = Math.max(0, now - expected)
      expected = now + intervalMs
      usePerformanceStore.getState().setSceneSnapshot({ eventLoopLagMs: Math.round(drift * 10) / 10 })
    }, intervalMs)

    return () => {
      active = false
      window.clearInterval(timer)
    }
  }, [])

  // Track long tasks (main-thread blocking) when supported.
  useEffect(() => {
    if (typeof window === 'undefined' || typeof PerformanceObserver === 'undefined') return

    let observer: PerformanceObserver | null = null
    try {
      observer = new PerformanceObserver((list) => {
        const entries = list.getEntries()
        for (const entry of entries) {
          usePerformanceStore.getState().recordLongTask(entry.duration)
        }
      })
      observer.observe({ entryTypes: ['longtask'] })
    } catch {
      // Long task API not available in this browser/runtime.
    }

    return () => {
      observer?.disconnect()
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

    const frameCallbackStart = performance.now()
    const bookkeepingStart = frameCallbackStart
    const now = performance.now()
    const delta = now - lastFrameTimeRef.current
    lastFrameTimeRef.current = now

    if (!glInfoInitializedRef.current) {
      state.gl.info.autoReset = false
      glInfoInitializedRef.current = true
    }
    const renderInfo = state.gl.info.render
    const memoryInfo = state.gl.info.memory
    const glInfo = state.gl.info as { programs?: unknown[] }
    const programs = Array.isArray(glInfo.programs) ? glInfo.programs.length : undefined
    const renderCalls = renderInfo.calls
    const renderTriangles = renderInfo.triangles
    const renderPoints = renderInfo.points
    const renderLines = renderInfo.lines

    // Record frame
    usePerformanceStore.getState().recordFrame(delta, {
      drawCalls: renderCalls,
      triangles: renderTriangles,
      points: renderPoints,
      lines: renderLines,
      geometries: memoryInfo.geometries,
      textures: memoryInfo.textures,
      programs,
    })
    state.gl.info.reset()

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

    bookkeepingSampleCounterRef.current++
    if (bookkeepingSampleCounterRef.current % 30 === 0) {
      usePerformanceStore.getState().recordSubsystemSample(
        'render.bookkeeping',
        performance.now() - bookkeepingStart
      )
    }

    sceneStatsCounterRef.current++
    if (sceneStatsCounterRef.current % 30 === 0) {
      let sceneObjects = 0
      let sceneMeshes = 0
      state.scene.traverse((obj) => {
        sceneObjects++
        const maybeMesh = obj as { isMesh?: boolean }
        if (maybeMesh.isMesh) {
          sceneMeshes++
        }
      })
      usePerformanceStore.getState().setSceneSnapshot({ sceneObjects, sceneMeshes })
    }

    usePerformanceStore.getState().recordFrameLoopAggregate(
      performance.now() - frameCallbackStart,
      1
    )
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
