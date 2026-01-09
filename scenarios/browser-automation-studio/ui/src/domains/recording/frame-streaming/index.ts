/**
 * Frame Streaming Module
 *
 * Consolidated exports for frame streaming functionality.
 * This module provides:
 * - Client-side frame statistics (useFrameStats)
 * - Server-side performance metrics (usePerfStats)
 * - Unified combined statistics (useUnifiedStats)
 * - Type definitions for all streaming concepts
 *
 * Usage:
 * ```tsx
 * import {
 *   useUnifiedStats,
 *   useFrameStats,
 *   usePerfStats,
 *   type FrameStats,
 *   type UnifiedStreamStats,
 * } from '@/domains/recording/frame-streaming';
 * ```
 */

// Types
export type {
  FrameStats,
  FrameStatsAggregated,
  BottleneckType,
  UnifiedStreamStats,
  StreamConnectionStatus,
  StreamSettings,
  BottleneckSeverity,
} from './types';

// Unified hook (recommended)
export { useUnifiedStats } from './useUnifiedStats';

// Individual hooks (for specific use cases)
export { useFrameStats, formatBytes, formatBandwidth } from '../hooks/useFrameStats';
export { usePerfStats, getBottleneckSeverity, formatMs } from '../hooks/usePerfStats';
