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
  PerformanceSubsystemTiming,
  SceneComplexitySnapshot,
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
  /** Rolling long-task count since last metrics publish */
  longTaskWindowCount: number
  /** Rolling long-task blocked ms since last metrics publish */
  longTaskWindowBlockedMs: number
  /** Worst long-task duration in current window */
  longTaskWindowWorstMs: number
  /** Subsystem timing buckets since last metrics publish */
  subsystemTimingBuckets: Record<string, { totalMs: number; maxMs: number; samples: number }>
  /** Scene complexity snapshot for perf correlation */
  sceneSnapshot: SceneComplexitySnapshot
  /** Number of invalidate() calls since last publish */
  invalidateWindowCount: number
  /** Number of demand render requests since last publish */
  renderRequestWindowCount: number
  /** Demand render request reason buckets since last publish */
  renderRequestReasonBuckets: Record<string, number>
  /** Last named demand render reason */
  lastRenderReason: string
  /** Start timestamp for invalidate-rate window */
  invalidateWindowStart: number
  /** Total sampled useFrame callback time accumulated in this publish window */
  frameLoopWindowTotalMs: number
  /** Total sampled useFrame callback count accumulated in this publish window */
  frameLoopWindowCallbackCount: number
  /** Rendered frame count in this publish window */
  frameWindowCount: number
  /** Start timestamp for generic diagnostics windowing */
  diagnosticsWindowStart: number
  /** Pointer-move event count in diagnostics window */
  pointerMoveWindowCount: number
  /** Interaction store write count in diagnostics window */
  interactionStoreWriteCount: number
  /** Raycast sample count in diagnostics window */
  raycastWindowCount: number
  /** Raycast total time in diagnostics window */
  raycastWindowTotalMs: number
  /** Raycast max time in diagnostics window */
  raycastWindowMaxMs: number
}

interface PerformanceActions {
  /** Start monitoring */
  startMonitoring: () => void
  /** Stop monitoring */
  stopMonitoring: () => void
  /** Record a frame (called from useFrame) */
  recordFrame: (deltaMs: number, renderStats?: {
    drawCalls: number
    triangles: number
    points?: number
    lines?: number
    geometries?: number
    textures?: number
    programs?: number
  }) => void
  /** Record a long-task entry duration (ms) */
  recordLongTask: (durationMs: number) => void
  /** Record a subsystem timing sample (ms) */
  recordSubsystemSample: (name: string, durationMs: number) => void
  /** Update scene complexity snapshot */
  setSceneSnapshot: (snapshot: Partial<SceneComplexitySnapshot>) => void
  /** Record invalidate() usage from demand frameloop */
  recordInvalidate: () => void
  /** Record a named demand-render request */
  recordRenderRequest: (reason: string) => void
  /** Record sampled useFrame aggregate for a component */
  recordFrameLoopAggregate: (totalMs: number, callbackCount: number) => void
  /** Record pointer-move interaction event */
  recordPointerMoveEvent: () => void
  /** Record raycast timing sample (ms) */
  recordRaycastSample: (durationMs: number) => void
  /** Record interaction-store write */
  recordInteractionStoreWrite: () => void
  /** Record a timeline marker event */
  recordTraceMarker: (type: PerformanceTraceMarker['type'], label: string) => void
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
  /** Update focus state for trace markers */
  setWindowFocus: (isFocused: boolean) => void
  /** Reset metrics */
  resetMetrics: () => void
  /** Get current metrics (for display) */
  getMetrics: () => PerformanceMetrics
}

type PerformanceStore = PerformanceState & PerformanceActions

const getDefaultMaxFpsForTier = (tier: PerformanceTier): number =>
  tier === 'low' || tier === 'medium' ? 30 : 60

const initialMetrics: PerformanceMetrics = {
  currentFps: 60,
  averageFps: 60,
  minFps: 60,
  maxFps: 60,
  frameTimeMs: 16.67,
  frameP50Ms: 16.67,
  frameP95Ms: 16.67,
  frameP99Ms: 16.67,
  overBudget16Pct: 0,
  overBudget33Pct: 0,
  isDegraded: false,
  memoryUsageMb: null,
  gpuMemoryMb: null,
  workload: {
    drawCalls: null,
    drawCallsAvg: null,
    triangles: null,
    trianglesAvg: null,
    points: null,
    pointsAvg: null,
    lines: null,
    linesAvg: null,
    geometries: null,
    textures: null,
    programs: null,
  },
  longTasks: {
    count: 0,
    blockedMs: 0,
    worstMs: 0,
  },
  subsystemTimings: [],
  useFrameTotalMs: 0,
  useFrameCallbacksPerFrame: 0,
  renderedFramesPerSecond: 0,
  pointerMoveRateHz: 0,
  interactionStoreWritesPerSec: 0,
  raycastAvgMs: 0,
  raycastMaxMs: 0,
  interactionMsPerFrame: 0,
  unaccountedFrameMs: 0,
}

const initialSceneSnapshot: SceneComplexitySnapshot = {
  agents: 0,
  mountedAgents: 0,
  furniture: 0,
  decorations: 0,
  stars: 0,
  sceneObjects: 0,
  sceneMeshes: 0,
  selectedNodes: 0,
  sceneType: 'unknown',
  tier: 'medium',
  dpr: '1',
  shadows: true,
  materialQuality: 'standard',
  maxFps: 'auto',
  effectiveMaxFps: 30,
  frameloopMode: 'always',
  forceAlwaysFrameloop: false,
  invalidateRateHz: 0,
  renderRequestRateHz: 0,
  lastRenderReason: 'initial',
  topRenderReasons: '',
  documentHidden: false,
  windowFocused: true,
  eventLoopLagMs: 0,
  lastVisibilityEvent: 'visible',
  lastFocusEvent: 'focus',
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
  longTaskWindowCount: 0,
  longTaskWindowBlockedMs: 0,
  longTaskWindowWorstMs: 0,
  subsystemTimingBuckets: {},
  sceneSnapshot: initialSceneSnapshot,
  invalidateWindowCount: 0,
  renderRequestWindowCount: 0,
  renderRequestReasonBuckets: {},
  lastRenderReason: 'initial',
  invalidateWindowStart: 0,
  frameLoopWindowTotalMs: 0,
  frameLoopWindowCallbackCount: 0,
  frameWindowCount: 0,
  diagnosticsWindowStart: 0,
  pointerMoveWindowCount: 0,
  interactionStoreWriteCount: 0,
  raycastWindowCount: 0,
  raycastWindowTotalMs: 0,
  raycastWindowMaxMs: 0,
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

const round2 = (value: number): number => Math.round(value * 100) / 100

const percentile = (values: number[], p: number): number => {
  if (values.length === 0) return 0
  const sorted = [...values].sort((a, b) => a - b)
  const idx = Math.min(sorted.length - 1, Math.max(0, Math.ceil((p / 100) * sorted.length) - 1))
  return sorted[idx] ?? 0
}

const summarizeSubsystemTimings = (
  buckets: Record<string, { totalMs: number; maxMs: number; samples: number }>
): PerformanceSubsystemTiming[] => {
  const results: PerformanceSubsystemTiming[] = []
  for (const [name, bucket] of Object.entries(buckets)) {
    if (bucket.samples <= 0) continue
    results.push({
      name,
      avgMs: round2(bucket.totalMs / bucket.samples),
      maxMs: round2(bucket.maxMs),
      totalMs: round2(bucket.totalMs),
      samples: bucket.samples,
    })
  }
  results.sort((a, b) => b.totalMs - a.totalMs)
  return results.slice(0, 6)
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
      points: null,
      lines: null,
      memoryUsageMb: null,
    })

    set({
      isMonitoring: true,
      samples: new Array(get().config.sampleSize).fill({
        fps: 60,
        timestamp: now,
        frameTimeMs: 16.67,
      }),
      sampleIndex: 0,
      traceRing,
      traceRingIndex: 0,
      traceRingCount: 0,
      publishedTraceSamples: [],
      traceMarkers: [],
      traceVersion: 0,
      lastDegradedState: false,
      longTaskWindowCount: 0,
      longTaskWindowBlockedMs: 0,
      longTaskWindowWorstMs: 0,
      subsystemTimingBuckets: {},
      invalidateWindowCount: 0,
      renderRequestWindowCount: 0,
      renderRequestReasonBuckets: {},
      lastRenderReason: 'monitor-start',
      invalidateWindowStart: now,
      frameLoopWindowTotalMs: 0,
      frameLoopWindowCallbackCount: 0,
      frameWindowCount: 0,
      diagnosticsWindowStart: now,
      pointerMoveWindowCount: 0,
      interactionStoreWriteCount: 0,
      raycastWindowCount: 0,
      raycastWindowTotalMs: 0,
      raycastWindowMaxMs: 0,
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
    samples[sampleIndex] = { fps, timestamp: now, frameTimeMs: deltaMs }

    // Update trace ring (direct mutation for performance)
    const traceRing = state.traceRing
    const traceWriteIndex = state.traceRingIndex
    traceRing[traceWriteIndex] = {
      timestamp: now,
      fps,
      frameTimeMs: deltaMs,
      drawCalls: renderStats?.drawCalls ?? null,
      triangles: renderStats?.triangles ?? null,
      points: renderStats?.points ?? null,
      lines: renderStats?.lines ?? null,
      memoryUsageMb: null,
    }
    const newTraceRingIndex = (traceWriteIndex + 1) % state.config.traceSampleSize
    const newTraceRingCount = Math.min(state.traceRingCount + 1, state.config.traceSampleSize)

    // Calculate new index
    const newIndex = (sampleIndex + 1) % state.config.sampleSize
    const frameWindowCount = state.frameWindowCount + 1

    // Only update React state every ~10 frames to reduce overhead
    if (newIndex % state.config.tracePublishIntervalFrames === 0) {
      // Calculate metrics from samples
      let sum = 0
      let min = Infinity
      let max = -Infinity
      let over16Count = 0
      let over33Count = 0
      const frameTimes: number[] = []

      for (const sample of samples) {
        sum += sample.fps
        if (sample.fps < min) min = sample.fps
        if (sample.fps > max) max = sample.fps
        frameTimes.push(sample.frameTimeMs)
        if (sample.frameTimeMs > 16.7) over16Count++
        if (sample.frameTimeMs > 33.3) over33Count++
      }

      const avgFps = sum / samples.length
      const isDegraded = avgFps < state.config.degradedThreshold
      const p50 = percentile(frameTimes, 50)
      const p95 = percentile(frameTimes, 95)
      const p99 = percentile(frameTimes, 99)
      const overBudget16Pct = samples.length > 0 ? (over16Count / samples.length) * 100 : 0
      const overBudget33Pct = samples.length > 0 ? (over33Count / samples.length) * 100 : 0

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

      let drawCallsSum = 0
      let drawCallsCount = 0
      let trianglesSum = 0
      let trianglesCount = 0
      let pointsSum = 0
      let pointsCount = 0
      let linesSum = 0
      let linesCount = 0

      for (let i = 0; i < newTraceRingCount; i++) {
        const s = traceRing[i]
        if (!s) continue
        if (s.drawCalls !== null) {
          drawCallsSum += s.drawCalls
          drawCallsCount++
        }
        if (s.triangles !== null) {
          trianglesSum += s.triangles
          trianglesCount++
        }
        if (s.points !== null) {
          pointsSum += s.points
          pointsCount++
        }
        if (s.lines !== null) {
          linesSum += s.lines
          linesCount++
        }
      }

      const subsystemTimings = summarizeSubsystemTimings(state.subsystemTimingBuckets)
      const longTaskBlockedMs = round2(state.longTaskWindowBlockedMs)
      const longTaskWorstMs = round2(state.longTaskWindowWorstMs)
      const longTaskCount = state.longTaskWindowCount
      const invalidateElapsedMs = Math.max(1, now - state.invalidateWindowStart)
      const invalidateRateHz = round2((state.invalidateWindowCount * 1000) / invalidateElapsedMs)
      const renderRequestRateHz = round2(
        (state.renderRequestWindowCount * 1000) / invalidateElapsedMs
      )
      const topRenderReasons = Object.entries(state.renderRequestReasonBuckets)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 4)
        .map(([reason, count]) => `${reason}:${count}`)
        .join(', ')
      const windowElapsedMs = Math.max(1, now - state.diagnosticsWindowStart)
      const renderedFramesPerSecond = round2((frameWindowCount * 1000) / windowElapsedMs)
      const pointerMoveRateHz = round2((state.pointerMoveWindowCount * 1000) / windowElapsedMs)
      const interactionStoreWritesPerSec = round2(
        (state.interactionStoreWriteCount * 1000) / windowElapsedMs
      )
      const raycastAvgMs = state.raycastWindowCount > 0
        ? round2(state.raycastWindowTotalMs / state.raycastWindowCount)
        : 0
      const raycastMaxMs = round2(state.raycastWindowMaxMs)
      const useFrameTotalMs = round2(state.frameLoopWindowTotalMs)
      const useFrameCallbacksPerFrame = frameWindowCount > 0
        ? round2(state.frameLoopWindowCallbackCount / frameWindowCount)
        : 0
      let interactionTotalMs = 0
      for (const timing of subsystemTimings) {
        if (timing.name.startsWith('interaction.')) {
          interactionTotalMs += timing.totalMs
        }
      }
      const interactionMsPerFrame = frameWindowCount > 0
        ? round2(interactionTotalMs / frameWindowCount)
        : 0
      const unaccountedFrameMs = round2(
        Math.max(0, deltaMs - useFrameTotalMs - interactionMsPerFrame)
      )

      set({
        sampleIndex: newIndex,
        traceRingIndex: newTraceRingIndex,
        traceRingCount: newTraceRingCount,
        publishedTraceSamples: orderedTraceSamples(traceRing, newTraceRingCount, newTraceRingIndex),
        traceMarkers,
        traceVersion: state.traceVersion + 1,
        lastDegradedState: isDegraded,
        longTaskWindowCount: 0,
        longTaskWindowBlockedMs: 0,
        longTaskWindowWorstMs: 0,
        subsystemTimingBuckets: {},
        invalidateWindowCount: 0,
        renderRequestWindowCount: 0,
        renderRequestReasonBuckets: {},
        invalidateWindowStart: now,
        frameLoopWindowTotalMs: 0,
        frameLoopWindowCallbackCount: 0,
        frameWindowCount: 0,
        diagnosticsWindowStart: now,
        pointerMoveWindowCount: 0,
        interactionStoreWriteCount: 0,
        raycastWindowCount: 0,
        raycastWindowTotalMs: 0,
        raycastWindowMaxMs: 0,
        sceneSnapshot: {
          ...state.sceneSnapshot,
          invalidateRateHz,
          renderRequestRateHz,
          lastRenderReason: state.lastRenderReason,
          topRenderReasons,
        },
        metrics: {
          currentFps: Math.round(fps),
          averageFps: Math.round(avgFps),
          minFps: Math.round(min),
          maxFps: Math.round(max),
          frameTimeMs: deltaMs,
          frameP50Ms: round2(p50),
          frameP95Ms: round2(p95),
          frameP99Ms: round2(p99),
          overBudget16Pct: round2(overBudget16Pct),
          overBudget33Pct: round2(overBudget33Pct),
          isDegraded,
          memoryUsageMb,
          gpuMemoryMb: null, // Not easily accessible in browsers
          workload: {
            drawCalls: renderStats?.drawCalls ?? null,
            drawCallsAvg: drawCallsCount > 0 ? Math.round(drawCallsSum / drawCallsCount) : null,
            triangles: renderStats?.triangles ?? null,
            trianglesAvg: trianglesCount > 0 ? Math.round(trianglesSum / trianglesCount) : null,
            points: renderStats?.points ?? null,
            pointsAvg: pointsCount > 0 ? Math.round(pointsSum / pointsCount) : null,
            lines: renderStats?.lines ?? null,
            linesAvg: linesCount > 0 ? Math.round(linesSum / linesCount) : null,
            geometries: renderStats?.geometries ?? null,
            textures: renderStats?.textures ?? null,
            programs: renderStats?.programs ?? null,
          },
          longTasks: {
            count: longTaskCount,
            blockedMs: longTaskBlockedMs,
            worstMs: longTaskWorstMs,
          },
          subsystemTimings,
          useFrameTotalMs,
          useFrameCallbacksPerFrame,
          renderedFramesPerSecond,
          pointerMoveRateHz,
          interactionStoreWritesPerSec,
          raycastAvgMs,
          raycastMaxMs,
          interactionMsPerFrame,
          unaccountedFrameMs,
        },
      })
    } else {
      // Avoid notifying subscribers every frame; sampleIndex is internal.
      state.sampleIndex = newIndex
      state.traceRingIndex = newTraceRingIndex
      state.traceRingCount = newTraceRingCount
      state.frameWindowCount = frameWindowCount
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
        points: null,
        lines: null,
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

  recordLongTask: (durationMs) => {
    const state = get()
    if (!state.isMonitoring) return

    state.longTaskWindowCount += 1
    state.longTaskWindowBlockedMs += durationMs
    if (durationMs > state.longTaskWindowWorstMs) {
      state.longTaskWindowWorstMs = durationMs
    }
  },

  recordSubsystemSample: (name, durationMs) => {
    const state = get()
    if (!state.isMonitoring || !Number.isFinite(durationMs)) return

    const buckets = state.subsystemTimingBuckets
    const bucket = buckets[name] ?? { totalMs: 0, maxMs: 0, samples: 0 }
    bucket.totalMs += durationMs
    bucket.samples += 1
    if (durationMs > bucket.maxMs) {
      bucket.maxMs = durationMs
    }
    buckets[name] = bucket
  },

  setSceneSnapshot: (snapshot) =>
    set((state) => ({
      sceneSnapshot: {
        ...state.sceneSnapshot,
        ...snapshot,
      },
    })),

  recordInvalidate: () => {
    const state = get()
    if (!state.isMonitoring) return
    state.invalidateWindowCount += 1
    if (state.invalidateWindowStart === 0) {
      state.invalidateWindowStart = performance.now()
    }
  },

  recordRenderRequest: (reason) => {
    const state = get()
    if (!state.isMonitoring) return
    const normalizedReason = reason.trim() || 'unknown'
    state.renderRequestWindowCount += 1
    state.lastRenderReason = normalizedReason
    state.renderRequestReasonBuckets[normalizedReason] =
      (state.renderRequestReasonBuckets[normalizedReason] ?? 0) + 1
    if (state.invalidateWindowStart === 0) {
      state.invalidateWindowStart = performance.now()
    }
  },

  recordFrameLoopAggregate: (totalMs, callbackCount) => {
    const state = get()
    if (!state.isMonitoring || !Number.isFinite(totalMs) || !Number.isFinite(callbackCount)) return
      state.frameLoopWindowTotalMs += totalMs
      state.frameLoopWindowCallbackCount += callbackCount
  },

  recordPointerMoveEvent: () => {
    const state = get()
    if (!state.isMonitoring) return
    state.pointerMoveWindowCount += 1
  },

  recordRaycastSample: (durationMs) => {
    const state = get()
    if (!state.isMonitoring || !Number.isFinite(durationMs)) return
    state.raycastWindowCount += 1
    state.raycastWindowTotalMs += durationMs
    if (durationMs > state.raycastWindowMaxMs) {
      state.raycastWindowMaxMs = durationMs
    }
  },

  recordInteractionStoreWrite: () => {
    const state = get()
    if (!state.isMonitoring) return
    state.interactionStoreWriteCount += 1
  },

  recordTraceMarker: (type, label) => {
    const marker: PerformanceTraceMarker = {
      timestamp: performance.now(),
      type,
      label,
    }
    set((state) => ({
      traceMarkers: [...state.traceMarkers, marker].slice(-state.config.traceMarkerLimit),
      traceVersion: state.traceVersion + 1,
    }))
  },

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
        sceneSnapshot: {
          ...state.sceneSnapshot,
          documentHidden: !isVisible,
          lastVisibilityEvent: isVisible ? 'visible' : 'hidden',
        },
      }
    })
  },

  setWindowFocus: (isFocused) => {
    const marker: PerformanceTraceMarker = {
      timestamp: performance.now(),
      type: isFocused ? 'focus' : 'blur',
      label: isFocused ? 'Window focused' : 'Window blurred',
    }
    set((state) => ({
      traceMarkers: [...state.traceMarkers, marker].slice(-state.config.traceMarkerLimit),
      traceVersion: state.traceVersion + 1,
      sceneSnapshot: {
        ...state.sceneSnapshot,
        windowFocused: isFocused,
        lastFocusEvent: isFocused ? 'focus' : 'blur',
      },
    }))
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
      longTaskWindowCount: 0,
      longTaskWindowBlockedMs: 0,
      longTaskWindowWorstMs: 0,
      subsystemTimingBuckets: {},
      sceneSnapshot: initialSceneSnapshot,
      invalidateWindowCount: 0,
      renderRequestWindowCount: 0,
      renderRequestReasonBuckets: {},
      lastRenderReason: 'reset',
      invalidateWindowStart: 0,
      frameLoopWindowTotalMs: 0,
      frameLoopWindowCallbackCount: 0,
      frameWindowCount: 0,
      diagnosticsWindowStart: 0,
      pointerMoveWindowCount: 0,
      interactionStoreWriteCount: 0,
      raycastWindowCount: 0,
      raycastWindowTotalMs: 0,
      raycastWindowMaxMs: 0,
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
 * Selector for effective FPS cap. Respects auto default by tier.
 */
export const selectEffectiveMaxFps = (state: PerformanceStore): number => (
  state.config.maxFps === 'auto'
    ? getDefaultMaxFpsForTier(state.currentTier)
    : state.config.maxFps
)

/**
 * Selector for trace overlay data.
 */
export const selectTraceData = (state: PerformanceStore) => ({
  samples: state.publishedTraceSamples,
  markers: state.traceMarkers,
  version: state.traceVersion,
})

/**
 * Selector for scene complexity snapshot.
 */
export const selectSceneSnapshot = (state: PerformanceStore) => state.sceneSnapshot
