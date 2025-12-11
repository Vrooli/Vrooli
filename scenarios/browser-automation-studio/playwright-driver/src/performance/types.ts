/**
 * Performance Debug Mode Types
 *
 * Type definitions for frame streaming performance instrumentation.
 * These types are shared between the playwright-driver and Go API.
 *
 * ## Wire Format Stability
 * These types define the JSON contract for performance data.
 * - Adding new optional fields is safe (omitempty)
 * - Removing or renaming fields is a BREAKING CHANGE
 * - Changes require coordinated updates to api/performance/types.go
 */

/**
 * Timing data for a single frame capture.
 * Collected at each stage of the streaming pipeline.
 */
export interface FrameTimings {
  /** Unique frame identifier for correlation: "{sessionId}-{sequenceNum}" */
  frame_id: string;

  /** Session this frame belongs to */
  session_id: string;

  /** Frame sequence number within session (monotonically increasing) */
  sequence_num: number;

  /** ISO 8601 timestamp when capture started */
  timestamp: string;

  // Driver-side timings (all in milliseconds)

  /** Time to capture screenshot from Playwright (page.screenshot()) */
  capture_ms: number;

  /** Time to compare with previous frame buffer (deduplication check) */
  compare_ms: number;

  /** Time to send frame over WebSocket to API */
  ws_send_ms: number;

  /** Total driver-side processing time for this frame */
  driver_total_ms: number;

  // API-side timings (added by Go API, optional in driver)

  /** Time to receive frame from driver WebSocket */
  api_receive_ms?: number;

  /** Time to broadcast frame to all subscribed clients */
  api_broadcast_ms?: number;

  /** Total API-side processing time */
  api_total_ms?: number;

  // UI-side timings (added by React, for future use)

  /** Time to decode JPEG to ImageBitmap */
  decode_ms?: number;

  /** Time to draw ImageBitmap to canvas */
  draw_ms?: number;

  // Frame metadata

  /** Frame size in bytes (JPEG payload) */
  frame_bytes: number;

  /** Whether frame was skipped (identical to previous) */
  skipped: boolean;
}

/**
 * Binary frame header sent with each frame when perf mode is enabled.
 * This is a subset of FrameTimings - just the driver-side data.
 *
 * Wire format:
 * [4 bytes: header length (uint32 big-endian)]
 * [N bytes: JSON of FrameHeader]
 * [remaining: JPEG data]
 */
export interface FrameHeader {
  frame_id: string;
  capture_ms: number;
  compare_ms: number;
  ws_send_ms: number;
  frame_bytes: number;
}

/**
 * Aggregated statistics over a window of frames.
 * Computed periodically and sent to clients for visualization.
 */
export interface FrameStatsAggregated {
  /** Session these stats belong to */
  session_id: string;

  /** When this stats window started */
  window_start_time: string;

  /** Duration of the stats window in milliseconds */
  window_duration_ms: number;

  /** Total frames captured in this window */
  frame_count: number;

  /** Frames skipped due to unchanged content */
  skipped_count: number;

  // Capture timing percentiles (milliseconds)

  /** 50th percentile (median) capture time */
  capture_p50_ms: number;

  /** 90th percentile capture time */
  capture_p90_ms: number;

  /** 99th percentile capture time */
  capture_p99_ms: number;

  /** Maximum capture time observed */
  capture_max_ms: number;

  // End-to-end timing percentiles (driver capture start -> API broadcast complete)

  /** 50th percentile end-to-end time */
  e2e_p50_ms: number;

  /** 90th percentile end-to-end time */
  e2e_p90_ms: number;

  /** 99th percentile end-to-end time */
  e2e_p99_ms: number;

  /** Maximum end-to-end time observed */
  e2e_max_ms: number;

  // Throughput metrics

  /** Actual frames per second achieved */
  actual_fps: number;

  /** Target FPS configured for the session */
  target_fps: number;

  /** Average frame size in bytes */
  avg_frame_bytes: number;

  /** Bandwidth in bytes per second */
  bandwidth_bytes_per_sec: number;

  // Bottleneck identification

  /** Primary bottleneck identified */
  primary_bottleneck: BottleneckType;

  /** Human-readable description of the bottleneck */
  bottleneck_description: string;
}

/** Possible bottleneck types in the streaming pipeline */
export type BottleneckType =
  | 'capture' // Screenshot capture is slow
  | 'encode' // JPEG encoding is slow (rare with Playwright)
  | 'network' // Network/WebSocket is slow
  | 'decode' // Client-side decoding is slow
  | 'draw' // Canvas rendering is slow
  | 'none'; // No significant bottleneck detected

/**
 * WebSocket message type for performance stats broadcast.
 * Sent periodically from API to subscribed UI clients.
 */
export interface PerfStatsMessage {
  type: 'perf_stats';
  session_id: string;
  stats: FrameStatsAggregated;
}

/**
 * Response format for GET /debug/performance/{sessionId}
 */
export interface DebugPerformanceResponse {
  session_id: string;
  enabled: boolean;
  current_stats: FrameStatsAggregated | null;
  recent_frames: FrameTimings[];
}
