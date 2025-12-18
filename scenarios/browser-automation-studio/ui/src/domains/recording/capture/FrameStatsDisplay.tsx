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
 *
 * When debug performance mode is enabled, the tooltip also shows:
 * - Pipeline timing breakdown (capture, network, etc.)
 * - Percentile latencies (P50/P90/P99)
 * - Bottleneck identification with color coding
 */

import { useMemo, useState } from 'react';
import type { FrameStats } from '../hooks/useFrameStats';
import { formatBytes, formatBandwidth } from '../hooks/useFrameStats';
import type { FrameStatsAggregated, BottleneckType } from '../hooks/usePerfStats';
import { getBottleneckSeverity, formatMs } from '../hooks/usePerfStats';

interface FrameStatsDisplayProps {
  /** Current frame statistics (client-side) */
  stats: FrameStats | null;
  /** Target FPS for comparison (from settings) */
  targetFps?: number;
  /** Debug performance stats from server (when perf mode enabled) */
  debugStats?: FrameStatsAggregated | null;
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

/**
 * Get display info for a bottleneck type.
 */
function getBottleneckDisplay(bottleneck: BottleneckType): {
  label: string;
  colorClass: string;
  bgClass: string;
} {
  const severity = getBottleneckSeverity(bottleneck);

  const labels: Record<BottleneckType, string> = {
    capture: 'Capture',
    encode: 'Encode',
    network: 'Network',
    decode: 'Decode',
    draw: 'Draw',
    none: 'None',
  };

  const colorClasses: Record<typeof severity, string> = {
    none: 'text-green-600 dark:text-green-400',
    warning: 'text-yellow-600 dark:text-yellow-400',
    critical: 'text-red-600 dark:text-red-400',
  };

  const bgClasses: Record<typeof severity, string> = {
    none: 'bg-green-100 dark:bg-green-900/30',
    warning: 'bg-yellow-100 dark:bg-yellow-900/30',
    critical: 'bg-red-100 dark:bg-red-900/30',
  };

  return {
    label: labels[bottleneck],
    colorClass: colorClasses[severity],
    bgClass: bgClasses[severity],
  };
}

export function FrameStatsDisplay({
  stats,
  targetFps = 6,
  debugStats,
  className = '',
}: FrameStatsDisplayProps) {
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

  // Memoize debug stats formatting
  const formattedDebugStats = useMemo(() => {
    if (!debugStats) return null;

    const bottleneckDisplay = getBottleneckDisplay(debugStats.primary_bottleneck);

    return {
      // Capture percentiles
      captureP50: formatMs(debugStats.capture_p50_ms),
      captureP90: formatMs(debugStats.capture_p90_ms),
      captureP99: formatMs(debugStats.capture_p99_ms),
      // E2E percentiles
      e2eP50: formatMs(debugStats.e2e_p50_ms),
      e2eP90: formatMs(debugStats.e2e_p90_ms),
      e2eP99: formatMs(debugStats.e2e_p99_ms),
      // Throughput
      actualFps: debugStats.actual_fps.toFixed(1),
      skippedCount: debugStats.skipped_count,
      frameCount: debugStats.frame_count,
      skippedPercent:
        debugStats.frame_count > 0
          ? Math.round((debugStats.skipped_count / debugStats.frame_count) * 100)
          : 0,
      // Bottleneck
      bottleneck: bottleneckDisplay,
      bottleneckDescription: debugStats.bottleneck_description,
    };
  }, [debugStats]);

  const fpsColorClass = stats
    ? getFpsColorClass(stats.avgFps, targetFps)
    : 'text-gray-500 dark:text-gray-400';

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
        <span className={`font-medium tabular-nums ${fpsColorClass}`}>
          {formattedStats.avgFps} fps
        </span>
        <span className="text-gray-400 dark:text-gray-500">·</span>
        <span className="text-gray-500 dark:text-gray-400 tabular-nums">
          {formattedStats.avgFrameSize}
        </span>
        {/* Show debug indicator when perf stats available */}
        {formattedDebugStats && (
          <>
            <span className="text-gray-400 dark:text-gray-500">·</span>
            <span
              className={`px-1 rounded text-[10px] font-medium ${formattedDebugStats.bottleneck.bgClass} ${formattedDebugStats.bottleneck.colorClass}`}
            >
              {formattedDebugStats.bottleneck.label}
            </span>
          </>
        )}
      </div>

      {/* Expanded tooltip on hover */}
      {isHovering && (
        <div
          className={`absolute right-0 top-full mt-2 z-50 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 overflow-hidden ${formattedDebugStats ? 'w-72' : 'w-56'}`}
        >
          {/* Header */}
          <div className="px-3 py-2 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
            <p className="text-xs font-medium text-gray-600 dark:text-gray-300">
              Stream Performance
            </p>
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
              <span className={`text-xs font-medium tabular-nums ${fpsColorClass}`}>
                {formattedStats.avgFps}
              </span>
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

            {/* Debug Performance Section - only shown when debugStats available */}
            {formattedDebugStats && (
              <>
                <div className="border-t border-gray-100 dark:border-gray-700 my-2" />

                {/* Debug header */}
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-xs font-medium text-orange-600 dark:text-orange-400">
                    Pipeline Debug
                  </span>
                  <span className="flex-1 h-px bg-orange-200 dark:bg-orange-800" />
                </div>

                {/* Bottleneck indicator */}
                <div className="flex justify-between items-center">
                  <span className="text-xs text-gray-500 dark:text-gray-400">Bottleneck</span>
                  <span
                    className={`text-xs font-medium px-1.5 py-0.5 rounded ${formattedDebugStats.bottleneck.bgClass} ${formattedDebugStats.bottleneck.colorClass}`}
                  >
                    {formattedDebugStats.bottleneck.label}
                  </span>
                </div>
                {formattedDebugStats.bottleneckDescription &&
                  formattedDebugStats.bottleneck.label !== 'None' && (
                    <p className="text-[10px] text-gray-500 dark:text-gray-400 italic pl-1">
                      {formattedDebugStats.bottleneckDescription}
                    </p>
                  )}

                {/* Capture latency */}
                <div className="mt-2">
                  <div className="flex justify-between items-center text-xs text-gray-500 dark:text-gray-400 mb-1">
                    <span>Capture Latency</span>
                  </div>
                  <div className="flex gap-2 text-xs">
                    <div className="flex-1 bg-gray-50 dark:bg-gray-900/50 rounded px-2 py-1">
                      <span className="text-gray-400 dark:text-gray-500 text-[10px]">P50</span>
                      <span className="ml-1 font-medium tabular-nums text-gray-700 dark:text-gray-200">
                        {formattedDebugStats.captureP50}
                      </span>
                    </div>
                    <div className="flex-1 bg-gray-50 dark:bg-gray-900/50 rounded px-2 py-1">
                      <span className="text-gray-400 dark:text-gray-500 text-[10px]">P90</span>
                      <span className="ml-1 font-medium tabular-nums text-gray-700 dark:text-gray-200">
                        {formattedDebugStats.captureP90}
                      </span>
                    </div>
                    <div className="flex-1 bg-gray-50 dark:bg-gray-900/50 rounded px-2 py-1">
                      <span className="text-gray-400 dark:text-gray-500 text-[10px]">P99</span>
                      <span className="ml-1 font-medium tabular-nums text-gray-700 dark:text-gray-200">
                        {formattedDebugStats.captureP99}
                      </span>
                    </div>
                  </div>
                </div>

                {/* E2E latency */}
                <div className="mt-2">
                  <div className="flex justify-between items-center text-xs text-gray-500 dark:text-gray-400 mb-1">
                    <span>End-to-End Latency</span>
                  </div>
                  <div className="flex gap-2 text-xs">
                    <div className="flex-1 bg-gray-50 dark:bg-gray-900/50 rounded px-2 py-1">
                      <span className="text-gray-400 dark:text-gray-500 text-[10px]">P50</span>
                      <span className="ml-1 font-medium tabular-nums text-gray-700 dark:text-gray-200">
                        {formattedDebugStats.e2eP50}
                      </span>
                    </div>
                    <div className="flex-1 bg-gray-50 dark:bg-gray-900/50 rounded px-2 py-1">
                      <span className="text-gray-400 dark:text-gray-500 text-[10px]">P90</span>
                      <span className="ml-1 font-medium tabular-nums text-gray-700 dark:text-gray-200">
                        {formattedDebugStats.e2eP90}
                      </span>
                    </div>
                    <div className="flex-1 bg-gray-50 dark:bg-gray-900/50 rounded px-2 py-1">
                      <span className="text-gray-400 dark:text-gray-500 text-[10px]">P99</span>
                      <span className="ml-1 font-medium tabular-nums text-gray-700 dark:text-gray-200">
                        {formattedDebugStats.e2eP99}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Skipped frames */}
                <div className="flex justify-between items-center mt-2">
                  <span className="text-xs text-gray-500 dark:text-gray-400">Frames Skipped</span>
                  <span className="text-xs tabular-nums text-gray-600 dark:text-gray-300">
                    {formattedDebugStats.skippedCount} ({formattedDebugStats.skippedPercent}%)
                  </span>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
