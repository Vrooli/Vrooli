/**
 * useFrameStats Hook
 *
 * Tracks frame delivery statistics for the live preview stream.
 * All metrics are calculated client-side from received frames.
 *
 * Metrics tracked:
 * - FPS (current and rolling average)
 * - Frame size (last and rolling average)
 * - Total frames and bytes received
 */

import { useCallback, useRef, useState } from 'react';

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
  const totalFramesRef = useRef(0);
  const totalBytesRef = useRef(0);

  /**
   * Record a new frame and update statistics.
   * @param frameSize Size of the frame in bytes
   */
  const recordFrame = useCallback((frameSize: number) => {
    const now = Date.now();
    const history = historyRef.current;

    // Add new frame to history
    history.push({ timestamp: now, size: frameSize });

    // Update totals
    totalFramesRef.current += 1;
    totalBytesRef.current += frameSize;

    // Prune old entries (older than HISTORY_WINDOW_MS)
    const cutoff = now - HISTORY_WINDOW_MS;
    while (history.length > 0 && history[0].timestamp < cutoff) {
      history.shift();
    }

    // Calculate statistics
    const currentCutoff = now - CURRENT_WINDOW_MS;

    // Frames in last 1 second (for current FPS)
    let currentFrameCount = 0;
    // Frames in last 5 seconds (for average)
    let totalSize = 0;

    for (const record of history) {
      totalSize += record.size;
      if (record.timestamp >= currentCutoff) {
        currentFrameCount += 1;
      }
    }

    // Calculate averages
    const historyDuration = history.length > 1 ? (now - history[0].timestamp) / 1000 : 1;
    const avgFps = history.length / Math.max(historyDuration, 0.1);
    const avgFrameSize = history.length > 0 ? totalSize / history.length : 0;
    const bytesPerSecond = historyDuration > 0 ? totalSize / historyDuration : 0;

    setStats({
      currentFps: currentFrameCount,
      avgFps: Math.round(avgFps * 10) / 10, // 1 decimal place
      lastFrameSize: frameSize,
      avgFrameSize: Math.round(avgFrameSize),
      totalFrames: totalFramesRef.current,
      totalBytes: totalBytesRef.current,
      bytesPerSecond: Math.round(bytesPerSecond),
    });
  }, []);

  /**
   * Reset all statistics (e.g., when session changes).
   */
  const reset = useCallback(() => {
    historyRef.current = [];
    totalFramesRef.current = 0;
    totalBytesRef.current = 0;
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
