/**
 * Performance monitoring types for runtime FPS tracking and dynamic adjustment.
 */

import type { PerformanceTier } from './graphics'

export type MaxFpsSetting = 'auto' | number

/** FPS sample for moving average calculation */
export interface FPSSample {
  fps: number
  timestamp: number
  frameTimeMs: number
}

/** Trace sample for FPS timeline visualization */
export interface PerformanceTraceSample {
  timestamp: number
  fps: number
  frameTimeMs: number
  drawCalls: number | null
  triangles: number | null
  points: number | null
  lines: number | null
  memoryUsageMb: number | null
}

/** Marker event on performance timeline */
export interface PerformanceTraceMarker {
  timestamp: number
  type:
    | 'tier-adjust'
    | 'degraded'
    | 'recovered'
    | 'hidden'
    | 'visible'
    | 'focus'
    | 'blur'
    | 'drag-start'
    | 'drag-end'
    | 'hover-start'
    | 'hover-end'
    | 'selection-change'
    | 'camera-mode-change'
    | 'render-request'
  label: string
}

/** Aggregated render workload counters */
export interface RenderWorkloadStats {
  drawCalls: number | null
  drawCallsAvg: number | null
  triangles: number | null
  trianglesAvg: number | null
  points: number | null
  pointsAvg: number | null
  lines: number | null
  linesAvg: number | null
  geometries: number | null
  textures: number | null
  programs: number | null
}

/** Main-thread long task summary from PerformanceObserver */
export interface LongTaskStats {
  count: number
  blockedMs: number
  worstMs: number
}

/** Subsystem timing summary for hotspot detection */
export interface PerformanceSubsystemTiming {
  name: string
  avgMs: number
  maxMs: number
  totalMs: number
  samples: number
}

/** Scene complexity snapshot for quick correlation */
export interface SceneComplexitySnapshot {
  agents: number
  mountedAgents: number
  furniture: number
  decorations: number
  stars: number
  sceneObjects: number
  sceneMeshes: number
  selectedNodes: number
  sceneType: string
  tier: PerformanceTier
  dpr: string
  shadows: boolean
  materialQuality: string
  maxFps: MaxFpsSetting
  effectiveMaxFps: number
  frameloopMode: 'always' | 'demand'
  forceAlwaysFrameloop: boolean
  invalidateRateHz: number
  renderRequestRateHz: number
  lastRenderReason: string
  topRenderReasons: string
  documentHidden: boolean
  windowFocused: boolean
  eventLoopLagMs: number
  lastVisibilityEvent: 'hidden' | 'visible'
  lastFocusEvent: 'focus' | 'blur'
}

/** Performance metrics snapshot */
export interface PerformanceMetrics {
  /** Current FPS (frames per second) */
  currentFps: number
  /** Average FPS over sample window */
  averageFps: number
  /** Minimum FPS in sample window */
  minFps: number
  /** Maximum FPS in sample window */
  maxFps: number
  /** Frame time in milliseconds */
  frameTimeMs: number
  /** Frame-time percentile (p50) in milliseconds */
  frameP50Ms: number
  /** Frame-time percentile (p95) in milliseconds */
  frameP95Ms: number
  /** Frame-time percentile (p99) in milliseconds */
  frameP99Ms: number
  /** Percent of samples over 16.7ms (missed 60fps budget) */
  overBudget16Pct: number
  /** Percent of samples over 33.3ms (missed 30fps budget) */
  overBudget33Pct: number
  /** Whether performance is degraded (below target) */
  isDegraded: boolean
  /** Memory usage if available (MB) */
  memoryUsageMb: number | null
  /** GPU memory usage if available (MB) */
  gpuMemoryMb: number | null
  /** Renderer workload counters */
  workload: RenderWorkloadStats
  /** Main-thread long task summary */
  longTasks: LongTaskStats
  /** Top subsystem timings in current sampling window */
  subsystemTimings: PerformanceSubsystemTiming[]
  /** Total sampled useFrame callback CPU time in current publish window (ms) */
  useFrameTotalMs: number
  /** Average sampled useFrame callbacks executed per rendered frame */
  useFrameCallbacksPerFrame: number
  /** Frames rendered per second in current publish window */
  renderedFramesPerSecond: number
  /** Pointer move events per second */
  pointerMoveRateHz: number
  /** Interaction-store writes per second */
  interactionStoreWritesPerSec: number
  /** Raycast cost in current publish window (ms avg) */
  raycastAvgMs: number
  /** Raycast worst-case cost in current publish window (ms max) */
  raycastMaxMs: number
  /** Total interaction handler cost per rendered frame (ms) */
  interactionMsPerFrame: number
  /** Frame time not explained by measured callbacks/interaction (ms) */
  unaccountedFrameMs: number
}

/** Configuration for FPS monitoring */
export interface FPSMonitorConfig {
  /** Target FPS to maintain */
  targetFps: number
  /** FPS threshold below which tier is decreased */
  degradedThreshold: number
  /** FPS threshold above which tier can be increased */
  upgradeThreshold: number
  /** Number of samples to average */
  sampleSize: number
  /** Minimum time between tier adjustments (ms) */
  adjustmentCooldown: number
  /** Whether to automatically adjust tier */
  autoAdjust: boolean
  /** Whether to show FPS overlay */
  showOverlay: boolean
  /** Max FPS cap. 'auto' uses tier defaults (low/medium=30, high/ultra=60). */
  maxFps: MaxFpsSetting
  /** Whether to render trace charts in overlay */
  showTraceCharts: boolean
  /** Number of samples retained for trace charts */
  traceSampleSize: number
  /** Number of frames between trace publish updates */
  tracePublishIntervalFrames: number
  /** Max marker events retained for trace chart */
  traceMarkerLimit: number
  /** Force `always` frameloop for A/B diagnostics */
  forceAlwaysFrameloop: boolean
}

/** Tier adjustment recommendation */
export interface TierAdjustment {
  /** Whether adjustment is recommended */
  shouldAdjust: boolean
  /** Direction of adjustment */
  direction: 'increase' | 'decrease' | 'none'
  /** Recommended tier */
  recommendedTier: PerformanceTier
  /** Reason for adjustment */
  reason: string
}

/** Default FPS monitor configuration */
export const DEFAULT_FPS_CONFIG: FPSMonitorConfig = {
  targetFps: 60,
  degradedThreshold: 45, // Below 45 FPS = degraded
  upgradeThreshold: 55,  // Above 55 FPS consistently = can upgrade
  sampleSize: 60,        // 1 second of samples at 60fps
  adjustmentCooldown: 5000, // 5 seconds between adjustments
  autoAdjust: true,
  showOverlay: false,
  maxFps: 'auto',
  showTraceCharts: true,
  traceSampleSize: 240, // ~4 seconds at 60fps
  tracePublishIntervalFrames: 12,
  traceMarkerLimit: 50,
  forceAlwaysFrameloop: false,
}

/** FPS thresholds for each tier */
export const TIER_FPS_THRESHOLDS: Record<PerformanceTier, { min: number; target: number }> = {
  low: { min: 20, target: 30 },
  medium: { min: 30, target: 45 },
  high: { min: 45, target: 60 },
  ultra: { min: 50, target: 60 },
}
