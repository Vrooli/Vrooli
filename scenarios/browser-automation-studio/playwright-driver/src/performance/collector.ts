/**
 * Performance Collector
 *
 * Collects and aggregates timing data for the frame streaming pipeline.
 * Uses a ring buffer to store recent frame timings for percentile analysis.
 *
 * ## Usage
 *
 * ```typescript
 * const collector = new PerfCollector(sessionId, config);
 *
 * // Record each frame
 * collector.recordFrame({
 *   captureMs: 45,
 *   compareMs: 2,
 *   wsSendMs: 5,
 *   frameBytes: 45000,
 *   skipped: false,
 * });
 *
 * // Get aggregated stats periodically
 * if (collector.shouldLogSummary()) {
 *   const stats = collector.getAggregatedStats();
 *   logger.info('Frame performance', stats);
 * }
 *
 * // Build header for WebSocket frame
 * const header = collector.buildFrameHeader(sequenceNum, captureMs, compareMs, frameBytes);
 * ```
 */

import type { Config } from '../config';
import type {
  FrameTimings,
  FrameHeader,
  FrameStatsAggregated,
  BottleneckType,
} from './types';

/** Configuration for the performance collector */
export interface PerfCollectorConfig {
  /** Number of frame timings to retain in the ring buffer */
  bufferSize: number;
  /** Log summary every N frames (0 = disabled) */
  logSummaryInterval: number;
  /** Target FPS for the session */
  targetFps: number;
}

/** Frame timing data for recording (partial - driver-side only) */
export interface FrameTimingInput {
  captureMs: number;
  compareMs: number;
  wsSendMs: number;
  frameBytes: number;
  skipped: boolean;
}

/**
 * Collects and aggregates frame timing data.
 * Thread-safe for single-threaded Node.js (no async operations mutate state).
 */
export class PerfCollector {
  private readonly sessionId: string;
  private readonly bufferSize: number;
  private readonly logSummaryInterval: number;
  private readonly targetFps: number;

  /** Ring buffer of frame timings */
  private readonly timings: { timing: FrameTimings; startMs: number }[] = [];
  private next = 0;

  // Wall time labels are anchored to monotonic observation time, not remote clocks.
  private startTime = new Date();
  private startMs = performance.now();
  private lastRecordedMs = this.startMs;

  /** Current sequence number */
  private sequenceNum = 0;

  constructor(sessionId: string, config: PerfCollectorConfig) {
    this.sessionId = sessionId;
    this.bufferSize = Math.max(1, config.bufferSize);
    this.logSummaryInterval = Math.max(0, config.logSummaryInterval);
    this.targetFps = config.targetFps;
  }

  /**
   * Create a PerfCollector from the global config.
   */
  static fromConfig(sessionId: string, config: Config, targetFps: number): PerfCollector {
    return new PerfCollector(sessionId, {
      bufferSize: config.performance.bufferSize,
      logSummaryInterval: config.performance.logSummaryInterval,
      targetFps,
    });
  }

  /**
   * Record a new frame timing.
   * Increments sequence number automatically.
   */
  recordFrame(input: FrameTimingInput): void {
    this.sequenceNum++;

    const timing: FrameTimings = {
      frame_id: `${this.sessionId}-${this.sequenceNum}`,
      session_id: this.sessionId,
      sequence_num: this.sequenceNum,
      timestamp: new Date().toISOString(),
      capture_ms: input.captureMs,
      compare_ms: input.compareMs,
      ws_send_ms: input.wsSendMs,
      driver_total_ms: input.captureMs + input.compareMs + input.wsSendMs,
      frame_bytes: input.frameBytes,
      skipped: input.skipped,
    };

    const sample = { timing, startMs: this.lastRecordedMs };
    this.lastRecordedMs = performance.now();
    this.timings[this.next] = sample;
    this.next = (this.next + 1) % this.bufferSize;
  }

  /**
   * Record a skipped frame (identical to previous).
   * Only capture and compare times are recorded.
   */
  recordSkipped(captureMs: number, compareMs: number): void {
    this.recordFrame({
      captureMs,
      compareMs,
      wsSendMs: 0,
      frameBytes: 0,
      skipped: true,
    });
  }

  /**
   * Build a binary frame header for WebSocket transmission.
   * Returns a Buffer containing the header length prefix and JSON header.
   */
  buildFrameHeader(
    captureMs: number,
    compareMs: number,
    wsSendMs: number,
    frameBytes: number
  ): Buffer {
    const header: FrameHeader = {
      frame_id: `${this.sessionId}-${this.sequenceNum + 1}`, // Next frame
      capture_ms: captureMs,
      compare_ms: compareMs,
      ws_send_ms: wsSendMs,
      frame_bytes: frameBytes,
    };

    const headerJson = Buffer.from(JSON.stringify(header), 'utf8');
    const headerLen = Buffer.alloc(4);
    headerLen.writeUInt32BE(headerJson.length, 0);

    return Buffer.concat([headerLen, headerJson]);
  }

  /**
   * Check if we should log a summary based on frame count.
   */
  shouldLogSummary(): boolean {
    return this.sequenceNum > 0 && this.logSummaryInterval > 0 && this.sequenceNum % this.logSummaryInterval === 0;
  }

  /**
   * Get current frame count.
   */
  getFrameCount(): number {
    return this.sequenceNum;
  }

  /**
   * Get current sequence number.
   */
  getSequenceNum(): number {
    return this.sequenceNum;
  }

  /**
   * Get aggregated statistics from the ring buffer.
   */
  getAggregatedStats(): FrameStatsAggregated {
    const first = this.timings[this.next % this.timings.length];
    const windowStartMs = first?.startMs ?? this.lastRecordedMs;
    const windowDurationMs = Math.max(0, performance.now() - windowStartMs);
    const samples = this.timings.map(({ timing }) => timing);
    const sent = samples.filter((t) => !t.skipped);
    const captureTimes = samples.map((t) => t.capture_ms).sort((a, b) => a-b);
    const processingTimes = sent.map((t) => t.driver_total_ms).sort((a, b) => a-b);
    const totalBytes = sent.reduce((total, t) => total + t.frame_bytes, 0);
    const captureP50 = percentile(captureTimes, .5);
    const captureP90 = percentile(captureTimes, .9);
    const processingP90 = percentile(processingTimes, .9);
    const { bottleneck, description } = samples.length
      ? identifyBottleneck(captureP50, captureP90, processingP90, this.targetFps)
      : { bottleneck: 'none' as const, description: 'No frames recorded yet' };

    return {
      session_id: this.sessionId,
      window_start_time: new Date(this.startTime.getTime() + windowStartMs - this.startMs).toISOString(),
      window_duration_ms: windowDurationMs,
      frame_count: samples.length,
      skipped_count: samples.length - sent.length,
      capture_p50_ms: round2(captureP50),
      capture_p90_ms: round2(captureP90),
      capture_p99_ms: round2(percentile(captureTimes, .99)),
      capture_max_ms: round2(percentile(captureTimes, 1)),
      // Legacy wire names: sums of processing durations, not measured transit/paint.
      e2e_p50_ms: round2(percentile(processingTimes, .5)),
      e2e_p90_ms: round2(processingP90),
      e2e_p99_ms: round2(percentile(processingTimes, .99)),
      e2e_max_ms: round2(percentile(processingTimes, 1)),
      actual_fps: windowDurationMs > 0 ? round2(sent.length * 1000 / windowDurationMs) : 0,
      target_fps: this.targetFps,
      avg_frame_bytes: sent.length ? Math.round(totalBytes / sent.length) : 0,
      bandwidth_bytes_per_sec: windowDurationMs > 0 ? Math.round(totalBytes * 1000 / windowDurationMs) : 0,
      primary_bottleneck: bottleneck,
      bottleneck_description: description,
    };
  }

  /**
   * Get the most recent frame timings.
   * @param limit Maximum number of frames to return (default: 10)
   */
  getRecentFrames(limit = 10): FrameTimings[] {
    const count = limit > 0 ? Math.min(limit, this.timings.length) : this.timings.length;
    return Array.from({ length: count }, (_, i) => ({
      ...this.timings[(this.next + this.timings.length - count + i) % this.timings.length]!.timing,
    }));
  }

  /**
   * Reset all statistics (e.g., when session restarts).
   */
  reset(): void {
    this.timings.length = 0;
    this.next = 0;
    this.startTime = new Date();
    this.startMs = performance.now();
    this.lastRecordedMs = this.startMs;
    this.sequenceNum = 0;
  }
}

/**
 * Calculate the p-th percentile of a sorted array.
 * Uses linear interpolation between nearest ranks.
 */
function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0;
  if (sorted.length === 1) return sorted[0] ?? 0;

  const index = p * (sorted.length - 1);
  const lower = Math.floor(index);
  const upper = Math.ceil(index);

  if (lower === upper) return sorted[lower] ?? 0;

  const fraction = index - lower;
  const lowerValue = sorted[lower] ?? 0;
  const upperValue = sorted[upper] ?? lowerValue;
  return lowerValue * (1 - fraction) + upperValue * fraction;
}

/**
 * Round to 2 decimal places.
 */
function round2(n: number): number {
  return Math.round(n * 100) / 100;
}

/**
 * Identify the primary bottleneck based on timing data.
 */
function identifyBottleneck(
  captureP50: number,
  captureP90: number,
  e2eP90: number,
  targetFps: number
): { bottleneck: BottleneckType; description: string } {
  const targetFrameTime = 1000 / targetFps;

  // If capture P90 > 80% of target frame time, capture is the bottleneck
  if (captureP90 > targetFrameTime * 0.8) {
    return {
      bottleneck: 'capture',
      description: `Screenshot capture P90 (${round2(captureP90)}ms) exceeds 80% of target frame time (${round2(targetFrameTime)}ms). Consider reducing quality or resolution.`,
    };
  }

  // If capture P50 > 100ms, capture is definitely slow
  if (captureP50 > 100) {
    return {
      bottleneck: 'capture',
      description: `Screenshot capture averaging ${round2(captureP50)}ms (>100ms threshold). The browser may be under heavy load.`,
    };
  }

  // Component durations cannot establish network transit or client rendering cost.
  if (e2eP90 > targetFrameTime * 1.5) {
    return {
      bottleneck: 'processing',
      description: `Processing P90 (${round2(e2eP90)}ms) exceeds the frame budget. Network transit and client rendering are not measured.`,
    };
  }

  return {
    bottleneck: 'none',
    description: 'No significant bottlenecks in measured processing. Network transit and client rendering are not measured.',
  };
}

/**
 * Export index for the performance module.
 */
export * from './types';
