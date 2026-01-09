/**
 * useUnifiedStats Hook
 *
 * Combines client-side frame statistics with server-side performance metrics
 * into a single unified view. This is the recommended hook for components
 * that need comprehensive streaming statistics.
 */

import { useMemo } from 'react';
import { useFrameStats } from '../hooks/useFrameStats';
import { usePerfStats } from '../hooks/usePerfStats';
import type { UnifiedStreamStats, BottleneckSeverity, BottleneckType } from './types';

interface UseUnifiedStatsOptions {
  /** Session ID for filtering server stats */
  sessionId: string | null;
  /** Whether server performance mode is enabled */
  serverStatsEnabled: boolean;
}

interface UseUnifiedStatsReturn {
  /** Unified statistics from both client and server */
  stats: UnifiedStreamStats;
  /** Record a new frame (updates client stats) */
  recordFrame: (frameSize: number) => void;
  /** Reset all statistics */
  reset: () => void;
  /** Get severity level for current bottleneck */
  bottleneckSeverity: BottleneckSeverity;
  /** Whether the streaming pipeline has a significant bottleneck */
  hasBottleneck: boolean;
}

/**
 * Combined hook for unified streaming statistics.
 *
 * Usage:
 * ```tsx
 * const { stats, recordFrame, bottleneckSeverity, hasBottleneck } = useUnifiedStats({
 *   sessionId,
 *   serverStatsEnabled: showStats,
 * });
 *
 * // In frame processing callback:
 * recordFrame(frameData.byteLength);
 *
 * // Display stats:
 * console.log(`Client FPS: ${stats.client.currentFps}`);
 * if (stats.server) {
 *   console.log(`Bottleneck: ${stats.server.primary_bottleneck}`);
 * }
 * ```
 */
export function useUnifiedStats({
  sessionId,
  serverStatsEnabled,
}: UseUnifiedStatsOptions): UseUnifiedStatsReturn {
  // Client-side frame statistics
  const {
    stats: clientStats,
    recordFrame,
    reset: resetClientStats,
  } = useFrameStats();

  // Server-side performance statistics
  const {
    stats: serverStats,
    isReceiving: isReceivingServerStats,
    reset: resetServerStats,
  } = usePerfStats(sessionId, serverStatsEnabled);

  // Combine into unified stats
  const stats = useMemo<UnifiedStreamStats>(
    () => ({
      client: clientStats,
      server: serverStats,
      serverStatsEnabled,
      isReceivingServerStats,
    }),
    [clientStats, serverStats, serverStatsEnabled, isReceivingServerStats]
  );

  // Calculate bottleneck severity
  const bottleneckSeverity = useMemo<BottleneckSeverity>(() => {
    if (!serverStats) return 'none';
    return getBottleneckSeverity(serverStats.primary_bottleneck);
  }, [serverStats]);

  // Check if there's a significant bottleneck
  const hasBottleneck = useMemo(() => {
    if (!serverStats) return false;
    return serverStats.primary_bottleneck !== 'none';
  }, [serverStats]);

  // Combined reset function
  const reset = () => {
    resetClientStats();
    resetServerStats();
  };

  return {
    stats,
    recordFrame,
    reset,
    bottleneckSeverity,
    hasBottleneck,
  };
}

/**
 * Get severity level for a bottleneck type.
 */
function getBottleneckSeverity(bottleneck: BottleneckType): BottleneckSeverity {
  switch (bottleneck) {
    case 'none':
      return 'none';
    case 'capture':
    case 'network':
      return 'critical'; // Most impactful bottlenecks
    case 'encode':
    case 'decode':
    case 'draw':
      return 'warning';
    default:
      return 'none';
  }
}
