import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { RecordedAction } from './types';
import { PlaywrightView, type FrameStats, type PageMetadata } from './components/PlaywrightView';
import { StreamSettings, useStreamSettings, type StreamSettingsValues } from './components/StreamSettings';
import { FrameStatsDisplay } from './components/FrameStatsDisplay';
import { BrowserUrlBar } from './components/BrowserUrlBar';
import { usePerfStats } from './hooks/usePerfStats';

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

  const [liveRefreshToken, setLiveRefreshToken] = useState(0);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const lastReportedViewportRef = useRef<{ width: number; height: number } | null>(null);
  // Track viewport for passing to PlaywrightView (for coordinate mapping)
  const [currentViewport, setCurrentViewport] = useState<{ width: number; height: number } | null>(null);

  // Stream settings management
  const { preset, settings: streamSettings, showStats, setPreset, setShowStats } = useStreamSettings();

  // Debug performance stats from server (enabled when showStats is on)
  const { stats: perfStats } = usePerfStats(sessionId ?? null, showStats);

  // Frame statistics from PlaywrightView
  const [frameStats, setFrameStats] = useState<FrameStats | null>(null);
  const handleStatsUpdate = useCallback((stats: FrameStats) => {
    setFrameStats(stats);
  }, []);

  // Page metadata (title, url) from PlaywrightView - used for history display
  const [pageTitle, setPageTitle] = useState<string>('');
  const handlePageMetadataChange = useCallback((metadata: PageMetadata) => {
    setPageTitle(metadata.title);
  }, []);

  // Notify parent when stream settings change (for session creation)
  useEffect(() => {
    onStreamSettingsChange?.(streamSettings);
  }, [streamSettings, onStreamSettingsChange]);

  // Handle URL navigation from BrowserUrlBar
  const handleUrlNavigate = useCallback(
    (url: string) => {
      onPreviewUrlChange(url);
    },
    [onPreviewUrlChange]
  );

  // Handle refresh
  const handleRefresh = useCallback(() => {
    setLiveRefreshToken((t) => t + 1);
  }, []);

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
      {/* Header with browser-like URL bar and settings */}
      <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-2.5 flex items-center gap-2">
        {/* Browser URL bar */}
        <BrowserUrlBar
          value={previewUrl}
          onChange={onPreviewUrlChange}
          onNavigate={handleUrlNavigate}
          onRefresh={handleRefresh}
          placeholder={lastUrl || 'Search or enter URL'}
          pageTitle={pageTitle}
        />

        {/* Frame stats display (conditionally shown) */}
        {showStats && (
          <FrameStatsDisplay
            stats={frameStats}
            targetFps={streamSettings.fps}
            debugStats={perfStats}
          />
        )}

        {/* Stream quality settings */}
        <StreamSettings
          sessionId={sessionId}
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
              onPageMetadataChange={handlePageMetadataChange}
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
