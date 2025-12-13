/**
 * FloatingActionBar Component
 *
 * A floating, draggable action bar for Record Mode that provides:
 * - Connection status indicator (green/blue/yellow/red)
 * - Timeline/Live view toggle
 * - URL bar expand toggle
 * - Performance stats (if enabled)
 * - Stream settings button
 * - End session button
 *
 * The bar auto-hides to the nearest edge when mouse moves away (if enabled).
 */

import { useState, useCallback } from 'react';
import { FloatingPanel } from '@/components/FloatingPanel';
import { BrowserUrlBar } from './BrowserUrlBar';
import { StreamSettings, type StreamSettingsValues, type StreamPreset } from './StreamSettings';
import { FrameStatsDisplay } from './FrameStatsDisplay';
import type { FrameStats } from '../hooks/useFrameStats';
import type { FrameStatsAggregated } from '../hooks/usePerfStats';

/** Connection/recording status */
export type ConnectionStatus = 'recording' | 'connected' | 'disconnected' | 'error';

interface FloatingActionBarProps {
  /** Current connection/recording status */
  status: ConnectionStatus;
  /** Number of recorded actions */
  actionCount: number;
  /** Whether recording is active */
  isRecording: boolean;
  /** Current view in sidebar ('timeline' | 'preview' | null for no sidebar) */
  sidebarView: 'timeline' | 'preview' | null;
  /** Callback to change sidebar view */
  onSidebarViewChange: (view: 'timeline' | 'preview' | null) => void;
  /** Current URL value */
  previewUrl: string;
  /** Callback when URL changes */
  onPreviewUrlChange: (url: string) => void;
  /** Callback when user navigates to URL */
  onNavigate: (url: string) => void;
  /** Callback for refresh */
  onRefresh: () => void;
  /** Session ID for stream settings */
  sessionId?: string | null;
  /** Stream settings preset */
  streamPreset: StreamPreset;
  /** Callback when stream preset changes */
  onStreamPresetChange: (preset: StreamPreset) => void;
  /** Callback when stream settings values change */
  onStreamSettingsChange: (settings: StreamSettingsValues) => void;
  /** Whether to show performance stats */
  showStats: boolean;
  /** Callback when show stats changes */
  onShowStatsChange: (show: boolean) => void;
  /** Frame stats for display */
  frameStats: FrameStats | null;
  /** Target FPS for stats display */
  targetFps: number;
  /** Debug stats from server */
  debugStats?: FrameStatsAggregated | null;
  /** Callback to end session */
  onEndSession: () => void;
  /** Whether auto-hide is enabled */
  autoHide?: boolean;
  /** Callback when auto-hide changes */
  onAutoHideChange?: (autoHide: boolean) => void;
  /** Page title for URL bar history */
  pageTitle?: string;
}

/** Status indicator colors */
const STATUS_COLORS: Record<ConnectionStatus, { bg: string; ring: string; title: string }> = {
  recording: {
    bg: 'bg-green-500',
    ring: 'ring-green-400/50',
    title: 'Recording - Connected to Playwright',
  },
  connected: {
    bg: 'bg-blue-500',
    ring: 'ring-blue-400/50',
    title: 'Connected - Not recording',
  },
  disconnected: {
    bg: 'bg-yellow-500',
    ring: 'ring-yellow-400/50',
    title: 'Disconnected - Reconnecting...',
  },
  error: {
    bg: 'bg-red-500',
    ring: 'ring-red-400/50',
    title: 'Error - Recording may not be saved',
  },
};

export function FloatingActionBar({
  status,
  actionCount,
  isRecording,
  sidebarView,
  onSidebarViewChange,
  previewUrl,
  onPreviewUrlChange,
  onNavigate,
  onRefresh,
  sessionId,
  streamPreset,
  onStreamPresetChange,
  onStreamSettingsChange,
  showStats,
  onShowStatsChange,
  frameStats,
  targetFps,
  debugStats,
  onEndSession,
  autoHide = false,
  onAutoHideChange,
  pageTitle,
}: FloatingActionBarProps) {
  const [isUrlExpanded, setIsUrlExpanded] = useState(false);
  const statusColor = STATUS_COLORS[status];

  // Toggle sidebar view
  const handleTimelineToggle = useCallback(() => {
    if (sidebarView === 'timeline') {
      onSidebarViewChange(null);
    } else {
      onSidebarViewChange('timeline');
    }
  }, [sidebarView, onSidebarViewChange]);

  return (
    <FloatingPanel
      id="record-mode-action-bar"
      defaultPosition={{ x: window.innerWidth / 2 - 200, y: 16 }}
      autoHide={autoHide}
      hiddenVisiblePx={8}
      showProximityPx={50}
      showDelayMs={200}
      zIndex={100}
      className="select-none"
    >
      <div className="flex flex-col bg-white/95 dark:bg-gray-900/95 backdrop-blur-sm border border-gray-200 dark:border-gray-700 rounded-xl shadow-lg">
        {/* Main action row */}
        <div className="flex items-center gap-2 px-3 py-2">
          {/* Status indicator */}
          <div
            className={`relative w-3 h-3 rounded-full ${statusColor.bg} ${isRecording ? 'animate-pulse' : ''}`}
            title={statusColor.title}
          >
            {isRecording && (
              <span className={`absolute inset-0 rounded-full ${statusColor.bg} animate-ping opacity-75`} />
            )}
          </div>

          {/* Grip handle for dragging */}
          <div className="text-gray-400 dark:text-gray-500 cursor-grab active:cursor-grabbing">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path d="M7 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 2zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 14zm6-8a2 2 0 1 0-.001-4.001A2 2 0 0 0 13 6zm0 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 14z" />
            </svg>
          </div>

          {/* Divider */}
          <div className="w-px h-5 bg-gray-200 dark:bg-gray-700" />

          {/* Timeline toggle button */}
          <button
            type="button"
            onClick={handleTimelineToggle}
            className={`relative p-1.5 rounded-lg transition-colors ${
              sidebarView === 'timeline'
                ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30'
                : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800'
            }`}
            title={sidebarView === 'timeline' ? 'Hide timeline' : 'Show timeline'}
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h10M4 18h7" />
            </svg>
            {/* Action count badge */}
            {actionCount > 0 && (
              <span className="absolute -top-1 -right-1 flex items-center justify-center min-w-[18px] h-[18px] px-1 text-[10px] font-semibold text-white bg-blue-500 rounded-full">
                {actionCount > 99 ? '99+' : actionCount}
              </span>
            )}
          </button>

          {/* URL bar expand toggle */}
          <button
            type="button"
            onClick={() => setIsUrlExpanded(!isUrlExpanded)}
            className={`p-1.5 rounded-lg transition-colors ${
              isUrlExpanded
                ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30'
                : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800'
            }`}
            title={isUrlExpanded ? 'Hide URL bar' : 'Show URL bar'}
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </button>

          {/* Performance stats (if enabled) */}
          {showStats && frameStats && (
            <>
              <div className="w-px h-5 bg-gray-200 dark:bg-gray-700" />
              <FrameStatsDisplay
                stats={frameStats}
                targetFps={targetFps}
                debugStats={debugStats}
              />
            </>
          )}

          {/* Divider */}
          <div className="w-px h-5 bg-gray-200 dark:bg-gray-700" />

          {/* Stream settings */}
          <StreamSettings
            sessionId={sessionId}
            preset={streamPreset as 'fast' | 'balanced' | 'sharp' | 'hidpi' | 'custom'}
            onPresetChange={onStreamPresetChange}
            onSettingsChange={onStreamSettingsChange}
            showStats={showStats}
            onShowStatsChange={onShowStatsChange}
            autoHide={autoHide}
            onAutoHideChange={onAutoHideChange}
          />

          {/* End session button */}
          <button
            type="button"
            onClick={onEndSession}
            className="p-1.5 text-gray-500 dark:text-gray-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
            title="End recording session"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
            </svg>
          </button>
        </div>

        {/* Expanded URL bar row */}
        {isUrlExpanded && (
          <div className="px-3 pb-2 min-w-[320px]">
            <BrowserUrlBar
              value={previewUrl}
              onChange={onPreviewUrlChange}
              onNavigate={onNavigate}
              onRefresh={onRefresh}
              placeholder="Search or enter URL"
              pageTitle={pageTitle}
            />
          </div>
        )}
      </div>
    </FloatingPanel>
  );
}
