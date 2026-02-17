/**
 * Performance monitoring types for runtime FPS tracking and dynamic adjustment.
 */

import type { PerformanceTier } from './graphics'

export type MaxFpsSetting = 'auto' | number

/** FPS sample for moving average calculation */
export interface FPSSample {
  fps: number
  timestamp: number
}

/** Trace sample for FPS timeline visualization */
export interface PerformanceTraceSample {
  timestamp: number
  fps: number
  frameTimeMs: number
  drawCalls: number | null
  triangles: number | null
  memoryUsageMb: number | null
}

/** Marker event on performance timeline */
export interface PerformanceTraceMarker {
  timestamp: number
  type: 'tier-adjust' | 'degraded' | 'recovered' | 'hidden' | 'visible'
  label: string
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
  /** Whether performance is degraded (below target) */
  isDegraded: boolean
  /** Memory usage if available (MB) */
  memoryUsageMb: number | null
  /** GPU memory usage if available (MB) */
  gpuMemoryMb: number | null
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
}

/** FPS thresholds for each tier */
export const TIER_FPS_THRESHOLDS: Record<PerformanceTier, { min: number; target: number }> = {
  low: { min: 20, target: 30 },
  medium: { min: 30, target: 45 },
  high: { min: 45, target: 60 },
  ultra: { min: 50, target: 60 },
}
