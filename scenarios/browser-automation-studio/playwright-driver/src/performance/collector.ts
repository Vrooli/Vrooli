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
  private readonly timings: FrameTimings[] = [];

  /** Total frames recorded (including those evicted from buffer) */
  private frameCount = 0;

  /** Total frames skipped */
  private skippedCount = 0;

  /** When collection started */
  private readonly startTime: Date;

  /** Current sequence number */
  private sequenceNum = 0;

  constructor(sessionId: string, config: PerfCollectorConfig) {
    this.sessionId = sessionId;
    this.bufferSize = Math.max(1, config.bufferSize);
    this.logSummaryInterval = Math.max(0, config.logSummaryInterval);
    this.targetFps = config.targetFps;
    this.startTime = new Date();
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
    this.frameCount++;
    if (input.skipped) {
      this.skippedCount++;
    }

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

    // Ring buffer: evict oldest if full
    if (this.timings.length >= this.bufferSize) {
      this.timings.shift();
    }
    this.timings.push(timing);
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
    if (this.logSummaryInterval <= 0) return false;
    return this.frameCount % this.logSummaryInterval === 0;
  }

  /**
   * Get current frame count.
   */
  getFrameCount(): number {
    return this.frameCount;
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
    const now = new Date();
    const windowDurationMs = now.getTime() - this.startTime.getTime();

    if (this.timings.length === 0) {
      return {
        session_id: this.sessionId,
        window_start_time: this.startTime.toISOString(),
        window_duration_ms: windowDurationMs,
        frame_count: 0,
        skipped_count: 0,
        capture_p50_ms: 0,
        capture_p90_ms: 0,
        capture_p99_ms: 0,
        capture_max_ms: 0,
        e2e_p50_ms: 0,
        e2e_p90_ms: 0,
        e2e_p99_ms: 0,
        e2e_max_ms: 0,
        actual_fps: 0,
        target_fps: this.targetFps,
        avg_frame_bytes: 0,
        bandwidth_bytes_per_sec: 0,
        primary_bottleneck: 'none',
        bottleneck_description: 'No frames recorded yet',
      };
    }

    // Extract timing arrays for percentile calculation
    const captureTimes = this.timings.map((t) => t.capture_ms);
    const e2eTimes = this.timings.map((t) => t.driver_total_ms);
    const frameSizes = this.timings.filter((t) => !t.skipped).map((t) => t.frame_bytes);

    // Sort for percentile calculation
    const sortedCapture = [...captureTimes].sort((a, b) => a - b);
    const sortedE2E = [...e2eTimes].sort((a, b) => a - b);

    // Calculate percentiles
    const captureP50 = percentile(sortedCapture, 0.5);
    const captureP90 = percentile(sortedCapture, 0.9);
    const captureP99 = percentile(sortedCapture, 0.99);
    const captureMax = sortedCapture[sortedCapture.length - 1];

    const e2eP50 = percentile(sortedE2E, 0.5);
    const e2eP90 = percentile(sortedE2E, 0.9);
    const e2eP99 = percentile(sortedE2E, 0.99);
    const e2eMax = sortedE2E[sortedE2E.length - 1];

    // Calculate throughput
    const actualFps = (this.frameCount / windowDurationMs) * 1000;
    const avgFrameBytes =
      frameSizes.length > 0
        ? Math.round(frameSizes.reduce((a, b) => a + b, 0) / frameSizes.length)
        : 0;
    const totalBytes = frameSizes.reduce((a, b) => a + b, 0);
    const bandwidthBytesPerSec = Math.round((totalBytes / windowDurationMs) * 1000);

    // Identify bottleneck
    const { bottleneck, description } = identifyBottleneck(
      captureP50,
      captureP90,
      e2eP90,
      this.targetFps
    );

    return {
      session_id: this.sessionId,
      window_start_time: this.startTime.toISOString(),
      window_duration_ms: windowDurationMs,
      frame_count: this.frameCount,
      skipped_count: this.skippedCount,
      capture_p50_ms: round2(captureP50),
      capture_p90_ms: round2(captureP90),
      capture_p99_ms: round2(captureP99),
      capture_max_ms: round2(captureMax),
      e2e_p50_ms: round2(e2eP50),
      e2e_p90_ms: round2(e2eP90),
      e2e_p99_ms: round2(e2eP99),
      e2e_max_ms: round2(e2eMax),
      actual_fps: round2(actualFps),
      target_fps: this.targetFps,
      avg_frame_bytes: avgFrameBytes,
      bandwidth_bytes_per_sec: bandwidthBytesPerSec,
      primary_bottleneck: bottleneck,
      bottleneck_description: description,
    };
  }

  /**
   * Get the most recent frame timings.
   * @param limit Maximum number of frames to return (default: 10)
   */
  getRecentFrames(limit = 10): FrameTimings[] {
    const start = Math.max(0, this.timings.length - limit);
    return this.timings.slice(start);
  }

  /**
   * Reset all statistics (e.g., when session restarts).
   */
  reset(): void {
    this.timings.length = 0;
    this.frameCount = 0;
    this.skippedCount = 0;
    this.sequenceNum = 0;
  }
}

/**
 * Calculate the p-th percentile of a sorted array.
 * Uses linear interpolation between nearest ranks.
 */
function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0;
  if (sorted.length === 1) return sorted[0];

  const index = p * (sorted.length - 1);
  const lower = Math.floor(index);
  const upper = Math.ceil(index);

  if (lower === upper) return sorted[lower];

  const fraction = index - lower;
  return sorted[lower] * (1 - fraction) + sorted[upper] * fraction;
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

  // If E2E P90 > 150% of target frame time and capture is fine, it's likely network
  if (e2eP90 > targetFrameTime * 1.5 && captureP90 < targetFrameTime * 0.5) {
    const networkTime = e2eP90 - captureP90;
    return {
      bottleneck: 'network',
      description: `End-to-end P90 (${round2(e2eP90)}ms) indicates network latency. Estimated network overhead: ${round2(networkTime)}ms.`,
    };
  }

  return {
    bottleneck: 'none',
    description: 'No significant bottlenecks detected. Performance is within expected bounds.',
  };
}

/**
 * Export index for the performance module.
 */
export * from './types';
