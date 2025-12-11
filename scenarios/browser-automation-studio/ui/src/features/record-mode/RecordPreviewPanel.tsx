import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { RecordedAction } from './types';
import { PlaywrightView, type FrameStats } from './components/PlaywrightView';
import { StreamSettings, useStreamSettings, type StreamSettingsValues } from './components/StreamSettings';
import { FrameStatsDisplay } from './components/FrameStatsDisplay';

interface RecordPreviewPanelProps {
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  actions: RecordedAction[];
  sessionId?: string | null;
  onViewportChange?: (size: { width: number; height: number }) => void;
  /** Callback when stream settings change (for session creation) */
  onStreamSettingsChange?: (settings: StreamSettingsValues) => void;
}

export function RecordPreviewPanel({
  previewUrl,
  onPreviewUrlChange,
  actions,
  sessionId,
  onViewportChange,
  onStreamSettingsChange,
}: RecordPreviewPanelProps) {
  const lastUrl = useMemo(() => {
    if (actions.length === 0) return '';
    return actions[actions.length - 1]?.url ?? '';
  }, [actions]);

  const effectiveUrl = previewUrl || lastUrl;

  const [urlInput, setUrlInput] = useState(previewUrl);
  const [liveRefreshToken, setLiveRefreshToken] = useState(0);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const lastReportedViewportRef = useRef<{ width: number; height: number } | null>(null);
  // Track viewport for passing to PlaywrightView (for coordinate mapping)
  const [currentViewport, setCurrentViewport] = useState<{ width: number; height: number } | null>(null);

  // Stream settings management
  const { preset, settings: streamSettings, showStats, setPreset, setShowStats } = useStreamSettings();

  // Frame statistics from PlaywrightView
  const [frameStats, setFrameStats] = useState<FrameStats | null>(null);
  const handleStatsUpdate = useCallback((stats: FrameStats) => {
    setFrameStats(stats);
  }, []);

  // Notify parent when stream settings change (for session creation)
  useEffect(() => {
    onStreamSettingsChange?.(streamSettings);
  }, [streamSettings, onStreamSettingsChange]);

  // Keep input in sync with upstream changes
  useEffect(() => {
    setUrlInput(previewUrl);
  }, [previewUrl]);

  // Debounce URL changes before notifying parent
  useEffect(() => {
    const debounce = setTimeout(() => {
      if (urlInput !== previewUrl) {
        onPreviewUrlChange(urlInput);
      }
    }, 400);
    return () => clearTimeout(debounce);
  }, [urlInput, previewUrl, onPreviewUrlChange]);

  // Observe preview container size and notify parent for viewport sizing.
  useEffect(() => {
    const node = previewContainerRef.current;
    if (!node) return;

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      const width = entry.contentRect.width;
      const height = entry.contentRect.height;
      if (width <= 0 || height <= 0) return;

      const size = { width, height };
      const prev = lastReportedViewportRef.current;
      if (prev && Math.abs(prev.width - width) < 1 && Math.abs(prev.height - height) < 1) {
        return;
      }
      lastReportedViewportRef.current = size;
      setCurrentViewport(size);
      if (onViewportChange) {
        onViewportChange(size);
      }
    });

    observer.observe(node);
    return () => observer.disconnect();
  }, [previewContainerRef, onViewportChange]);

  return (
    <div className="flex flex-col h-full">
      {/* Header with URL input and settings */}
      <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-3 flex items-center gap-2">
        <div className="flex-1 relative">
          <input
            className="w-full pl-4 pr-10 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-full bg-white dark:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder={lastUrl || 'Enter a URL to preview'}
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
          />
          <button
            className="absolute right-1 top-1/2 -translate-y-1/2 p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-full transition-colors"
            onClick={() => setLiveRefreshToken((t) => t + 1)}
            title="Refresh page"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>

        {/* Frame stats display (conditionally shown) */}
        {showStats && <FrameStatsDisplay stats={frameStats} targetFps={streamSettings.fps} />}

        {/* Stream quality settings */}
        <StreamSettings
          hasActiveSession={!!sessionId}
          preset={preset}
          onPresetChange={setPreset}
          onSettingsChange={(newSettings) => onStreamSettingsChange?.(newSettings)}
          showStats={showStats}
          onShowStatsChange={setShowStats}
        />
      </div>
      <div className="flex-1 overflow-hidden" ref={previewContainerRef}>
        {sessionId ? (
          effectiveUrl ? (
            <PlaywrightView
              sessionId={sessionId}
              refreshToken={liveRefreshToken}
              viewport={currentViewport ?? undefined}
              quality={streamSettings.quality}
              fps={streamSettings.fps}
              onStatsUpdate={handleStatsUpdate}
            />
          ) : (
            <EmptyState title="Add a URL to load the live preview" subtitle="Live preview renders the actual Playwright session." />
          )
        ) : (
          <EmptyState title="Start a recording session" subtitle="Create or resume a recording session to view the live browser." />
        )}
      </div>
    </div>
  );
}

function EmptyState({ title, subtitle, actionLabel, onAction }: { title: string; subtitle?: string; actionLabel?: string; onAction?: () => void }) {
  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-6">
      <svg className="w-10 h-10 text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6v6l4 2" />
      </svg>
      <p className="text-sm font-medium text-gray-800 dark:text-gray-200">{title}</p>
      {subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
      {actionLabel && onAction && (
        <button
          className="mt-3 px-3 py-1.5 text-xs font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600"
          onClick={onAction}
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}
