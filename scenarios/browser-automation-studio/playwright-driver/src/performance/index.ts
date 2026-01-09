/**
 * Performance Debug Mode Module
 *
 * Provides instrumentation for the frame streaming pipeline to identify
 * bottlenecks and track optimization progress.
 *
 * ## Quick Start
 *
 * ```typescript
 * import { PerfCollector } from './performance';
 * import { loadConfig } from './config';
 *
 * const config = loadConfig();
 * const collector = PerfCollector.fromConfig(sessionId, config, targetFps);
 *
 * // In frame loop:
 * const captureStart = performance.now();
 * const buffer = await captureFrame();
 * const captureMs = performance.now() - captureStart;
 *
 * // ... compare and send ...
 *
 * collector.recordFrame({ captureMs, compareMs, wsSendMs, frameBytes, skipped: false });
 *
 * if (collector.shouldLogSummary()) {
 *   logger.info('Frame perf', collector.getAggregatedStats());
 * }
 * ```
 */

export { PerfCollector } from './collector';
export type { PerfCollectorConfig, FrameTimingInput } from './collector';
export type {
  FrameTimings,
  FrameHeader,
  FrameStatsAggregated,
  BottleneckType,
  PerfStatsMessage,
  DebugPerformanceResponse,
} from './types';
