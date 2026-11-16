/**
 * useFrameStats Hook
 *
 * Tracks frame delivery statistics for the live preview stream.
 * All metrics are calculated client-side from received frames.
 *
 * PERFORMANCE: This hook is optimized for high-frequency frame updates (30-60 FPS).
 * It uses ref-based tracking internally and only updates React state every second
 * to prevent React reconciliation on every frame, which was causing FPS drops.
 *
 * Metrics tracked:
 * - FPS (current and rolling average)
 * - Frame size (last and rolling average)
 * - Total frames and bytes received
 */

import { useCallback, useEffect, useRef, useState } from 'react';

/** Frame statistics snapshot */
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

/** Entry in the frame history ring buffer */
interface FrameRecord {
  timestamp: number; // ms since epoch
  size: number; // bytes
}

const HISTORY_WINDOW_MS = 5000; // 5 seconds of history
const CURRENT_WINDOW_MS = 1000; // 1 second for "current" FPS
const STATE_UPDATE_INTERVAL_MS = 1000; // Keep UI stats readable without pushing page-wide renders.
const COMPACT_HEAD_THRESHOLD = 256;

/**
 * Hook for tracking frame delivery statistics.
 *
 * Usage:
 * ```tsx
 * const { stats, recordFrame, reset } = useFrameStats();
 *
 * // Call when a frame is received:
 * recordFrame(frameBlob.size);
 *
 * // Access stats:
 * console.log(`FPS: ${stats.currentFps}, Avg: ${stats.avgFps}`);
 * ```
 */
export function useFrameStats() {
  const [stats, setStats] = useState<FrameStats>({
    currentFps: 0,
    avgFps: 0,
    lastFrameSize: 0,
    avgFrameSize: 0,
    totalFrames: 0,
    totalBytes: 0,
    bytesPerSecond: 0,
  });

  // Ring buffer of frame records (timestamp + size)
  const historyRef = useRef<FrameRecord[]>([]);
  const historyHeadRef = useRef(0);
  const totalFramesRef = useRef(0);
  const totalBytesRef = useRef(0);
  const lastFrameSizeRef = useRef(0);

  // Track if stats have changed since last state update
  const statsDirtyRef = useRef(false);

  /**
   * Calculate current statistics from history.
   * Pure function - doesn't update any state.
   */
  const calculateStats = useCallback((): FrameStats => {
    const now = Date.now();
    const history = historyRef.current;

    // Calculate statistics
    const currentCutoff = now - CURRENT_WINDOW_MS;

    // Frames in last 1 second (for current FPS)
    let currentFrameCount = 0;
    // Frames in last 5 seconds (for average)
    let totalSize = 0;

    for (let index = historyHeadRef.current; index < history.length; index += 1) {
      const record = history[index];
      if (!record) continue;
      totalSize += record.size;
      if (record.timestamp >= currentCutoff) {
        currentFrameCount += 1;
      }
    }

    // Calculate averages
    const activeLength = history.length - historyHeadRef.current;
    const firstEntry = history[historyHeadRef.current];
    const historyDuration = activeLength > 1 && firstEntry ? (now - firstEntry.timestamp) / 1000 : 1;
    const avgFps = activeLength / Math.max(historyDuration, 0.1);
    const avgFrameSize = activeLength > 0 ? totalSize / activeLength : 0;
    const bytesPerSecond = historyDuration > 0 ? totalSize / historyDuration : 0;

    return {
      currentFps: currentFrameCount,
      avgFps: Math.round(avgFps * 10) / 10, // 1 decimal place
      lastFrameSize: lastFrameSizeRef.current,
      avgFrameSize: Math.round(avgFrameSize),
      totalFrames: totalFramesRef.current,
      totalBytes: totalBytesRef.current,
      bytesPerSecond: Math.round(bytesPerSecond),
    };
  }, []);

  /**
   * Record a new frame.
   * Updates refs immediately but batches React state updates.
   * @param frameSize Size of the frame in bytes
   */
  const recordFrame = useCallback((frameSize: number) => {
    const now = Date.now();
    const history = historyRef.current;

    // Add new frame to history (ref update - no React render)
    history.push({ timestamp: now, size: frameSize });

    // Update totals (ref updates - no React render)
    totalFramesRef.current += 1;
    totalBytesRef.current += frameSize;
    lastFrameSizeRef.current = frameSize;

    // Prune old entries (older than HISTORY_WINDOW_MS)
    const cutoff = now - HISTORY_WINDOW_MS;
    while (historyHeadRef.current < history.length) {
      const firstHistoryEntry = history[historyHeadRef.current];
      if (!firstHistoryEntry || firstHistoryEntry.timestamp >= cutoff) break;
      historyHeadRef.current += 1;
    }

    // Pruning advances a cursor in O(1) per expired frame. Compact only after
    // the inactive prefix is substantial so steady high-rate streams do not
    // shift the entire retained window on every incoming frame.
    if (historyHeadRef.current >= COMPACT_HEAD_THRESHOLD && historyHeadRef.current * 2 >= history.length) {
      historyRef.current = history.slice(historyHeadRef.current);
      historyHeadRef.current = 0;
    }

    // Mark stats as dirty for the next interval update
    statsDirtyRef.current = true;
  }, []);

  /**
   * Periodically update React state from refs.
   * This batches updates to prevent React reconciliation on every frame.
   */
  useEffect(() => {
    const intervalId = setInterval(() => {
      if (statsDirtyRef.current) {
        statsDirtyRef.current = false;
        setStats(calculateStats());
      }
    }, STATE_UPDATE_INTERVAL_MS);

    return () => clearInterval(intervalId);
  }, [calculateStats]);

  /**
   * Reset all statistics (e.g., when session changes).
   */
  const reset = useCallback(() => {
    historyRef.current = [];
    historyHeadRef.current = 0;
    totalFramesRef.current = 0;
    totalBytesRef.current = 0;
    lastFrameSizeRef.current = 0;
    statsDirtyRef.current = false;
    setStats({
      currentFps: 0,
      avgFps: 0,
      lastFrameSize: 0,
      avgFrameSize: 0,
      totalFrames: 0,
      totalBytes: 0,
      bytesPerSecond: 0,
    });
  }, []);

  return { stats, recordFrame, reset };
}

/**
 * Format bytes as human-readable string.
 * @param bytes Number of bytes
 * @param decimals Number of decimal places (default: 1)
 */
export function formatBytes(bytes: number, decimals = 1): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const value = bytes / Math.pow(k, i);
  return `${value.toFixed(decimals)} ${sizes[i]}`;
}

/**
 * Format bytes per second as bandwidth string.
 * @param bytesPerSecond Bytes per second
 */
export function formatBandwidth(bytesPerSecond: number): string {
  return `${formatBytes(bytesPerSecond)}/s`;
}
