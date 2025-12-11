/**
 * FrameStatsDisplay Component
 *
 * Displays frame streaming statistics as a compact inline badge
 * with an expanded tooltip on hover showing detailed metrics.
 *
 * Design:
 * - Inline badge: "6 fps · 42KB" - always visible, minimal footprint
 * - Hover tooltip: Detailed breakdown of all metrics
 * - Color coding: Green when hitting target FPS, yellow/red when degraded
 */

import { useMemo, useState } from 'react';
import type { FrameStats } from '../hooks/useFrameStats';
import { formatBytes, formatBandwidth } from '../hooks/useFrameStats';

interface FrameStatsDisplayProps {
  /** Current frame statistics */
  stats: FrameStats | null;
  /** Target FPS for comparison (from settings) */
  targetFps?: number;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Determine the color class based on how close actual FPS is to target.
 * - >= 80% of target: green
 * - >= 50% of target: yellow
 * - < 50% of target: red
 */
function getFpsColorClass(actualFps: number, targetFps: number): string {
  if (targetFps <= 0) return 'text-gray-500 dark:text-gray-400';
  const ratio = actualFps / targetFps;
  if (ratio >= 0.8) return 'text-green-600 dark:text-green-400';
  if (ratio >= 0.5) return 'text-yellow-600 dark:text-yellow-400';
  return 'text-red-600 dark:text-red-400';
}

export function FrameStatsDisplay({ stats, targetFps = 6, className = '' }: FrameStatsDisplayProps) {
  const [isHovering, setIsHovering] = useState(false);

  // Memoize formatted values to avoid recalculation on every render
  const formattedStats = useMemo(() => {
    if (!stats) {
      return {
        fps: '—',
        avgFps: '—',
        frameSize: '—',
        avgFrameSize: '—',
        bandwidth: '—',
        totalFrames: '—',
        totalBytes: '—',
      };
    }
    return {
      fps: stats.currentFps.toString(),
      avgFps: stats.avgFps.toFixed(1),
      frameSize: formatBytes(stats.lastFrameSize),
      avgFrameSize: formatBytes(stats.avgFrameSize),
      bandwidth: formatBandwidth(stats.bytesPerSecond),
      totalFrames: stats.totalFrames.toLocaleString(),
      totalBytes: formatBytes(stats.totalBytes),
    };
  }, [stats]);

  const fpsColorClass = stats ? getFpsColorClass(stats.avgFps, targetFps) : 'text-gray-500 dark:text-gray-400';

  // Don't show anything if we haven't received any frames
  if (!stats || stats.totalFrames === 0) {
    return (
      <div className={`text-xs text-gray-400 dark:text-gray-500 ${className}`}>
        <span className="opacity-50">— fps</span>
      </div>
    );
  }

  return (
    <div
      className={`relative ${className}`}
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
    >
      {/* Inline badge - always visible */}
      <div className="flex items-center gap-1.5 text-xs cursor-help select-none">
        <span className={`font-medium tabular-nums ${fpsColorClass}`}>{formattedStats.avgFps} fps</span>
        <span className="text-gray-400 dark:text-gray-500">·</span>
        <span className="text-gray-500 dark:text-gray-400 tabular-nums">{formattedStats.avgFrameSize}</span>
      </div>

      {/* Expanded tooltip on hover */}
      {isHovering && (
        <div className="absolute right-0 top-full mt-2 z-50 w-56 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
          {/* Header */}
          <div className="px-3 py-2 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
            <p className="text-xs font-medium text-gray-600 dark:text-gray-300">Stream Performance</p>
          </div>

          {/* Stats grid */}
          <div className="p-3 space-y-2">
            {/* FPS section */}
            <div className="flex justify-between items-center">
              <span className="text-xs text-gray-500 dark:text-gray-400">Current FPS</span>
              <span className={`text-xs font-medium tabular-nums ${fpsColorClass}`}>
                {formattedStats.fps} / {targetFps}
              </span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-xs text-gray-500 dark:text-gray-400">Avg FPS (5s)</span>
              <span className={`text-xs font-medium tabular-nums ${fpsColorClass}`}>{formattedStats.avgFps}</span>
            </div>

            <div className="border-t border-gray-100 dark:border-gray-700 my-2" />

            {/* Size section */}
            <div className="flex justify-between items-center">
              <span className="text-xs text-gray-500 dark:text-gray-400">Last Frame</span>
              <span className="text-xs font-medium tabular-nums text-gray-700 dark:text-gray-200">
                {formattedStats.frameSize}
              </span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-xs text-gray-500 dark:text-gray-400">Avg Frame</span>
              <span className="text-xs font-medium tabular-nums text-gray-700 dark:text-gray-200">
                {formattedStats.avgFrameSize}
              </span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-xs text-gray-500 dark:text-gray-400">Bandwidth</span>
              <span className="text-xs font-medium tabular-nums text-gray-700 dark:text-gray-200">
                {formattedStats.bandwidth}
              </span>
            </div>

            <div className="border-t border-gray-100 dark:border-gray-700 my-2" />

            {/* Totals section */}
            <div className="flex justify-between items-center">
              <span className="text-xs text-gray-500 dark:text-gray-400">Total Frames</span>
              <span className="text-xs tabular-nums text-gray-600 dark:text-gray-300">
                {formattedStats.totalFrames}
              </span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-xs text-gray-500 dark:text-gray-400">Total Data</span>
              <span className="text-xs tabular-nums text-gray-600 dark:text-gray-300">
                {formattedStats.totalBytes}
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
