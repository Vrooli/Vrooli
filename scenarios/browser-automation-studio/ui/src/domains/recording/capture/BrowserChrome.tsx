/**
 * BrowserChrome Component
 *
 * The browser chrome UI (URL bar, frame stats, replay style toggle, settings)
 * that appears above the browser viewport. Used in both recording and execution modes.
 *
 * The Replay Style and Settings buttons are always visible regardless of mode.
 * Frame stats are shown when live streaming (recording mode with showStats=true).
 */

import { Palette, SlidersHorizontal } from 'lucide-react';
import { BrowserUrlBar } from './BrowserUrlBar';
import { FrameStatsDisplay } from './FrameStatsDisplay';
import type { FrameStats } from '../hooks/useFrameStats';
import type { FrameStatsAggregated } from '../hooks/usePerfStats';
import clsx from 'clsx';

export type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

interface BrowserChromeProps {
  // URL bar
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  onNavigate?: (url: string) => void;
  onRefresh?: () => void;
  pageTitle?: string;
  placeholder?: string;

  // Frame stats (optional - shown during live streaming)
  frameStats?: FrameStats | null;
  targetFps?: number;
  debugStats?: FrameStatsAggregated | null;
  showStats?: boolean;

  // Replay style toggle (always visible)
  showReplayStyleToggle?: boolean;
  showReplayStyle?: boolean;
  onReplayStyleToggle?: () => void;

  // Settings (always visible)
  onSettingsClick?: () => void;

  // Mode context (for URL bar behavior)
  mode?: 'recording' | 'execution';
  executionStatus?: ExecutionStatus;

  // Read-only (execution playback - disables URL editing)
  readOnly?: boolean;

  // Additional class name
  className?: string;
}

export function BrowserChrome({
  previewUrl,
  onPreviewUrlChange,
  onNavigate,
  onRefresh,
  pageTitle,
  placeholder = 'Search or enter URL',
  frameStats,
  targetFps,
  debugStats,
  showStats = false,
  showReplayStyleToggle = true,
  showReplayStyle = false,
  onReplayStyleToggle,
  onSettingsClick,
  readOnly = false,
  className,
}: BrowserChromeProps) {
  // In read-only mode, we don't allow navigation or refresh
  const handleNavigate = readOnly ? undefined : onNavigate;
  const handleRefresh = readOnly ? undefined : onRefresh;

  // Default handlers for URL bar when read-only
  const handleUrlChange = readOnly ? () => {} : onPreviewUrlChange;
  const noopNavigate = () => {};
  const noopRefresh = () => {};

  return (
    <div className={clsx(
      'border-b border-gray-200 dark:border-gray-700 px-4 py-2.5 flex items-center gap-2',
      className
    )}>
      {/* Browser URL bar */}
      <BrowserUrlBar
        value={previewUrl}
        onChange={handleUrlChange}
        onNavigate={handleNavigate ?? noopNavigate}
        onRefresh={handleRefresh ?? noopRefresh}
        placeholder={placeholder}
        pageTitle={pageTitle}
        disabled={readOnly}
      />

      {/* Frame stats display (shown during live streaming) */}
      {showStats && (
        <FrameStatsDisplay
          stats={frameStats ?? null}
          targetFps={targetFps}
          debugStats={debugStats ?? null}
        />
      )}

      {/* Replay style toggle */}
      {showReplayStyleToggle && onReplayStyleToggle && (
        <button
          type="button"
          role="switch"
          aria-checked={showReplayStyle}
          onClick={onReplayStyleToggle}
          className={clsx(
            'flex items-center gap-2 px-2 py-1.5 text-xs rounded-lg border transition-colors',
            showReplayStyle
              ? 'border-flow-accent/50 bg-flow-accent/10 text-flow-accent'
              : 'border-gray-700 bg-gray-800/70 text-gray-300 hover:text-surface',
          )}
          title="Toggle replay styling for the live preview"
        >
          <Palette size={14} />
          Replay Style
        </button>
      )}

      {/* Settings button */}
      {onSettingsClick && (
        <button
          type="button"
          onClick={onSettingsClick}
          className="flex items-center gap-2 px-2 py-1.5 text-xs text-gray-300 hover:text-surface bg-gray-800/70 border border-gray-700 rounded-lg transition-colors"
          title="Configure stream and replay settings"
        >
          <SlidersHorizontal size={14} />
          Settings
        </button>
      )}
    </div>
  );
}
