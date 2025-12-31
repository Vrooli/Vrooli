/**
 * BrowserChrome Component
 *
 * The browser chrome UI (URL bar, frame stats, settings) that appears
 * above the browser viewport. Extracted from RecordPreviewPanel to be
 * reusable in both recording and execution modes.
 */

import { Palette, SlidersHorizontal, CheckCircle, Loader2, XCircle, Clock, AlertCircle } from 'lucide-react';
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

  // Frame stats (optional)
  frameStats?: FrameStats | null;
  targetFps?: number;
  debugStats?: FrameStatsAggregated | null;
  showStats?: boolean;

  // Replay style toggle
  showReplayStyleToggle?: boolean;
  showReplayStyle?: boolean;
  onReplayStyleToggle?: () => void;

  // Settings
  onSettingsClick?: () => void;

  // Mode context
  mode?: 'recording' | 'execution';
  executionStatus?: ExecutionStatus;

  // Read-only (execution playback)
  readOnly?: boolean;

  // Additional class name
  className?: string;
}

/** Status indicator for execution mode */
function ExecutionStatusIndicator({ status }: { status: ExecutionStatus }) {
  const config: Record<ExecutionStatus, { icon: typeof Clock; text: string; className: string; spin?: boolean }> = {
    pending: { icon: Clock, text: 'Pending', className: 'text-gray-500 dark:text-gray-400' },
    running: { icon: Loader2, text: 'Running', className: 'text-blue-500', spin: true },
    completed: { icon: CheckCircle, text: 'Completed', className: 'text-green-500' },
    failed: { icon: XCircle, text: 'Failed', className: 'text-red-500' },
    cancelled: { icon: AlertCircle, text: 'Cancelled', className: 'text-amber-500' },
  };

  const { icon: Icon, text, className, spin } = config[status];

  return (
    <div className={clsx('flex items-center gap-1.5 text-xs font-medium', className)}>
      <Icon size={14} className={spin ? 'animate-spin' : ''} />
      <span>{text}</span>
    </div>
  );
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
  mode = 'recording',
  executionStatus,
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

      {/* Execution status indicator (execution mode only) */}
      {mode === 'execution' && executionStatus && (
        <ExecutionStatusIndicator status={executionStatus} />
      )}

      {/* Frame stats display (conditionally shown) */}
      {showStats && (
        <FrameStatsDisplay
          stats={frameStats ?? null}
          targetFps={targetFps}
          debugStats={debugStats ?? null}
        />
      )}

      {/* Replay style toggle (optional, typically recording mode) */}
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
