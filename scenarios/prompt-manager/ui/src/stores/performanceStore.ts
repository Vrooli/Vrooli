/**
 * Performance monitoring store for runtime FPS tracking and dynamic tier adjustment.
 * Enables automatic graphics tier adjustment based on actual device performance.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#performance-monitoring

import { create } from 'zustand'
import type {
  FPSSample,
  PerformanceMetrics,
  FPSMonitorConfig,
  TierAdjustment,
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
}

interface PerformanceActions {
  /** Start monitoring */
  startMonitoring: () => void
  /** Stop monitoring */
  stopMonitoring: () => void
  /** Record a frame (called from useFrame) */
  recordFrame: (deltaMs: number) => void
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
    set({
      isMonitoring: true,
      samples: new Array(get().config.sampleSize).fill({
        fps: 60,
        timestamp: performance.now(),
      }),
      sampleIndex: 0,
    })
  },

  stopMonitoring: () => {
    set({ isMonitoring: false })
  },

  recordFrame: (deltaMs) => {
    const state = get()
    if (!state.isMonitoring) return

    // Calculate FPS from delta
    const fps = deltaMs > 0 ? 1000 / deltaMs : 60
    const now = performance.now()

    // Update ring buffer (direct mutation for performance)
    const samples = state.samples
    const sampleIndex = state.sampleIndex
    samples[sampleIndex] = { fps, timestamp: now }

    // Calculate new index
    const newIndex = (sampleIndex + 1) % state.config.sampleSize

    // Only update React state every ~10 frames to reduce overhead
    if (newIndex % 10 === 0) {
      // Calculate metrics from samples
      let sum = 0
      let min = Infinity
      let max = -Infinity

      for (const sample of samples) {
        if (sample && typeof sample.fps === 'number') {
          sum += sample.fps
          if (sample.fps < min) min = sample.fps
          if (sample.fps > max) max = sample.fps
        }
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

      set({
        sampleIndex: newIndex,
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
      // Just update index without React update
      state.sampleIndex = newIndex
    }
  },

  setConfig: (config) =>
    set((state) => ({
      config: { ...state.config, ...config },
    })),

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
    set({ lastAdjustmentTime: performance.now() })
  },

  setCurrentTier: (tier) => {
    set({ currentTier: tier })
  },

  resetMetrics: () => {
    set({
      metrics: initialMetrics,
      samples: [],
      sampleIndex: 0,
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
