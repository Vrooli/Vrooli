/**
 * usePerfStats Hook
 *
 * Receives and manages debug performance statistics from the server.
 * Stats are sent via WebSocket when debug performance mode is enabled.
 *
 * These stats provide pipeline timing data (capture, compare, send, receive, broadcast)
 * that complement the client-side frame stats from useFrameStats.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { useWebSocket } from '@/contexts/WebSocketContext';

/** Possible bottleneck types in the streaming pipeline */
export type BottleneckType =
  | 'capture' // Screenshot capture is slow
  | 'encode' // JPEG encoding is slow (rare with Playwright)
  | 'network' // Network/WebSocket is slow
  | 'decode' // Client-side decoding is slow
  | 'draw' // Canvas rendering is slow
  | 'none'; // No significant bottleneck detected

/**
 * Aggregated statistics over a window of frames.
 * Computed periodically by the server and sent via WebSocket.
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

/**
 * WebSocket message type for performance stats broadcast.
 */
interface PerfStatsMessage {
  type: 'perf_stats';
  session_id: string;
  stats: FrameStatsAggregated;
}

/**
 * Hook for receiving debug performance statistics from the server.
 *
 * Usage:
 * ```tsx
 * const { stats, isEnabled } = usePerfStats(sessionId, debugPerfModeEnabled);
 *
 * // Access stats when available:
 * if (stats) {
 *   console.log(`Capture P50: ${stats.capture_p50_ms}ms`);
 *   console.log(`Bottleneck: ${stats.primary_bottleneck}`);
 * }
 * ```
 *
 * @param sessionId - The session ID to filter stats for
 * @param enabled - Whether debug performance mode is enabled
 */
export function usePerfStats(sessionId: string | null, enabled: boolean) {
  const [stats, setStats] = useState<FrameStatsAggregated | null>(null);
  const { lastMessage } = useWebSocket();

  // Track if we've received any stats (indicates server has perf mode active)
  const hasReceivedStatsRef = useRef(false);
  const [isReceiving, setIsReceiving] = useState(false);

  // Reset stats when session changes or mode is disabled
  useEffect(() => {
    if (!enabled || !sessionId) {
      setStats(null);
      hasReceivedStatsRef.current = false;
      setIsReceiving(false);
    }
  }, [enabled, sessionId]);

  // Handle incoming perf_stats messages
  useEffect(() => {
    if (!lastMessage || !enabled || !sessionId) return;

    const msg = lastMessage as unknown as PerfStatsMessage;

    if (msg.type === 'perf_stats' && msg.session_id === sessionId) {
      setStats(msg.stats);
      if (!hasReceivedStatsRef.current) {
        hasReceivedStatsRef.current = true;
        setIsReceiving(true);
      }
    }
  }, [lastMessage, enabled, sessionId]);

  /**
   * Reset stats (e.g., when starting a new recording).
   */
  const reset = useCallback(() => {
    setStats(null);
    hasReceivedStatsRef.current = false;
    setIsReceiving(false);
  }, []);

  return {
    /** Current aggregated stats, or null if not yet received */
    stats,
    /** Whether debug perf mode is enabled */
    isEnabled: enabled,
    /** Whether we're actively receiving stats from server */
    isReceiving,
    /** Reset stats */
    reset,
  };
}

/**
 * Get severity level for a bottleneck type.
 * Used for color coding in the UI.
 */
export function getBottleneckSeverity(
  bottleneck: BottleneckType
): 'none' | 'warning' | 'critical' {
  switch (bottleneck) {
    case 'none':
      return 'none';
    case 'capture':
    case 'network':
      return 'critical'; // These are the most impactful bottlenecks
    case 'encode':
    case 'decode':
    case 'draw':
      return 'warning';
    default:
      return 'none';
  }
}

/**
 * Format milliseconds for display.
 */
export function formatMs(ms: number): string {
  if (ms < 1) return '<1ms';
  if (ms < 10) return `${ms.toFixed(1)}ms`;
  return `${Math.round(ms)}ms`;
}
