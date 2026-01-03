/**
 * BrowserChrome Component
 *
 * The browser chrome UI (URL bar, frame stats, replay style toggle, settings)
 * that appears above the browser viewport. Used in both recording and execution modes.
 *
 * The Replay Style and Settings buttons are always visible regardless of mode.
 * Frame stats are shown when live streaming (recording mode with showStats=true).
 */

import { Palette, SlidersHorizontal, PanelLeft } from 'lucide-react';
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

  // Left sidebar toggle
  isSidebarOpen?: boolean;
  onToggleSidebar?: () => void;
  actionCount?: number;

  // Frame stats (optional - shown during live streaming)
  frameStats?: FrameStats | null;
  targetFps?: number;
  debugStats?: FrameStatsAggregated | null;
  showStats?: boolean;

  // Replay style toggle (always visible)
  showReplayStyleToggle?: boolean;
  showReplayStyle?: boolean;
  onReplayStyleToggle?: () => void;

  // Settings panel
  onSettingsClick?: () => void;
  /** Whether the settings panel is open (hides the settings button when true) */
  isSettingsPanelOpen?: boolean;

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
  isSidebarOpen,
  onToggleSidebar,
  actionCount = 0,
  frameStats,
  targetFps,
  debugStats,
  showStats = false,
  showReplayStyleToggle = true,
  showReplayStyle = false,
  onReplayStyleToggle,
  onSettingsClick,
  isSettingsPanelOpen = false,
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
      'border-b border-gray-200 dark:border-gray-700 px-4 py-2.5 flex items-center gap-2 min-h-[52px]',
      className
    )}>
      {/* Left sidebar toggle - hidden when sidebar is open (sidebar has its own close button) */}
      {onToggleSidebar && !isSidebarOpen && (
        <button
          onClick={onToggleSidebar}
          className="relative flex-shrink-0 p-1.5 rounded-lg border border-gray-700 bg-gray-800/70 text-gray-300 hover:text-white transition-colors"
          title="Show timeline"
        >
          <PanelLeft size={16} />
          {actionCount > 0 && (
            <span className="absolute -top-1.5 -right-1.5 inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 text-[10px] font-semibold text-white bg-blue-500 rounded-full">
              {actionCount > 99 ? '99+' : actionCount}
            </span>
          )}
        </button>
      )}

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
          data-testid="browser-replay-style-button"
          type="button"
          role="switch"
          aria-checked={showReplayStyle}
          aria-label="Toggle replay style"
          onClick={onReplayStyleToggle}
          className={clsx(
            'flex-shrink-0 p-1.5 rounded-lg border transition-colors',
            showReplayStyle
              ? 'border-flow-accent/50 bg-flow-accent/10 text-flow-accent'
              : 'border-gray-700 bg-gray-800/70 text-gray-300 hover:text-white',
          )}
          title="Toggle replay styling for the live preview"
        >
          <Palette size={16} />
        </button>
      )}

      {/* Settings button - hidden when settings panel is open (panel has its own close button) */}
      {onSettingsClick && !isSettingsPanelOpen && (
        <button
          type="button"
          aria-label="Preview settings"
          onClick={onSettingsClick}
          className="flex-shrink-0 p-1.5 text-gray-300 hover:text-white bg-gray-800/70 border border-gray-700 rounded-lg transition-colors"
          title="Configure stream and replay settings"
        >
          <SlidersHorizontal size={16} />
        </button>
      )}
    </div>
  );
}
