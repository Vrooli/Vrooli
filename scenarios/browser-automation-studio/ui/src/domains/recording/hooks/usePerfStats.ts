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
import { useWebSocketMessage } from '@/contexts/WebSocketContext';

import type { FrameStatsAggregated, BottleneckType } from '../frame-streaming/types';

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
  useWebSocketMessage((lastMessage) => {
    if (!enabled || !sessionId) return;

    const msg = lastMessage as unknown as PerfStatsMessage;

    if (msg.type === 'perf_stats' && msg.session_id === sessionId) {
      setStats(msg.stats);
      if (!hasReceivedStatsRef.current) {
        hasReceivedStatsRef.current = true;
        setIsReceiving(true);
      }
    }
  });

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
    case 'processing':
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
