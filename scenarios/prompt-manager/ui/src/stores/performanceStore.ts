/**
 * Performance monitoring store for runtime FPS tracking and dynamic tier adjustment.
 * Enables automatic graphics tier adjustment based on actual device performance.
 */
// AI_CHECK: REACT_PERFORMANCE=1 | LAST: 2026-02-17
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#performance-monitoring

import { create } from 'zustand'
import type {
  FPSSample,
  PerformanceMetrics,
  FPSMonitorConfig,
  TierAdjustment,
  PerformanceTraceSample,
  PerformanceTraceMarker,
} from '@/types/performance'
import {
  DEFAULT_FPS_CONFIG,
  TIER_FPS_THRESHOLDS,
} from '@/types/performance'
import type { PerformanceTier } from '@/types/graphics'

interface PerformanceState {
  /** Current metrics snapshot */
  metrics: PerformanceMetrics
  /** FPS monitor configuration */
  config: FPSMonitorConfig
  /** FPS samples for averaging (ring buffer) */
  samples: FPSSample[]
  /** Index for ring buffer */
  sampleIndex: number
  /** Last tier adjustment timestamp */
  lastAdjustmentTime: number
  /** Whether monitoring is active */
  isMonitoring: boolean
  /** Current tier (synced from graphics store) */
  currentTier: PerformanceTier
  /** Ring buffer storage for trace samples */
  traceRing: PerformanceTraceSample[]
  /** Current write index for trace ring */
  traceRingIndex: number
  /** Number of valid samples in trace ring */
  traceRingCount: number
  /** Published trace samples (ordered oldest->newest) for overlay rendering */
  publishedTraceSamples: PerformanceTraceSample[]
  /** Published trace marker events */
  traceMarkers: PerformanceTraceMarker[]
  /** Version counter for throttled trace updates */
  traceVersion: number
  /** Last known degradation state for marker transition detection */
  lastDegradedState: boolean
  /** Last known tab visibility state */
  isTabVisible: boolean
}

interface PerformanceActions {
  /** Start monitoring */
  startMonitoring: () => void
  /** Stop monitoring */
  stopMonitoring: () => void
  /** Record a frame (called from useFrame) */
  recordFrame: (deltaMs: number, renderStats?: { drawCalls: number; triangles: number }) => void
  /** Update configuration */
  setConfig: (config: Partial<FPSMonitorConfig>) => void
  /** Toggle FPS overlay */
  toggleOverlay: () => void
  /** Toggle auto-adjust */
  toggleAutoAdjust: () => void
  /** Get tier adjustment recommendation */
  getTierAdjustment: () => TierAdjustment
  /** Mark that a tier adjustment was made */
  recordAdjustment: () => void
  /** Update current tier (called when graphics store changes) */
  setCurrentTier: (tier: PerformanceTier) => void
  /** Update visibility state for trace markers */
  setTabVisibility: (isVisible: boolean) => void
  /** Reset metrics */
  resetMetrics: () => void
  /** Get current metrics (for display) */
  getMetrics: () => PerformanceMetrics
}

type PerformanceStore = PerformanceState & PerformanceActions

const initialMetrics: PerformanceMetrics = {
  currentFps: 60,
  averageFps: 60,
  minFps: 60,
  maxFps: 60,
  frameTimeMs: 16.67,
  isDegraded: false,
  memoryUsageMb: null,
  gpuMemoryMb: null,
}

const initialState: PerformanceState = {
  metrics: initialMetrics,
  config: DEFAULT_FPS_CONFIG,
  samples: [],
  sampleIndex: 0,
  lastAdjustmentTime: 0,
  isMonitoring: false,
  currentTier: 'medium',
  traceRing: [],
  traceRingIndex: 0,
  traceRingCount: 0,
  publishedTraceSamples: [],
  traceMarkers: [],
  traceVersion: 0,
  lastDegradedState: false,
  isTabVisible: true,
}

const orderedTraceSamples = (
  ring: PerformanceTraceSample[],
  count: number,
  head: number,
): PerformanceTraceSample[] => {
  if (count === 0) return []
  if (count < ring.length) {
    return ring.slice(0, count)
  }

  const ordered: PerformanceTraceSample[] = []
  for (let i = 0; i < ring.length; i++) {
    const idx = (head + i) % ring.length
    const sample = ring[idx]
    if (sample) {
      ordered.push(sample)
    }
  }
  return ordered
}

/**
 * Zustand store for performance monitoring.
 *
 * IMPORTANT: recordFrame is called from useFrame - it uses direct mutation
 * and only triggers React updates periodically to avoid performance impact.
 */
export const usePerformanceStore = create<PerformanceStore>((set, get) => ({
  ...initialState,

  startMonitoring: () => {
    const now = performance.now()
    const traceRing = new Array(get().config.traceSampleSize).fill({
      timestamp: now,
      fps: 60,
      frameTimeMs: 16.67,
      drawCalls: null,
      triangles: null,
      memoryUsageMb: null,
    })

    set({
      isMonitoring: true,
      samples: new Array(get().config.sampleSize).fill({
        fps: 60,
        timestamp: now,
      }),
      sampleIndex: 0,
      traceRing,
      traceRingIndex: 0,
      traceRingCount: 0,
      publishedTraceSamples: [],
      traceMarkers: [],
      traceVersion: 0,
      lastDegradedState: false,
    })
  },

  stopMonitoring: () => {
    set({ isMonitoring: false })
  },

  recordFrame: (deltaMs, renderStats) => {
    const state = get()
    if (!state.isMonitoring) return

    // Calculate FPS from delta
    const fps = deltaMs > 0 ? 1000 / deltaMs : 60
    const now = performance.now()

    // Update FPS ring buffer (direct mutation for performance)
    const samples = state.samples
    const sampleIndex = state.sampleIndex
    samples[sampleIndex] = { fps, timestamp: now }

    // Update trace ring (direct mutation for performance)
    const traceRing = state.traceRing
    const traceWriteIndex = state.traceRingIndex
    traceRing[traceWriteIndex] = {
      timestamp: now,
      fps,
      frameTimeMs: deltaMs,
      drawCalls: renderStats?.drawCalls ?? null,
      triangles: renderStats?.triangles ?? null,
      memoryUsageMb: null,
    }
    const newTraceRingIndex = (traceWriteIndex + 1) % state.config.traceSampleSize
    const newTraceRingCount = Math.min(state.traceRingCount + 1, state.config.traceSampleSize)

    // Calculate new index
    const newIndex = (sampleIndex + 1) % state.config.sampleSize

    // Only update React state every ~10 frames to reduce overhead
    if (newIndex % state.config.tracePublishIntervalFrames === 0) {
      // Calculate metrics from samples
      let sum = 0
      let min = Infinity
      let max = -Infinity

      for (const sample of samples) {
        sum += sample.fps
        if (sample.fps < min) min = sample.fps
        if (sample.fps > max) max = sample.fps
      }

      const avgFps = sum / samples.length
      const isDegraded = avgFps < state.config.degradedThreshold

      // Get memory info if available
      let memoryUsageMb: number | null = null
      if (typeof performance !== 'undefined' && 'memory' in performance) {
        const memory = (performance as Performance & { memory?: { usedJSHeapSize: number } }).memory
        if (memory) {
          memoryUsageMb = Math.round(memory.usedJSHeapSize / 1024 / 1024)
        }
      }

      const markerEvents: PerformanceTraceMarker[] = []
      if (isDegraded !== state.lastDegradedState) {
        markerEvents.push({
          timestamp: now,
          type: isDegraded ? 'degraded' : 'recovered',
          label: isDegraded ? 'FPS degraded' : 'FPS recovered',
        })
      }

      const traceMarkers = markerEvents.length > 0
        ? [...state.traceMarkers, ...markerEvents].slice(-state.config.traceMarkerLimit)
        : state.traceMarkers

      // Backfill memory usage for latest sample
      if (newTraceRingCount > 0) {
        const latestIdx =
          (newTraceRingIndex - 1 + state.config.traceSampleSize) % state.config.traceSampleSize
        const latestSample = traceRing[latestIdx]
        if (latestSample) {
          latestSample.memoryUsageMb = memoryUsageMb
        }
      }

      set({
        sampleIndex: newIndex,
        traceRingIndex: newTraceRingIndex,
        traceRingCount: newTraceRingCount,
        publishedTraceSamples: orderedTraceSamples(traceRing, newTraceRingCount, newTraceRingIndex),
        traceMarkers,
        traceVersion: state.traceVersion + 1,
        lastDegradedState: isDegraded,
        metrics: {
          currentFps: Math.round(fps),
          averageFps: Math.round(avgFps),
          minFps: Math.round(min),
          maxFps: Math.round(max),
          frameTimeMs: deltaMs,
          isDegraded,
          memoryUsageMb,
          gpuMemoryMb: null, // Not easily accessible in browsers
        },
      })
    } else {
      // Avoid notifying subscribers every frame; sampleIndex is internal.
      state.sampleIndex = newIndex
      state.traceRingIndex = newTraceRingIndex
      state.traceRingCount = newTraceRingCount
    }
  },

  setConfig: (config) =>
    set((state) => {
      const nextConfig = { ...state.config, ...config }
      const needsTraceReset =
        nextConfig.traceSampleSize !== state.config.traceSampleSize

      if (!needsTraceReset) {
        return { config: nextConfig }
      }

      const now = performance.now()
      const traceRing = new Array(nextConfig.traceSampleSize).fill({
        timestamp: now,
        fps: 60,
        frameTimeMs: 16.67,
        drawCalls: null,
        triangles: null,
        memoryUsageMb: null,
      })

      return {
        config: nextConfig,
        traceRing,
        traceRingIndex: 0,
        traceRingCount: 0,
        publishedTraceSamples: [],
        traceVersion: state.traceVersion + 1,
      }
    }),

  toggleOverlay: () =>
    set((state) => ({
      config: { ...state.config, showOverlay: !state.config.showOverlay },
    })),

  toggleAutoAdjust: () =>
    set((state) => ({
      config: { ...state.config, autoAdjust: !state.config.autoAdjust },
    })),

  getTierAdjustment: () => {
    const state = get()
    const { metrics, config, currentTier, lastAdjustmentTime } = state

    const now = performance.now()
    const cooldownElapsed = now - lastAdjustmentTime > config.adjustmentCooldown

    // Default: no adjustment
    const noAdjustment: TierAdjustment = {
      shouldAdjust: false,
      direction: 'none',
      recommendedTier: currentTier,
      reason: 'Performance is acceptable',
    }

    if (!cooldownElapsed) {
      return {
        ...noAdjustment,
        reason: 'Waiting for adjustment cooldown',
      }
    }

    const tierOrder: PerformanceTier[] = ['low', 'medium', 'high', 'ultra']
    const currentIndex = tierOrder.indexOf(currentTier)

    // Check if we should decrease tier (poor performance)
    if (metrics.isDegraded && currentIndex > 0) {
      const lowerTier = tierOrder[currentIndex - 1] as PerformanceTier
      return {
        shouldAdjust: true,
        direction: 'decrease' as const,
        recommendedTier: lowerTier,
        reason: `FPS (${metrics.averageFps}) below threshold (${config.degradedThreshold})`,
      }
    }

    // Check if we should increase tier (good performance)
    if (
      metrics.averageFps > config.upgradeThreshold &&
      metrics.minFps > config.degradedThreshold &&
      currentIndex < tierOrder.length - 1
    ) {
      const higherTier = tierOrder[currentIndex + 1] as PerformanceTier
      const targetFps = TIER_FPS_THRESHOLDS[higherTier].min

      // Only upgrade if current FPS can handle higher tier
      if (metrics.averageFps > targetFps + 10) {
        return {
          shouldAdjust: true,
          direction: 'increase' as const,
          recommendedTier: higherTier,
          reason: `FPS (${metrics.averageFps}) exceeds upgrade threshold`,
        }
      }
    }

    return noAdjustment
  },

  recordAdjustment: () => {
    const now = performance.now()
    const marker: PerformanceTraceMarker = {
      timestamp: now,
      type: 'tier-adjust',
      label: `Tier -> ${get().currentTier}`,
    }
    set((state) => ({
      lastAdjustmentTime: now,
      traceMarkers: [...state.traceMarkers, marker].slice(-state.config.traceMarkerLimit),
      traceVersion: state.traceVersion + 1,
    }))
  },

  setCurrentTier: (tier) => {
    set({ currentTier: tier })
  },

  setTabVisibility: (isVisible) => {
    if (get().isTabVisible === isVisible) {
      return
    }
    set((state) => {
      const now = performance.now()
      const marker: PerformanceTraceMarker = {
        timestamp: now,
        type: isVisible ? 'visible' : 'hidden',
        label: isVisible ? 'Tab visible' : 'Tab hidden',
      }
      return {
        isTabVisible: isVisible,
        traceMarkers: [...state.traceMarkers, marker].slice(-state.config.traceMarkerLimit),
        traceVersion: state.traceVersion + 1,
      }
    })
  },

  resetMetrics: () => {
    set({
      metrics: initialMetrics,
      samples: [],
      sampleIndex: 0,
      traceRing: [],
      traceRingIndex: 0,
      traceRingCount: 0,
      publishedTraceSamples: [],
      traceMarkers: [],
      traceVersion: 0,
      lastDegradedState: false,
      isTabVisible: true,
    })
  },

  getMetrics: () => get().metrics,
}))

/**
 * Selector for metrics (safe for React components)
 */
export const selectMetrics = (state: PerformanceStore) => state.metrics

/**
 * Selector for config
 */
export const selectConfig = (state: PerformanceStore) => state.config

/**
 * Selector for overlay visibility
 */
export const selectShowOverlay = (state: PerformanceStore) => state.config.showOverlay

/**
 * Selector for trace overlay data.
 */
export const selectTraceData = (state: PerformanceStore) => ({
  samples: state.publishedTraceSamples,
  markers: state.traceMarkers,
  version: state.traceVersion,
})
