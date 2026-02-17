/**
 * Tests for Performance monitoring store.
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { usePerformanceStore } from '@/stores/performanceStore'

describe('Performance Store', () => {
  beforeEach(() => {
    // Reset store state completely before each test
    usePerformanceStore.getState().resetMetrics()
    usePerformanceStore.getState().stopMonitoring()
    // Reset adjustment cooldown
    usePerformanceStore.setState({
      lastAdjustmentTime: 0,
      currentTier: 'medium',
    })
  })

  afterEach(() => {
    usePerformanceStore.getState().stopMonitoring()
  })

  describe('monitoring lifecycle', () => {
    it('starts monitoring', () => {
      usePerformanceStore.getState().startMonitoring()
      expect(usePerformanceStore.getState().isMonitoring).toBe(true)
    })

    it('stops monitoring', () => {
      usePerformanceStore.getState().startMonitoring()
      usePerformanceStore.getState().stopMonitoring()
      expect(usePerformanceStore.getState().isMonitoring).toBe(false)
    })
  })

  describe('recordFrame', () => {
    it('ignores frames when not monitoring', () => {
      const initialMetrics = usePerformanceStore.getState().metrics

      usePerformanceStore.getState().recordFrame(16.67) // 60 FPS

      expect(usePerformanceStore.getState().metrics).toEqual(initialMetrics)
    })

    it('records frame times when monitoring', () => {
      usePerformanceStore.getState().startMonitoring()

      // Record 10 frames to trigger metrics update
      for (let i = 0; i < 10; i++) {
        usePerformanceStore.getState().recordFrame(16.67) // 60 FPS
      }

      const metrics = usePerformanceStore.getState().metrics
      expect(metrics.currentFps).toBeGreaterThan(0)
      expect(metrics.averageFps).toBeGreaterThan(0)
    })

    it('publishes trace samples with render stats on interval', () => {
      usePerformanceStore.getState().setConfig({
        traceSampleSize: 16,
        tracePublishIntervalFrames: 4,
      })
      usePerformanceStore.getState().startMonitoring()

      for (let i = 0; i < 4; i++) {
        usePerformanceStore.getState().recordFrame(16.67, {
          drawCalls: 100 + i,
          triangles: 200 + i,
        })
      }

      const state = usePerformanceStore.getState()
      expect(state.publishedTraceSamples.length).toBe(4)
      expect(state.traceVersion).toBeGreaterThan(0)
      expect(state.publishedTraceSamples[3]?.drawCalls).toBe(103)
      expect(state.publishedTraceSamples[3]?.triangles).toBe(203)
    })

    it('keeps only latest trace samples up to traceSampleSize', () => {
      usePerformanceStore.getState().setConfig({
        traceSampleSize: 5,
        tracePublishIntervalFrames: 5,
      })
      usePerformanceStore.getState().startMonitoring()

      for (let i = 0; i < 10; i++) {
        usePerformanceStore.getState().recordFrame(16.67)
      }

      const state = usePerformanceStore.getState()
      expect(state.publishedTraceSamples).toHaveLength(5)
      const firstTs = state.publishedTraceSamples[0]?.timestamp ?? 0
      const lastTs = state.publishedTraceSamples[4]?.timestamp ?? 0
      expect(lastTs).toBeGreaterThanOrEqual(firstTs)
    })
  })

  describe('config', () => {
    it('allows updating config', () => {
      usePerformanceStore.getState().setConfig({ targetFps: 30 })
      expect(usePerformanceStore.getState().config.targetFps).toBe(30)
    })

    it('toggles overlay', () => {
      const initialValue = usePerformanceStore.getState().config.showOverlay
      usePerformanceStore.getState().toggleOverlay()
      expect(usePerformanceStore.getState().config.showOverlay).toBe(!initialValue)
    })

    it('toggles auto adjust', () => {
      const initialValue = usePerformanceStore.getState().config.autoAdjust
      usePerformanceStore.getState().toggleAutoAdjust()
      expect(usePerformanceStore.getState().config.autoAdjust).toBe(!initialValue)
    })
  })

  describe('tier adjustment', () => {
    it('returns no adjustment when performance is acceptable', () => {
      usePerformanceStore.getState().setCurrentTier('medium')
      usePerformanceStore.getState().startMonitoring()

      // Simulate good FPS
      for (let i = 0; i < 60; i++) {
        usePerformanceStore.getState().recordFrame(16.67) // ~60 FPS
      }

      const adjustment = usePerformanceStore.getState().getTierAdjustment()
      expect(adjustment.direction).toBe('none')
    })

    it('recommends decreasing tier when FPS is low', () => {
      // Set up state for tier adjustment test
      // Set lastAdjustmentTime far in the past to ensure cooldown has passed
      const pastTime = performance.now() - 10000 // 10 seconds ago

      usePerformanceStore.setState({
        currentTier: 'high',
        isMonitoring: true,
        lastAdjustmentTime: pastTime,
        metrics: {
          currentFps: 30,
          averageFps: 30,
          minFps: 25,
          maxFps: 35,
          frameTimeMs: 33.33,
          isDegraded: true, // FPS is below threshold
          memoryUsageMb: null,
          gpuMemoryMb: null,
        },
      })

      const adjustment = usePerformanceStore.getState().getTierAdjustment()
      expect(adjustment.direction).toBe('decrease')
      expect(adjustment.recommendedTier).toBe('medium')
    })
  })

  describe('tier tracking', () => {
    it('tracks current tier', () => {
      usePerformanceStore.getState().setCurrentTier('high')
      expect(usePerformanceStore.getState().currentTier).toBe('high')

      usePerformanceStore.getState().setCurrentTier('low')
      expect(usePerformanceStore.getState().currentTier).toBe('low')
    })

    it('records adjustment time', () => {
      const before = usePerformanceStore.getState().lastAdjustmentTime

      usePerformanceStore.getState().recordAdjustment()

      const after = usePerformanceStore.getState().lastAdjustmentTime
      expect(after).toBeGreaterThan(before)
    })

    it('adds trace markers for tab visibility changes', () => {
      const before = usePerformanceStore.getState().traceMarkers.length
      usePerformanceStore.getState().setTabVisibility(false)
      usePerformanceStore.getState().setTabVisibility(true)
      const markers = usePerformanceStore.getState().traceMarkers
      expect(markers.length).toBe(before + 2)
      expect(markers[markers.length - 2]?.type).toBe('hidden')
      expect(markers[markers.length - 1]?.type).toBe('visible')
    })
  })
})
