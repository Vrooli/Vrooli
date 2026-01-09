/**
 * BrowserChrome Component
 *
 * The browser chrome UI (URL bar, frame stats, replay style toggle, settings)
 * that appears above the browser viewport. Used in both recording and execution modes.
 *
 * The Replay Style and Settings buttons are always visible regardless of mode.
 * Frame stats are shown when live streaming (recording mode with showStats=true).
 *
 * Viewport indicator shows:
 * - Current browser viewport dimensions
 * - Warning when actual browser dimensions differ from requested (e.g., session profile override)
 * - Sync status when viewport is being updated
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { Palette, SlidersHorizontal, PanelLeft, Monitor, AlertTriangle, Loader2, ArrowLeft, ArrowRight, RotateCw, History } from 'lucide-react';
import { BrowserUrlBar } from './BrowserUrlBar';
import { FrameStatsDisplay } from './FrameStatsDisplay';
import type { FrameStats } from '../hooks/useFrameStats';
import type { FrameStatsAggregated } from '../hooks/usePerfStats';
import clsx from 'clsx';

/** A single entry in the navigation stack */
export interface NavigationStackEntry {
  url: string;
  title: string;
  timestamp: string;
}

/** Navigation stack data from the driver */
export interface NavigationStackData {
  backStack: NavigationStackEntry[];
  current: NavigationStackEntry | null;
  forwardStack: NavigationStackEntry[];
}

export type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

interface BrowserChromeProps {
  // URL bar
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  onNavigate?: (url: string) => void;
  pageTitle?: string;
  placeholder?: string;

  // Navigation controls (back, forward, refresh)
  onGoBack?: () => void;
  onGoForward?: () => void;
  onRefresh?: () => void;
  canGoBack?: boolean;
  canGoForward?: boolean;

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

  // Navigation history popup (right-click on back/forward buttons)
  /** Fetch navigation stack data for the popup */
  onFetchNavigationStack?: () => Promise<NavigationStackData | null>;
  /** Navigate to a specific number of steps back or forward */
  onNavigateToIndex?: (delta: number) => void;
  /** Open the full history view in session settings */
  onOpenHistorySettings?: () => void;

  // Viewport indicator props (Priority 3)
  /** Current browser viewport dimensions (what we requested from Playwright) */
  browserViewport?: { width: number; height: number } | null;
  /** Actual browser dimensions with source attribution (may differ due to session profile override) */
  actualBrowserViewport?: {
    width: number;
    height: number;
    source?: 'requested' | 'fingerprint' | 'fingerprint_partial' | 'default';
    reason?: string;
  } | null;
  /** Whether there's a mismatch between requested and actual dimensions */
  hasDimensionMismatch?: boolean;
  /** Whether viewport sync is in progress */
  isViewportSyncing?: boolean;
  /** Human-readable explanation of what determined the viewport dimensions */
  viewportReason?: string;

  // Additional class name
  className?: string;
}

export function BrowserChrome({
  previewUrl,
  onPreviewUrlChange,
  onNavigate,
  pageTitle,
  placeholder = 'Search or enter URL',
  onGoBack,
  onGoForward,
  onRefresh,
  canGoBack = false,
  canGoForward = false,
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
  onFetchNavigationStack,
  onNavigateToIndex,
  onOpenHistorySettings,
  browserViewport,
  actualBrowserViewport,
  hasDimensionMismatch = false,
  isViewportSyncing = false,
  viewportReason,
  className,
}: BrowserChromeProps) {
  // In read-only mode, we don't allow navigation
  const handleNavigate = readOnly ? undefined : onNavigate;
  const handleGoBack = readOnly ? undefined : onGoBack;
  const handleGoForward = readOnly ? undefined : onGoForward;
  const handleRefresh = readOnly ? undefined : onRefresh;

  // Default handlers for URL bar when read-only
  const handleUrlChange = readOnly ? () => {} : onPreviewUrlChange;
  const noopNavigate = () => {};

  // Navigation history popup state
  const [historyPopup, setHistoryPopup] = useState<{
    type: 'back' | 'forward';
    position: { x: number; y: number };
    entries: NavigationStackEntry[];
  } | null>(null);
  // Note: historyLoading could be used for a loading indicator, but for now
  // the fetch is fast enough that we skip showing it
  const [, setHistoryLoading] = useState(false);
  const popupRef = useRef<HTMLDivElement>(null);

  // Close popup when clicking outside
  useEffect(() => {
    if (!historyPopup) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (popupRef.current && !popupRef.current.contains(e.target as Node)) {
        setHistoryPopup(null);
      }
    };

    // Small delay to prevent immediate close
    const timeoutId = setTimeout(() => {
      document.addEventListener('click', handleClickOutside);
    }, 0);

    return () => {
      clearTimeout(timeoutId);
      document.removeEventListener('click', handleClickOutside);
    };
  }, [historyPopup]);

  // Handle right-click on back/forward buttons
  const handleHistoryContextMenu = useCallback(async (
    e: React.MouseEvent,
    type: 'back' | 'forward'
  ) => {
    e.preventDefault();
    if (readOnly || !onFetchNavigationStack) return;

    setHistoryLoading(true);
    try {
      const data = await onFetchNavigationStack();
      if (!data) {
        setHistoryPopup(null);
        return;
      }

      const entries = type === 'back' ? data.backStack : data.forwardStack;
      if (entries.length === 0) {
        setHistoryPopup(null);
        return;
      }

      // Position popup below the button
      const rect = (e.target as HTMLElement).closest('button')?.getBoundingClientRect();
      setHistoryPopup({
        type,
        position: {
          x: rect?.left ?? e.clientX,
          y: (rect?.bottom ?? e.clientY) + 4,
        },
        entries,
      });
    } finally {
      setHistoryLoading(false);
    }
  }, [readOnly, onFetchNavigationStack]);

  // Handle navigation from popup
  const handlePopupNavigate = useCallback((index: number) => {
    if (!historyPopup || !onNavigateToIndex) return;

    // For back stack, index 0 is the most recent (delta = -1)
    // For forward stack, index 0 is the next page (delta = +1)
    const delta = historyPopup.type === 'back' ? -(index + 1) : (index + 1);
    onNavigateToIndex(delta);
    setHistoryPopup(null);
  }, [historyPopup, onNavigateToIndex]);

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

      {/* Navigation controls - back, forward, refresh */}
      <div className="flex-shrink-0 flex items-center gap-0.5">
        {/* Back button */}
        <button
          type="button"
          onClick={handleGoBack}
          onContextMenu={(e) => handleHistoryContextMenu(e, 'back')}
          disabled={readOnly || !canGoBack}
          className={clsx(
            'p-1.5 rounded-lg transition-colors',
            canGoBack && !readOnly
              ? 'text-gray-300 hover:text-white hover:bg-gray-700/50'
              : 'text-gray-600 cursor-not-allowed'
          )}
          title={canGoBack ? 'Go back (right-click for history)' : 'No back history'}
          aria-label="Go back"
        >
          <ArrowLeft size={16} />
        </button>

        {/* Forward button */}
        <button
          type="button"
          onClick={handleGoForward}
          onContextMenu={(e) => handleHistoryContextMenu(e, 'forward')}
          disabled={readOnly || !canGoForward}
          className={clsx(
            'p-1.5 rounded-lg transition-colors',
            canGoForward && !readOnly
              ? 'text-gray-300 hover:text-white hover:bg-gray-700/50'
              : 'text-gray-600 cursor-not-allowed'
          )}
          title={canGoForward ? 'Go forward (right-click for history)' : 'No forward history'}
          aria-label="Go forward"
        >
          <ArrowRight size={16} />
        </button>

        {/* Refresh button */}
        <button
          type="button"
          onClick={handleRefresh}
          disabled={readOnly}
          className={clsx(
            'p-1.5 rounded-lg transition-colors',
            readOnly
              ? 'text-gray-600 cursor-not-allowed'
              : 'text-gray-300 hover:text-white hover:bg-gray-700/50'
          )}
          title="Refresh page"
          aria-label="Refresh page"
        >
          <RotateCw size={16} />
        </button>
      </div>

      {/* Browser URL bar */}
      <BrowserUrlBar
        value={previewUrl}
        onChange={handleUrlChange}
        onNavigate={handleNavigate ?? noopNavigate}
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

      {/* Viewport indicator - shows current browser dimensions and sync status */}
      {browserViewport && (
        <div
          className={clsx(
            'flex-shrink-0 flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-mono transition-colors',
            hasDimensionMismatch
              ? 'bg-amber-500/10 border border-amber-500/30 text-amber-400'
              : 'bg-gray-800/50 border border-gray-700/50 text-gray-400',
          )}
          title={
            // Build a detailed tooltip showing viewport attribution
            (() => {
              const displayWidth = hasDimensionMismatch && actualBrowserViewport
                ? actualBrowserViewport.width
                : browserViewport.width;
              const displayHeight = hasDimensionMismatch && actualBrowserViewport
                ? actualBrowserViewport.height
                : browserViewport.height;
              const source = actualBrowserViewport?.source;
              const reason = viewportReason || actualBrowserViewport?.reason;

              if (reason) {
                return `${displayWidth}x${displayHeight}: ${reason}`;
              }
              if (hasDimensionMismatch) {
                return `Browser using ${displayWidth}x${displayHeight} (session profile override)`;
              }
              if (source && source !== 'requested') {
                const sourceLabel = source === 'fingerprint' ? 'browser profile' :
                                   source === 'fingerprint_partial' ? 'partial profile' :
                                   source === 'default' ? 'default' : source;
                return `${displayWidth}x${displayHeight} (from ${sourceLabel})`;
              }
              return `Browser viewport: ${displayWidth}x${displayHeight}`;
            })()
          }
        >
          {isViewportSyncing ? (
            <Loader2 size={12} className="animate-spin" />
          ) : hasDimensionMismatch ? (
            <AlertTriangle size={12} />
          ) : (
            <Monitor size={12} />
          )}
          <span>
            {hasDimensionMismatch && actualBrowserViewport
              ? `${actualBrowserViewport.width}x${actualBrowserViewport.height}`
              : `${browserViewport.width}x${browserViewport.height}`}
          </span>
          {/* Show source attribution inline when there's a mismatch or non-requested source */}
          {hasDimensionMismatch ? (
            <span className="text-[10px] opacity-70">(profile)</span>
          ) : actualBrowserViewport?.source && actualBrowserViewport.source !== 'requested' ? (
            <span className="text-[10px] opacity-70">
              ({actualBrowserViewport.source === 'fingerprint' ? 'profile' :
                actualBrowserViewport.source === 'default' ? 'default' :
                actualBrowserViewport.source})
            </span>
          ) : null}
        </div>
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

      {/* Navigation history popup */}
      {historyPopup && (
        <div
          ref={popupRef}
          className="fixed z-50 min-w-[280px] max-w-[400px] bg-gray-900 border border-gray-700 rounded-lg shadow-xl overflow-hidden"
          style={{
            left: historyPopup.position.x,
            top: historyPopup.position.y,
          }}
        >
          {/* Header */}
          <div className="flex items-center gap-2 px-3 py-2 border-b border-gray-700 bg-gray-800/50">
            <History size={14} className="text-gray-400" />
            <span className="text-xs font-medium text-gray-300">
              {historyPopup.type === 'back' ? 'Back History' : 'Forward History'}
            </span>
            <span className="text-xs text-gray-500 ml-auto">
              {historyPopup.entries.length} {historyPopup.entries.length === 1 ? 'page' : 'pages'}
            </span>
          </div>

          {/* Entries list */}
          <div className="max-h-[300px] overflow-y-auto">
            {historyPopup.entries.map((entry, index) => (
              <button
                key={`${entry.url}-${index}`}
                onClick={() => handlePopupNavigate(index)}
                className="w-full text-left px-3 py-2 hover:bg-gray-800 border-b border-gray-800 last:border-b-0 transition-colors group"
              >
                <div className="text-sm text-gray-200 truncate group-hover:text-white">
                  {entry.title || '(No title)'}
                </div>
                <div className="text-xs text-gray-500 truncate mt-0.5 font-mono">
                  {entry.url}
                </div>
              </button>
            ))}
          </div>

          {/* Footer with "Show All History" link */}
          {onOpenHistorySettings && (
            <div className="px-3 py-2 border-t border-gray-700 bg-gray-800/30">
              <button
                onClick={() => {
                  setHistoryPopup(null);
                  onOpenHistorySettings();
                }}
                className="text-xs text-blue-400 hover:text-blue-300 hover:underline"
              >
                Show All History...
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
