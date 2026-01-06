/**
 * Frame Streaming Types
 *
 * Consolidated type definitions for the frame streaming system.
 * Includes both client-side statistics and server-side performance metrics.
 */

// =============================================================================
// Client-Side Frame Statistics
// =============================================================================

/**
 * Client-side frame statistics tracked locally.
 * Calculated from received frames without server round-trip.
 */
export interface FrameStats {
  /** Frames received in the last 1 second */
  currentFps: number;
  /** Average FPS over the last 5 seconds */
  avgFps: number;
  /** Size of last received frame in bytes */
  lastFrameSize: number;
  /** Average frame size in bytes over last 5 seconds */
  avgFrameSize: number;
  /** Total frames received since tracking started */
  totalFrames: number;
  /** Total bytes received since tracking started */
  totalBytes: number;
  /** Bandwidth in bytes per second (rolling average) */
  bytesPerSecond: number;
}

// =============================================================================
// Server-Side Performance Statistics
// =============================================================================

/** Possible bottleneck types in the streaming pipeline */
export type BottleneckType =
  | 'capture' // Screenshot capture is slow
  | 'encode' // JPEG encoding is slow (rare with Playwright)
  | 'network' // Network/WebSocket is slow
  | 'decode' // Client-side decoding is slow
  | 'draw' // Canvas rendering is slow
  | 'none'; // No significant bottleneck detected

/**
 * Aggregated server-side statistics over a window of frames.
 * Computed periodically by the server and sent via WebSocket.
 * Provides detailed pipeline timing for bottleneck identification.
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

// =============================================================================
// Unified Statistics
// =============================================================================

/**
 * Combined statistics from both client and server.
 * Provides a complete view of streaming performance.
 */
export interface UnifiedStreamStats {
  /** Client-side frame statistics */
  client: FrameStats;
  /** Server-side aggregated statistics (null if not enabled/received) */
  server: FrameStatsAggregated | null;
  /** Whether server performance mode is enabled */
  serverStatsEnabled: boolean;
  /** Whether we're actively receiving server stats */
  isReceivingServerStats: boolean;
}

// =============================================================================
// Stream Connection Status
// =============================================================================

/** Connection status for the live preview stream */
export interface StreamConnectionStatus {
  /** Whether frames are being received */
  isConnected: boolean;
  /** Whether connected via WebSocket (true) or polling fallback (false) */
  isWebSocket: boolean;
  /** Timestamp of last received frame */
  lastFrameTime?: string;
}

// =============================================================================
// Stream Settings
// =============================================================================

/** Stream quality settings */
export interface StreamSettings {
  /** JPEG quality (1-100) */
  quality: number;
  /** Target frames per second (1-60) */
  fps: number;
  /** Scale mode: 'css' = 1x scale, 'device' = device pixel ratio */
  scale: 'css' | 'device';
}

// =============================================================================
// Utility Types
// =============================================================================

/** Severity level for bottleneck display */
export type BottleneckSeverity = 'none' | 'warning' | 'critical';
